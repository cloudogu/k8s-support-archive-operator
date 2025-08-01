package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	libv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type SyncArchiveUseCase struct {
	supportArchivesInterface supportArchiveV1Interface
	supportArchiveRepository supportArchiveRepository
	supportArchiveHandler    deleteArchiveHandler
	namespace                string
	syncInterval             time.Duration
	reconciliationTrigger    chan<- event.GenericEvent
}

func NewSyncArchiveUseCase(
	supportArchivesInterface supportArchiveV1Interface,
	supportArchiveRepository supportArchiveRepository,
	supportArchiveHandler deleteArchiveHandler,
	syncInterval time.Duration,
	namespace string,
	reconciliationTrigger chan<- event.GenericEvent,
) *SyncArchiveUseCase {
	return &SyncArchiveUseCase{
		supportArchivesInterface: supportArchivesInterface,
		supportArchiveRepository: supportArchiveRepository,
		supportArchiveHandler:    supportArchiveHandler,
		syncInterval:             syncInterval,
		namespace:                namespace,
		reconciliationTrigger:    reconciliationTrigger,
	}
}

func (s *SyncArchiveUseCase) SyncArchivesWithInterval(ctx context.Context) error {
	logger := log.FromContext(ctx).
		WithName("support archive sync interval handler")
	logger.Info(fmt.Sprintf("started regularly syncing support archives with interval %s", s.syncInterval))

	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	var errs []error
	for {
		select {
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
			return errors.Join(errs...)
		case <-ticker.C:
			logger.Info("regularly syncing support archives...")
			err := s.syncArchives(ctx)
			if err != nil {
				logger.Error(err, "failed to regularly sync support archives")
				errs = append(errs, err)
			}
		}
	}
}

func (s *SyncArchiveUseCase) syncArchives(ctx context.Context) error {
	logger := log.FromContext(ctx).
		WithName("sync support archives")

	storedSupportArchives, err := s.supportArchiveRepository.List(ctx)
	if len(storedSupportArchives) != 0 && err != nil {
		logger.Error(err, "partial failure when listing support archives")
	} else if err != nil {
		return fmt.Errorf("failed to list stored support archives: %w", err)
	}

	supportArchiveDescriptors, err := s.supportArchivesInterface.SupportArchives(s.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list support archive descriptors: %w", err)
	}

	toDelete, toAdd := s.diff(storedSupportArchives, supportArchiveDescriptors.Items)

	var errs []error
	for _, archive := range toDelete {
		err := s.supportArchiveHandler.Delete(ctx, archive)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete support archive %q: %w", archive.Name, err))
		}
	}

	for _, archive := range toAdd {
		s.reconciliationTrigger <- event.GenericEvent{Object: &archive}
	}

	return errors.Join(errs...)
}

func (s *SyncArchiveUseCase) diff(storedSupportArchives []domain.SupportArchiveID, supportArchiveDescriptors []libv1.SupportArchive) ([]domain.SupportArchiveID, []libv1.SupportArchive) {
	toDelete := make([]domain.SupportArchiveID, len(storedSupportArchives))
	copy(toDelete, storedSupportArchives)
	toAdd := make([]libv1.SupportArchive, len(supportArchiveDescriptors))
	copy(toAdd, supportArchiveDescriptors)

	for i, storedArchive := range storedSupportArchives {
		for j, archiveDescriptor := range supportArchiveDescriptors {
			if storedArchive.Name == archiveDescriptor.Name && storedArchive.Namespace == archiveDescriptor.Namespace {
				toDelete = remove(toDelete, i)
				toAdd = remove(toAdd, j)
			}
		}
	}

	return toDelete, toAdd
}

func remove[T any](s []T, i int) []T {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
