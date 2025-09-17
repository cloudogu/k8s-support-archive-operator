package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const loggerName = "LokiLogsProvider"

type LokiLogsProvider struct {
	serviceURL                 string
	httpClient                 *http.Client
	username                   string
	password                   string
	maxQueryResultCount        int
	maxQueryTimeWindowNanoSecs int64
}

type labelValuesResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

type queryLogsResponse struct {
	Status string        `json:"status"`
	Data   queryLogsData `json:"data"`
}

type queryLogsData struct {
	ResultType string            `json:"resultType"`
	Result     []queryLogsResult `json:"result"`
}

type queryLogsResult struct {
	Stream queryLogsStream
	Values [][]string `json:"values"`
}

type queryLogsStream struct {
	LogLevel  string `json:"detected_level"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
}

func NewLokiLogsProvider(httpClient *http.Client, httpAPIUrl, username, password string, maxQueryResultCount int, maxQueryTimeWindow time.Duration) *LokiLogsProvider {
	return &LokiLogsProvider{
		serviceURL:                 httpAPIUrl,
		httpClient:                 httpClient,
		username:                   username,
		password:                   password,
		maxQueryResultCount:        maxQueryResultCount,
		maxQueryTimeWindowNanoSecs: maxQueryTimeWindow.Nanoseconds(),
	}
}

func (lp *LokiLogsProvider) FindLogs(
	ctx context.Context,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	namespace string,
	resultChan chan<- *domain.LogLine,
) error {
	var reqStartTime, reqEndTime = int64(0), startTimeInNanoSec
	for {
		reqStartTime, reqEndTime = findLogsNextTimeWindow(reqEndTime, endTimeInNanoSec, lp.maxQueryTimeWindowNanoSecs)
		httpResp, err := lp.httpFindLogs(ctx, reqStartTime, reqEndTime, namespace)
		if err != nil {
			return fmt.Errorf("find logs; %v", err)
		}

		logLines, err := convertQueryLogsResponseToLogLines(httpResp)
		if err != nil {
			return fmt.Errorf("convert http response to LogLines; %v", err)
		}
		if len(logLines) == 0 {
			return nil
		}

		for _, ll := range logLines {
			resultChan <- &ll
		}

		reqEndTime = findLatestTimestamp(logLines)
	}
}

func (lp *LokiLogsProvider) httpFindLogs(
	ctx context.Context,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	namespace string,
) (*queryLogsResponse, error) {
	logger := log.FromContext(ctx).WithName(loggerName)

	query, err := buildFindLogsHttpQuery(lp.serviceURL, namespace, startTimeInNanoSec, endTimeInNanoSec, lp.maxQueryResultCount)
	if err != nil {
		return nil, fmt.Errorf("build logs query; %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, query, nil)
	if err != nil {
		return nil, fmt.Errorf("create http request for query '%s'; %w", query, err)
	}
	req.SetBasicAuth(lp.username, lp.password)

	resp, err := lp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call loki http api; %w", err)
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			logger.Error(err, "failed to close body of http response")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, extractErrorFromResponse(resp)
	}

	return parseQueryLogsResponse(resp.Body)
}

func buildFindLogsHttpQuery(
	serviceURL string,
	namespace string,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	maxQueryResultCount int,
) (string, error) {
	baseUrl, err := url.Parse(serviceURL)
	if err != nil {
		return "", fmt.Errorf("parse service URL; %w", err)
	}
	baseUrl = baseUrl.JoinPath("loki/api/v1/query_range")

	params := baseUrl.Query()
	params.Set("query", fmt.Sprintf("{namespace=\"%s\"}", namespace))
	params.Set("start", fmt.Sprintf("%d", startTimeInNanoSec))
	params.Set("end", fmt.Sprintf("%d", endTimeInNanoSec))
	params.Set("limit", fmt.Sprintf("%d", maxQueryResultCount))

	baseUrl.RawQuery = params.Encode()

	return baseUrl.String(), nil
}

func extractErrorFromResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil || len(body) == 0 {
		return fmt.Errorf("http request failed with status: %s, body: empty", resp.Status)
	}
	return fmt.Errorf("http request failed with status: %s, body: %s", resp.Status, body)
}

func parseQueryLogsResponse(httpRespBody io.Reader) (*queryLogsResponse, error) {
	result := &queryLogsResponse{}
	err := json.NewDecoder(httpRespBody).Decode(result)
	if err != nil {
		return nil, fmt.Errorf("decode query logs http response; %w", err)
	}
	return result, nil
}

func findLogsNextTimeWindow(startTimeInNanoSec int64, maxEndTimeInNanoSec int64, maxTimeWindowNanoSecs int64) (int64, int64) {
	timeWindowEndInNanoSec := minInt64(startTimeInNanoSec+maxTimeWindowNanoSecs, maxEndTimeInNanoSec)
	return startTimeInNanoSec, timeWindowEndInNanoSec
}

func convertQueryLogsResponseToLogLines(httpResp *queryLogsResponse) ([]domain.LogLine, error) {
	var result []domain.LogLine
	for _, respResult := range httpResp.Data.Result {
		for _, respValue := range respResult.Values {
			timestampAsInt, err := strconv.ParseInt(respValue[0], 10, 64)
			if err != nil {
				return []domain.LogLine{}, fmt.Errorf("parse results timestamp '%s'; %w", respValue[0], err)
			}

			newLogLine, err := appendTimeFields(domain.LogLine{
				Timestamp: time.Unix(0, timestampAsInt),
				Value:     respValue[1],
			})
			if err != nil {
				return []domain.LogLine{}, fmt.Errorf("append time fields to logline '%s'; %w", respValue[1], err)
			}

			result = append(result, newLogLine)
		}
	}

	return result, nil
}

func findLatestTimestamp(loglines []domain.LogLine) int64 {
	var latest int64
	for _, ll := range loglines {
		if ll.Timestamp.UnixNano() > latest {
			latest = ll.Timestamp.UnixNano()
		}
	}
	return latest
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func daysToNanoSec(days int) int64 {
	return time.Hour.Nanoseconds() * 24 * int64(days)
}

func appendTimeFields(logLine domain.LogLine) (domain.LogLine, error) {
	jsonDecoder := json.NewDecoder(strings.NewReader(logLine.Value))

	var data map[string]interface{}
	err := jsonDecoder.Decode(&data)
	if err != nil {
		return domain.LogLine{}, fmt.Errorf("decode logline; %w", err)
	}

	data["time"] = logLine.Timestamp.String()
	data["time_unix_nano"] = strconv.FormatInt(logLine.Timestamp.UnixNano(), 10)
	data["time_year"] = logLine.Timestamp.Year()
	data["time_month"] = logLine.Timestamp.Month()
	data["time_day"] = logLine.Timestamp.Day()

	result := bytes.NewBufferString("")
	jsonEncoder := json.NewEncoder(result)
	err = jsonEncoder.Encode(data)
	if err != nil {
		return domain.LogLine{}, fmt.Errorf("encode event")
	}

	newLogLine := domain.LogLine{
		Timestamp: logLine.Timestamp,
		Value:     strings.Replace(result.String(), "\n", "", -1),
	}
	return newLogLine, nil
}
