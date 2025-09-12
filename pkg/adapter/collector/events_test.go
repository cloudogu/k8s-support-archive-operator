package collector

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollect(t *testing.T) {
	t.Run("should send EventSets to result channel", func(t *testing.T) {
		t.Skip("TODO")

		ctx := context.TODO()
		startTime := time.Now()
		endTime := startTime.AddDate(0, 0, 10)

		logPrvMock := NewMockLogsProvider(t)
		logPrvMock.EXPECT().
			FindValuesOfLabel(ctx, startTime.UnixNano(), endTime.UnixNano(), "kind").
			Return([]string{"Pod", "Dogu"}, nil)
		logPrvMock.EXPECT().
			FindLogs(ctx, startTime.UnixNano(), endTime.UnixNano(), "aNamespace", "Pod").
			Return(
				[]LogLine{
					{Timestamp: startTime.AddDate(0, 0, 1), Value: "{msg=\"pod message 1\""},
					{Timestamp: startTime.AddDate(0, 0, 2), Value: "{msg=\"pod message 2\""},
				},
				nil,
			)
		logPrvMock.EXPECT().
			FindLogs(ctx, startTime.UnixNano(), endTime.UnixNano(), "aNamespace", "Dogu").
			Return(
				[]LogLine{
					{Timestamp: startTime.AddDate(0, 0, 1), Value: "{msg=\"dogu message 1\""},
				},
				nil,
			)

		eventsCol := NewEventsCollector(logPrvMock)

		res := receiveResult()
		err := eventsCol.Collect(ctx, "aNamespace", startTime, endTime, res.channel)

		res.wait()
		require.NoError(t, err)

		assert.Equal(t, 2, len(res.eventSets))

		assert.Equal(t, "aNamespace", res.eventSets[0].Namespace)
		assert.Equal(t, "Pod", res.eventSets[0].Kind)

		assert.Equal(t, 2, len(res.eventSets[0].Events))
		assert.Equal(t, "pod message 1", res.eventSets[0].Events[0].Message)
		assert.Equal(t, "pod message 2", res.eventSets[0].Events[1].Message)

		assert.Equal(t, "aNamespace", res.eventSets[1].Namespace)
		assert.Equal(t, "Pod", res.eventSets[1].Kind)

		assert.Equal(t, 1, len(res.eventSets[1].Events))
		assert.Equal(t, "dogu message 1", res.eventSets[1].Events[0].Message)
	})

	t.Run("should create new EventSet", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("should convert LogLine to Event", func(t *testing.T) {
		t.Skip("TODO")
	})

	t.Run("should calculate event's year, month and day fields", func(t *testing.T) {
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
