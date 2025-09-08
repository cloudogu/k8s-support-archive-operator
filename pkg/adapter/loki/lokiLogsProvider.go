package loki

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const maxQueryTimeWindowInDays = 30

type lokiLogsProvider struct {
	httpAPIUrl string
	httpClient *http.Client
}

func NewLokiLogsProvider(httpClient *http.Client, httpAPIUrl string) *lokiLogsProvider {
	return &lokiLogsProvider{
		httpAPIUrl: httpAPIUrl,
		httpClient: httpClient,
	}
}

func (lp *lokiLogsProvider) getValuesOfLabel(ctx context.Context, startTime, endTime time.Time, label string) ([]string, error) {
	var reqStartTime int64
	var resEndTime int64 = startTime.UnixNano()
	var hasNext = true
	for hasNext {
		reqStartTime, resEndTime, hasNext = nextTimeWindow(resEndTime, endTime.UnixNano(), maxQueryTimeWindowInDays)
		_, err := lp.doLokiHttpQuery(reqStartTime, resEndTime)
		if err != nil {
			return []string{}, err
		}
	}

	return []string{}, nil
}

func (lp *lokiLogsProvider) getLogs(ctx context.Context, start, end time.Time, namespace string, kind string) ([]string, error) {
	return []string{}, nil
}

func (lp *lokiLogsProvider) doLokiHttpQuery(startTimeInUnixNano int64, endTimeInUnixNano int64) ([]string, error) {

	url, err := buildLokiQuery(lp.httpAPIUrl, startTimeInUnixNano, endTimeInUnixNano)
	if err != nil {
		return []string{}, err
	}

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return []string{}, err
	}
	_, err = lp.httpClient.Do(request)
	if err != nil {
		return []string{}, err
	}

	return []string{}, nil
}

func buildLokiQuery(httpAPIUrl string, startTimeInUnixNano int64, endTimeInUnixNano int64) (string, error) {
	baseUrl, err := url.Parse(httpAPIUrl)
	if err != nil {
		return "", err
	}

	baseUrl = baseUrl.JoinPath("/loki/api/v1/query_range")

	params := baseUrl.Query()
	params.Set("start", fmt.Sprintf("%d", startTimeInUnixNano))
	params.Set("end", fmt.Sprintf("%d", endTimeInUnixNano))

	baseUrl.RawQuery = params.Encode()

	return baseUrl.String(), nil
}

func nextTimeWindow(startTimeInNanoSec int64, maxEndTimeInNanoSec int64, maxTimeWindowInDays int) (int64, int64, bool) {
	maxTimeWindowInNanoSec := time.Hour.Nanoseconds() * 24 * int64(maxTimeWindowInDays)
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
