package domain

import "time"

type VolumeMetrics struct {
	Name      string
	Timestamp time.Time      `yaml:"timestamp"`
	Items     []VolumeMetric `yaml:"items"`
}

type VolumeMetric struct {
	Name            string `yaml:"name"`
	Capacity        int64  `yaml:"capacity"`
	Used            int64  `yaml:"used"`
	PercentageUsage string `yaml:"percentageUsage"`
	Phase           string `yaml:"phase"`
}
