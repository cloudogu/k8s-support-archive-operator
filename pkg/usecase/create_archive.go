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
	supportArchivesInterface supportArchiveInterface
	stateHandler             stateHandler
}

func NewCreateArchiveUseCase(supportArchivesInterface supportArchiveInterface, stateHandler stateHandler) *CreateArchiveUseCase {
	return &CreateArchiveUseCase{
		supportArchivesInterface: supportArchivesInterface,
		stateHandler:             stateHandler,
	}
}

func (c CreateArchiveUseCase) HandleArchiveRequest(ctx context.Context, cr *libapi.SupportArchive) (bool, error) {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase")

	targetCollectors := []collector{col.NewLogCollector()}

	// get actualState
	currentCollectors, err := c.stateHandler.Read(cr.Name, cr.Namespace)
	if err != nil {
		return true, fmt.Errorf("failed to read state: %w", err)
	}
	logger.Info("Got actual state", "currentCollectors", currentCollectors)

	// get diff
	var requiredCollectors []collector
	for _, tc := range targetCollectors {
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
	logger.Info("Determined next collector", "collector", name)
	err = nextCollect.Collect(ctx, cr.Name, cr.Namespace, c.stateHandler)
	if err != nil {
		return true, fmt.Errorf("failed to execute collector %s: %w", name, err)
	}
	logger.Info("Successfully executed collector", "collector", name)

	// TODO Set condition CollectorXYDone

	return true, nil
}

func getCollectorStringSlice(collector []collector) []string {
	result := make([]string, len(collector))
	for i := range collector {
		result[i] = collector[i].Name()
	}

	return result
}

func (c CreateArchiveUseCase) finalize(ctx context.Context, cr *libapi.SupportArchive) (bool, error) {
	logger := log.FromContext(ctx).WithName("CreateArchiveUseCase")
	client := c.supportArchivesInterface.SupportArchives(cr.Namespace)

	downloadURL := c.stateHandler.GetDownloadURL(cr.Name, cr.Namespace)
	_, err := client.UpdateStatusWithRetry(ctx, cr, func(status libapi.SupportArchiveStatus) libapi.SupportArchiveStatus {
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
