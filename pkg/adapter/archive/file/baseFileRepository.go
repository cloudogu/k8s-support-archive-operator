package file

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	stateFileName = ".done"
)

type createFn[DATATYPE domain.CollectorUnionDataType] = func(context.Context, domain.SupportArchiveID, *DATATYPE) error
type deleteFn = func(context.Context, domain.SupportArchiveID) error
type finishFn = func(context.Context, domain.SupportArchiveID) error
type closeFn = func(context.Context, domain.SupportArchiveID) error

type baseFileRepository struct {
	workPath     string
	collectorDir string
	filesystem   volumeFs
}

func NewBaseFileRepository(workPath string, collectorDir string, filesystem volumeFs) *baseFileRepository {
	return &baseFileRepository{
		workPath:     workPath,
		collectorDir: collectorDir,
		filesystem:   filesystem,
	}
}

func (l *baseFileRepository) finishCollection(ctx context.Context, id domain.SupportArchiveID) error {
	logger := log.FromContext(ctx).WithName("baseFileRepository.finishCollection")
	stateFilePath := getStateFilePath(l.workPath, id, l.collectorDir)

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

func (l *baseFileRepository) IsCollected(_ context.Context, id domain.SupportArchiveID) (bool, error) {
	stateFilePath := getStateFilePath(l.workPath, id, l.collectorDir)
	_, err := l.filesystem.Stat(stateFilePath)

	if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check if file %s exists: %w", stateFilePath, err)
	}
	return true, nil
}

func (l *baseFileRepository) Delete(_ context.Context, id domain.SupportArchiveID) error {
	dirPath := filepath.Join(l.workPath, id.Namespace, id.Name, l.collectorDir)
	err := l.filesystem.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("failed to remove %s directory %s: %w", l.collectorDir, dirPath, err)
	}

	return nil
}

// create receives elements from the stream and calls the concrete createFn for each element.
// If an error occurs, create executes deleteFn to tidy up.
// If the stream is closed, create will end and call the finishFn.
func create[DATATYPE domain.CollectorUnionDataType](ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *DATATYPE, createFn createFn[DATATYPE], deleteFn deleteFn, finishFn finishFn, closeFn closeFn) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-dataStream:
			if ok {
				err := createFn(ctx, id, data)
				if err != nil {
					return handleCreateErr(ctx, id, err, closeFn, deleteFn)
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

func handleCreateErr(ctx context.Context, id domain.SupportArchiveID, err error, closeFn closeFn, deleteFn deleteFn) error {
	if err != nil {
		if closeFn != nil {
			closeErr := closeFn(ctx, id)
			if closeErr != nil {
				err = fmt.Errorf("error during close function: %w", closeErr)
			}
		}

		err = fmt.Errorf("error creating element from data stream: %w", err)
		cleanErr := deleteFn(ctx, id)
		if cleanErr != nil {
			return errors.Join(err, fmt.Errorf("failed to clean up data after error: %w", cleanErr))
		}

		return err
	}
	return nil
}

func (l *baseFileRepository) Stream(ctx context.Context, id domain.SupportArchiveID, stream *domain.Stream) error {
	dirPath := filepath.Join(l.workPath, id.Namespace, id.Name, l.collectorDir)

	err := l.filesystem.WalkDir(dirPath, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		writeSaveToChannel(ctx, domain.StreamData{
			ID:                info.Name(),
			StreamConstructor: l.createStreamConstructor(path),
		}, stream.Data)

		return nil
	})

	if err != nil {
		return err
	}

	close(stream.Data)

	return nil
}

func (l *baseFileRepository) createStreamConstructor(path string) domain.StreamConstructor {
	return func() (io.Reader, domain.CloseStreamFunc, error) {
		open, openErr := l.filesystem.Open(path)
		if openErr != nil {
			return nil, nil, fmt.Errorf("failed to open file %s: %w", path, openErr)
		}

		return bufio.NewReader(open), open.Close, nil
	}
}

func writeSaveToChannel[T any](ctx context.Context, data T, dataChannel chan<- T) {
	select {
	case <-ctx.Done():
		return
	case dataChannel <- data:
		return
	}
}

func getStateFilePath(workPath string, id domain.SupportArchiveID, collectorDir string) string {
	return filepath.Join(workPath, id.Namespace, id.Name, collectorDir, stateFileName)
}
