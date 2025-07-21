package usecase

import (
	"context"
	"fmt"
	libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	col "github.com/cloudogu/k8s-support-archive-operator/pkg/collector"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"slices"
	"time"
)

type CreateArchiveUseCase struct {
	supportArchivesInterface supportArchiveV1Interface
	stateHandler             stateHandler
	targetCollectors         []archiveDataCollector
}

func NewCreateArchiveUseCase(supportArchivesInterface supportArchiveV1Interface, stateHandler stateHandler) *CreateArchiveUseCase {
	return &CreateArchiveUseCase{
		supportArchivesInterface: supportArchivesInterface,
		stateHandler:             stateHandler,
		// TargetCollectors should be defined by the custom resource. This is a fake implementation.
		// With working collectors, targetCollectors should be created in HandleArchiveRequest.
		targetCollectors: []archiveDataCollector{col.NewLogCollector()},
	}
}

// HandleArchiveRequest processes the support archive custom resource.
// It reads the actual state and executes the next data collector.
// If there are remaining collectors after execution, the method returns true, nil to indicate a necessary requeue.
// If there are no remaining collectors, the method returns false, nil.
func (c CreateArchiveUseCase) HandleArchiveRequest(ctx context.Context, cr *libapi.SupportArchive) (bool, error) {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase")

	// get actualState
	currentCollectors, done, err := c.stateHandler.Read(ctx, cr.Name, cr.Namespace)
	if err != nil {
		return true, fmt.Errorf("failed to read state: %w", err)
	}
	logger.Info("Got actual state", "currentCollectors", currentCollectors)

	if done {
		logger.Info("All collectors have been executed")
		return false, nil
	}

	// get diff
	var requiredCollectors []archiveDataCollector
	for _, tc := range c.targetCollectors {
		if !slices.Contains(currentCollectors, tc.Name()) {
			requiredCollectors = append(requiredCollectors, tc)
		}
	}
	logger.Info("Determined required collectors", "requiredCollectors", getCollectorStringSlice(requiredCollectors))

	if len(requiredCollectors) == 0 {
		logger.Info("No collectors remaining")
		return c.finalize(ctx, cr)
	}

	// apply the next step - we could execute all remaining collectors, but this could block other archives in multitenant environments.
	nextCollect := requiredCollectors[0]
	name := nextCollect.Name()
	logger.Info("Determined next archiveDataCollector", "archiveDataCollector", name)
	err = nextCollect.Collect(ctx, cr.Name, cr.Namespace, c.stateHandler)
	if err != nil {
		return true, fmt.Errorf("failed to execute archiveDataCollector %s: %w", name, err)
	}
	logger.Info("Successfully executed archiveDataCollector", "archiveDataCollector", name)
	err = c.stateHandler.WriteState(ctx, cr.Name, cr.Namespace, name)
	if err != nil {
		return true, fmt.Errorf("failed to write state %s for %s/%s: %w", name, cr.Namespace, cr.Name, err)
	}

	// TODO Set condition CollectorXYDone

	return true, nil
}

func getCollectorStringSlice(collector []archiveDataCollector) []string {
	result := make([]string, len(collector))
	for i := range collector {
		result[i] = collector[i].Name()
	}

	return result
}

func (c CreateArchiveUseCase) finalize(ctx context.Context, cr *libapi.SupportArchive) (bool, error) {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase")

	err := c.stateHandler.Finalize(ctx, cr.Name, cr.Namespace)
	if err != nil {
		return true, fmt.Errorf("failed to finalize state: %w", err)
	}

	client := c.supportArchivesInterface.SupportArchives(cr.Namespace)

	downloadURL := c.stateHandler.GetDownloadURL(ctx, cr.Name, cr.Namespace)
	_, err = client.UpdateStatusWithRetry(ctx, cr, func(status libapi.SupportArchiveStatus) libapi.SupportArchiveStatus {
		meta.SetStatusCondition(&status.Conditions, getSuccessfulArchiveCreatedCondition(downloadURL))
		status.DownloadPath = downloadURL
		status.Phase = libapi.StatusPhaseCreated
		return status
	}, metav1.UpdateOptions{})
	if err != nil {
		return true, fmt.Errorf("failed to set status for archive %s/%s: %w", cr.Namespace, cr.Name, err)
	}
	logger.Info("Successfully set download url for archive for target", "url", downloadURL)

	return false, nil
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
