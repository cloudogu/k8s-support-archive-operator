package loki

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const maxQueryTimeWindowInDays = 30

type LokiLogsProvider struct {
	httpAPIUrl string
	httpClient *http.Client
}

type valuesOfLabelsResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

func NewLokiLogsProvider(httpClient *http.Client, httpAPIUrl string) *LokiLogsProvider {
	return &LokiLogsProvider{
		httpAPIUrl: httpAPIUrl,
		httpClient: httpClient,
	}
}

func (lp *LokiLogsProvider) FindValuesOfLabel(ctx context.Context, startTimeInNanoSec, endTimeInNanoSec int64, label string) ([]string, error) {
	var result []string
	var reqStartTime, reqEndTime, hasNext = int64(0), startTimeInNanoSec, true
	for hasNext {
		reqStartTime, reqEndTime, hasNext = nextTimeWindow(reqEndTime, endTimeInNanoSec, maxQueryTimeWindowInDays)
		resp, err := lp.doLokiValuesOfLabelQuery(label, reqStartTime, reqEndTime)
		if err != nil {
			//return []string{}, err
			return []string{}, fmt.Errorf("query loki for values of label \"%s\", %v", label, err)
		}
		result = append(result, resp.Data...)
	}

	return removeDuplicates(result), nil
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

func (lp *LokiLogsProvider) FindLogs(ctx context.Context, startTimeInNanoSec, endTimeInNanoSec int64, namespace string, kind string) ([]string, error) {
	return []string{}, nil
}

func (lp *LokiLogsProvider) doLokiValuesOfLabelQuery(label string, startTimeInUnixNano int64, endTimeInUnixNano int64) (*valuesOfLabelsResponse, error) {

	query, err := buildLokiValuesOfLabelQuery(lp.httpAPIUrl, label, startTimeInUnixNano, endTimeInUnixNano)
	if err != nil {
		return nil, err // url.Error
	}

	request, err := http.NewRequest(http.MethodGet, query, nil)
	if err != nil {
		return nil, err // fmt.Errorf or errors.New
	}

	resp, err := lp.httpClient.Do(request)
	if err != nil {
		//return nil, err url.Error
		return nil, fmt.Errorf("call loki HTTP API, %w", err)
	}

	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			// TODO log error
		}
	}(resp.Body)

	return parseValuesOfLabelResponse(resp.Body)
}

func parseValuesOfLabelResponse(lokiResp io.Reader) (*valuesOfLabelsResponse, error) {
	result := &valuesOfLabelsResponse{}
	err := json.NewDecoder(lokiResp).Decode(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func buildLokiValuesOfLabelQuery(httpAPIUrl string, label string, startTimeInUnixNano int64, endTimeInUnixNano int64) (string, error) {
	baseUrl, err := url.Parse(httpAPIUrl)
	if err != nil {
		return "", err // url.Error
	}

	path := fmt.Sprintf("/loki/api/v1/label/%s/values", label)
	baseUrl = baseUrl.JoinPath(path)

	params := baseUrl.Query()
	params.Set("start", fmt.Sprintf("%d", startTimeInUnixNano))
	params.Set("end", fmt.Sprintf("%d", endTimeInUnixNano))

	baseUrl.RawQuery = params.Encode()

	return baseUrl.String(), nil
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
