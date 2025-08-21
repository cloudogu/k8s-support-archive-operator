package domain

import (
	"strconv"
	"time"
)

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

type LabeledSample struct {
	MetricName string
	ID         string
	Value      float64
	Time       time.Time
}

func (ls *LabeledSample) GetHeader() []string {
	return []string{"label", "value", "time"}
}

func (ls *LabeledSample) GetRow() []string {
	return []string{ls.ID, strconv.FormatFloat(ls.Value, 'f', 2, 64), ls.Time.Format("2006-01-02T15:04:05-07:00")}
}
