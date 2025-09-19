package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/config"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	loggerName         = "LokiLogsProvider"
	lokiQueryrangePath = "loki/api/v1/query_range"
)

type LokiLogsProvider struct {
	serviceURL                 string
	httpClient                 *http.Client
	username                   string
	password                   string
	maxQueryResultCount        int
	maxQueryTimeWindowNanoSecs int64
	logEventSourceName         string
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

type ReturnType int

const (
	onlyLogs ReturnType = iota
	onlyEvents
)

func (r ReturnType) GetQuery(namespace, logEventSourceName string) (string, error) {
	switch r {
	case onlyLogs:
		return fmt.Sprintf("{namespace=\"%s\", job!=\"%s\"}", namespace, logEventSourceName), nil
	case onlyEvents:
		return fmt.Sprintf("{namespace=\"%s\", job=\"%s\"}", namespace, logEventSourceName), nil
	default:
		return "", fmt.Errorf("invalid ReturnType: %v", r)
	}
}

func NewLokiLogsProvider(httpClient *http.Client, operatorConfig *config.OperatorConfig) *LokiLogsProvider {
	return &LokiLogsProvider{
		httpClient:                 httpClient,
		serviceURL:                 operatorConfig.LogGatewayConfig.Url,
		username:                   operatorConfig.LogGatewayConfig.Username,
		password:                   operatorConfig.LogGatewayConfig.Password,
		maxQueryResultCount:        operatorConfig.LogsMaxQueryResultCount,
		maxQueryTimeWindowNanoSecs: operatorConfig.LogsMaxQueryTimeWindow.Nanoseconds(),
		logEventSourceName:         operatorConfig.LogsEventSourceName,
	}
}

func (lp *LokiLogsProvider) FindLogs(
	ctx context.Context,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	namespace string,
	resultChan chan<- *domain.LogLine,
) error {
	return lp.findLogs(ctx, startTimeInNanoSec, endTimeInNanoSec, namespace, resultChan, onlyLogs)
}

func (lp *LokiLogsProvider) FindEvents(
	ctx context.Context,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	namespace string,
	resultChan chan<- *domain.LogLine,
) error {
	return lp.findLogs(ctx, startTimeInNanoSec, endTimeInNanoSec, namespace, resultChan, onlyEvents)
}

func (lp *LokiLogsProvider) findLogs(
	ctx context.Context,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	namespace string,
	resultChan chan<- *domain.LogLine,
	returnType ReturnType,
) error {
	var reqStartTime, reqEndTime = int64(0), startTimeInNanoSec
	for {
		reqStartTime, reqEndTime = findLogsNextTimeWindow(reqEndTime, endTimeInNanoSec, lp.maxQueryTimeWindowNanoSecs)

		httpResp, err := lp.httpFindLogs(ctx, reqStartTime, reqEndTime, namespace, returnType)
		if err != nil {
			return fmt.Errorf("finding logs: %w", err)
		}

		logLines, err := convertQueryLogsResponseToLogLines(httpResp)
		if err != nil {
			return fmt.Errorf("convert http response to LogLines: %w", err)
		}

		for _, ll := range logLines {
			resultChan <- &ll
		}

		if reqEndTime == endTimeInNanoSec && len(logLines) != lp.maxQueryResultCount {
			return nil
		}

		if len(logLines) == 0 || len(logLines) != lp.maxQueryResultCount {
			reqEndTime += lp.maxQueryTimeWindowNanoSecs
			continue
		}

		reqEndTime = findLatestTimestamp(logLines)
		// if we reach the limit and the last timestamp is this starting timestamp, possible other logs can't be queried
		// we use a high limit to avoid that.
		if reqEndTime == reqStartTime {
			reqEndTime += 1
		}
	}
}

func (lp *LokiLogsProvider) httpFindLogs(ctx context.Context, startTimeInNanoSec int64, endTimeInNanoSec int64, namespace string, returnType ReturnType) (*queryLogsResponse, error) {
	logger := log.FromContext(ctx).WithName(loggerName)

	query, err := returnType.GetQuery(namespace, lp.logEventSourceName)
	if err != nil {
		return nil, err
	}

	query, err = buildFindLogsHttpQuery(lp.serviceURL, startTimeInNanoSec, endTimeInNanoSec, lp.maxQueryResultCount, query)
	if err != nil {
		return nil, fmt.Errorf("building logs query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, query, nil)
	if err != nil {
		return nil, fmt.Errorf("create http request for query %q: %w", query, err)
	}
	req.SetBasicAuth(lp.username, lp.password)

	resp, err := lp.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call loki http api: %w", err)
	}

	defer func(body io.ReadCloser) {
		closeErr := body.Close()
		if closeErr != nil {
			logger.Error(closeErr, "failed to close body of http response")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, extractErrorFromResponse(resp)
	}

	return parseQueryLogsResponse(resp.Body)
}

func buildFindLogsHttpQuery(
	serviceURL string,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	maxQueryResultCount int,
	query string,
) (string, error) {
	baseUrl, err := url.Parse(serviceURL)
	if err != nil {
		return "", fmt.Errorf("parse service URL: %w", err)
	}
	baseUrl = baseUrl.JoinPath(lokiQueryrangePath)

	params := baseUrl.Query()
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%d", startTimeInNanoSec))
	params.Set("end", fmt.Sprintf("%d", endTimeInNanoSec))
	params.Set("limit", fmt.Sprintf("%d", maxQueryResultCount))
	params.Set("direction", "forward")

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
				return []domain.LogLine{}, fmt.Errorf("parse results timestamp %q: %w", respValue[0], err)
			}
			logTimestamp := time.Unix(0, timestampAsInt)

			jsonLog, err := plainLogToJsonLog(respValue[1])
			if err != nil {
				return []domain.LogLine{}, fmt.Errorf("convert plain text logline to json logline: %w", err)
			}

			jsonLogWithTimeFields, err := enrichLogLineWithTimeFields(logTimestamp, jsonLog)
			if err != nil {
				return []domain.LogLine{}, fmt.Errorf("enrich logline with time fields: %w", err)
			}

			newLogLine := domain.LogLine{
				Timestamp: logTimestamp,
				Value:     jsonLogWithTimeFields,
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

func enrichLogLineWithTimeFields(timestamp time.Time, jsonLogLine string) (string, error) {
	var data map[string]interface{}
	jsonDecoder := json.NewDecoder(strings.NewReader(jsonLogLine))
	err := jsonDecoder.Decode(&data)
	if err != nil {
		return "", fmt.Errorf("decode json logline: %w", err)
	}

	data["time"] = timestamp.String()
	data["time_unix_nano"] = strconv.FormatInt(timestamp.UnixNano(), 10)
	data["time_year"] = timestamp.Year()
	data["time_month"] = timestamp.Month()
	data["time_day"] = timestamp.Day()

	result := bytes.NewBufferString("")
	jsonEncoder := json.NewEncoder(result)
	err = jsonEncoder.Encode(data)
	if err != nil {
		return "", fmt.Errorf("encode json logline: %w", err)
	}

	return strings.Replace(result.String(), "\n", "", -1), nil
}

func plainLogToJsonLog(plainLog string) (string, error) {
	if json.Valid([]byte(plainLog)) {
		return plainLog, nil
	}

	data := make(map[string]string)
	data["message"] = plainLog

	result := bytes.NewBufferString("")
	jsonEncoder := json.NewEncoder(result)
	err := jsonEncoder.Encode(data)
	if err != nil {
		return "", fmt.Errorf("encode json with plain text as field: %w", err)
	}
	return result.String(), nil
}
