package loki

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/loki-find-logs-response.json
var lokiFindLogsResponse []byte

//go:embed testdata/loki-find-logs-response-empty.json
var lokiFindLogsEmptyResponse []byte

func TestLokiLogsProviderFindLogs(t *testing.T) {
	t.Run("should start next call with the latest timestamp from result", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		resultTimestamp := startTime + daysToNanoSec(1)
		lastestResultTimestamp := startTime + daysToNanoSec(2)

		var callCount int
		var httpServerCalls []httpServerCall
		var callError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, callError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, callError)

			if callCount == 1 {
				resp, err := newQueryRangeResponseWithResultTimestamps([]int64{
					resultTimestamp,
					lastestResultTimestamp,
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 2 {
				_, err := w.Write(lokiFindLogsEmptyResponse)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		_, err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", "aKind")
		require.NoError(t, err)

		assert.Equal(t, 2, callCount)

		assert.Equal(t, startTime, httpServerCalls[0].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[0].reqEndTime)

		assert.Equal(t, lastestResultTimestamp, httpServerCalls[1].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[1].reqEndTime)
	})

	t.Run("should call API if latest timestamp == endTime", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		resultTimestamp := startTime + daysToNanoSec(1)
		lastestResultTimestamp := endTime

		var callCount int
		var httpServerCalls []httpServerCall
		var callError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, callError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, callError)

			if callCount == 1 {
				resp, err := newQueryRangeResponseWithResultTimestamps([]int64{
					resultTimestamp,
					lastestResultTimestamp,
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 2 {
				_, err := w.Write(lokiFindLogsEmptyResponse)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		_, err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", "aKind")
		require.NoError(t, err)

		assert.Equal(t, 2, callCount)

		assert.Equal(t, startTime, httpServerCalls[0].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[0].reqEndTime)

		assert.Equal(t, lastestResultTimestamp, httpServerCalls[1].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[1].reqEndTime)
	})

	t.Run("should stop calling API if an empty response was received", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		resultTimestamp := startTime + daysToNanoSec(1)
		lastestResultTimestamp := endTime

		var callCount int
		var httpServerCalls []httpServerCall
		var callError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, callError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, callError)

			if callCount == 1 {
				resp, err := newQueryRangeResponseWithResultTimestamps([]int64{
					resultTimestamp,
					lastestResultTimestamp,
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 2 {
				resp, err := newQueryRangeResponseWithResultTimestamps([]int64{
					lastestResultTimestamp,
					lastestResultTimestamp,
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 3 {
				_, err := w.Write(lokiFindLogsEmptyResponse)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		_, err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", "aKind")
		require.NoError(t, err)

		assert.Equal(t, 3, callCount)
	})

	t.Run("should parse response from API", func(t *testing.T) {
		var startTime int64 = 1757484951000000000 // earliest timestamp in result
		var endTime int64 = 1757507346000000000   // latest timestamp in result

		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			if callCount == 1 {
				_, err := w.Write(lokiFindLogsResponse)
				require.NoError(t, err)
			}
			if callCount == 2 {
				_, err := w.Write(lokiFindLogsEmptyResponse)
				require.NoError(t, err)
			}
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		logLines, err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "ecosystem", "Pod")
		require.NoError(t, err)

		assert.Equal(t, 6, len(logLines))

		assert.True(t, containsLogLine(logLines, time.Unix(0, 1757507346000000000), "\"count\":1280"))
		assert.True(t, containsLogLine(logLines, time.Unix(0, 1757507084000000000), "\"count\":1259"))
		assert.True(t, containsLogLine(logLines, time.Unix(0, 1757506748000000000), "\"count\":1243"))
		assert.True(t, containsLogLine(logLines, time.Unix(0, 1757500152000000000), "\"count\":41"))
		assert.True(t, containsLogLine(logLines, time.Unix(0, 1757486049000000000), "\"count\":8"))
		assert.True(t, containsLogLine(logLines, time.Unix(0, 1757484951000000000), "\"count\":4"))
	})

	t.Run("should use basic authentification", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(10)

		var reqAuthUsername string
		var reqAuthPassword string
		var ok bool
		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			if callCount == 1 {
				reqAuthUsername, reqAuthPassword, ok = r.BasicAuth()
				require.True(t, ok)

				_, err := w.Write(lokiFindLogsResponse)
				require.NoError(t, err)
			}

			if callCount == 2 {
				_, err := w.Write(lokiFindLogsEmptyResponse)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "aUser", "aPassword")

		_, err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", "aKind")
		require.NoError(t, err)

		assert.Equal(t, "aUser", reqAuthUsername)
		assert.Equal(t, "aPassword", reqAuthPassword)
	})
}

func TestBuildFindLogsHttpQuery(t *testing.T) {
	t.Run("should create url for querying logs", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		apiUrl, err := buildFindLogsHttpQuery("http://example.com:8080", "aNamespace", "aKind", startTime, endTime)
		require.NoError(t, err)

		parsedApiUrl, err := url.Parse(apiUrl)
		require.NoError(t, err)

		assert.Equal(t, "http", parsedApiUrl.Scheme)
		assert.Equal(t, "example.com:8080", parsedApiUrl.Host)
		assert.Equal(t, "/loki/api/v1/query_range", parsedApiUrl.Path)

		assert.Equal(t, 4, len(parsedApiUrl.Query()))
		assert.Equal(t, "{namespace=\"aNamespace\", kind=\"aKind\"}", parsedApiUrl.Query().Get("query"))
		assert.Equal(t, fmt.Sprintf("%v", startTime), parsedApiUrl.Query().Get("start"))
		assert.Equal(t, fmt.Sprintf("%v", endTime), parsedApiUrl.Query().Get("end"))
		assert.Equal(t, fmt.Sprintf("%v", maxQueryResultCount), parsedApiUrl.Query().Get("limit"))
	})
}

func containsLogLine(logLines []collector.LogLine, timestamp time.Time, valueContains string) bool {
	for _, ll := range logLines {
		if ll.Timestamp.Equal(timestamp) && strings.Contains(ll.Value, valueContains) {
			return true
		}
	}
	return false
}

func newQueryRangeResponseWithResultTimestamps(timestamps []int64) ([]byte, error) {
	var values [][]string
	for _, ts := range timestamps {
		values = append(values, []string{strconv.FormatInt(ts, 10), ""})
	}

	result := queryRangeResponse{
		Data: queryRangeData{
			ResultType: "stream",
			Result: []queryRangeResult{
				{
					Values: values,
				},
			},
		},
	}
	return json.Marshal(result)
}

func appendHttpServerCall(calls []httpServerCall, request *http.Request) ([]httpServerCall, error) {
	reqStartTime, reqEndTime, err := parseStartAndEndTime(request)
	if err != nil {
		return []httpServerCall{}, err
	}
	result := append(calls, httpServerCall{
		reqStartTime: reqStartTime,
		reqEndTime:   reqEndTime,
		reqError:     err,
	})
	return result, nil
}
