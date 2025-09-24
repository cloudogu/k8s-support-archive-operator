package collector

import (
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestLogCollector_Collect(t *testing.T) {
	t.Run("should call log provider to find logs and close result channel", func(t *testing.T) {
		// given
		startTime := time.Now()
		endTime := startTime.AddDate(0, 0, 10)
		resultChannel := make(chan *domain.LogLine)

		logPrvMock := NewMockLogsProvider(t)
		logPrvMock.EXPECT().FindLogs(testCtx, startTime, endTime, testNamespace, mock.Anything).Return(nil)

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

		sut := NewLogCollector(logPrvMock)

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
		resultChannel := make(chan<- *domain.LogLine)

		logPrvMock := NewMockLogsProvider(t)
		logPrvMock.EXPECT().FindLogs(testCtx, startTime, endTime, testNamespace, resultChannel).Return(assert.AnError)

		logsCol := NewLogCollector(logPrvMock)

		// when
		err := logsCol.Collect(testCtx, testNamespace, startTime, endTime, resultChannel)

		// then
		assert.Error(t, err)
		assert.ErrorContains(t, err, "failed to find logs")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestLogCollector_Name(t *testing.T) {
	t.Run("should return correct name", func(t *testing.T) {
		sut := &LogCollector{}

		assert.Equal(t, "Logs", sut.Name())
	})
}
