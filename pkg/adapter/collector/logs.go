package collector

import (
	"context"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"time"
)

type LogCollector struct{}

func NewLogCollector() *LogCollector {
	return &LogCollector{}
}

func (l *LogCollector) Name() string {
	return "Logs"
}

// Do not close resultChan on error. Closing the channel should only indicate that the collection finished successfully.
func (l *LogCollector) Collect(ctx context.Context, _ string, startTime, endTime time.Time, resultChan chan<- *domain.PodLog) error {
	doguLog := &domain.PodLog{
		PodName:   "cas",
		StartTime: startTime,
		EndTime:   endTime,
		Entries:   []string{"log entry"},
	}

	writeSaveToChannel(ctx, doguLog, resultChan)

	doguLog = &domain.PodLog{
		PodName:   "ldap",
		StartTime: startTime,
		EndTime:   endTime,
		Entries:   []string{"log entry"},
	}

	writeSaveToChannel(ctx, doguLog, resultChan)
	close(resultChan)

	return nil
}

// Select ctx.Done on writing to avoid blocking this process if the receiver throws an error and does not read the channel anymore.
// The context muss be derived from the shared error group.
func writeSaveToChannel[T any](ctx context.Context, data T, dataChannel chan<- T) {
	select {
	case <-ctx.Done():
		return
	case dataChannel <- data:
		return
	}
}
