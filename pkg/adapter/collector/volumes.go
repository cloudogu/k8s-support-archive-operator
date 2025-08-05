package collector

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"time"
)

const pvcVolumeMetricName = "persistentVolumeClaims"

type VolumesCollector struct {
	coreV1Interface coreV1Interface
	metricsProvider metricsProvider
}

func NewVolumesCollector(coreV1Interface coreV1Interface, provider metricsProvider) *VolumesCollector {
	return &VolumesCollector{coreV1Interface: coreV1Interface, metricsProvider: provider}
}

func (vc *VolumesCollector) Name() string {
	return string(domain.CollectorTypVolumeInfo)
}

func (vc *VolumesCollector) Collect(ctx context.Context, namespace string, _, end time.Time, resultChan chan<- *domain.VolumeInfo) error {
	list, err := vc.coreV1Interface.PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing pvcs: %w", err)
	}

	if len(list.Items) == 0 {
		close(resultChan)
		return nil
	}

	result := &domain.VolumeInfo{Name: pvcVolumeMetricName, Timestamp: end, Items: make([]domain.VolumeInfoItem, 0, len(list.Items))}

	for _, pvc := range list.Items {
		i, itemErr := vc.getOutputItem(ctx, pvc.Name, namespace, string(pvc.Status.Phase), end)
		if itemErr != nil {
			return fmt.Errorf("error getting output item for pvc %s: %w", pvc.Name, itemErr)
		}
		result.Items = append(result.Items, i)
	}

	writeSaveToChannel(ctx, result, resultChan)
	close(resultChan)
	return nil
}

func (vc *VolumesCollector) getOutputItem(ctx context.Context, pvcName, namespace, phase string, timestamp time.Time) (domain.VolumeInfoItem, error) {
	capacityBytes, err := vc.metricsProvider.GetCapacityBytesForPVC(ctx, namespace, pvcName, timestamp)
	if err != nil {
		return domain.VolumeInfoItem{}, fmt.Errorf("failed to get capacity bytes: %w", err)
	}

	usedBytes, err := vc.metricsProvider.GetUsedBytesForPVC(ctx, namespace, pvcName, timestamp)
	if err != nil {
		return domain.VolumeInfoItem{}, fmt.Errorf("failed to get used bytes: %w", err)
	}

	var usagePercentage string
	if capacityBytes != 0 {
		usagePercentage = strconv.FormatFloat(float64(usedBytes)/float64(capacityBytes)*100, 'f', 2, 64)
	}

	return domain.VolumeInfoItem{
		Name:            pvcName,
		Capacity:        capacityBytes,
		Used:            usedBytes,
		PercentageUsage: usagePercentage,
		Phase:           phase,
	}, nil
}
