package prometheus

import (
	"fmt"
	"github.com/prometheus/client_golang/api"
	"net/http"
)

type TokenTransport struct {
	http.RoundTripper
	token string
}

func (tt *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if tt.token != "" {
		req.Header.Add("Authorization", tt.token)
	}

	return tt.RoundTripper.RoundTrip(req)
}

func GetClient(address, token string) (api.Client, error) {
	transport := api.DefaultRoundTripper

	tokenTransport := &TokenTransport{
		RoundTripper: transport,
		token:        token,
	}

	cfg := api.Config{
		Address:      address,
		RoundTripper: tokenTransport,
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create prometheus client: %w", err)
	}

	return client, nil
}
