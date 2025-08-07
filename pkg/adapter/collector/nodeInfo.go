package collector

import (
	"context"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"time"
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

func (nic *NodeInfoCollector) Collect(ctx context.Context, _ string, start, end time.Time, resultChan chan<- *domain.NodeInfo) error {
	nodeInfo := &domain.NodeInfo{}

	name, err := nic.metricsProvider.GetNodeNames(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.Name = name

	count, err := nic.metricsProvider.GetNodeCount(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.Count = count

	storage, err := nic.metricsProvider.GetNodeStorage(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.Storage = storage

	storageFree, err := nic.metricsProvider.GetNodeFreeStorage(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.StorageFree = storageFree

	storageFreeRelative, err := nic.metricsProvider.GetNodeFreeRelativeStorage(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.StorageFree = storageFreeRelative

	cpuCores, err := nic.metricsProvider.GetNodeCPUCores(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.CPUCores = cpuCores

	cpuUsage, err := nic.metricsProvider.GetNodeCPUUsage(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.CPUUsage = cpuUsage

	cpuUsageRelative, err := nic.metricsProvider.GetNodeCPUUsageRelative(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.CPUUsageRelative = cpuUsageRelative

	ram, err := nic.metricsProvider.GetNodeRAM(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.RAM = ram

	ramFree, err := nic.metricsProvider.GetNodeRAMFree(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.RAMFree = ramFree

	ramUsedRelative, err := nic.metricsProvider.GetNodeRAMUsedRelative(ctx, start, end)
	if err != nil {
		return err
	}
	nodeInfo.RAMUsedRelative = ramUsedRelative

	writeSaveToChannel(ctx, nodeInfo, resultChan)
	close(resultChan)
	return nil
}

func debugprint[t domain.NodeStorageInfo | domain.NodeCPUInfo | domain.NodeRAMInfo](m t) {
	for _, sample := range m {
		for i, v := range sample.Labels {
			println("Label: ", i, v)
		}
		for _, s := range sample.Samples {
			println("Value: ", s.Value, "Time: ", s.Time.Format(time.RFC3339))
		}

	}
}
