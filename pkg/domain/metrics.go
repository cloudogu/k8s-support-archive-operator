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
	NodeCountRange NodeCountRange
	NodeNameRange  NodeNameRange
}

type NodeNameRange []StringSample

type StringSample struct {
	Value string
	Time  time.Time
}

type NodeCountRange []Sample[int]
type NodeStorageInfo []LabeledSamples[float64]
type NodeRAMInfo []LabeledSamples[float64]
type NodeCPUInfo []LabeledSamples[float64]
type NodeContainerNetworkInfo []LabeledSamples[int]

type Number interface {
	int | float64
}

type LabeledSamples[n Number] struct {
	Labels  map[string]string
	Samples []Sample[n]
}

type Sample[n Number] struct {
	Value n
	Time  time.Time
}
