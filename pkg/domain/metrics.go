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
