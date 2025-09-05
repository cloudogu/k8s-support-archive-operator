package file

import (
	"context"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

const (
	archiveEventsDirName = "k8/events"
)

type EventsFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
}

func (repo *EventsFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, data <-chan *domain.Events) error {
	return create(ctx, id, data, repo.writeToFile, repo.Delete, repo.finishCollection, nil)
}

func (repo *EventsFileRepository) writeToFile(ctx context.Context, id domain.SupportArchiveID, events *domain.Events) error {
	//directory := filepath.Join(v.workPath, id.Namespace, id.Name, archiveEventsDirName, events.Namespace, events.Kind)
	//filePath := fmt.Sprintf("%s.logs", directory)
	return nil
}
