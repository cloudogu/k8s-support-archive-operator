package collector

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollect(t *testing.T) {
	t.Run("should send EventSets to result channel", func(t *testing.T) {
		t.Skip("TODO: loki logs provider has changed")
		/*
			ctx := context.TODO()
			startTime := time.Now()
			endTime := startTime.AddDate(0, 0, 10)

			podTimestamp1 := startTime.AddDate(0, 0, 1)
			podTimestamp2 := startTime.AddDate(0, 0, 2)
			doguTimestamp1 := startTime.AddDate(0, 0, 1)

			logPrvMock := NewMockLogsProvider(t)
			logPrvMock.EXPECT().
				FindValuesOfLabel(ctx, startTime.UnixNano(), endTime.UnixNano(), "kind").
				Return([]string{"Pod", "Dogu"}, nil)
			logPrvMock.EXPECT().
				FindLogs(ctx, startTime.UnixNano(), endTime.UnixNano(), "aNamespace", "Pod").
				Return(
					[]LogLine{
						{Timestamp: podTimestamp1, Value: "{\"msg\":\"pod message 1\"}"},
						{Timestamp: podTimestamp2, Value: "{\"msg\":\"pod message 2\"}"},
					},
					nil,
				)
			logPrvMock.EXPECT().
				FindLogs(ctx, startTime.UnixNano(), endTime.UnixNano(), "aNamespace", "Dogu").
				Return(
					[]LogLine{
						{Timestamp: doguTimestamp1, Value: "{\"msg\":\"dogu message 1\"}"},
					},
					nil,
				)

			eventsCol := NewEventsCollector(logPrvMock)

			res := receiveResult()
			err := eventsCol.Collect(ctx, "aNamespace", startTime, endTime, res.channel)

			res.wait()
			require.NoError(t, err)

			assert.Equal(t, 2, len(res.eventSets))

			// EventSet 0: Pod
			assert.Equal(t, "aNamespace", res.eventSets[0].Namespace)
			assert.Equal(t, "Pod", res.eventSets[0].Kind)

			assert.Equal(t, 2, len(res.eventSets[0].Events))

			podMsg1, err := valueOfJsonField(res.eventSets[0].Events[0], "msg")
			require.NoError(t, err)
			assert.Equal(t, "pod message 1", podMsg1)

			podTimeYear1, err := valueOfJsonFieldInt(res.eventSets[0].Events[0], "time_year")
			require.NoError(t, err)
			assert.Equal(t, podTimestamp1.Year(), podTimeYear1)

			podMsg2, err := valueOfJsonField(res.eventSets[0].Events[1], "msg")
			require.NoError(t, err)
			assert.Equal(t, "pod message 2", podMsg2)

			podTimeYear2, err := valueOfJsonFieldInt(res.eventSets[0].Events[1], "time_year")
			require.NoError(t, err)
			assert.Equal(t, podTimestamp2.Year(), podTimeYear2)

			// EventSet 1: Dogu
			assert.Equal(t, "aNamespace", res.eventSets[1].Namespace)
			assert.Equal(t, "Dogu", res.eventSets[1].Kind)

			doguMsg1, err := valueOfJsonField(res.eventSets[1].Events[0], "msg")
			require.NoError(t, err)
			assert.Equal(t, "dogu message 1", doguMsg1)

			dogTimeYear1, err := valueOfJsonFieldInt(res.eventSets[1].Events[0], "time_year")
			require.NoError(t, err)
			assert.Equal(t, doguTimestamp1.Year(), dogTimeYear1)
		
		*/
	})

	t.Run("should convert LogLine to Event and add time fields", func(t *testing.T) {
		t.Skip("TODO: loki logs provider has changed")

		aTime := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		logLine := LogLine{
			Timestamp: aTime,
			Value:     "{\"msg\": \"dogu message 1\"}",
		}

		event, err := logLineToEvent(logLine)
		require.NoError(t, err)

		jsonMsg, err := valueOfJsonField(event, "msg")
		require.NoError(t, err)
		assert.Equal(t, "dogu message 1", jsonMsg)

		jsonTime, err := valueOfJsonField(event, "time")
		require.NoError(t, err)
		assert.Equal(t, "2009-11-17 20:34:58.651387237 +0000 UTC", jsonTime)

		jsonTimeUnixNano, err := valueOfJsonField(event, "time_unix_nano")
		require.NoError(t, err)
		assert.Equal(t, strconv.FormatInt(aTime.UnixNano(), 10), jsonTimeUnixNano)

		jsonTimeYear, err := valueOfJsonFieldInt(event, "time_year")
		require.NoError(t, err)
		assert.Equal(t, 2009, jsonTimeYear)

		jsonTimeMonth, err := valueOfJsonFieldInt(event, "time_month")
		require.NoError(t, err)
		assert.Equal(t, 11, jsonTimeMonth)

		jsonTimeDay, err := valueOfJsonFieldInt(event, "time_day")
		require.NoError(t, err)
		assert.Equal(t, 17, jsonTimeDay)

	})

	t.Run("should encode event as one line", func(t *testing.T) {
		t.Skip("TODO: loki logs provider has changed")

		aTime := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		logLine := LogLine{
			Timestamp: aTime,
			Value:     "{\"msg\": \"dogu message 1\"}",
		}

		event, err := logLineToEvent(logLine)
		require.NoError(t, err)

		assert.NotContains(t, event, "\n")
	})

	t.Run("error handling", func(t *testing.T) {
		t.Skip("TODO")
	})

}

type result struct {
	channel   chan *domain.EventSet
	eventSets []*domain.EventSet
	waitGroup sync.WaitGroup
}

func receiveResult() *result {
	res := &result{
		channel:   make(chan *domain.EventSet),
		eventSets: []*domain.EventSet{},
		waitGroup: sync.WaitGroup{},
	}
	res.receive()
	return res
}

func (res *result) receive() {
	res.waitGroup.Add(1)
	go func(resultChan <-chan *domain.EventSet) {
		for r := range res.channel {
			res.eventSets = append(res.eventSets, r)
		}
		res.waitGroup.Done()
	}(res.channel)
}

func (res *result) wait() {
	res.waitGroup.Wait()
}

func valueOfJsonField(jsonAsString string, field string) (string, error) {
	jsonDecoder := json.NewDecoder(strings.NewReader(jsonAsString))

	var decodedData map[string]interface{}
	err := jsonDecoder.Decode(&decodedData)
	if err != nil {
		return "", err
	}
	value, containsField := decodedData[field]
	if !containsField {
		return "", nil
	}
	return value.(string), nil
}

func valueOfJsonFieldInt(jsonAsString string, field string) (int, error) {
	jsonDecoder := json.NewDecoder(strings.NewReader(jsonAsString))
	jsonDecoder.UseNumber()

	var decodedData map[string]interface{}
	err := jsonDecoder.Decode(&decodedData)
	if err != nil {
		return 0, err
	}
	value, containsField := decodedData[field]
	if !containsField {
		return 0, nil
	}

	valueInt64, err := value.(json.Number).Int64()
	if err != nil {
		return 0, err
	}

	return int(valueInt64), nil
}
