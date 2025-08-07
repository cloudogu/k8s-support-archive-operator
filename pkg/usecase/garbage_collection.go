package usecase

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	libv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type GarbageCollectionUseCase struct {
	supportArchivesInterface supportArchiveInterface
	supportArchiveRepository supportArchiveRepository
	interval                 time.Duration
	numberToKeep             int
}

func NewGarbageCollectionUseCase(supportArchivesInterface supportArchiveInterface, supportArchiveRepository supportArchiveRepository, interval time.Duration, numberToKeep int) *GarbageCollectionUseCase {
	return &GarbageCollectionUseCase{supportArchivesInterface: supportArchivesInterface, supportArchiveRepository: supportArchiveRepository, interval: interval, numberToKeep: numberToKeep}
}

func (g *GarbageCollectionUseCase) CollectGarbageWithInterval(ctx context.Context) error {
	logger := log.FromContext(ctx).
		WithName("support archive garbage collection handler")

	logger.Info("start collecting garbage...")

	if g.interval == 0 {
		logger.Info("garbage collection interval set to 0; disabling garbage collection")
		return nil
	}

	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()

	var errs []error
	for {
		select {
		case <-ctx.Done():
			return errors.Join(errs...)
		case <-ticker.C:
			logger.Info("regularly collecting garbage...")
			err := g.collectGarbage(ctx)
			if err != nil {
				logger.Error(err, "failed to regularly garbage collect support archives")
				errs = append(errs, err)
			}
		}
	}
}

func (g *GarbageCollectionUseCase) collectGarbage(ctx context.Context) error {
	archiveList, err := g.supportArchivesInterface.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list support archives: %w", err)
	}

	var errs []error
	toDelete, err := g.findArchivesToDelete(ctx, archiveList.Items)
	errs = append(errs, err)

	err = g.deleteArchives(ctx, toDelete)
	errs = append(errs, err)

	return errors.Join(errs...)
}

func (g *GarbageCollectionUseCase) findArchivesToDelete(ctx context.Context, archives []libv1.SupportArchive) ([]libv1.SupportArchive, error) {
	completedArchives, err := g.getCompletedArchives(ctx, archives)

	return g.getOldestArchivesExcludingTheOnesToKeep(completedArchives), err
}

func (g *GarbageCollectionUseCase) getCompletedArchives(ctx context.Context, archives []libv1.SupportArchive) ([]libv1.SupportArchive, error) {
	var errs []error
	completedArchives := slices.Collect(func(yield func(libv1.SupportArchive) bool) {
		for _, archive := range archives {
			archiveID := domain.SupportArchiveID{
				Namespace: archive.Namespace,
				Name:      archive.Name,
			}
			exists, err := g.supportArchiveRepository.Exists(ctx, archiveID)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to check if support archive %s/%s exists: %w", archiveID.Namespace, archiveID.Name, err))
			}

			if exists {
				if !yield(archive) {
					return
				}
			}
		}
	})

	return completedArchives, errors.Join(errs...)
}

func (g *GarbageCollectionUseCase) getOldestArchivesExcludingTheOnesToKeep(completedArchives []libv1.SupportArchive) []libv1.SupportArchive {
	if len(completedArchives) <= g.numberToKeep {
		return nil
	}

	slices.SortFunc(completedArchives, func(a, b libv1.SupportArchive) int {
		return a.CreationTimestamp.Compare(b.CreationTimestamp.Time)
	})

	return completedArchives[g.numberToKeep:]
}

func (g *GarbageCollectionUseCase) deleteArchives(ctx context.Context, toDelete []libv1.SupportArchive) error {
	var errs []error
	for _, archive := range toDelete {
		err := g.supportArchivesInterface.Delete(ctx, archive.Name, metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete support archive %s/%s: %w", archive.Namespace, archive.Name, err))
		}
	}

	return errors.Join(errs...)
}
