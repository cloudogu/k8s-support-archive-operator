package file

import (
	"context"
	"fmt"
	"os"
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
	logFiles   map[domain.SupportArchiveID]closableRWFile
}

func NewLogFileRepository(workPath string, fs volumeFs) *LogFileRepository {
	return &LogFileRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, archiveLogDirName, fs),
		logFiles:     make(map[domain.SupportArchiveID]closableRWFile),
	}
}

func (l *LogFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.LogLine) error {
	return create(ctx, id, dataStream, l.createPodLog, l.Delete, l.finishCollection, l.close)
}

func (l *LogFileRepository) createPodLog(ctx context.Context, id domain.SupportArchiveID, data *domain.LogLine) error {
	logger := log.FromContext(ctx).WithName("LogFileRepository.createPodLog")

	if l.logFiles[id] == nil {
		filePath := filepath.Join(l.workPath, id.Namespace, id.Name, archiveLogDirName, fmt.Sprintf("%s%s", "logs", ".log"))
		err := l.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
		}
		l.logFiles[id], err = l.filesystem.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0666))
		if err != nil {
			return fmt.Errorf("failed to create log file %s: %w", filePath, err)
		}
		_, err = l.logFiles[id].Write([]byte("LOGS\n"))
		if err != nil {
			return fmt.Errorf("failed to write header to log file %s: %w", filePath, err)
		}
		logger.Info(fmt.Sprintf("Created log file %s", filePath))
	}

	_, err := l.logFiles[id].Write([]byte(fmt.Sprintf("%s%s", data.Value, "\n")))
	if err != nil {
		return fmt.Errorf("failed to write data to log file %s: %w", id, err)
	}

	return nil
}

func (l *LogFileRepository) close(_ context.Context, id domain.SupportArchiveID) error {
	if l.logFiles == nil || l.logFiles[id] == nil {
		return nil
	}

	err := l.logFiles[id].Close()
	if err != nil {
		return fmt.Errorf("failed to close log file %s: %w", id, err)
	}

	return nil
}
