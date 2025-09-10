package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	col "github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/collector"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const loggerName = "LokiLogsProvider"
const maxQueryTimeWindowInDays = 30

type LokiLogsProvider struct {
	serviceURL string
	httpClient *http.Client
	username   string
	password   string
}

type valuesOfLabelsResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

type logsResponse struct {
	Status string           `json:"status"`
	Data   logsResponseData `json:"data"`
}

type logsResponseData struct {
	ResultType string               `json:"resultType"`
	Result     []logsResponseStream `json:"result"`
}

type logsResponseStream struct {
	Values [][]string `json:"values"`
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
		reqStartTime, reqEndTime, hasNext = nextTimeWindow(reqEndTime, endTimeInNanoSec, maxQueryTimeWindowInDays)
		resp, err := lp.httpFindValuesOfLabel(ctx, label, reqStartTime, reqEndTime)
		if err != nil {
			return []string{}, fmt.Errorf("query loki for values of label \"%s\"; %v", label, err)
		}
		result = append(result, resp.Data...)
	}

	return removeDuplicates(result), nil
}

func (lp *LokiLogsProvider) httpFindValuesOfLabel(ctx context.Context, label string, startTimeInNanoSec int64, endTimeInNanoSec int64) (*valuesOfLabelsResponse, error) {
	logger := log.FromContext(ctx).WithName(loggerName)

	query, err := buildLokiValuesOfLabelQuery(lp.serviceURL, label, startTimeInNanoSec, endTimeInNanoSec)
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

	return parseValuesOfLabelResponse(resp.Body)
}

func buildLokiValuesOfLabelQuery(serviceURL string, label string, startTimeInNanoSec int64, endTimeInNanoSec int64) (string, error) {
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

func parseValuesOfLabelResponse(lokiResp io.Reader) (*valuesOfLabelsResponse, error) {
	result := &valuesOfLabelsResponse{}
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

func (lp *LokiLogsProvider) FindLogs(ctx context.Context, startTimeInNanoSec, endTimeInNanoSec int64, namespace string, kind string) ([]col.LogLine, error) {

	return []col.LogLine{}, nil
}

func httpFindLogs(ctx context.Context, startTimeInNanoSec, endTimeInNanoSec int64, namespace string, kind string) ([]col.LogLine, error) {

	return []col.LogLine{}, nil
}

func parseFindLogsResponse(lokiResp io.Reader) ([]col.LogLine, error) {
	result := &logsResponse{}
	err := json.NewDecoder(lokiResp).Decode(result)
	if err != nil {
		return nil, err
	}
	return []col.LogLine{}, nil
}

func nextTimeWindow(startTimeInNanoSec int64, maxEndTimeInNanoSec int64, maxTimeWindowInDays int) (int64, int64, bool) {
	maxTimeWindowInNanoSec := daysToNanoSec(maxTimeWindowInDays)
	timeWindowEndInNanoSec := minInt64(startTimeInNanoSec+maxTimeWindowInNanoSec, maxEndTimeInNanoSec)
	hasNext := timeWindowEndInNanoSec < maxEndTimeInNanoSec
	return startTimeInNanoSec, timeWindowEndInNanoSec, hasNext
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
