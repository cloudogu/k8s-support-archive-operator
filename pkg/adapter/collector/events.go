package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

type EventsCollector struct {
	logsProvider LogsProvider
}

func NewEventsCollector(logsProvider LogsProvider) *EventsCollector {
	return &EventsCollector{
		logsProvider: logsProvider,
	}
}

func (ec *EventsCollector) Collect(ctx context.Context, namespace string, startTime, endTime time.Time, resultChan chan<- *domain.LogLine) error {
	defer close(resultChan)

	err := ec.logsProvider.FindEvents(ctx, startTime.UnixNano(), endTime.UnixNano(), namespace, resultChan)
	if err != nil {
		return fmt.Errorf("call log provider to find logs; %v", err)
	}

	return nil
}

func (ec *EventsCollector) Name() string {
	return string(domain.CollectorTypeEvents)
}
