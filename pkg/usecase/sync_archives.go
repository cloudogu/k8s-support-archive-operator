package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	k8sErrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type SyncArchiveUseCase struct {
	supportArchivesInterface supportArchiveV1Interface
	supportArchiveRepository supportArchiveRepository
	supportArchiveHandler    deleteArchiveHandler
	syncInterval             time.Duration
}

func NewSyncArchiveUseCase(supportArchivesInterface supportArchiveV1Interface, supportArchiveRepository supportArchiveRepository, supportArchiveHandler deleteArchiveHandler, syncInterval time.Duration) *SyncArchiveUseCase {
	return &SyncArchiveUseCase{supportArchivesInterface: supportArchivesInterface, supportArchiveRepository: supportArchiveRepository, supportArchiveHandler: supportArchiveHandler, syncInterval: syncInterval}
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

	supportArchives, err := s.supportArchiveRepository.List(ctx)
	if len(supportArchives) != 0 && err != nil {
		logger.Error(err, "partial failure when listing support archives")
	} else if err != nil {
		return fmt.Errorf("failed to list support archives: %w", err)
	}

	var errs []error
	for _, archive := range supportArchives {
		_, err := s.supportArchivesInterface.SupportArchives(archive.Namespace).Get(ctx, archive.Name, metav1.GetOptions{})
		if k8sErrs.IsNotFound(err) {
			err := s.supportArchiveHandler.Delete(ctx, archive)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to delete support archive %q: %w", archive.Name, err))
			}
		} else if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
