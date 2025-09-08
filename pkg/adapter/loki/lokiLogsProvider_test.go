package loki

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type httpServerCall struct {
	reqStartTime int64
	reqEndTime   int64
	reqError     error
}

func TestLokiLogsProvider(t *testing.T) {

	t.Run("should call API one time if endTime == startTime + maxTimeWindow", func(t *testing.T) {
		startTime := time.Now()
		endTime := startTime.AddDate(0, 0, maxQueryTimeWindowInDays)

		var callCount int
		var reqStartTime, reqEndTime int64
		var reqError error

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			reqStartTime, reqEndTime, reqError = parseStartAndEndTime(r)
			require.NoError(t, reqError)
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL)

		_, _ = lokiLogsPrv.getValuesOfLabel(context.TODO(), startTime, endTime, "aKind")

		assert.Equal(t, 1, callCount)

		assert.Equal(t, startTime.UnixNano(), reqStartTime)
		assert.Equal(t, endTime.UnixNano(), reqEndTime)
	})

	t.Run("should call API two times if endTime == startTime + 2*maxTimeWindow ", func(t *testing.T) {
		startTime := time.Now()
		endTime := startTime.AddDate(0, 0, 2*maxQueryTimeWindowInDays)

		var callCount int
		var httpServerCalls []httpServerCall
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			reqStartTime, reqEndTime, reqError := parseStartAndEndTime(r)
			httpServerCalls = append(httpServerCalls, httpServerCall{
				reqStartTime: reqStartTime,
				reqEndTime:   reqEndTime,
				reqError:     reqError,
			})
			require.NoError(t, reqError)
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL)

		_, _ = lokiLogsPrv.getValuesOfLabel(context.TODO(), startTime, endTime, "aKind")

		assert.Equal(t, 2, callCount)

		assert.Equal(t, startTime.UnixNano(), httpServerCalls[0].reqStartTime)
		assert.Equal(t, startTime.AddDate(0, 0, maxQueryTimeWindowInDays).UnixNano(), httpServerCalls[0].reqEndTime)

		assert.Equal(t, startTime.AddDate(0, 0, maxQueryTimeWindowInDays).UnixNano(), httpServerCalls[1].reqStartTime)
		assert.Equal(t, startTime.AddDate(0, 0, 2*maxQueryTimeWindowInDays).UnixNano(), httpServerCalls[1].reqEndTime)
	})
}

func TestNextTimeWindow(t *testing.T) {
	t.Run("should calculate one time window if startTime + maxTimeWindow == maxEndTime", func(t *testing.T) {
		startTime := time.Now()
		var maxTimeWindowInDays = 20
		maxEndTime := startTime.AddDate(0, 0, 20)

		nextStart, nextEnd, hasNext := nextTimeWindow(startTime.UnixNano(), maxEndTime.UnixNano(), maxTimeWindowInDays)

		assert.Equal(t, startTime.UnixNano(), nextStart)
		assert.Equal(t, maxEndTime.UnixNano(), nextEnd)
		assert.False(t, hasNext)
	})

	t.Run("should calculate more than one time window if startTime + maxTimeWindow < maxEndTime", func(t *testing.T) {
		startTime := time.Now()
		var maxTimeWindowInDays = 20
		maxEndTime := startTime.AddDate(0, 0, 40)

		nextStart, nextEnd, hasNext := nextTimeWindow(startTime.UnixNano(), maxEndTime.UnixNano(), maxTimeWindowInDays)

		assert.Equal(t, startTime.UnixNano(), nextStart)
		assert.Equal(t, startTime.AddDate(0, 0, maxTimeWindowInDays).UnixNano(), nextEnd)
		assert.True(t, hasNext)
	})

	t.Run("should calculate two time windows if startTime + 2 * maxTimeWindow == maxEndTime", func(t *testing.T) {
		startTime := time.Now()
		var maxTimeWindowInDays = 24
		maxEndTime := startTime.AddDate(0, 0, 2*maxTimeWindowInDays)

		nextStart1, nextEnd1, hasNext1 := nextTimeWindow(startTime.UnixNano(), maxEndTime.UnixNano(), maxTimeWindowInDays)
		nextStart2, nextEnd2, hasNext2 := nextTimeWindow(nextEnd1, maxEndTime.UnixNano(), maxTimeWindowInDays)

		assert.Equal(t, startTime.UnixNano(), nextStart1)
		assert.Equal(t, startTime.AddDate(0, 0, maxTimeWindowInDays).UnixNano(), nextEnd1)
		assert.True(t, hasNext1)

		assert.Equal(t, startTime.AddDate(0, 0, maxTimeWindowInDays).UnixNano(), nextStart2)
		assert.Equal(t, maxEndTime.UnixNano(), nextEnd2)
		assert.False(t, hasNext2)
	})

	t.Run("should calculate two time windows where the second ends at maxEndTime if maxEndTime is inside the second time window", func(t *testing.T) {
		startTime := time.Now()
		var maxTimeWindowInDays = 20
		maxEndTime := startTime.AddDate(0, 0, 2*maxTimeWindowInDays-maxTimeWindowInDays/2)

		nextStart1, nextEnd1, hasNext1 := nextTimeWindow(startTime.UnixNano(), maxEndTime.UnixNano(), maxTimeWindowInDays)
		nextStart2, nextEnd2, hasNext2 := nextTimeWindow(nextEnd1, maxEndTime.UnixNano(), maxTimeWindowInDays)

		assert.Equal(t, startTime.UnixNano(), nextStart1)
		assert.Equal(t, startTime.AddDate(0, 0, maxTimeWindowInDays).UnixNano(), nextEnd1)
		assert.True(t, hasNext1)

		assert.Equal(t, startTime.AddDate(0, 0, maxTimeWindowInDays).UnixNano(), nextStart2)
		assert.Equal(t, maxEndTime.UnixNano(), nextEnd2)
		assert.False(t, hasNext2)
	})
}

func parseStartAndEndTime(r *http.Request) (int64, int64, error) {
	start, err := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.ParseInt(r.URL.Query().Get("end"), 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return start, end, nil
}
