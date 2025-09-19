package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type SingleLogFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
	files      map[domain.SupportArchiveID]closableRWFile
	dirName    string
}

func NewSingleLogFileRepository(workPath, dirname string, fs volumeFs) *SingleLogFileRepository {
	return &SingleLogFileRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, dirname, fs),
		files:        make(map[domain.SupportArchiveID]closableRWFile),
		dirName:      dirname,
	}
}

func (l *SingleLogFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.LogLine) error {
	return create(ctx, id, dataStream, l.createLog, l.Delete, l.finishCollection, l.close)
}

func (l *SingleLogFileRepository) createLog(ctx context.Context, id domain.SupportArchiveID, data *domain.LogLine) error {
	logger := log.FromContext(ctx).WithName("LogFileRepository.createLog")

	if l.files[id] == nil {
		filePath := filepath.Join(l.workPath, id.Namespace, id.Name, l.dirName, fmt.Sprintf("%s%s", "logs", ".log"))
		err := l.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
		}
		l.files[id], err = l.filesystem.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0666))
		if err != nil {
			return fmt.Errorf("failed to create log file %s: %w", filePath, err)
		}
		_, err = l.files[id].Write([]byte("LOGS\n"))
		if err != nil {
			return fmt.Errorf("failed to write header to log file %s: %w", filePath, err)
		}
		logger.Info(fmt.Sprintf("Created log file %s", filePath))
	}

	_, err := l.files[id].Write([]byte(fmt.Sprintf("%s%s", data.Value, "\n")))
	if err != nil {
		return fmt.Errorf("failed to write data to log file %s: %w", id, err)
	}

	return nil
}

func (l *SingleLogFileRepository) close(_ context.Context, id domain.SupportArchiveID) error {
	if l.files == nil || l.files[id] == nil {
		return nil
	}

	err := l.files[id].Close()
	if err != nil {
		return fmt.Errorf("failed to close log file %s: %w", id, err)
	}

	return nil
}
