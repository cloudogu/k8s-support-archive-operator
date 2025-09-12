package loki

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
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
		var anError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, anError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, anError)

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimestamp), ""},
					{asString(lastestResultTimestamp), ""},
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

	t.Run("should calling API until the response is empty", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		resultTimestamp := startTime + daysToNanoSec(1)
		lastestResultTimestamp := endTime

		var callCount int
		var httpServerCalls []httpServerCall
		var anError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, anError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, anError)

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimestamp), "logline"},
					{asString(lastestResultTimestamp), "logline A"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 2 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(lastestResultTimestamp), "logline B"},
					{asString(lastestResultTimestamp), "logline C"},
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

		assert.Equal(t, startTime, httpServerCalls[0].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[0].reqEndTime)

		assert.Equal(t, lastestResultTimestamp, httpServerCalls[1].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[1].reqEndTime)

		assert.Equal(t, lastestResultTimestamp, httpServerCalls[2].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[2].reqEndTime)
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

	t.Run("should remove duplicate loglines", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		resultTimestamp1 := startTime + daysToNanoSec(1)
		resultTimestamp2 := startTime + daysToNanoSec(2)
		resultTimestamp3 := startTime + daysToNanoSec(3)

		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimestamp1), "logline_1"},
					{asString(resultTimestamp1), "logline_1"},
					{asString(resultTimestamp2), "logline_2"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}
			if callCount == 2 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimestamp1), "logline_1"},
					{asString(resultTimestamp2), "logline_2"},
					{asString(resultTimestamp3), "logline_3"},
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

		logLines, err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", "aKind")
		require.NoError(t, err)

		assert.Equal(t, 3, len(logLines))

		assert.True(t, containsLogLine(logLines, time.Unix(0, resultTimestamp1), "logline_1"))
		assert.True(t, containsLogLine(logLines, time.Unix(0, resultTimestamp2), "logline_2"))
		assert.True(t, containsLogLine(logLines, time.Unix(0, resultTimestamp3), "logline_3"))
	})

	t.Run("should issue an error if underlying error occurs", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)
		httpAPIUrl := "\n"

		lokiLogsPrv := NewLokiLogsProvider(http.DefaultClient, httpAPIUrl, "", "")
		_, err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", "aKind")

		assert.Error(t, err)
		assert.ErrorContains(t, err, "find logs")
		assert.NoError(t, errors.Unwrap(err)) // not expose implementation details through errors
	})

	t.Run("should issue an error if response can not be converted to LogLines", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(10)

		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{"not a timestamp", "logline_1"},
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

		lokiLogsPrv := NewLokiLogsProvider(http.DefaultClient, server.URL, "", "")
		_, err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", "aKind")

		assert.Error(t, err)
		assert.ErrorContains(t, err, "convert http response to LogLines")
		assert.NoError(t, errors.Unwrap(err)) // not expose implementation details through errors
	})

	t.Run("should issue an error if the result size exceeds the limit", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + daysToNanoSec(maxQueryTimeWindowInDays)

		responseMessage := "message from response"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(responseMessage))
			require.NoError(t, err)
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "")

		_, err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", "aKind")

		assert.Error(t, err)
		assert.ErrorContains(t, err, responseMessage)
		assert.NoError(t, errors.Unwrap(err)) // not expose implementation details through errors
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

	t.Run("error handling", func(t *testing.T) {
		_, err := buildFindLogsHttpQuery("\n", "", "", 0, 0)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "parse service URL")
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

func newQueryRangeResponse(values [][]string) ([]byte, error) {
	result := queryLogsResponse{
		Data: queryLogsData{
			ResultType: "stream",
			Result: []queryLogsResult{
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

func asString(value int64) string {
	return strconv.FormatInt(value, 10)
}
