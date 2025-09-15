package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

func (ec *EventsCollector) Collect(ctx context.Context, namespace string, startTime, endTime time.Time, resultChan chan<- *domain.EventSet) error {
	defer close(resultChan)

	kindValues, err := ec.logsProvider.FindValuesOfLabel(ctx, startTime.UnixNano(), endTime.UnixNano(), "kind")
	if err != nil {
		return err
	}

	for _, kind := range kindValues {
		logLines, err2 := ec.logsProvider.FindLogs(ctx, startTime.UnixNano(), endTime.UnixNano(), namespace, kind)
		if err2 != nil {
			return err2
		}
		events, err3 := logLinesToEvents(logLines)
		if err3 != nil {
			return err3
		}
		eventSet := &domain.EventSet{
			Namespace: namespace,
			Kind:      kind,
			Events:    events,
		}
		writeSaveToChannel(ctx, eventSet, resultChan)
	}

	return nil
}

func logLinesToEvents(logLines []LogLine) ([]string, error) {
	var result []string
	for _, ll := range logLines {
		event, err := logLineToEvent(ll)
		if err != nil {
			return []string{}, err
		}
		result = append(result, event)
	}
	return result, nil
}

func logLineToEvent(logLine LogLine) (string, error) {
	jsonDecoder := json.NewDecoder(strings.NewReader(logLine.Value))

	var data map[string]interface{}
	err := jsonDecoder.Decode(&data)
	if err != nil {
		return "", fmt.Errorf("convert logline to event; %w", err)
	}

	data["time"] = logLine.Timestamp.String()
	data["time_unix_nano"] = strconv.FormatInt(logLine.Timestamp.UnixNano(), 10)
	data["time_year"] = logLine.Timestamp.Year()
	data["time_month"] = logLine.Timestamp.Month()
	data["time_day"] = logLine.Timestamp.Day()

	result := bytes.NewBufferString("")
	jsonEncoder := json.NewEncoder(result)
	err = jsonEncoder.Encode(data)
	if err != nil {
		return "", err
	}

	return strings.Replace(result.String(), "\n", "", -1), nil
}

func (ec *EventsCollector) Name() string {
	return string(domain.CollectorTypeEvents)
}
