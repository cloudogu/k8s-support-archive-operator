package domain

import "time"

type PodLog struct {
	PodName   string
	StartTime time.Time
	EndTime   time.Time
	Entries   []string
}

type LogLine struct {
	Timestamp time.Time
	Value     string
}
