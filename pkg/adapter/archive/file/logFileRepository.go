package file

import (
	"bufio"
	"context"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	archiveDirName = "logs"
)

type LogFileRepository struct {
	*baseFileRepository
	workPath   string
	filesystem volumeFs
}

func NewLogFileRepository(workPath string, fs volumeFs, repository *baseFileRepository) *LogFileRepository {
	return &LogFileRepository{
		workPath:           workPath,
		filesystem:         fs,
		baseFileRepository: repository,
	}
}

func (l *LogFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.PodLog) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-dataStream:
			if ok {
				err := l.createPodLog(ctx, id, data)
				if err != nil {
					return fmt.Errorf("error creating pod log: %w", err)
				}
			} else {
				err := l.FinishCollection(ctx, id)
				if err != nil {
					return fmt.Errorf("error finishing collection: %w", err)
				}
				return nil
			}
		}
	}
}

func (l *LogFileRepository) createPodLog(ctx context.Context, id domain.SupportArchiveID, data *domain.PodLog) error {
	logger := log.FromContext(ctx).WithName("LogFileRepository.createPodLog")
	filePath := filepath.Join(l.workPath, id.Namespace, id.Name, archiveDirName, fmt.Sprintf("%s%s", data.PodName, ".log"))
	err := l.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
	}

	open, err := l.filesystem.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filePath, err)
	}
	defer func() {
		closeErr := open.Close()
		if closeErr != nil {
			logger.Error(closeErr, "failed to close file")
		}
	}()

	for _, logLine := range data.Entries {
		_, writeErr := open.Write([]byte(logLine))
		if writeErr != nil {
			return fmt.Errorf("failed to write to file %s: %w", filePath, writeErr)
		}
	}
	logger.Info(fmt.Sprintf("Created file %s", filePath))

	return nil
}

func (l *LogFileRepository) Stream(ctx context.Context, id domain.SupportArchiveID, stream domain.Stream) error {
	logger := log.FromContext(ctx).WithName("LogFileRepository.Stream")

	dirPath := filepath.Join(l.workPath, id.Namespace, id.Name, archiveDirName)
	dir, err := l.filesystem.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	var filesToClose []closableRWFile
	defer func() {
		for _, file := range filesToClose {
			closeErr := file.Close()
			if closeErr != nil {
				logger.Error(closeErr, "failed to close file")
			}
		}
	}()

	for _, file := range dir {
		filePath := filepath.Join(dirPath, file.Name())
		open, openErr := l.filesystem.Open(filePath)
		if openErr != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, openErr)
		}
		filesToClose = append(filesToClose, open)

		writeSaveToChannel(ctx, domain.StreamData{
			ID:     file.Name(),
			Reader: bufio.NewReader(open),
		}, stream.Data)
	}
	close(stream.Data)

	return nil
}

func writeSaveToChannel[T any](ctx context.Context, data T, dataChannel chan<- T) {
	select {
	case <-ctx.Done():
		return
	case dataChannel <- data:
		return
	}
}

func (l *LogFileRepository) Delete(_ context.Context, id domain.SupportArchiveID) error {
	dirPath := filepath.Join(l.workPath, id.Namespace, id.Name, archiveDirName)
	err := l.filesystem.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("failed to remove logs directory %s: %w", dirPath, err)
	}

	return nil
}
