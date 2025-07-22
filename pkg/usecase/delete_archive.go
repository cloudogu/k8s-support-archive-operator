package usecase

import (
	"context"
	"errors"
	"fmt"
	libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

type DeleteArchiveUseCase struct {
	supportArchivesInterface supportArchiveV1Interface
	supportArchiveRepository supportArchiveRepository
	collectorMapping         CollectorMapping
}

func NewDeleteArchiveUseCase(supportArchivesInterface supportArchiveV1Interface, collectorMapping CollectorMapping, supportArchiveRepository supportArchiveRepository) *DeleteArchiveUseCase {
	return &DeleteArchiveUseCase{
		supportArchivesInterface: supportArchivesInterface,
		supportArchiveRepository: supportArchiveRepository,
		collectorMapping:         collectorMapping,
	}
}

func (d *DeleteArchiveUseCase) Delete(ctx context.Context, cr *libapi.SupportArchive) error {
	id := domain.SupportArchiveID{
		Namespace: cr.Namespace,
		Name:      cr.Name,
	}

	var multiErr []error
	// Always try to delete all collector files to avoid zombie data.
	for col := range d.collectorMapping {
		err := deleteCollectorRepositoryData(ctx, id, col, d.collectorMapping)
		if err != nil {
			multiErr = append(multiErr, err)
		}
	}

	err := d.supportArchiveRepository.Delete(ctx, id)
	if err != nil {
		multiErr = append(multiErr, err)
	}

	return errors.Join(multiErr...)
}

func deleteCollectorRepositoryData(ctx context.Context, id domain.SupportArchiveID, col domain.CollectorType, collectorMapping CollectorMapping) error {
	baseRepo, err := getBaseRepositoryForCollector(col, collectorMapping)
	if err != nil {
		return err
	}
	err = baseRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete %s collector repository: %w", col, err)
	}

	return nil
}

func getBaseRepositoryForCollector(collectorType domain.CollectorType, collectorMapping CollectorMapping) (baseCollectorRepository, error) {
	baseRepo, ok := collectorMapping[collectorType].Repository.(baseCollectorRepository)
	if !ok {
		return nil, fmt.Errorf("invalid base repository type for collector %s", collectorType)
	}
	return baseRepo, nil
}
