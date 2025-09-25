package collector

import (
	"sync"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollect(t *testing.T) {
	t.Run("should call log provider to find events and close result channel", func(t *testing.T) {
		// given
		startTime := time.Now()
		endTime := startTime.AddDate(0, 0, 10)
		resultChannel := make(chan *domain.LogLine)

		logPrvMock := NewMockLogsProvider(t)
		logPrvMock.EXPECT().FindEvents(testCtx, startTime, endTime, testNamespace, mock.Anything).Return(nil)

		group := sync.WaitGroup{}
		group.Add(1)
		timeout := time.NewTimer(time.Minute)
		failed := false
		go func() {
			for {
				select {
				case _, closed := <-resultChannel:
					if !closed {
						group.Done()
						return
					}
				case <-timeout.C:
					failed = true
					group.Done()
					return
				}
			}
		}()

		sut := NewEventsCollector(logPrvMock)

		// when
		err := sut.Collect(testCtx, testNamespace, startTime, endTime, resultChannel)

		// then
		group.Wait()
		if failed {
			t.Fatal("failed waiting for channel closing")
		}
		require.NoError(t, err)
	})

	t.Run("should issue an error if log provider returns one", func(t *testing.T) {
		// given
		startTime := time.Now()
		endTime := startTime.AddDate(0, 0, 10)
		resultChannel := make(chan *domain.LogLine)

		logPrvMock := NewMockLogsProvider(t)
		logPrvMock.EXPECT().FindEvents(testCtx, startTime, endTime, testNamespace, mock.Anything).Return(assert.AnError)

		group := sync.WaitGroup{}
		group.Add(1)
		timeout := time.NewTimer(time.Minute)
		failed := false
		go func() {
			for {
				select {
				case _, closed := <-resultChannel:
					if !closed {
						group.Done()
						return
					}
				case <-timeout.C:
					failed = true
					group.Done()
					return
				}
			}
		}()

		eventsCol := NewEventsCollector(logPrvMock)

		// when
		err := eventsCol.Collect(testCtx, testNamespace, startTime, endTime, resultChannel)

		// then
		group.Wait()
		if failed {
			t.Fatal("failed waiting for channel closing")
		}

		assert.Error(t, err)
		assert.ErrorContains(t, err, "error finding events")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestEventsCollector_Name(t *testing.T) {
	t.Run("should return correct name", func(t *testing.T) {
		sut := &EventsCollector{}

		assert.Equal(t, "Events", sut.Name())
	})
}
