package domain

import "time"

type PodLog struct {
	Timestamp time.Time
	Value     string
}
