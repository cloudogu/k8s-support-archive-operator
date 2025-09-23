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
	serviceURL          string
	httpClient          *http.Client
	username            string
	password            string
	maxQueryResultCount int
	maxQueryTimeWindow  time.Duration
	logEventSourceName  string
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
		httpClient:          httpClient,
		serviceURL:          operatorConfig.LogGatewayConfig.Url,
		username:            operatorConfig.LogGatewayConfig.Username,
		password:            operatorConfig.LogGatewayConfig.Password,
		maxQueryResultCount: operatorConfig.LogsMaxQueryResultCount,
		maxQueryTimeWindow:  operatorConfig.LogsMaxQueryTimeWindow,
		logEventSourceName:  operatorConfig.LogsEventSourceName,
	}
}

func (lp *LokiLogsProvider) FindLogs(ctx context.Context, startTimeInNanoSec, endTimeInNanoSec time.Time, namespace string, resultChan chan<- *domain.LogLine) error {
	return lp.findLogs(ctx, startTimeInNanoSec, endTimeInNanoSec, namespace, resultChan, onlyLogs)
}

func (lp *LokiLogsProvider) FindEvents(ctx context.Context, start, end time.Time, namespace string, resultChan chan<- *domain.LogLine) error {
	return lp.findLogs(ctx, start, end, namespace, resultChan, onlyEvents)
}

func (lp *LokiLogsProvider) findLogs(ctx context.Context, start, end time.Time, namespace string, resultChan chan<- *domain.LogLine, returnType ReturnType) error {
	var reqStartTime time.Time
	reqEndTime := start
	for {
		reqStartTime, reqEndTime = findLogsNextTimeWindow(reqEndTime, end, lp.maxQueryTimeWindow)

		httpResp, err := lp.httpFindLogs(ctx, reqStartTime, reqEndTime, namespace, returnType)
		if err != nil {
			return fmt.Errorf("finding logs: %w", err)
		}

		latestTimestamp, logLineCount, err := writeResponse(ctx, resultChan, httpResp)
		if err != nil {
			return fmt.Errorf("error writing response: %w", err)
		}

		// if the actual time window is the last time window and the result size limit is not reached
		if reqEndTime.Equal(end) && logLineCount != lp.maxQueryResultCount {
			return nil
		}

		// if result count is less than max result count than we have all logs in this time window
		if logLineCount != lp.maxQueryResultCount {
			continue
		}

		reqEndTime = latestTimestamp
		// if we reach the limit and the last timestamp is this starting timestamp, possible other logs can't be queried
		// we use a high limit to avoid that.
		if reqEndTime.Equal(reqStartTime) {
			reqEndTime = reqEndTime.Add(time.Nanosecond)
		}
	}
}

// writeResponse iterates over all log lines in the http response, parses all lines and writes them to the result channel.
// On success, it returns the timestamp from the latest logline and the number of all logs.
func writeResponse(ctx context.Context, resultChan chan<- *domain.LogLine, httpResp *queryLogsResponse) (time.Time, int, error) {
	var latestTimestamp time.Time
	var count int
	for _, respResult := range httpResp.Data.Result {
		for _, respValue := range respResult.Values {
			logLine, err := httpToDomainLogLine(respValue[0], respValue[1])
			if err != nil {
				return time.Time{}, 0, err
			}

			writeSaveToChannel(ctx, &logLine, resultChan)

			if logLine.Timestamp.After(latestTimestamp) {
				latestTimestamp = logLine.Timestamp
			}

			count++
		}
	}

	return latestTimestamp, count, nil
}

func httpToDomainLogLine(timestamp, line string) (domain.LogLine, error) {
	timestampAsInt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return domain.LogLine{}, fmt.Errorf("parse results timestamp %q: %w", timestamp, err)
	}
	logTimestamp := time.Unix(0, timestampAsInt)

	jsonLog, err := plainLogToJsonLog(line)
	if err != nil {
		return domain.LogLine{}, fmt.Errorf("convert plain text logline to json logline: %w", err)
	}

	jsonLogWithTimeFields, err := enrichLogLineWithTimeFields(logTimestamp, jsonLog)
	if err != nil {
		return domain.LogLine{}, fmt.Errorf("enrich logline with time fields: %w", err)
	}

	return domain.LogLine{
		Timestamp: logTimestamp,
		Value:     jsonLogWithTimeFields,
	}, nil
}

func (lp *LokiLogsProvider) httpFindLogs(ctx context.Context, start, end time.Time, namespace string, returnType ReturnType) (*queryLogsResponse, error) {
	logger := log.FromContext(ctx).WithName(loggerName)

	query, err := returnType.GetQuery(namespace, lp.logEventSourceName)
	if err != nil {
		return nil, err
	}

	query, err = buildFindLogsHttpQuery(lp.serviceURL, query, start, end, lp.maxQueryResultCount)
	if err != nil {
		return nil, fmt.Errorf("building logs query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, query, nil)
	if err != nil {
		return nil, fmt.Errorf("create http request for query %q: %w", query, err)
	}
	req.SetBasicAuth(lp.username, lp.password)
	// nolint:bodyclose
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

func buildFindLogsHttpQuery(serviceURL, query string, start, end time.Time, maxQueryResultCount int) (string, error) {
	baseUrl, err := url.Parse(serviceURL)
	if err != nil {
		return "", fmt.Errorf("parse service URL: %w", err)
	}
	baseUrl = baseUrl.JoinPath(lokiQueryrangePath)

	params := baseUrl.Query()
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%d", start.UnixNano()))
	params.Set("end", fmt.Sprintf("%d", end.UnixNano()))
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

func findLogsNextTimeWindow(startTime time.Time, maxEndTime time.Time, maxTimeWindow time.Duration) (time.Time, time.Time) {
	timeWindowEnd := maxEndTime
	window := startTime.Add(maxTimeWindow)
	if window.Before(timeWindowEnd) {
		timeWindowEnd = window
	}

	return startTime, timeWindowEnd
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

	return strings.ReplaceAll(result.String(), "\n", ""), nil
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

func writeSaveToChannel[T any](ctx context.Context, data T, dataChannel chan<- T) {
	select {
	case <-ctx.Done():
		return
	case dataChannel <- data:
		return
	}
}
