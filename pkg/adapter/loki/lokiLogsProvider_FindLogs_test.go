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
	"sync"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/config"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
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
	testStartTime           = time.Now()
)

func TestLokiLogsProviderFindLogs(t *testing.T) {
	t.Run("should call API once if result size < limit and queried time == max time window", func(t *testing.T) {
		endTime := testStartTime.Add(time.Hour * 24 * 10)

		result1Timestamp := testStartTime.Add(time.Hour * 24 * 2)
		result2Timestamp := testStartTime.Add(time.Hour * 24 * 5)

		var callCount int
		var httpServerCalls []httpServerCall
		var anError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, anError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, anError)

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(result1Timestamp.UnixNano()), "{\"msg\":\"msg1\"}"},
					{asString(result2Timestamp.UnixNano()), "{\"msg\":\"msg2\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProviderWithLimits(server.Client(), server.URL, 10, 3)

		res := receiveLogLineResults(2)
		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", res.channel)

		res.wait()
		require.NoError(t, err)

		assert.Equal(t, 1, callCount)
		assert.True(t, testStartTime.Equal(httpServerCalls[0].reqStart))
		assert.True(t, endTime.Equal(httpServerCalls[0].reqEnd))
	})

	t.Run("should call API twice using the latest result timestamp as start time if result size == limit and queried time == max time window", func(t *testing.T) {
		endTime := testStartTime.Add(time.Hour * 24 * 10)

		result1Timestamp := testStartTime.Add(time.Hour * 24 * 2)
		result2Timestamp := testStartTime.Add(time.Hour * 24 * 5)
		result3Timestamp := testStartTime.Add(time.Hour * 24 * 8)

		result1TimestampSecondCall := testStartTime.Add(time.Hour * 24 * 9)

		var callCount int
		var httpServerCalls []httpServerCall
		var anError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, anError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, anError)

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(result1Timestamp.UnixNano()), "{\"msg\":\"msg1\"}"},
					{asString(result2Timestamp.UnixNano()), "{\"msg\":\"msg2\"}"},
					{asString(result3Timestamp.UnixNano()), "{\"msg\":\"msg3\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 2 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(result1TimestampSecondCall.UnixNano()), "{\"msg\":\"msg1 second call\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProviderWithLimits(server.Client(), server.URL, 10, 3)

		defer func() {
			// recover panic if the channel is closed correctly from the test
			if r := recover(); r != nil {
				assert.Failf(t, "test failed", "implementation wrote to channel but expected no element: %v", r)
				return
			}
		}()

		res := receiveLogLineResults(4)
		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", res.channel)

		res.wait()
		require.NoError(t, err)

		assert.Equal(t, 2, callCount)

		assert.True(t, testStartTime.Equal(httpServerCalls[0].reqStart))
		assert.True(t, endTime.Equal(httpServerCalls[0].reqEnd))

		assert.True(t, result3Timestamp.Equal(httpServerCalls[1].reqStart))
		assert.True(t, endTime.Equal(httpServerCalls[1].reqEnd))
	})

	t.Run("should call API twice if queried time == 2 * max time window", func(t *testing.T) {
		endTime := testStartTime.Add(time.Hour * 24 * 20)

		resultTimeStampFirstResponse := testStartTime.Add(time.Hour * 24 * 2)
		resultTimeStampSecondResponse := testStartTime.Add(time.Hour * 24 * 15)

		var callCount int
		var httpServerCalls []httpServerCall
		var anError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, anError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, anError)

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimeStampFirstResponse.UnixNano()), "{\"msg\":\"first response\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 2 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimeStampSecondResponse.UnixNano()), "{\"msg\":\"second response\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProviderWithLimits(server.Client(), server.URL, 10, 3)

		res := receiveLogLineResults(2)
		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", res.channel)

		res.wait()
		require.NoError(t, err)

		assert.Equal(t, 2, callCount)

		assert.True(t, testStartTime.Equal(httpServerCalls[0].reqStart))
		assert.True(t, testStartTime.Add(time.Hour*24*10).Equal(httpServerCalls[0].reqEnd))

		assert.True(t, testStartTime.Add(time.Hour*24*10).Equal(httpServerCalls[1].reqStart))
		assert.True(t, endTime.Equal(httpServerCalls[1].reqEnd))
	})

	t.Run("should be able to handle empty results", func(t *testing.T) {
		endTime := testStartTime.Add(time.Hour * 24 * 30)

		resultTimeStampFirstResponse := testStartTime.Add(time.Hour * 24 * 5)
		resultTimeStampThirdResponse := testStartTime.Add(time.Hour * 24 * 25)

		var callCount int
		var httpServerCalls []httpServerCall
		var anError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, anError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, anError)

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimeStampFirstResponse.UnixNano()), "{\"msg\":\"first response\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 2 {
				_, err := w.Write(lokiFindLogsEmptyResponse)
				require.NoError(t, err)
			}

			if callCount == 3 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimeStampThirdResponse.UnixNano()), "{\"msg\":\"third response\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProviderWithLimits(server.Client(), server.URL, 10, 3)

		res := receiveLogLineResults(2)
		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", res.channel)

		res.wait()

		assert.NoError(t, err)
		assert.Equal(t, 3, callCount)
		assert.Equal(t, 2, len(res.logLines))
	})

	t.Run("should call API twice with a shorter second time window if end time == 1.5 * max time window", func(t *testing.T) {
		endTime := testStartTime.Add(time.Hour * 24 * 15)

		resultTimeStampFirstResponse := testStartTime.Add(time.Hour * 24 * 8)
		resultTimeStampSecondResponse := testStartTime.Add(time.Hour * 24 * 12)

		var callCount int
		var httpServerCalls []httpServerCall
		var anError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, anError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, anError)

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimeStampFirstResponse.UnixNano()), "{\"msg\":\"first response\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

			if callCount == 2 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimeStampSecondResponse.UnixNano()), "{\"msg\":\"second response\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProviderWithLimits(server.Client(), server.URL, 10, 3)

		res := receiveLogLineResults(2)
		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", res.channel)

		require.NoError(t, err)
		res.wait()

		assert.Equal(t, 2, callCount)

		assert.True(t, testStartTime.Equal(httpServerCalls[0].reqStart))
		assert.True(t, testStartTime.Add(time.Hour*24*10).Equal(httpServerCalls[0].reqEnd))

		assert.True(t, testStartTime.Add(time.Hour*24*10).Equal(httpServerCalls[1].reqStart))
		assert.True(t, endTime.Equal(httpServerCalls[1].reqEnd))
	})

	t.Run("should calculate next time window", func(t *testing.T) {
		endTime := testStartTime.Add(time.Hour * 24 * 20)

		nextStart, nextEnd := findLogsNextTimeWindow(testStartTime, endTime, time.Hour*24*10)
		assert.True(t, testStartTime.Equal(nextStart))
		assert.True(t, testStartTime.Add(time.Hour*24*10).Equal(nextEnd))

		nextStart, nextEnd = findLogsNextTimeWindow(nextEnd, endTime, time.Hour*24*10)
		assert.True(t, testStartTime.Add(time.Hour*24*10).Equal(nextStart))
		assert.True(t, endTime.Equal(nextEnd))
	})

	t.Run("should parse response from API", func(t *testing.T) {
		startTime := time.Unix(0, 1757484951000000000) // earliest timestamp in result
		endTime := time.Unix(0, 1757507346000000000)   // latest timestamp in result

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
			if callCount == 3 {
				_, err := w.Write(lokiFindLogsEmptyResponse)
				require.NoError(t, err)
			}
		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProvider(server.Client(), server.URL)

		res := receiveLogLineResults(6)
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

	t.Run("should convert plain text logs to json logs", func(t *testing.T) {
		endTime := testStartTime.Add(testMaxQueryTimeWindow)

		var callCount int
		var httpServerCalls []httpServerCall
		var anError error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
			httpServerCalls, anError = appendHttpServerCall(httpServerCalls, r)
			require.NoError(t, anError)

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(testStartTime.Add(testMaxQueryTimeWindow).UnixNano()), "plain text with \"quotes\""},
					{asString(testStartTime.Add(testMaxQueryTimeWindow).UnixNano()), "1"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}
		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProvider(server.Client(), server.URL)

		res := receiveLogLineResults(2)
		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", res.channel)

		res.wait()
		require.NoError(t, err)

		require.Equal(t, 1, callCount)
		require.True(t, testStartTime.Equal(httpServerCalls[0].reqStart))
		require.True(t, endTime.Equal(httpServerCalls[0].reqEnd))

		msg, err := valueOfJsonField(res.logLines[0].Value, "message")
		require.NoError(t, err)
		assert.Equal(t, "plain text with \"quotes\"", msg)

		msg2, err := valueOfJsonField(res.logLines[1].Value, "message")
		require.NoError(t, err)
		assert.Equal(t, "1", msg2)

	})

	t.Run("should append time fields to http response", func(t *testing.T) {
		endTime := testStartTime.Add(testMaxQueryTimeWindow)

		resultTimestamp := testStartTime.Add(time.Hour * 24)

		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1

			if callCount == 1 {
				resp, err := newQueryRangeResponse([][]string{
					{asString(resultTimestamp.UnixNano()), "{\"msg\":\"msg1\"}"},
				})
				require.NoError(t, err)

				_, err = w.Write(resp)
				require.NoError(t, err)
			}
		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProvider(server.Client(), server.URL)

		res := receiveLogLineResults(1)
		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "ecosystem", res.channel)

		res.wait()
		require.NoError(t, err)

		require.Equal(t, 1, callCount)

		assert.True(t, containsJsonField(res.logLines[0].Value, "time"))
		assert.True(t, containsJsonField(res.logLines[0].Value, "time_unix_nano"))
		assert.True(t, containsJsonField(res.logLines[0].Value, "time_year"))
		assert.True(t, containsJsonField(res.logLines[0].Value, "time_month"))
		assert.True(t, containsJsonField(res.logLines[0].Value, "time_day"))

	})

	t.Run("should enrich json log time fields", func(t *testing.T) {
		aTime := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

		logLine, err := enrichLogLine(aTime, "{\"msg\": \"dogu message 1\"}", make(map[string]string))
		require.NoError(t, err)

		jsonMsg, err := valueOfJsonField(logLine, "msg")
		require.NoError(t, err)
		assert.Equal(t, "dogu message 1", jsonMsg)

		jsonTime, err := valueOfJsonField(logLine, "time")
		require.NoError(t, err)
		assert.Equal(t, "2009-11-17 20:34:58.651387237 +0000 UTC", jsonTime)

		jsonTimeUnixNano, err := valueOfJsonField(logLine, "time_unix_nano")
		require.NoError(t, err)
		assert.Equal(t, strconv.FormatInt(aTime.UnixNano(), 10), jsonTimeUnixNano)

		jsonTimeYear, err := valueOfJsonFieldInt(logLine, "time_year")
		require.NoError(t, err)
		assert.Equal(t, 2009, jsonTimeYear)

		jsonTimeMonth, err := valueOfJsonFieldInt(logLine, "time_month")
		require.NoError(t, err)
		assert.Equal(t, 11, jsonTimeMonth)

		jsonTimeDay, err := valueOfJsonFieldInt(logLine, "time_day")
		require.NoError(t, err)
		assert.Equal(t, 17, jsonTimeDay)

	})

	t.Run("should enrich json with stream fields", func(t *testing.T) {
		aTime := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

		stream := make(map[string]string)
		stream["detected_level"] = "Normal"
		stream["job"] = "loki.source.kubernetes_events"

		logLine, err := enrichLogLine(aTime, "{\"msg\": \"dogu message 1\"}", stream)
		require.NoError(t, err)

		jsonMsg, err := valueOfJsonField(logLine, "stream_detected_level")
		require.NoError(t, err)
		assert.Equal(t, "Normal", jsonMsg)

		jsonMsg2, err := valueOfJsonField(logLine, "stream_job")
		require.NoError(t, err)
		assert.Equal(t, "loki.source.kubernetes_events", jsonMsg2)
	})

	t.Run("should encode LogLine.Value as one line", func(t *testing.T) {
		aTime := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

		logLine, err := enrichLogLine(aTime, "{\"msg\": \"dogu message 1\"}", make(map[string]string))
		require.NoError(t, err)

		assert.NotContains(t, logLine, "\n")
		assert.NotContains(t, logLine, "\t")
	})

	t.Run("should use basic authentification", func(t *testing.T) {
		endTime := testStartTime.Add(time.Hour * 24 * 10)

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

			if callCount == 3 {
				_, err := w.Write(lokiFindLogsEmptyResponse)
				require.NoError(t, err)
			}

			if callCount == 4 {
				_, err := w.Write(lokiFindLogsEmptyResponse)
				require.NoError(t, err)
			}

		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProviderWithCredentials(server.Client(), server.URL, "aUser", "aPassword")

		res := receiveLogLineResults(6)
		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", res.channel)

		res.wait()
		require.NoError(t, err)

		assert.Equal(t, "aUser", reqAuthUsername)
		assert.Equal(t, "aPassword", reqAuthPassword)
	})

	t.Run("should issue an error if underlying error occurs", func(t *testing.T) {
		endTime := testStartTime.Add(testMaxQueryTimeWindow)
		httpAPIUrl := "\n"

		lokiLogsPrv := newTestLokiLogsProvider(http.DefaultClient, httpAPIUrl)

		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", make(chan *domain.LogLine))

		assert.Error(t, err)
		assert.ErrorContains(t, err, "finding logs:")
	})

	t.Run("should issue an error if response can not be converted to LogLines", func(t *testing.T) {
		endTime := testStartTime.Add(time.Hour * 24 * 10)

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

		lokiLogsPrv := newTestLokiLogsProvider(http.DefaultClient, server.URL)

		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", make(chan *domain.LogLine))

		assert.Error(t, err)
		assert.ErrorContains(t, err, "error writing response: parse results timestamp \"not a timestamp\"")
	})

	t.Run("should issue an error if the result size exceeds the limit", func(t *testing.T) {
		endTime := testStartTime.Add(testMaxQueryTimeWindow)

		responseMessage := "message from response"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(responseMessage))
			require.NoError(t, err)
		}))
		defer server.Close()

		lokiLogsPrv := newTestLokiLogsProvider(server.Client(), server.URL)

		err := lokiLogsPrv.FindLogs(context.TODO(), testStartTime, endTime, "aNamespace", make(chan *domain.LogLine))

		assert.Error(t, err)
		assert.ErrorContains(t, err, responseMessage)
	})
}

func TestBuildFindLogsHttpQuery(t *testing.T) {
	t.Run("should create url for querying logs", func(t *testing.T) {
		endTime := testStartTime.Add(testMaxQueryTimeWindow)

		apiUrl, err := buildFindLogsHttpQuery(
			"http://example.com:8080",
			"aQuery",
			testStartTime,
			endTime,
			testMaxQueryResultCount,
		)
		require.NoError(t, err)

		parsedApiUrl, err := url.Parse(apiUrl)
		require.NoError(t, err)

		assert.Equal(t, "http", parsedApiUrl.Scheme)
		assert.Equal(t, "example.com:8080", parsedApiUrl.Host)
		assert.Equal(t, "/loki/api/v1/query_range", parsedApiUrl.Path)

		assert.Equal(t, 5, len(parsedApiUrl.Query()))
		assert.Equal(t, "aQuery", parsedApiUrl.Query().Get("query"))
		assert.Equal(t, fmt.Sprintf("%v", testStartTime.UnixNano()), parsedApiUrl.Query().Get("start"))
		assert.Equal(t, fmt.Sprintf("%v", endTime.UnixNano()), parsedApiUrl.Query().Get("end"))
		assert.Equal(t, fmt.Sprintf("%v", testMaxQueryResultCount), parsedApiUrl.Query().Get("limit"))
		assert.Equal(t, "forward", parsedApiUrl.Query().Get("direction"))
	})

	t.Run("should issue an error if service url is not a valid", func(t *testing.T) {
		_, err := buildFindLogsHttpQuery("\n", "", time.Now(), time.Now(), 0)

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
	reqStart time.Time
	reqEnd   time.Time
	reqError error
}

func appendHttpServerCall(calls []httpServerCall, request *http.Request) ([]httpServerCall, error) {
	reqStartTime, reqEndTime, err := parseStartAndEndTime(request)
	if err != nil {
		return []httpServerCall{}, err
	}
	result := append(calls, httpServerCall{
		reqStart: reqStartTime,
		reqEnd:   reqEndTime,
		reqError: err,
	})
	return result, nil
}

func asString(value int64) string {
	return strconv.FormatInt(value, 10)
}

type logLineResult struct {
	channel       chan *domain.LogLine
	logLines      []*domain.LogLine
	waitGroup     sync.WaitGroup
	resultsToRead int
}

func (res *logLineResult) receive() {
	if res.resultsToRead <= 0 {
		close(res.channel)
		return
	}
	res.waitGroup.Add(1)
	timer := time.NewTimer(5 * time.Minute)
	go func(channel <-chan *domain.LogLine) {
		defer res.waitGroup.Done()
		for {
			select {
			case <-timer.C:
				return
			case ll, isOpen := <-res.channel:
				if isOpen {
					res.logLines = append(res.logLines, ll)
					res.resultsToRead -= 1
					if res.resultsToRead <= 0 {
						close(res.channel)
						return
					}
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

func receiveLogLineResults(resultsToRead int) *logLineResult {
	res := &logLineResult{
		channel:       make(chan *domain.LogLine),
		logLines:      []*domain.LogLine{},
		waitGroup:     sync.WaitGroup{},
		resultsToRead: resultsToRead,
	}
	res.receive()
	return res
}

func parseStartAndEndTime(r *http.Request) (time.Time, time.Time, error) {
	start, err := strconv.ParseInt(r.URL.Query().Get("start"), 10, 64)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := strconv.ParseInt(r.URL.Query().Get("end"), 10, 64)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return time.Unix(0, start), time.Unix(0, end), nil
}

func daysToDuration(days int) time.Duration {
	return time.Duration(time.Hour.Nanoseconds() * int64(24) * int64(days))
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
	s, ok := value.(string)
	if !ok {
		return "", errors.New("value is not a string")
	}
	return s, nil
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

	number, ok := value.(json.Number)
	if !ok {
		return 0, errors.New("value is not a json number")
	}
	valueInt64, err := number.Int64()
	if err != nil {
		return 0, err
	}

	return int(valueInt64), nil
}

func containsJsonField(jsonAsString string, field string) bool {
	jsonDecoder := json.NewDecoder(strings.NewReader(jsonAsString))
	jsonDecoder.UseNumber()

	var decodedData map[string]interface{}
	err := jsonDecoder.Decode(&decodedData)
	if err != nil {
		return false
	}

	_, containsField := decodedData[field]
	return containsField
}

func newTestLokiLogsProvider(httpClient *http.Client, serviceUrl string) *LokiLogsProvider {
	return newTestLokiLogsProviderWithCredentials(httpClient, serviceUrl, "", "")
}

func newTestLokiLogsProviderWithCredentials(httpClient *http.Client, serviceUrl string, username string, password string) *LokiLogsProvider {
	cfg := &config.OperatorConfig{
		LogGatewayConfig: config.LogGatewayConfig{
			Url:      serviceUrl,
			Username: username,
			Password: password,
		},
		LogsMaxQueryTimeWindow:  testMaxQueryTimeWindow,
		LogsMaxQueryResultCount: testMaxQueryResultCount,
	}

	return NewLokiLogsProvider(httpClient, cfg)
}

func newTestLokiLogsProviderWithLimits(httpClient *http.Client, serviceUrl string, maxTimeWindowInDays int, maxQueryResultCount int) *LokiLogsProvider {
	cfg := &config.OperatorConfig{
		LogGatewayConfig: config.LogGatewayConfig{
			Url:      serviceUrl,
			Username: "",
			Password: "",
		},
		LogsMaxQueryTimeWindow:  daysToDuration(maxTimeWindowInDays),
		LogsMaxQueryResultCount: maxQueryResultCount,
	}

	return NewLokiLogsProvider(httpClient, cfg)
}
