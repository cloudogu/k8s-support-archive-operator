package loki

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/loki-find-values-of-label-response.json
var lokiValuesOfLabelResponse []byte

//go:embed testdata/loki-find-values-of-label-response-time-window-1.json
var lokiValuesOfLabelResponseTimeWindow1 []byte

//go:embed testdata/loki-find-values-of-label-response-time-window-2.json
var lokiValuesOfLabelResponseTimeWindow2 []byte

type httpServerCall struct {
	reqStartTime int64
	reqEndTime   int64
	reqError     error
}

func TestLokiLogsProviderFindValuesOfLabel(t *testing.T) {
	t.Run("should call API one time if endTime == startTime + maxTimeWindow", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		var callCount int
		var reqStartTime, reqEndTime int64
		var reqError error

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			reqStartTime, reqEndTime, reqError = parseStartAndEndTime(r)
			require.NoError(t, reqError)

			_, err := w.Write(lokiValuesOfLabelResponse)
			require.NoError(t, err)
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		_, err := lokiLogsPrv.FindValuesOfLabel(context.TODO(), startTime, endTime, "aKind")
		require.NoError(t, err)

		assert.Equal(t, 1, callCount)

		assert.Equal(t, startTime, reqStartTime)
		assert.Equal(t, endTime, reqEndTime)
	})

	t.Run("should call API three times if startTime + 2.5 * maxTimeWindow == endTime", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(2.5*maxQueryTimeWindowInDays)

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

			_, err := w.Write(lokiValuesOfLabelResponse)
			require.NoError(t, err)
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		_, err := lokiLogsPrv.FindValuesOfLabel(context.TODO(), startTime, endTime, "aKind")
		require.NoError(t, err)

		assert.Equal(t, 3, callCount)

		assert.Equal(t, startTime, httpServerCalls[0].reqStartTime)
		assert.Equal(t, startTime+daysToNanoSec(maxQueryTimeWindowInDays), httpServerCalls[0].reqEndTime)

		assert.Equal(t, startTime+daysToNanoSec(maxQueryTimeWindowInDays), httpServerCalls[1].reqStartTime)
		assert.Equal(t, startTime+daysToNanoSec(2*maxQueryTimeWindowInDays), httpServerCalls[1].reqEndTime)

		assert.Equal(t, startTime+daysToNanoSec(2*maxQueryTimeWindowInDays), httpServerCalls[2].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[2].reqEndTime)
	})

	t.Run("should parse response from API", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(10)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write(lokiValuesOfLabelResponse)
			require.NoError(t, err)
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		values, err := lokiLogsPrv.FindValuesOfLabel(context.TODO(), startTime, endTime, "aKind")
		require.NoError(t, err)

		assert.Equal(t, 2, len(values))
		assert.Contains(t, values, "Pod")
		assert.Contains(t, values, "PersistentVolumeClaim")
	})

	t.Run("should remove duplicates after consecutive api calls", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(2*maxQueryTimeWindowInDays)

		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			if callCount == 1 {
				_, err := w.Write(lokiValuesOfLabelResponseTimeWindow1)
				require.NoError(t, err)
			}
			if callCount == 2 {
				_, err := w.Write(lokiValuesOfLabelResponseTimeWindow2)
				require.NoError(t, err)
			}
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		values, err := lokiLogsPrv.FindValuesOfLabel(context.TODO(), startTime, endTime, "aKind")
		require.NoError(t, err)

		assert.Equal(t, 3, len(values))
		assert.Contains(t, values, "Pod")
		assert.Contains(t, values, "PersistentVolumeClaim")
		assert.Contains(t, values, "Dogu")
	})

	t.Run("should issue an error if underlying error occurs", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		lokiLogsPrv := NewLokiLogsProvider(http.DefaultClient, "", "", "")
		_, err := lokiLogsPrv.FindValuesOfLabel(context.TODO(), startTime, endTime, "aKind")

		assert.Error(t, err)
	})

	t.Run("should not expose underlying implementation errors", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		lokiLogsPrv := NewLokiLogsProvider(http.DefaultClient, "", "", "")
		_, err := lokiLogsPrv.FindValuesOfLabel(context.TODO(), startTime, endTime, "aKind")

		assert.NoError(t, errors.Unwrap(err))
	})

	t.Run("should use basic authentification", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(10)

		var reqAuthUsername string
		var reqAuthPassword string
		var ok bool
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqAuthUsername, reqAuthPassword, ok = r.BasicAuth()
			require.True(t, ok)

			_, err := w.Write(lokiValuesOfLabelResponse)
			require.NoError(t, err)
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "aUser", "aPassword")

		_, err := lokiLogsPrv.FindValuesOfLabel(context.TODO(), startTime, endTime, "aKind")
		require.NoError(t, err)

		assert.Equal(t, "aUser", reqAuthUsername)
		assert.Equal(t, "aPassword", reqAuthPassword)
	})

	t.Run("should issue an error if the time window exceeds the limit", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays+10)

		responseMessage := "message from response"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(responseMessage))
			require.NoError(t, err)
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		_, err := lokiLogsPrv.FindValuesOfLabel(context.TODO(), startTime, endTime, "aKind")

		assert.Error(t, err)
		assert.ErrorContains(t, err, responseMessage)
	})
}

func TestBuildFindValuesOfLabelHttpQuery(t *testing.T) {
	t.Run("should create url for querying values of a label", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		apiUrl, err := buildFindValuesOfLabelHttpQuery("http://example.com:8080", "aLabel", startTime, endTime)
		require.NoError(t, err)

		parsedApiUrl, err := url.Parse(apiUrl)
		require.NoError(t, err)

		assert.Equal(t, "http", parsedApiUrl.Scheme)
		assert.Equal(t, "example.com:8080", parsedApiUrl.Host)
		assert.Equal(t, "/loki/api/v1/label/aLabel/values", parsedApiUrl.Path)

		assert.Equal(t, 2, len(parsedApiUrl.Query()))
		assert.Equal(t, fmt.Sprintf("%v", startTime), parsedApiUrl.Query().Get("start"))
		assert.Equal(t, fmt.Sprintf("%v", endTime), parsedApiUrl.Query().Get("end"))
	})
}

func TestNextTimeWindow(t *testing.T) {
	t.Run("should calculate one time window if startTime + maxTimeWindow == maxEndTime", func(t *testing.T) {
		startTime := time.Now().Unix()
		var maxTimeWindowInDays = 20
		maxEndTime := startTime + daysToNanoSec(maxTimeWindowInDays)

		nextStart, nextEnd, hasNext := findValuesOfLabelNextTimeWindow(startTime, maxEndTime, maxTimeWindowInDays)

		assert.Equal(t, startTime, nextStart)
		assert.Equal(t, maxEndTime, nextEnd)
		assert.False(t, hasNext)
	})

	t.Run("should calculate more than one time window if startTime + maxTimeWindow < maxEndTime", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		var maxTimeWindowInDays = 20
		maxEndTime := startTime + daysToNanoSec(21)

		nextStart, nextEnd, hasNext := findValuesOfLabelNextTimeWindow(startTime, maxEndTime, maxTimeWindowInDays)

		assert.Equal(t, startTime, nextStart)
		assert.Equal(t, startTime+daysToNanoSec(maxTimeWindowInDays), nextEnd)
		assert.True(t, hasNext)
	})

	t.Run("should calculate two time windows if startTime + 2 * maxTimeWindow == maxEndTime", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		var maxTimeWindowInDays = 35
		maxEndTime := startTime + daysToNanoSec(2*maxTimeWindowInDays)

		nextStart1, nextEnd1, hasNext1 := findValuesOfLabelNextTimeWindow(startTime, maxEndTime, maxTimeWindowInDays)
		nextStart2, nextEnd2, hasNext2 := findValuesOfLabelNextTimeWindow(nextEnd1, maxEndTime, maxTimeWindowInDays)

		assert.Equal(t, startTime, nextStart1)
		assert.Equal(t, startTime+daysToNanoSec(maxTimeWindowInDays), nextEnd1)
		assert.True(t, hasNext1)

		assert.Equal(t, startTime+daysToNanoSec(maxTimeWindowInDays), nextStart2)
		assert.Equal(t, maxEndTime, nextEnd2)
		assert.False(t, hasNext2)
	})

	t.Run("should calculate two time windows where the second ends at maxEndTime if maxEndTime is inside the second time window", func(t *testing.T) {
		startTime := time.Now().Unix()
		var maxTimeWindowInDays = 20
		maxEndTime := startTime + daysToNanoSec(2*maxTimeWindowInDays-maxTimeWindowInDays/2)

		nextStart1, nextEnd1, hasNext1 := findValuesOfLabelNextTimeWindow(startTime, maxEndTime, maxTimeWindowInDays)
		nextStart2, nextEnd2, hasNext2 := findValuesOfLabelNextTimeWindow(nextEnd1, maxEndTime, maxTimeWindowInDays)

		assert.Equal(t, startTime, nextStart1)
		assert.Equal(t, startTime+daysToNanoSec(maxTimeWindowInDays), nextEnd1)
		assert.True(t, hasNext1)

		assert.Equal(t, startTime+daysToNanoSec(maxTimeWindowInDays), nextStart2)
		assert.Equal(t, maxEndTime, nextEnd2)
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
