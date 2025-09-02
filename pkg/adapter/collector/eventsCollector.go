package collector

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

type EventsCollector struct{}

func (ec *EventsCollector) Collect(ctx context.Context, namespace string, startTime, endTime time.Time, resultChan chan<- *domain.Events) error {
	return nil
}
func (ec *EventsCollector) Name() string {
	return ""
}
