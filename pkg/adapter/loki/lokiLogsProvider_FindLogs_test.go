package loki

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/loki-find-logs-response.json
var lokiFindLogsResponse []byte

//go:embed testdata/loki-find-logs-response-empty.json
var lokiFindLogsEmptyResponse []byte

var (
	testMaxQueryTimeWindow  = time.Hour * 24 * 30
	testMaxQueryResultCount = 2000
)

func TestLokiLogsProviderFindLogs(t *testing.T) {
	closeChannelAfterLastReadDuration := 5 * time.Millisecond

	t.Run("should start next call with the latest timestamp from result", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + testMaxQueryTimeWindow.Nanoseconds()

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
					{asString(resultTimestamp), "{\"msg\":\"msg1\"}"},
					{asString(lastestResultTimestamp), "{\"msg\":\"msg1\"}"},
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

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "", testMaxQueryResultCount, testMaxQueryTimeWindow)

		res := receiveLogLineResults(closeChannelAfterLastReadDuration)
		err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", res.channel)

		res.wait()
		require.NoError(t, err)

		assert.Equal(t, 2, callCount)

		assert.Equal(t, startTime, httpServerCalls[0].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[0].reqEndTime)

		assert.Equal(t, lastestResultTimestamp, httpServerCalls[1].reqStartTime)
		assert.Equal(t, endTime, httpServerCalls[1].reqEndTime)
	})

	t.Run("should calling API until the response is empty", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + testMaxQueryTimeWindow.Nanoseconds()

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
					{asString(resultTimestamp), "{\"msg\":\"msg1\"}"},
					{asString(lastestResultTimestamp), "{\"msg\":\"msg2\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 2 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(lastestResultTimestamp), "{\"msg\":\"msg3\"}"},
					{asString(lastestResultTimestamp), "{\"msg\":\"msg4\"}"},
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

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "", testMaxQueryResultCount, testMaxQueryTimeWindow)

		res := receiveLogLineResults(closeChannelAfterLastReadDuration)
		err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", res.channel)

		res.wait()
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

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "", testMaxQueryResultCount, testMaxQueryTimeWindow)

		res := receiveLogLineResults(closeChannelAfterLastReadDuration)
		err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "ecosystem", res.channel)

		res.wait()
		require.NoError(t, err)

		assert.Equal(t, 6, len(res.logLines))

		assert.True(t, containsLogLine(res.logLines, time.Unix(0, 1757507346000000000), "\"count\":1280"))
		assert.True(t, containsLogLine(res.logLines, time.Unix(0, 1757507084000000000), "\"count\":1259"))
		assert.True(t, containsLogLine(res.logLines, time.Unix(0, 1757506748000000000), "\"count\":1243"))
		assert.True(t, containsLogLine(res.logLines, time.Unix(0, 1757500152000000000), "\"count\":41"))
		assert.True(t, containsLogLine(res.logLines, time.Unix(0, 1757486049000000000), "\"count\":8"))
		assert.True(t, containsLogLine(res.logLines, time.Unix(0, 1757484951000000000), "\"count\":4"))
	})

	t.Run("should append time fields to http response", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + testMaxQueryTimeWindow.Nanoseconds()

		resultTimestamp := startTime + daysToNanoSec(1)

		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimestamp), "{\"msg\":\"msg1\"}"},
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

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "", testMaxQueryResultCount, testMaxQueryTimeWindow)

		res := receiveLogLineResults(closeChannelAfterLastReadDuration)
		err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "ecosystem", res.channel)

		res.wait()
		require.NoError(t, err)

		assert.True(t, testutils.ContainsJsonField(res.logLines[0].Value, "time"))
		assert.True(t, testutils.ContainsJsonField(res.logLines[0].Value, "time_unix_nano"))
		assert.True(t, testutils.ContainsJsonField(res.logLines[0].Value, "time_year"))
		assert.True(t, testutils.ContainsJsonField(res.logLines[0].Value, "time_month"))
		assert.True(t, testutils.ContainsJsonField(res.logLines[0].Value, "time_day"))

	})

	t.Run("should append time fields to LogLine.Value", func(t *testing.T) {
		aTime := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		logLine := domain.LogLine{
			Timestamp: aTime,
			Value:     "{\"msg\": \"dogu message 1\"}",
		}

		logLine, err := appendTimeFields(logLine)
		require.NoError(t, err)

		jsonMsg, err := testutils.ValueOfJsonField(logLine.Value, "msg")
		require.NoError(t, err)
		assert.Equal(t, "dogu message 1", jsonMsg)

		jsonTime, err := testutils.ValueOfJsonField(logLine.Value, "time")
		require.NoError(t, err)
		assert.Equal(t, "2009-11-17 20:34:58.651387237 +0000 UTC", jsonTime)

		jsonTimeUnixNano, err := testutils.ValueOfJsonField(logLine.Value, "time_unix_nano")
		require.NoError(t, err)
		assert.Equal(t, strconv.FormatInt(aTime.UnixNano(), 10), jsonTimeUnixNano)

		jsonTimeYear, err := testutils.ValueOfJsonFieldInt(logLine.Value, "time_year")
		require.NoError(t, err)
		assert.Equal(t, 2009, jsonTimeYear)

		jsonTimeMonth, err := testutils.ValueOfJsonFieldInt(logLine.Value, "time_month")
		require.NoError(t, err)
		assert.Equal(t, 11, jsonTimeMonth)

		jsonTimeDay, err := testutils.ValueOfJsonFieldInt(logLine.Value, "time_day")
		require.NoError(t, err)
		assert.Equal(t, 17, jsonTimeDay)

	})

	t.Run("should encode LogLine.Value as one line", func(t *testing.T) {
		aTime := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
		baseLogLine := domain.LogLine{
			Timestamp: aTime,
			Value:     "{\"msg\": \"dogu message 1\"}",
		}

		logLine, err := appendTimeFields(baseLogLine)
		require.NoError(t, err)

		assert.NotContains(t, logLine.Value, "\n")
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

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "aUser", "aPassword", 0, time.Hour)

		res := receiveLogLineResults(closeChannelAfterLastReadDuration)
		err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", res.channel)

		res.wait()
		require.NoError(t, err)

		assert.Equal(t, "aUser", reqAuthUsername)
		assert.Equal(t, "aPassword", reqAuthPassword)
	})

	t.Run("should issue an error if underlying error occurs", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + testMaxQueryTimeWindow.Nanoseconds()
		httpAPIUrl := "\n"

		lokiLogsPrv := NewLokiLogsProvider(http.DefaultClient, httpAPIUrl, "", "", 0, time.Hour)

		res := receiveLogLineResults(closeChannelAfterLastReadDuration)
		err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", res.channel)

		res.wait()

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

		lokiLogsPrv := NewLokiLogsProvider(http.DefaultClient, server.URL, "", "", 0, time.Hour)

		res := receiveLogLineResults(closeChannelAfterLastReadDuration)
		err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", res.channel)

		res.wait()

		assert.Error(t, err)
		assert.ErrorContains(t, err, "convert http response to LogLines")
		assert.NoError(t, errors.Unwrap(err)) // not expose implementation details through errors
	})

	t.Run("should issue an error if the result size exceeds the limit", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + testMaxQueryTimeWindow.Nanoseconds()

		responseMessage := "message from response"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(responseMessage))
			require.NoError(t, err)
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client(), server.URL, "", "", 0, time.Hour)

		res := receiveLogLineResults(closeChannelAfterLastReadDuration)
		err := lokiLogsPrv.FindLogs(context.TODO(), startTime, endTime, "aNamespace", res.channel)

		res.wait()

		assert.Error(t, err)
		assert.ErrorContains(t, err, responseMessage)
		assert.NoError(t, errors.Unwrap(err)) // not expose implementation details through errors
	})
}

func TestBuildFindLogsHttpQuery(t *testing.T) {
	t.Run("should create url for querying logs", func(t *testing.T) {
		startTime := time.Now().UnixNano()
		endTime := startTime + testMaxQueryTimeWindow.Nanoseconds()

		apiUrl, err := buildFindLogsHttpQuery("http://example.com:8080", "aNamespace", startTime, endTime, testMaxQueryResultCount)
		require.NoError(t, err)

		parsedApiUrl, err := url.Parse(apiUrl)
		require.NoError(t, err)

		assert.Equal(t, "http", parsedApiUrl.Scheme)
		assert.Equal(t, "example.com:8080", parsedApiUrl.Host)
		assert.Equal(t, "/loki/api/v1/query_range", parsedApiUrl.Path)

		assert.Equal(t, 4, len(parsedApiUrl.Query()))
		assert.Equal(t, "{namespace=\"aNamespace\"}", parsedApiUrl.Query().Get("query"))
		assert.Equal(t, fmt.Sprintf("%v", startTime), parsedApiUrl.Query().Get("start"))
		assert.Equal(t, fmt.Sprintf("%v", endTime), parsedApiUrl.Query().Get("end"))
		assert.Equal(t, fmt.Sprintf("%v", testMaxQueryResultCount), parsedApiUrl.Query().Get("limit"))
	})

	t.Run("should issue an error if service url is not a valid", func(t *testing.T) {
		_, err := buildFindLogsHttpQuery("\n", "", 0, 0, 0)

		assert.Error(t, err)
		assert.ErrorContains(t, err, "parse service URL")
	})
}

func containsLogLine(logLines []*domain.LogLine, timestamp time.Time, valueContains string) bool {
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

type httpServerCall struct {
	reqStartTime int64
	reqEndTime   int64
	reqError     error
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

type logLineResult struct {
	channel   chan *domain.LogLine
	logLines  []*domain.LogLine
	waitGroup sync.WaitGroup
}

func (res *logLineResult) receive(closeChannelAfterLastRead time.Duration) {
	res.waitGroup.Add(1)
	timer := time.NewTimer(closeChannelAfterLastRead)
	go func(channel <-chan *domain.LogLine) {
		defer res.waitGroup.Done()
		for {
			select {
			case <-timer.C:
				return
			case ll, isOpen := <-res.channel:
				if isOpen {
					res.logLines = append(res.logLines, ll)
					timer.Reset(closeChannelAfterLastRead)
				} else {
					return
				}
			}
		}
	}(res.channel)
}

func (res *logLineResult) wait() {
	res.waitGroup.Wait()
}

func receiveLogLineResults(closeChannelAfterLastRead time.Duration) *logLineResult {
	res := &logLineResult{
		channel:   make(chan *domain.LogLine),
		logLines:  []*domain.LogLine{},
		waitGroup: sync.WaitGroup{},
	}
	res.receive(closeChannelAfterLastRead)
	return res
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
