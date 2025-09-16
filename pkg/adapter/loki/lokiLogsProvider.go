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
const maxQueryTimeWindowInDays = 30 // Loki's max time range is 30d1h
const maxQueryResultCount = 2000    // Loki's max entries limit per query is 5000

type LokiLogsProvider struct {
	serviceURL string
	httpClient *http.Client
	username   string
	password   string
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

func NewLokiLogsProvider(httpClient *http.Client, httpAPIUrl string, username string, password string) *LokiLogsProvider {
	return &LokiLogsProvider{
		serviceURL: httpAPIUrl,
		httpClient: httpClient,
		username:   username,
		password:   password,
	}
}

func (lp *LokiLogsProvider) FindLogs(
	ctx context.Context,
	startTimeInNanoSec int64,
	endTimeInNanoSec int64,
	namespace string,
	resultChan chan<- *col.LogLine,
) error {
	var reqStartTime, reqEndTime = int64(0), startTimeInNanoSec
	for {
		reqStartTime, reqEndTime = findLogsNextTimeWindow(reqEndTime, endTimeInNanoSec, maxQueryTimeWindowInDays)
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

	query, err := buildFindLogsHttpQuery(lp.serviceURL, namespace, startTimeInNanoSec, endTimeInNanoSec)
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

func findLogsNextTimeWindow(startTimeInNanoSec int64, maxEndTimeInNanoSec int64, maxTimeWindowInDays int) (int64, int64) {
	maxTimeWindowInNanoSec := daysToNanoSec(maxTimeWindowInDays)
	timeWindowEndInNanoSec := minInt64(startTimeInNanoSec+maxTimeWindowInNanoSec, maxEndTimeInNanoSec)
	return startTimeInNanoSec, timeWindowEndInNanoSec
}

func convertQueryLogsResponseToLogLines(httpResp *queryLogsResponse) ([]col.LogLine, error) {
	var result []col.LogLine
	for _, respResult := range httpResp.Data.Result {
		for _, respValue := range respResult.Values {
			timestampAsInt, err := strconv.ParseInt(respValue[0], 10, 64)
			if err != nil {
				return []col.LogLine{}, fmt.Errorf("parse results timestamp '%s'; %w", respValue[0], err)
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

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func daysToNanoSec(days int) int64 {
	return time.Hour.Nanoseconds() * 24 * int64(days)
}
