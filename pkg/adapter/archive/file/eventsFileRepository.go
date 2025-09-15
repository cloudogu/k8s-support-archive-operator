package file

import (
	"context"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

const (
	archiveEventsDirName = "Events"
)

type EventsFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
}

func NewEventsRepository(workPath string, fs volumeFs) *EventsFileRepository {
	return &EventsFileRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, archiveEventsDirName, fs),
	}
}

func (repo *EventsFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, data <-chan *domain.EventSet) error {
	return create(ctx, id, data, repo.writeToFile, repo.Delete, repo.finishCollection, nil)
}

func (repo *EventsFileRepository) writeToFile(ctx context.Context, id domain.SupportArchiveID, events *domain.EventSet) error {
	// encode event as string and remove newlines
	//directory := filepath.Join(v.workPath, id.Namespace, id.Name, archiveEventsDirName, events.Namespace, events.Kind)
	//filePath := fmt.Sprintf("%s.logs", directory)
	return nil
}
