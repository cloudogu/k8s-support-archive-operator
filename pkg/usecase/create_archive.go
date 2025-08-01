package usecase

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"time"

	libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	if !cr.Spec.ExcludedContents.VolumeInfo {
		mapping[domain.CollectorTypVolumeInfo] = cm[domain.CollectorTypVolumeInfo]
	}

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

	nextCollector := collectorsToExecute[0]
	err = c.executeNextCollector(ctx, id, nextCollector)
	conditionErr := c.setConditionForCollector(ctx, cr, nextCollector, err)
	if err != nil {
		if conditionErr != nil {
			logger.Error(err, "could not add collector condition")
		}
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
	streamMap := make(map[domain.CollectorType]*domain.Stream)

	errGroup, errCtx := errgroup.WithContext(ctx)

	for col := range requiredCollectors {
		var stream *domain.Stream
		var err error
		logger.Info("collecting stream for collector", "collector", col)
		switch col {
		case domain.CollectorTypeLog:
			stream, err = fetchRepoAndStreamWithErrorGroup[domain.PodLog](errCtx, errGroup, col, c.collectorMapping, id)
		case domain.CollectorTypVolumeInfo:
			stream, err = fetchRepoAndStreamWithErrorGroup[domain.VolumeInfo](errCtx, errGroup, col, c.collectorMapping, id)
		default:
			return "", errors.New("invalid collector type")
		}

		if err != nil {
			return "", err
		}

		streamMap[col] = stream
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

func fetchRepoAndStreamWithErrorGroup[DATATYPE domain.CollectorUnionDataType](errCtx context.Context, group *errgroup.Group, col domain.CollectorType, collectorMapping CollectorMapping, id domain.SupportArchiveID) (*domain.Stream, error) {
	_, repo, typeErr := getCollectorAndRepositoryForType[DATATYPE](col, collectorMapping)
	if typeErr != nil {
		return nil, typeErr
	}

	resultChan := make(chan domain.StreamData)
	stream := &domain.Stream{
		Data: resultChan,
	}
	group.Go(func() error {
		var err error
		err = streamFromRepository[DATATYPE](errCtx, repo, id, stream)
		if err != nil {
			return fmt.Errorf("could not stream from repository for collector %s: %w", col, err)
		}
		return nil
	})

	return stream, nil
}

func (c *CreateArchiveUseCase) setConditionForCollector(ctx context.Context, cr *libapi.SupportArchive, collectorType domain.CollectorType, err error) error {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase.setConditionForCollector")
	client := c.supportArchivesInterface.SupportArchives(cr.Namespace)
	var condition metav1.Condition
	if err == nil {
		condition = getSuccessfulCollectorCondition(collectorType)
	} else {
		condition = getErrorCollectorCondition(collectorType, err)
	}

	_, err = client.UpdateStatusWithRetry(ctx, cr, func(status libapi.SupportArchiveStatus) libapi.SupportArchiveStatus {
		meta.SetStatusCondition(&status.Conditions, condition)
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to add condition for collector %s: %w", collectorType, err)
	}
	logger.Info("Condition set successfully", "collector", collectorType)
	return nil
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

func (c *CreateArchiveUseCase) executeNextCollector(ctx context.Context, id domain.SupportArchiveID, next domain.CollectorType) error {
	var err error
	switch next {
	case domain.CollectorTypeLog:
		col, repo, typeErr := getCollectorAndRepositoryForType[domain.PodLog](next, c.collectorMapping)
		if typeErr != nil {
			return typeErr
		}

		err = startCollector(ctx, id, time.Now(), time.Now(), col, repo)
	case domain.CollectorTypVolumeInfo:
		col, repo, typeErr := getCollectorAndRepositoryForType[domain.VolumeInfo](next, c.collectorMapping)
		if typeErr != nil {
			return typeErr
		}

		err = startCollector(ctx, id, time.Now(), time.Now(), col, repo)
	default:
		return fmt.Errorf("collector type %s is not supported", next)
	}

	if err != nil {
		return fmt.Errorf("failed to execute collector %s: %w", next, err)
	}

	return nil
}

func getCollectorAndRepositoryForType[DATATYPE domain.CollectorUnionDataType](collectorType domain.CollectorType, mapping CollectorMapping) (collector[DATATYPE], collectorRepository[DATATYPE], error) {
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

func startCollector[DATATYPE domain.CollectorUnionDataType](ctx context.Context, id domain.SupportArchiveID, startTime, endTime time.Time, collector collector[DATATYPE], repository collectorRepository[DATATYPE]) error {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase.startCollector")
	resultChan := make(chan *DATATYPE)
	errGroup, errCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		logger.Info("starting collector")
		return collector.Collect(errCtx, id.Namespace, startTime, endTime, resultChan)
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

func getSuccessfulCollectorCondition(collectorType domain.CollectorType) metav1.Condition {
	return metav1.Condition{
		Type:               collectorType.GetConditionType(),
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "CollectorExecuted",
		Message:            fmt.Sprintf("Successfully executed collector %s", collectorType),
	}
}

func getErrorCollectorCondition(collectorType domain.CollectorType, err error) metav1.Condition {
	return metav1.Condition{
		Type:               collectorType.GetConditionType(),
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Time{Time: time.Now()},
		Reason:             "ErrorDuringExecution",
		Message:            err.Error(),
	}
}

func streamFromRepository[DATATYPE domain.CollectorUnionDataType](ctx context.Context, repository collectorRepository[DATATYPE], id domain.SupportArchiveID, stream *domain.Stream) error {
	isCollected, err := repository.IsCollected(ctx, id)
	if err != nil {
		return fmt.Errorf("error during is collected call for collector: %w", err)
	}

	if !isCollected {
		return errors.New("collector is not completed")
	}

	return repository.Stream(ctx, id, stream)
}
