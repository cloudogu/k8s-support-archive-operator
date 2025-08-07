package domain

import "time"

type VolumeInfo struct {
	Name      string
	Timestamp time.Time        `yaml:"timestamp"`
	Items     []VolumeInfoItem `yaml:"items"`
}

type VolumeInfoItem struct {
	Name            string `yaml:"name"`
	Capacity        int64  `yaml:"capacity"`
	Used            int64  `yaml:"used"`
	PercentageUsage string `yaml:"percentageUsage"`
	Phase           string `yaml:"phase"`
}

type NodeInfo struct {
	Name                          NodeNameRange            `json:"name,omitempty"`
	Count                         NodeCountRange           `json:"count,omitempty"`
	Storage                       NodeStorageInfo          `json:"storage,omitempty"`
	StorageFree                   NodeStorageInfo          `json:"storageFree,omitempty"`
	StorageRelative               NodeStorageInfo          `json:"storageRelative,omitempty"`
	RAM                           NodeRAMInfo              `json:"ram,omitempty"`
	RAMFree                       NodeRAMInfo              `json:"ramFree,omitempty"`
	RAMUsedRelative               NodeRAMInfo              `json:"ramUsedRelative,omitempty"`
	CPUCores                      NodeCPUInfo              `json:"cpuCores,omitempty"`
	CPUUsage                      NodeCPUInfo              `json:"cpuUsage,omitempty"`
	CPUUsageRelative              NodeCPUInfo              `json:"cpuUsageRelative,omitempty"`
	NetworkContainerBytesReceived NodeContainerNetworkInfo `json:"networkContainerBytesReceived,omitempty"`
	NetworkContainerBytesSent     NodeContainerNetworkInfo `json:"networkContainerBytesSent,omitempty"`
}

type NodeNameRange []StringSample

type StringSample struct {
	Value string
	Time  time.Time
}

type NodeCountRange []LabeledSamples[int]
type NodeStorageInfo []LabeledSamples[float64]
type NodeRAMInfo []LabeledSamples[float64]
type NodeCPUInfo []LabeledSamples[float64]
type NodeContainerNetworkInfo []LabeledSamples[int]

type Number interface {
	int | float64
}

type LabeledSamples[n Number] struct {
	Labels  map[string]string `json:"labels"`
	Samples []Sample[n]       `json:"samples"`
}

type Sample[n Number] struct {
	Value n         `json:"value"`
	Time  time.Time `json:"time"`
}
