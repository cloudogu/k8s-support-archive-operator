package collector

import (
	"context"
	"errors"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"sync"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCollect(t *testing.T) {
	t.Run("should call log provider to find logs", func(t *testing.T) {
		startTime := time.Now()
		endTime := startTime.AddDate(0, 0, 10)

		resultTimestamp1 := startTime.AddDate(0, 0, 1)
		resultTimestamp2 := startTime.AddDate(0, 0, 2)

		logPrvMock := NewMockLogsProvider(t)
		logPrvMock.EXPECT().
			FindEvents(testCtx, startTime.UnixNano(), endTime.UnixNano(), "aNamespace", mock.Anything).
			RunAndReturn(func(ctx context.Context, i int64, i2 int64, s string, results chan<- *domain.LogLine) error {
				results <- &domain.LogLine{Timestamp: resultTimestamp1, Value: "{\"msg\":\"message 1\"}"}
				results <- &domain.LogLine{Timestamp: resultTimestamp2, Value: "{\"msg\":\"message 2\"}"}
				return nil
			})

		eventsCol := NewEventsCollector(logPrvMock)

		res := receiveLogLinesResult()
		err := eventsCol.Collect(testCtx, "aNamespace", startTime, endTime, res.channel)

		res.wait()
		require.NoError(t, err)

		assert.Equal(t, 2, len(res.logLines))

		assert.Equal(t, resultTimestamp1, res.logLines[0].Timestamp)

		msg1, err := testutils.ValueOfJsonField(res.logLines[0].Value, "msg")
		require.NoError(t, err)
		assert.Equal(t, "message 1", msg1)

		assert.Equal(t, resultTimestamp2, res.logLines[1].Timestamp)

		msg2, err := testutils.ValueOfJsonField(res.logLines[1].Value, "msg")
		require.NoError(t, err)
		assert.Equal(t, "message 2", msg2)
	})

	t.Run("should issue an error if log provider returns one", func(t *testing.T) {
		startTime := time.Now()
		endTime := startTime.AddDate(0, 0, 10)

		logPrvMock := NewMockLogsProvider(t)
		logPrvMock.EXPECT().
			FindEvents(testCtx, startTime.UnixNano(), endTime.UnixNano(), "aNamespace", mock.Anything).
			RunAndReturn(func(ctx context.Context, i int64, i2 int64, s string, results chan<- *domain.LogLine) error {
				return errors.New("a log provider error")
			})

		eventsCol := NewEventsCollector(logPrvMock)

		res := receiveLogLinesResult()
		err := eventsCol.Collect(testCtx, "aNamespace", startTime, endTime, res.channel)

		res.wait()

		assert.Error(t, err)
		assert.ErrorContains(t, err, "call log provider")
		assert.NoError(t, errors.Unwrap(err)) // not expose implementation details through errors

	})
}

type logLineResult struct {
	channel   chan *domain.LogLine
	logLines  []*domain.LogLine
	waitGroup sync.WaitGroup
}

func receiveLogLinesResult() *logLineResult {
	res := &logLineResult{
		channel:   make(chan *domain.LogLine),
		logLines:  []*domain.LogLine{},
		waitGroup: sync.WaitGroup{},
	}
	res.receive()
	return res
}

func (res *logLineResult) receive() {
	res.waitGroup.Add(1)
	go func(resultChan <-chan *domain.LogLine) {
		for r := range res.channel {
			res.logLines = append(res.logLines, r)
		}
		res.waitGroup.Done()
	}(res.channel)
}

func (res *logLineResult) wait() {
	res.waitGroup.Wait()
}
