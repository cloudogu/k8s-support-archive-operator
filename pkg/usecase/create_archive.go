package usecase

import (
	"context"
	"errors"
	"fmt"
	libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

type CollectorAndRepository struct {
	// We have to use any here because of the different data types.
	Collector  any
	Repository any
}

type CollectorMapping map[domain.CollectorType]CollectorAndRepository

func (cm CollectorMapping) getRequiredCollectorMapping(cr *libapi.SupportArchive) CollectorMapping {
	mapping := make(CollectorMapping)
	if !cr.Spec.ExcludedContents.Logs {
		mapping[domain.CollectorTypeLog] = cm[domain.CollectorTypeLog]
	}
	// TODO extend with more collectors
	/*if !cr.Spec.ExcludedContents.VolumeInfo {
		mapping[domain.CollectorTypVolume] = cm[domain.CollectorTypVolume]
	}*/

	return mapping
}

type CreateArchiveUseCase struct {
	supportArchivesInterface supportArchiveV1Interface
	supportArchiveRepository supportArchiveRepository
	collectorMapping         CollectorMapping
}

func NewCreateArchiveUseCase(supportArchivesInterface supportArchiveV1Interface, collectorMapping CollectorMapping, supportArchiveRepository supportArchiveRepository) *CreateArchiveUseCase {
	return &CreateArchiveUseCase{
		supportArchivesInterface: supportArchivesInterface,
		supportArchiveRepository: supportArchiveRepository,
		collectorMapping:         collectorMapping,
	}
}

// HandleArchiveRequest processes the support archive custom resource.
// It reads the actual state and executes the next data collector.
// If there are remaining collectors after execution, the method returns (true, nil) to indicate a necessary requeue.
// If there are no remaining collectors, the method returns (false, nil).
func (c *CreateArchiveUseCase) HandleArchiveRequest(ctx context.Context, cr *libapi.SupportArchive) (bool, error) {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase.HandleArchiveRequest")

	id := domain.SupportArchiveID{
		Namespace: cr.GetNamespace(),
		Name:      cr.GetName(),
	}
	requiredCollectorMapping := c.collectorMapping.getRequiredCollectorMapping(cr)
	completedCollectorList, err := c.getAlreadyExecutedCollectors(ctx, id)
	if err != nil {
		return true, fmt.Errorf("could not get already executed collectors: %w", err)
	}
	// If the user changes required collectors, we have to clean up old unused data.
	c.deleteUnusedRepositoryData(ctx, id, requiredCollectorMapping, completedCollectorList)

	collectorsToExecute := getCollectorTypesToExecute(requiredCollectorMapping, completedCollectorList)
	exists, err := c.supportArchiveRepository.Exists(ctx, id)
	if err != nil {
		return true, fmt.Errorf("could not check if the support archive exists: %w", err)
	}
	if len(collectorsToExecute) == 0 && !exists {
		logger.Info("all collectors are executed")
		url, createErr := c.createArchive(ctx, id, requiredCollectorMapping)
		if createErr != nil {
			return true, fmt.Errorf("could not create archive: %w", createErr)
		}
		statusErr := c.updateFinalStatus(ctx, cr, url)
		if statusErr != nil {
			return true, fmt.Errorf("could not update status: %w", statusErr)
		}

		return false, nil
	} else if len(collectorsToExecute) == 0 {
		logger.Info("archive exists")
		return false, nil
	}

	err = c.executeNextCollector(ctx, id, collectorsToExecute)
	if err != nil {
		return true, fmt.Errorf("could not execute next collector: %w", err)
	}

	return true, nil
}

func (c *CreateArchiveUseCase) deleteUnusedRepositoryData(ctx context.Context, id domain.SupportArchiveID, requiredCollectorMapping CollectorMapping, executedCollectors []domain.CollectorType) {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase.deleteUnusedRepositoryData")
	for _, col := range executedCollectors {
		_, ok := requiredCollectorMapping[col]
		if ok {
			continue
		}
		err := deleteCollectorRepositoryData(ctx, id, col, c.collectorMapping)
		if err != nil {
			logger.Error(err, "failed remove no longer required repository data", "collector", col)
		}
	}
}

func (c *CreateArchiveUseCase) createArchive(ctx context.Context, id domain.SupportArchiveID, requiredCollectors CollectorMapping) (string, error) {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase.createArchive")
	streamMap := make(map[domain.CollectorType]domain.Stream)
	streamFinalizers := make([]func() error, len(requiredCollectors))
	defer func() {
		finalizeStreams(streamFinalizers, logger)
	}()

	errGroup, errCtx := errgroup.WithContext(ctx)

	for col := range requiredCollectors {
		logger.Info("collecting stream for collector", "collector", col)
		switch col {
		case domain.CollectorTypeLog:
			_, repo, typeErr := getCollectorAndRepositoryForType[domain.PodLog](col, c.collectorMapping)
			if typeErr != nil {
				return "", typeErr
			}

			resultChan := make(chan domain.StreamData)
			stream := domain.Stream{
				Data: resultChan,
			}

			errGroup.Go(func() error {
				finalizer, err := streamFromRepository[domain.PodLog](errCtx, repo, id, stream)
				streamFinalizers = append(streamFinalizers, finalizer)
				if err != nil {
					return fmt.Errorf("could not stream from repository for collector %s: %w", col, err)
				}
				return nil
			})
			streamMap[col] = stream
		case domain.CollectorTypVolume:
			return "", errors.New("not implemented yet")
		default:
			return "", errors.New("invalid collector type")
		}
	}

	var url string
	errGroup.Go(func() error {
		var createErr error
		url, createErr = c.supportArchiveRepository.Create(errCtx, id, streamMap)
		return createErr
	})

	err := errGroup.Wait()
	if err != nil {
		return "", fmt.Errorf("error creating support archive: %w", err)
	}
	logger.Info("Created support archive successfully")

	return url, nil
}

func finalizeStreams(finalizers []func() error, logger logr.Logger) {
	for _, finalizer := range finalizers {
		if finalizer == nil {
			continue
		}
		err := finalizer()
		if err != nil {
			logger.Error(err, "could not finalize collector")
		}
	}
}

func (c *CreateArchiveUseCase) updateFinalStatus(ctx context.Context, cr *libapi.SupportArchive, url string) error {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase.updateFinalStatus")
	client := c.supportArchivesInterface.SupportArchives(cr.Namespace)
	_, err := client.UpdateStatusWithRetry(ctx, cr, func(status libapi.SupportArchiveStatus) libapi.SupportArchiveStatus {
		meta.SetStatusCondition(&status.Conditions, getSuccessfulArchiveCreatedCondition(url))
		status.DownloadPath = url
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to set status for archive %s/%s: %w", cr.Namespace, cr.Name, err)
	}
	logger.Info("Successfully set download url for archive for target", "url", url)

	return nil
}

func (c *CreateArchiveUseCase) executeNextCollector(ctx context.Context, id domain.SupportArchiveID, collectorsToExecute []domain.CollectorType) error {
	next := collectorsToExecute[0]
	var err error
	switch next {
	case domain.CollectorTypeLog:
		col, repo, typeErr := getCollectorAndRepositoryForType[domain.PodLog](next, c.collectorMapping)
		if typeErr != nil {
			return typeErr
		}

		err = startCollector(ctx, id, time.Now(), time.Now(), col, repo)
	case domain.CollectorTypVolume:
		err = fmt.Errorf("not implemented yet")
	default:
		return fmt.Errorf("collector type %s is not supported", next)
	}

	if err != nil {
		return fmt.Errorf("failed to execute collector %s: %w", next, err)
	}

	// TODO Add collector as condition in cr.status

	return nil
}

func getCollectorAndRepositoryForType[DATATYPE any](collectorType domain.CollectorType, mapping CollectorMapping) (collector[DATATYPE], collectorRepository[DATATYPE], error) {
	col, ok := mapping[collectorType].Collector.(collector[DATATYPE])
	if !ok {
		return nil, nil, fmt.Errorf("invalid collector type for collector %s", collectorType)
	}

	repo, ok := mapping[collectorType].Repository.(collectorRepository[DATATYPE])
	if !ok {
		return nil, nil, fmt.Errorf("invalid repository type for collector %s", collectorType)
	}

	return col, repo, nil
}

func startCollector[DATATYPE any](ctx context.Context, id domain.SupportArchiveID, startTime, endTime time.Time, collector collector[DATATYPE], repository collectorRepository[DATATYPE]) error {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase.startCollector")
	resultChan := make(chan *DATATYPE)
	errGroup, errCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		logger.Info("starting collector")
		return collector.Collect(errCtx, startTime, endTime, resultChan)
	})

	errGroup.Go(func() error {
		logger.Info("starting reading from collector")
		return repository.Create(errCtx, id, resultChan)
	})

	err := errGroup.Wait()
	if err != nil {
		return fmt.Errorf("error from error group %s: %w", collector.Name(), err)
	}

	return nil
}

func (c *CreateArchiveUseCase) getAlreadyExecutedCollectors(ctx context.Context, id domain.SupportArchiveID) ([]domain.CollectorType, error) {
	logger := log.FromContext(ctx).WithName("GetAlreadyExecutedCollectors.getAlreadyExecutedCollectors")
	var completedCollectorList []domain.CollectorType
	// Get actual state
	for colType := range c.collectorMapping {
		baseRepo, err := getBaseRepositoryForCollector(colType, c.collectorMapping)
		if err != nil {
			return nil, err
		}

		finished, err := baseRepo.IsCollected(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to determine if collector %s is already finished: %w", colType, err)
		}
		if !finished {
			logger.Info(fmt.Sprintf("collector %s is not finished", colType))
			continue
		}

		logger.Info(fmt.Sprintf("collector %s is already finished", colType))
		completedCollectorList = append(completedCollectorList, colType)
	}
	return completedCollectorList, nil
}

func getCollectorTypesToExecute(requiredCollectors CollectorMapping, completedCollectorListToExecute []domain.CollectorType) []domain.CollectorType {
	var result []domain.CollectorType

	for i := range requiredCollectors {
		found := false
		for _, j := range completedCollectorListToExecute {
			if i == j {
				found = true
				break
			}
		}
		if !found {
			result = append(result, i)
		}
	}

	return result
}

func getSuccessfulArchiveCreatedCondition(downloadURL string) metav1.Condition {
	return metav1.Condition{
		Type:               libapi.ConditionSupportArchiveCreated,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "AllCollectorsExecuted",
		Message:            fmt.Sprintf("It is available for download under following url: %s", downloadURL),
	}
}

func streamFromRepository[DATATYPE any](ctx context.Context, repository collectorRepository[DATATYPE], id domain.SupportArchiveID, stream domain.Stream) (func() error, error) {
	isCollected, err := repository.IsCollected(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error during is collected call for collector: %w", err)
	}

	if !isCollected {
		return nil, errors.New("collector is not completed")
	}

	return repository.Stream(ctx, id, stream)
}
