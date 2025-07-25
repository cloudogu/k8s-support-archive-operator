package file

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"io/fs"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	stateFileName = ".done"
)

type createFn[DATATYPE domain.CollectorUnionDataType] = func(context.Context, domain.SupportArchiveID, *DATATYPE) error
type deleteFn = func(context.Context, domain.SupportArchiveID) error
type finishFn = func(context.Context, domain.SupportArchiveID) error

type baseFileRepository struct {
	workPath   string
	filesystem volumeFs
}

func NewBaseFileRepository(workPath string, filesystem volumeFs) *baseFileRepository {
	return &baseFileRepository{
		workPath:   workPath,
		filesystem: filesystem,
	}
}

func (l *baseFileRepository) FinishCollection(ctx context.Context, id domain.SupportArchiveID, collectorDir string) error {
	logger := log.FromContext(ctx).WithName("baseFileRepository.FinishCollection")
	stateFilePath := getStateFilePath(l.workPath, id, collectorDir)

	err := l.filesystem.MkdirAll(filepath.Dir(stateFilePath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	stateFile, err := l.filesystem.Create(stateFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", stateFilePath, err)
	}
	defer func() {
		createErr := stateFile.Close()
		if err != nil {
			logger.Error(createErr, fmt.Sprintf("failed to close file %s", id))
		}
	}()
	_, err = stateFile.Write([]byte("done"))
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", stateFilePath, err)
	}

	return nil
}

func (l *baseFileRepository) IsCollected(_ context.Context, id domain.SupportArchiveID, collectorDir string) (bool, error) {
	stateFilePath := getStateFilePath(l.workPath, id, collectorDir)
	_, err := l.filesystem.Stat(stateFilePath)

	if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check if file %s exists: %w", stateFilePath, err)
	}
	return true, nil
}

func (l *baseFileRepository) Delete(_ context.Context, id domain.SupportArchiveID, collectorDir string) error {
	dirPath := filepath.Join(l.workPath, id.Namespace, id.Name, collectorDir)
	err := l.filesystem.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("failed to remove %s directory %s: %w", collectorDir, dirPath, err)
	}

	return nil
}

// create queries elements from the stream and calls the concrete createFn for each element.
// If an error occurs, create executes deleteFn to tidy up.
// If the stream is closed, create will end and call the finishFn.
func create[DATATYPE domain.CollectorUnionDataType](ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *DATATYPE, createFn createFn[DATATYPE], deleteFn deleteFn, finishFn finishFn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-dataStream:
			if ok {
				err := createFn(ctx, id, data)
				if err != nil {
					err = fmt.Errorf("error creating element from data stream: %w", err)
					cleanErr := deleteFn(ctx, id)
					if cleanErr != nil {
						return errors.Join(err, fmt.Errorf("failed to clean up data after error: %w", cleanErr))
					}
					return err
				}
			} else {
				err := finishFn(ctx, id)
				if err != nil {
					return fmt.Errorf("error finishing collection: %w", err)
				}
				return nil
			}
		}
	}
}

func (l *baseFileRepository) stream(ctx context.Context, id domain.SupportArchiveID, directory string, stream *domain.Stream) (func() error, error) {
	dirPath := filepath.Join(l.workPath, id.Namespace, id.Name, directory)
	var filesToClose []closableRWFile

	err := l.filesystem.WalkDir(dirPath, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := l.streamFile(ctx, path, info.Name(), stream)
		if err != nil {
			return err
		}

		if file != nil {
			filesToClose = append(filesToClose, file)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	close(stream.Data)

	return getFileFinalizerFunc(filesToClose), nil
}

func (l *baseFileRepository) streamFile(ctx context.Context, path, filename string, stream *domain.Stream) (closableRWFile, error) {
	open, openErr := l.filesystem.Open(path)
	if openErr != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, openErr)
	}

	writeSaveToChannel(ctx, domain.StreamData{
		ID:     filename,
		Reader: bufio.NewReader(open),
	}, stream.Data)

	return open, nil
}

func writeSaveToChannel[T any](ctx context.Context, data T, dataChannel chan<- T) {
	select {
	case <-ctx.Done():
		return
	case dataChannel <- data:
		return
	}
}

func getFileFinalizerFunc(filesToClose []closableRWFile) func() error {
	return func() error {
		var multiErr []error
		for _, file := range filesToClose {
			closeErr := file.Close()
			if closeErr != nil {
				multiErr = append(multiErr, fmt.Errorf("failed to close file: %w", closeErr))
			}
		}
		return errors.Join(multiErr...)
	}
}

func getStateFilePath(workPath string, id domain.SupportArchiveID, collectorDir string) string {
	return filepath.Join(workPath, id.Namespace, id.Name, collectorDir, stateFileName)
}
