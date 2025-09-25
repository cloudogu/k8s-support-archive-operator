package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

type LogCollector struct {
	logProvider LogsProvider
}

func NewLogCollector(logProvider LogsProvider) *LogCollector {
	return &LogCollector{logProvider: logProvider}
}

func (l *LogCollector) Name() string {
	return string(domain.CollectorTypeLog)
}

func (l *LogCollector) Collect(ctx context.Context, namespace string, startTime, endTime time.Time, resultChan chan<- *domain.LogLine) error {
	defer close(resultChan)

	err := l.logProvider.FindLogs(ctx, startTime, endTime, namespace, resultChan)
	if err != nil {
		return fmt.Errorf("failed to find logs: %w", err)
	}

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
