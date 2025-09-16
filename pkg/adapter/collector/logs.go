package collector

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

type LogCollector struct{}

func NewLogCollector() *LogCollector {
	return &LogCollector{}
}

func (l *LogCollector) Name() string {
	return string(domain.CollectorTypeLog)
}

func (l *LogCollector) Collect(ctx context.Context, _ string, startTime, endTime time.Time, resultChan chan<- *domain.PodLog) error {

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
