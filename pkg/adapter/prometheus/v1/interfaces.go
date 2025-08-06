package v1

import (
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
)

type v1API interface {
	v1.API
}

type client interface {
	api.Client
}
