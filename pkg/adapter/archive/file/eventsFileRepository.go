package file

import (
	"context"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

type EventsFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
}

func (repo *EventsFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, data <-chan *domain.Events) error {
	return create(ctx, id, data, repo.writeToFile, repo.Delete, repo.FinishCollection)
}

func (repo *EventsFileRepository) writeToFile(ctx context.Context, id domain.SupportArchiveID, data *domain.Events) error {
	return nil
}
