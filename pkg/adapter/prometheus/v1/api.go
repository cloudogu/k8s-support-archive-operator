package v1

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	maxSamples = 11000 // TODO make configurable. depends on prometheus instance
)

type PrometheusMetricsV1API struct {
	v1API
}

func NewPrometheusMetricsV1API(client client) *PrometheusMetricsV1API {
	return &PrometheusMetricsV1API{v1.NewAPI(client)}
}

func (p *PrometheusMetricsV1API) GetCapacityBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error) {
	query := fmt.Sprintf(capacityBytesQueryFmt, namespace, pvcName)

	return p.queryInt64(ctx, query, ts)
}

func (p *PrometheusMetricsV1API) GetUsedBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error) {
	query := fmt.Sprintf(usedBytesQueryFmt, namespace, pvcName)

	return p.queryInt64(ctx, query, ts)
}

func (p *PrometheusMetricsV1API) GetNodeCount(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeCountMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeNames(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeNameMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeStorage(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeStorageMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeStorageFree(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeStorageFreeMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeStorageFreeRelative(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeStorageFreeRelativeMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeRAM(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeRAMMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeRAMFree(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeRAMFreeMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeRAMUsedRelative(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeRAMUsedRelativeMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeCPUCores(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeCPUCoresMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeCPUUsage(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeCPUUsageMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeCPUUsageRelative(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeCPUUsageRelativeMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeNetworkContainerBytesReceived(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeNetworkContainerBytesReceivedMetric, start, end, step, resultChan, maxSamples)
}

func (p *PrometheusMetricsV1API) GetNodeNetworkContainerBytesSend(ctx context.Context, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample) error {
	return p.queryRange(ctx, nodeNetworkContainerBytesSentMetric, start, end, step, resultChan, maxSamples)
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
	logger := log.FromContext(ctx).WithName("PrometheusMetricsV1API.query")
	value, warnings, err := p.Query(ctx, query, ts)
	if err != nil {
		return "", err
	}

	logWarnings(logger, warnings)

	return parseValue(value, logger, query)
}

//nolint:unparam // ignore that we always set pageSampleSize to maxSamples
func (p *PrometheusMetricsV1API) queryRange(ctx context.Context, metric metric, start, end time.Time, step time.Duration, resultChan chan<- *domain.LabeledSample, pageSampleSize int) error {
	logger := log.FromContext(ctx).WithName("PrometheusMetricsV1API.queryRange")

	pageStart := start
	pageEnd := end
	lastPage := false
	pageIndex := 1

	for {
		pageEnd = pageStart.Add(step * time.Duration(pageSampleSize))

		if pageEnd.After(end) || pageEnd.Equal(end) {
			pageEnd = end
			lastPage = true
		}

		r := v1.Range{
			Start: pageStart,
			End:   pageEnd,
			Step:  step,
		}

		query, err := metric.getQuery()
		if err != nil {
			return err
		}
		logger.Info("do range query", "query", query, "start", start, "end", end, "step", step, "pageIndex", pageIndex, "lastPage", lastPage)
		value, warnings, pageErr := p.QueryRange(ctx, query, r)
		if pageErr != nil {
			return fmt.Errorf("metric range error: %w", pageErr)
		}
		logWarnings(logger, warnings)

		// write to channel
		err = writeMatrixToChannel(value, metric, resultChan)
		if err != nil {
			return err
		}

		if lastPage {
			break
		}

		pageIndex++
		pageStart = pageEnd
	}

	return nil
}

func writeMatrixToChannel(value model.Value, metric metric, ch chan<- *domain.LabeledSample) error {
	matrix, ok := value.(model.Matrix)
	if !ok {
		return fmt.Errorf("invalid value type: %T", value)
	}

	for _, sampleStream := range matrix {
		id := sampleStream.Metric["node"]

		for _, sample := range sampleStream.Values {
			ch <- &domain.LabeledSample{
				MetricName: string(metric),
				ID:         string(id),
				Value:      float64(sample.Value),
				Time:       sample.Timestamp.Time(),
			}
		}
	}

	return nil
}

func parseValue(value model.Value, logger logr.Logger, query string) (string, error) {
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
		return "", errors.New("string type not implemented")
	default:
		return "", errors.New("unknown prometheus return type")
	}
}

func logWarnings(logger logr.Logger, warnings []string) {
	for _, warning := range warnings {
		logger.Info(fmt.Sprintf("Warning: %s", warning))
	}
}
