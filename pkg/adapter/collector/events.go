package collector

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

type EventsCollector struct {
	logsProvider logsProvider
}

func (ec *EventsCollector) Collect(ctx context.Context, namespace string, startTime, endTime time.Time, resultChan chan<- *domain.Events) error {
	/*
		kindValues, _ := ec.logsProvider.getValuesOfLabel(ctx, startTime, endTime, "kind")
		for _, kind := range kindValues {
			logs, _ := ec.logsProvider.getLogs(ctx, startTime, endTime, namespace, kind)
			events := &domain.Events{
				Namespace: namespace,
				Kind:      kind,
				Logs:      logs,
			}
			writeSaveToChannel(ctx, events, resultChan)
		}
		close(resultChan)
	*/
	return nil

}
func (ec *EventsCollector) Name() string {
	return string(domain.CollectorTypEvents)
}
