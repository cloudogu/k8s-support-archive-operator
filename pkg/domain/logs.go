package domain

import "time"

type LogLine struct {
	Timestamp time.Time
	Value     string
}
