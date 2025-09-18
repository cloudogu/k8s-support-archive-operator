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
	archiveEventsDirName = "Events"
)

type EventRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
	eventFiles map[domain.SupportArchiveID]closableRWFile
}

func NewEventRepository(workPath string, fs volumeFs) *EventRepository {
	return &EventRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, archiveEventsDirName, fs),
		eventFiles:   make(map[domain.SupportArchiveID]closableRWFile),
	}
}

func (l *EventRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.LogLine) error {
	return create(ctx, id, dataStream, l.createEventLog, l.Delete, l.finishCollection, l.close)
}

func (l *EventRepository) createEventLog(ctx context.Context, id domain.SupportArchiveID, data *domain.LogLine) error {
	logger := log.FromContext(ctx).WithName("EventRepository.createEventLog")

	if l.eventFiles[id] == nil {
		filePath := filepath.Join(l.workPath, id.Namespace, id.Name, archiveEventsDirName, fmt.Sprintf("%s%s", "events", ".log"))
		err := l.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
		}
		l.eventFiles[id], err = l.filesystem.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0666))
		if err != nil {
			return fmt.Errorf("failed to create event file %s: %w", filePath, err)
		}
		_, err = l.eventFiles[id].Write([]byte("LOGS\n"))
		if err != nil {
			return fmt.Errorf("failed to write header to event file %s: %w", filePath, err)
		}
		logger.Info(fmt.Sprintf("Created event file %s", filePath))
	}

	_, err := l.eventFiles[id].Write([]byte(fmt.Sprintf("%s%s", data.Value, "\n")))
	if err != nil {
		return fmt.Errorf("failed to write data to log file %s: %w", id, err)
	}

	return nil
}

func (l *EventRepository) close(_ context.Context, id domain.SupportArchiveID) error {
	if l.eventFiles == nil || l.eventFiles[id] == nil {
		return nil
	}

	err := l.eventFiles[id].Close()
	if err != nil {
		return fmt.Errorf("failed to close event file %s: %w", id, err)
	}

	return nil
}
