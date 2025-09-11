package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	col "github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/collector"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const loggerName = "LokiLogsProvider"
const maxQueryTimeWindowInDays = 30
const maxQueryResultCount = 2000

type LokiLogsProvider struct {
	serviceURL string
	httpClient *http.Client
	username   string
	password   string
}

type findValuesOfLabelsHttpResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

type queryRangeResponse struct {
	Status string         `json:"status"`
	Data   queryRangeData `json:"data"`
}

type queryRangeData struct {
	ResultType string             `json:"resultType"`
	Result     []queryRangeResult `json:"result"`
}

type queryRangeResult struct {
	Stream queryRangeStream
	Values [][]string `json:"values"`
}

type queryRangeStream struct {
	LogLevel  string `json:"detected_level"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
}

func NewLokiLogsProvider(httpClient *http.Client, httpAPIUrl string, username string, password string) *LokiLogsProvider {
	return &LokiLogsProvider{
		serviceURL: httpAPIUrl,
		httpClient: httpClient,
		username:   username,
		password:   password,
	}
}

func (lp *LokiLogsProvider) FindValuesOfLabel(ctx context.Context, startTimeInNanoSec, endTimeInNanoSec int64, label string) ([]string, error) {
	var result []string
	var reqStartTime, reqEndTime, hasNext = int64(0), startTimeInNanoSec, true
	for hasNext {
		reqStartTime, reqEndTime, hasNext = findValuesOfLabelNextTimeWindow(reqEndTime, endTimeInNanoSec, maxQueryTimeWindowInDays)
		resp, err := lp.httpFindValuesOfLabel(ctx, label, reqStartTime, reqEndTime)
		if err != nil {
			return []string{}, fmt.Errorf("query loki for values of label \"%s\"; %v", label, err)
		}
		result = append(result, resp.Data...)
	}

	return removeDuplicates(result), nil
}

func (lp *LokiLogsProvider) httpFindValuesOfLabel(ctx context.Context, label string, startTimeInNanoSec int64, endTimeInNanoSec int64) (*findValuesOfLabelsHttpResponse, error) {
	logger := log.FromContext(ctx).WithName(loggerName)

	query, err := buildFindValuesOfLabelHttpQuery(lp.serviceURL, label, startTimeInNanoSec, endTimeInNanoSec)
	if err != nil {
		return nil, fmt.Errorf("build http query to get values of label \"%s\"; %w", label, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, query, nil)
	if err != nil {
		return nil, fmt.Errorf("create new req for query %s; %w", query, err)
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
		body, err := io.ReadAll(resp.Body)
		if err != nil || len(body) == 0 {
			return nil, fmt.Errorf("http request failed with status: %s, body: empty", resp.Status)
		}
		return nil, fmt.Errorf("http request failed with status: %s, body: %s", resp.Status, body)
	}

	return parseFindValuesOfLabelHttpResponse(resp.Body)
}

func buildFindValuesOfLabelHttpQuery(serviceURL string, label string, startTimeInNanoSec int64, endTimeInNanoSec int64) (string, error) {
	baseUrl, err := url.Parse(serviceURL)
	if err != nil {
		return "", fmt.Errorf("parse service URL; %w", err)
	}
	path := fmt.Sprintf("/loki/api/v1/label/%s/values", label)
	baseUrl = baseUrl.JoinPath(path)

	params := baseUrl.Query()
	params.Set("start", fmt.Sprintf("%d", startTimeInNanoSec))
	params.Set("end", fmt.Sprintf("%d", endTimeInNanoSec))

	baseUrl.RawQuery = params.Encode()

	return baseUrl.String(), nil
}

func parseFindValuesOfLabelHttpResponse(lokiResp io.Reader) (*findValuesOfLabelsHttpResponse, error) {
	result := &findValuesOfLabelsHttpResponse{}
	err := json.NewDecoder(lokiResp).Decode(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func removeDuplicates(data []string) []string {
	var uniqueMap = make(map[string]bool)
	var result []string
	for _, elem := range data {
		if _, mapContainsKey := uniqueMap[elem]; mapContainsKey {
			continue
		} else {
			result = append(result, elem)
			uniqueMap[elem] = true
		}
	}
	return result
}

func (lp *LokiLogsProvider) FindLogs(
	ctx context.Context,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	namespace string,
	kind string,
) ([]col.LogLine, error) {
	var result []col.LogLine
	var reqStartTime, reqEndTime = int64(0), startTimeInNanoSec
	for {
		reqStartTime, reqEndTime = findLogsNextTimeWindow(reqEndTime, endTimeInNanoSec, maxQueryTimeWindowInDays)
		httpResp, err := lp.httpFindLogs(ctx, reqStartTime, reqEndTime, namespace, kind)
		if err != nil {
			return []col.LogLine{}, err
		}
		logLines, err := queryRangeResponseToLogLines(httpResp)
		if err != nil {
			return []col.LogLine{}, err
		}
		if len(logLines) == 0 {
			return result, nil
		}
		result = append(result, logLines...)
		reqEndTime = findLatestTimestamp(logLines)
	}
}

func (lp *LokiLogsProvider) httpFindLogs(
	ctx context.Context,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	namespace string,
	kind string,
) (*queryRangeResponse, error) {
	logger := log.FromContext(ctx).WithName(loggerName)

	query, err := buildFindLogsHttpQuery(lp.serviceURL, namespace, kind, startTimeInNanoSec, endTimeInNanoSec)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, query, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(lp.username, lp.password)

	resp, err := lp.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			logger.Error(err, "failed to close body of http response")
		}
	}(resp.Body)

	return parseQueryRangeResponse(resp.Body)
}

func buildFindLogsHttpQuery(
	serviceURL string,
	namespace string,
	kind string,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
) (string, error) {
	baseUrl, err := url.Parse(serviceURL)
	if err != nil {
		return "", fmt.Errorf("parse service URL; %w", err)
	}
	baseUrl = baseUrl.JoinPath("loki/api/v1/query_range")

	params := baseUrl.Query()
	params.Set("query", fmt.Sprintf("{namespace=\"%s\", kind=\"%s\"}", namespace, kind))
	params.Set("start", fmt.Sprintf("%d", startTimeInNanoSec))
	params.Set("end", fmt.Sprintf("%d", endTimeInNanoSec))
	params.Set("limit", fmt.Sprintf("%d", maxQueryResultCount))

	baseUrl.RawQuery = params.Encode()

	return baseUrl.String(), nil
}

func parseQueryRangeResponse(httpRespBody io.Reader) (*queryRangeResponse, error) {
	result := &queryRangeResponse{}
	err := json.NewDecoder(httpRespBody).Decode(result)
	if err != nil {
		return nil, fmt.Errorf("decode json http response; %w", err)
	}
	return result, nil
}

func queryRangeResponseToLogLines(httpResp *queryRangeResponse) ([]col.LogLine, error) {
	var result []col.LogLine
	for _, respResult := range httpResp.Data.Result {
		for _, respValue := range respResult.Values {
			timestampAsInt, err := strconv.ParseInt(respValue[0], 10, 64)
			if err != nil {
				return []col.LogLine{}, err
			}
			result = append(result, col.LogLine{
				Timestamp: time.Unix(0, timestampAsInt),
				Value:     respValue[1],
			})
		}
	}

	return result, nil
}

func findLatestTimestamp(loglines []col.LogLine) int64 {
	var latest int64
	for _, ll := range loglines {
		if ll.Timestamp.UnixNano() > latest {
			latest = ll.Timestamp.UnixNano()
		}
	}
	return latest
}

func findValuesOfLabelNextTimeWindow(startTimeInNanoSec int64, maxEndTimeInNanoSec int64, maxTimeWindowInDays int) (int64, int64, bool) {
	maxTimeWindowInNanoSec := daysToNanoSec(maxTimeWindowInDays)
	timeWindowEndInNanoSec := minInt64(startTimeInNanoSec+maxTimeWindowInNanoSec, maxEndTimeInNanoSec)
	hasNext := timeWindowEndInNanoSec < maxEndTimeInNanoSec
	return startTimeInNanoSec, timeWindowEndInNanoSec, hasNext
}

func findLogsNextTimeWindow(startTimeInNanoSec int64, maxEndTimeInNanoSec int64, maxTimeWindowInDays int) (int64, int64) {
	maxTimeWindowInNanoSec := daysToNanoSec(maxTimeWindowInDays)
	timeWindowEndInNanoSec := minInt64(startTimeInNanoSec+maxTimeWindowInNanoSec, maxEndTimeInNanoSec)
	return startTimeInNanoSec, timeWindowEndInNanoSec
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
