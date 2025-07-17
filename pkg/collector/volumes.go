package collector

import (
	"bytes"
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"time"
)

type outputFile struct {
	Items []item `yaml:"items"`
}

type item struct {
	Name            string `yaml:"name"`
	Capacity        int64  `yaml:"capacity"`
	Used            int64  `yaml:"used"`
	PercentageUsage string `yaml:"percentageUsage"`
	Phase           string `yaml:"phase"`
}

type VolumesCollector struct {
	coreV1Interface coreV1Interface
	metricsProvider metricsProvider
}

func NewVolumesCollector(coreV1Interface coreV1Interface, provider metricsProvider) *VolumesCollector {
	return &VolumesCollector{coreV1Interface: coreV1Interface, metricsProvider: provider}
}

func (vc *VolumesCollector) Name() string {
	return "Volumes"
}

func (vc *VolumesCollector) Collect(ctx context.Context, name, namespace string, writer StateWriter) error {
	list, err := vc.coreV1Interface.PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error listing pvcs: %w", err)
	}

	result := &outputFile{Items: make([]item, 0, len(list.Items))}

	for _, pvc := range list.Items {
		i, itemErr := vc.getOutputItem(ctx, pvc.Name, namespace, string(pvc.Status.Phase))
		if itemErr != nil {
			return fmt.Errorf("error getting output item for pvc %s: %w", pvc.Name, itemErr)
		}
		result.Items = append(result.Items, i)
	}

	outBytes, err := yaml.Marshal(result)
	if err != nil {
		return fmt.Errorf("error marshalling result: %w", err)
	}

	err = writer.Write(ctx, vc.Name(), name, namespace, fmt.Sprintf("%s/volumes.yaml", vc.Name()), func(w io.Writer) error {
		_, writeErr := io.Copy(w, bytes.NewReader(outBytes))
		return writeErr
	})

	return err
}

func (vc *VolumesCollector) getOutputItem(ctx context.Context, pvcName, namespace, phase string) (item, error) {
	capacityBytes, err := vc.metricsProvider.GetCapacityBytesForPVC(ctx, namespace, pvcName, time.Now())
	if err != nil {
		return item{}, err
	}

	usedBytes, err := vc.metricsProvider.GetUsedBytesForPVC(ctx, namespace, pvcName, time.Now())
	if err != nil {
		return item{}, err
	}

	var usagePercentage string
	if capacityBytes != 0 {
		usagePercentage = strconv.FormatFloat(float64(usedBytes)/float64(capacityBytes)*100, 'f', 2, 64)
	}

	return item{
		Name:            pvcName,
		Capacity:        capacityBytes,
		Used:            usedBytes,
		PercentageUsage: usagePercentage,
		Phase:           phase,
	}, nil
}
