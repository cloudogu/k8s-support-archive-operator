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
	supportArchivesInterface    supportArchiveV1Interface
	supportArchiveRepository    supportArchiveRepository
	supportArchiveDeleteHandler deleteArchiveHandler
	namespace                   string
	syncInterval                time.Duration
	reconciliationTrigger       chan<- event.GenericEvent
}

func NewSyncArchiveUseCase(
	supportArchivesInterface supportArchiveV1Interface,
	supportArchiveRepository supportArchiveRepository,
	supportArchiveDeleteHandler deleteArchiveHandler,
	syncInterval time.Duration,
	namespace string,
	reconciliationTrigger chan<- event.GenericEvent,
) *SyncArchiveUseCase {
	return &SyncArchiveUseCase{
		supportArchivesInterface:    supportArchivesInterface,
		supportArchiveRepository:    supportArchiveRepository,
		supportArchiveDeleteHandler: supportArchiveDeleteHandler,
		syncInterval:                syncInterval,
		namespace:                   namespace,
		reconciliationTrigger:       reconciliationTrigger,
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

	toDelete := s.diff(storedSupportArchives, supportArchiveDescriptors.Items)
	var errs []error
	for archive := range toDelete {
		err := s.supportArchiveDeleteHandler.Delete(ctx, archive)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete support archive %q: %w", archive.Name, err))
		}
	}

	for _, archive := range supportArchiveDescriptors.Items {
		s.reconciliationTrigger <- event.GenericEvent{Object: &archive}
	}

	return errors.Join(errs...)
}

func (s *SyncArchiveUseCase) diff(storedSupportArchives []domain.SupportArchiveID, supportArchiveDescriptors []libv1.SupportArchive) map[domain.SupportArchiveID]struct{} {
	toDelete := make(map[domain.SupportArchiveID]struct{}, len(storedSupportArchives))
	for _, archive := range storedSupportArchives {
		toDelete[archive] = struct{}{}
	}

	for _, storedArchive := range storedSupportArchives {
		for _, archiveDescriptor := range supportArchiveDescriptors {
			if storedArchive.Name == archiveDescriptor.Name && storedArchive.Namespace == archiveDescriptor.Namespace {
				delete(toDelete, storedArchive)
				break // we can only break because we know there are no duplicates
			}
		}
	}

	return toDelete
}
