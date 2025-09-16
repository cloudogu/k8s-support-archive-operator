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

func (ec *EventsCollector) Collect(ctx context.Context, namespace string, startTime, endTime time.Time, resultChan chan<- *domain.LogLine) error {
	defer close(resultChan)

	err := ec.logsProvider.FindLogs(ctx, startTime.UnixNano(), endTime.UnixNano(), namespace, resultChan)
	if err != nil {
		return err
	}

	return nil
}

func logLinesToEvents(logLines []domain.LogLine) ([]string, error) {
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

func logLineToEvent(logLine domain.LogLine) (string, error) {
	jsonDecoder := json.NewDecoder(strings.NewReader(logLine.Value))

	var data map[string]interface{}
	err := jsonDecoder.Decode(&data)
	if err != nil {
		return "", fmt.Errorf("decode logline; %w", err)
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
		return "", fmt.Errorf("encode event")
	}

	return strings.Replace(result.String(), "\n", "", -1), nil
}

func (ec *EventsCollector) Name() string {
	return string(domain.CollectorTypeEvents)
}
