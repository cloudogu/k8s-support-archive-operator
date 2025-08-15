package collector

import (
	"context"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"time"
)

const (
	// defaultUsageMetricStep is used to identify resource usage spikes like cpu or ram usage.
	// Since metrics are already calculated over (intersect over time) it is not necessary to use a low duration, e.g., one second here.
	defaultUsageMetricStep = time.Second * 30 // TODO configurable
	// defaultHardwareMetricStep is used to identify changes in the underlying hardware like increasing ram or add a new node.
	defaultHardwareMetricStep = time.Minute * 30
)

type NodeInfoCollector struct {
	metricsProvider metricsProvider
}

func NewNodeInfoCollector(provider metricsProvider) *NodeInfoCollector {
	return &NodeInfoCollector{metricsProvider: provider}
}

func (nic *NodeInfoCollector) Name() string {
	return string(domain.CollectorTypeNodeInfo)
}

func (nic *NodeInfoCollector) Collect(ctx context.Context, _ string, start, end time.Time, resultChan chan<- *domain.LabeledSample) error {
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

	close(resultChan)
	return nil
}

func (nic *NodeInfoCollector) getGeneralInfo(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeNames(ctx, start, end, defaultHardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeCount(ctx, start, end, defaultHardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	return err
}

func (nic *NodeInfoCollector) getStorage(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeStorage(ctx, start, end, defaultHardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeStorageFree(ctx, start, end, defaultUsageMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeStorageFreeRelative(ctx, start, end, defaultUsageMetricStep, resultChan)
	if err != nil {
		return err
	}
	return err
}

func (nic *NodeInfoCollector) getCPU(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeCPUCores(ctx, start, end, defaultHardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeCPUUsage(ctx, start, end, defaultUsageMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeCPUUsageRelative(ctx, start, end, defaultUsageMetricStep, resultChan)
	if err != nil {
		return err
	}

	return err
}

func (nic *NodeInfoCollector) getRAM(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeRAM(ctx, start, end, defaultHardwareMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeRAMFree(ctx, start, end, defaultUsageMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeRAMUsedRelative(ctx, start, end, defaultUsageMetricStep, resultChan)
	if err != nil {
		return err
	}

	return err
}

func (nic *NodeInfoCollector) getNetwork(ctx context.Context, start time.Time, end time.Time, resultChan chan<- *domain.LabeledSample) error {
	err := nic.metricsProvider.GetNodeNetworkContainerBytesReceived(ctx, start, end, defaultUsageMetricStep, resultChan)
	if err != nil {
		return err
	}

	err = nic.metricsProvider.GetNodeNetworkContainerBytesSend(ctx, start, end, defaultUsageMetricStep, resultChan)
	if err != nil {
		return err
	}

	return err
}
