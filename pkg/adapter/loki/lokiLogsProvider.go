package loki

import (
	"context"
	"net/http"
	"time"
)

type lokiLogsProvider struct {
	httpClient *http.Client
}

func NewLokiLogsProvider(httpClient *http.Client) *lokiLogsProvider {
	return &lokiLogsProvider{
		httpClient,
	}
}

func (lp *lokiLogsProvider) getValuesOfLabel(ctx context.Context, start, end time.Time, label string) ([]string, error) {
	_, err := lp.httpClient.Get("http://example.com")
	if err != nil {
		return []string{}, err
	}
	return []string{}, nil
}

func (lp *lokiLogsProvider) getLogs(ctx context.Context, start, end time.Time, namespace string, kind string) ([]string, error) {
	return []string{}, nil
}
