package v1

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/prometheus"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"
	"time"
)

const (
	capacityBytesQueryFmt = "kubelet_volume_stats_capacity_bytes{namespace=\"%s\", persistentvolumeclaim=\"%s\"}"
	usedBytesQueryFmt     = "kubelet_volume_stats_used_bytes{namespace=\"%s\", persistentvolumeclaim=\"%s\"}"
)

type PrometheusMetricsV1API struct {
	v1.API
}

func NewPrometheusMetricsV1API(address string, token string) (*PrometheusMetricsV1API, error) {
	client, err := prometheus.GetClient(address, token)
	if err != nil {
		return nil, fmt.Errorf("unable to create prometheus client: %w", err)
	}

	v1Api := v1.NewAPI(client)

	return &PrometheusMetricsV1API{v1Api}, nil
}

func (p *PrometheusMetricsV1API) GetCapacityBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error) {
	query := fmt.Sprintf(capacityBytesQueryFmt, namespace, pvcName)

	return p.queryInt64(ctx, query, ts)
}

func (p *PrometheusMetricsV1API) GetUsedBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error) {
	query := fmt.Sprintf(usedBytesQueryFmt, namespace, pvcName)

	return p.queryInt64(ctx, query, ts)
}

func (p *PrometheusMetricsV1API) queryInt64(ctx context.Context, query string, ts time.Time) (int64, error) {
	result, err := p.query(ctx, query, ts)
	if err != nil {
		return 0, err
	}

	if result == "" {
		return 0, nil
	}

	bytesInt, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse string %s to int64: %w", result, err)
	}

	return bytesInt, nil
}

func (p *PrometheusMetricsV1API) query(ctx context.Context, query string, ts time.Time) (string, error) {
	logger := log.FromContext(ctx).WithName("Prometheus query")
	value, warnings, err := p.API.Query(ctx, query, ts)
	if err != nil {
		return "", err
	}

	logWarnings(logger, warnings)

	switch v := value.(type) {
	case model.Vector:
		if len(v) == 0 {
			logger.Info(fmt.Sprintf("empty vector result for query %s", query))
			return "", nil
		}
		logger.Info(fmt.Sprintf("vector result: %s", v.String()))
		// Query should always return one value
		return v[0].Value.String(), nil
	case model.Matrix:
		return "", errors.New("matrix type not implemented")
	case *model.Scalar:
		return "", errors.New("scalar type not implemented")
	case *model.String:
		return v.Value, errors.New("string type not implemented")
	default:
		return "", errors.New("unknown prometheus return type")
	}
}

func logWarnings(logger logr.Logger, warnings []string) {
	for _, warning := range warnings {
		logger.Info(fmt.Sprintf("Warning: %s", warning))
	}
}
