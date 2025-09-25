package collector

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

type NodeInfoCollector struct {
	metricsProvider metricsProvider
	// usageMetricStep is used to identify resource usage spikes like cpu or ram usage.
	// Since metrics are already calculated over (intersect over time) it is not necessary to use a low duration, e.g., one second here.
	usageMetricStep time.Duration
	// hardwareMetricStep is used to identify changes in the underlying hardware like increasing ram or add a new node.
	hardwareMetricStep time.Duration
}

// NewNodeInfoCollector creates a NodeInfoCollector with configurable steps.
func NewNodeInfoCollector(provider metricsProvider, usageStep, hardwareStep time.Duration) *NodeInfoCollector {
	return &NodeInfoCollector{metricsProvider: provider, usageMetricStep: usageStep, hardwareMetricStep: hardwareStep}
}

func (nic *NodeInfoCollector) Name() string {
	return string(domain.CollectorTypeNodeInfo)
}

func (nic *NodeInfoCollector) Collect(ctx context.Context, _ string, start, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	defer close(resultChan)

	err := nic.getGeneralInfo(ctx, start, end, resultChan)
	if err != nil {
		return err
	}

	err = nic.getStorage(ctx, start, end, resultChan)
	if err != nil {
		return err
	}

	err = nic.getCPU(ctx, start, end, resultChan)
	if err != nil {
		return err
	}

	err = nic.getRAM(ctx, start, end, resultChan)
	if err != nil {
		return err
	}

	err = nic.getNetwork(ctx, start, end, resultChan)
	if err != nil {
		return err
	}

	return nil
}

func (nic *NodeInfoCollector) getGeneralInfo(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeNames(ctx, start, end, nic.hardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeCount(ctx, start, end, nic.hardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	return err
}

func (nic *NodeInfoCollector) getStorage(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeStorage(ctx, start, end, nic.hardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeStorageFree(ctx, start, end, nic.usageMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeStorageFreeRelative(ctx, start, end, nic.usageMetricStep, resultChan)
	if err != nil {
		return err
	}
	return err
}

func (nic *NodeInfoCollector) getCPU(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeCPUCores(ctx, start, end, nic.hardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeCPUUsage(ctx, start, end, nic.usageMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeCPUUsageRelative(ctx, start, end, nic.usageMetricStep, resultChan)
	if err != nil {
		return err
	}

	return err
}

func (nic *NodeInfoCollector) getRAM(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeRAM(ctx, start, end, nic.hardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeRAMFree(ctx, start, end, nic.usageMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeRAMUsedRelative(ctx, start, end, nic.usageMetricStep, resultChan)
	if err != nil {
		return err
	}

	return err
}

func (nic *NodeInfoCollector) getNetwork(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeNetworkContainerBytesReceived(ctx, start, end, nic.usageMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeNetworkContainerBytesSend(ctx, start, end, nic.usageMetricStep, resultChan)
	if err != nil {
		return err
	}

	return err
}
