package file

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	archiveLogDirName = "Logs"
)

type LogFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
}

func NewLogFileRepository(workPath string, fs volumeFs) *LogFileRepository {
	return &LogFileRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, archiveLogDirName, fs),
	}
}

func (l *LogFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.PodLog) error {
	return create(ctx, id, dataStream, l.createPodLog, l.Delete, l.finishCollection, nil)
}

func (l *LogFileRepository) createPodLog(ctx context.Context, id domain.SupportArchiveID, data *domain.PodLog) error {
	logger := log.FromContext(ctx).WithName("LogFileRepository.createPodLog")
	filePath := filepath.Join(l.workPath, id.Namespace, id.Name, archiveLogDirName, fmt.Sprintf("%s%s", data.PodName, ".log"))
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
