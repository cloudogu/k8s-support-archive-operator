package v1

import "errors"

const (
	capacityBytesQueryFmt = "kubelet_volume_stats_capacity_bytes{namespace=\"%s\", persistentvolumeclaim=\"%s\"}"
	usedBytesQueryFmt     = "kubelet_volume_stats_used_bytes{namespace=\"%s\", persistentvolumeclaim=\"%s\"}"
)

const (
	nodeCountMetric               = "count"
	nodeNameMetric                = "name"
	nodeStorageMetric             = "storage"
	nodeStorageFreeMetric         = "storageFree"
	nodeStorageFreeRelativeMetric = "storageFreeRelative"

	nodeRAMMetric             = "ram"
	nodeRAMFreeMetric         = "ramFree"
	nodeRAMUsedRelativeMetric = "ramFreeRelative"

	nodeCPUCoresMetric         = "cpuCores"
	nodeCPUUsageMetric         = "cpuUsage"
	nodeCPUUsageRelativeMetric = "cpuUsageRelative"

	nodeNetworkContainerBytesReceivedMetric = "containerNetworkBytesReceived"
	nodeNetworkContainerBytesSentMetric     = "containerNetworkBytesSent"
)

type metric string

func (q metric) getQuery() (string, error) {
	switch q {
	case nodeCountMetric:
		return "count(kube_node_info)", nil
	case nodeNameMetric:
		return "count(kube_node_info) by (node)", nil
	case nodeStorageMetric:
		return "node_filesystem_size_bytes{mountpoint=\"/\",fstype!=\"rootfs\"}", nil
	case nodeStorageFreeMetric:
		return "node_filesystem_avail_bytes{mountpoint=\"/\",fstype!=\"rootfs\"}", nil
	case nodeStorageFreeRelativeMetric:
		return "100 - ((node_filesystem_avail_bytes{mountpoint=\"/\",fstype!=\"rootfs\"} * 100) / node_filesystem_size_bytes{mountpoint=\"/\",fstype!=\"rootfs\"})", nil
	case nodeRAMMetric:
		return "machine_memory_bytes", nil
	case nodeRAMFreeMetric:
		return "avg_over_time(node_memory_MemFree_bytes[5m]) + avg_over_time(node_memory_Cached_bytes[10m]) + avg_over_time(node_memory_Buffers_bytes[5m])", nil
	case nodeRAMUsedRelativeMetric:
		return "100 * (1- ((avg_over_time(node_memory_MemFree_bytes[5m]) + avg_over_time(node_memory_Cached_bytes[10m]) + avg_over_time(node_memory_Buffers_bytes[5m])) / avg_over_time(node_memory_MemTotal_bytes[5m])))", nil
	case nodeCPUCoresMetric:
		return "machine_cpu_cores", nil
	case nodeCPUUsageMetric:
		return "sum(rate (container_cpu_usage_seconds_total{id=~\"/.*\"}[2m])) by (node)", nil
	case nodeCPUUsageRelativeMetric:
		return "100 * avg(1 - rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) by (node)", nil
	case nodeNetworkContainerBytesReceivedMetric:
		return "sum (rate (container_network_receive_bytes_total[2m])) by (node)", nil
	case nodeNetworkContainerBytesSentMetric:
		return "sum (rate (container_network_transmit_bytes_total[2m])) by (node)", nil
	default:
		return "", errors.New("no query for metric")
	}
}
