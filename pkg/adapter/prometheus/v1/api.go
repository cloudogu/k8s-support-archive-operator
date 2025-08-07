package v1

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
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

	nodeCountQuery               = "count(kube_node_info)"
	nodeNameQuery                = "count(kube_node_info) by (node)"
	nodeStorageQuery             = "node_filesystem_size_bytes{mountpoint=\"/\",fstype!=\"rootfs\"}"
	nodeStorageFreeQuery         = "node_filesystem_avail_bytes{mountpoint=\"/\",fstype!=\"rootfs\"}"
	nodeStorageFreeRelativeQuery = "100 - ((node_filesystem_avail_bytes{mountpoint=\"/\",fstype!=\"rootfs\"} * 100) / node_filesystem_size_bytes{mountpoint=\"/\",fstype!=\"rootfs\"})"

	nodeRAMQuery             = "machine_memory_bytes"
	nodeRAMFreeQuery         = "avg_over_time(node_memory_MemFree_bytes[5m]) + avg_over_time(node_memory_Cached_bytes[10m]) + avg_over_time(node_memory_Buffers_bytes[5m])"
	nodeRAMUsedRelativeQuery = "100 * (1- ((avg_over_time(node_memory_MemFree_bytes[5m]) + avg_over_time(node_memory_Cached_bytes[10m]) + avg_over_time(node_memory_Buffers_bytes[5m])) / avg_over_time(node_memory_MemTotal_bytes[5m])))"

	nodeCPUCoresQuery    = "machine_cpu_cores"
	nodeCPUUsage         = "sum(rate (container_cpu_usage_seconds_total{id=~\"/.*\"}[2m])) by (node)"
	nodeCPUUsageRelative = "100 * avg(1 - rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) by (node)"

	nodeNetworkContainerBytesReceived = "sum (rate (container_network_receive_bytes_total[2m])) by (node)"
	nodeNetworkContainerBytesSent     = "sum (rate (container_network_transmit_bytes_total[2m])) by (node)"
)

var (
	convertMatrixErr = errors.New("failed to convert values to matrix model")
)

type PrometheusMetricsV1API struct {
	v1API
}

func (p *PrometheusMetricsV1API) GetNodeRAM(ctx context.Context, start, end time.Time) (domain.NodeRAMInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeRAMQuery, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[float64](matrix), nil
}

func (p *PrometheusMetricsV1API) GetNodeRAMFree(ctx context.Context, start, end time.Time) (domain.NodeRAMInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeRAMFreeQuery, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[float64](matrix), nil
}

func (p *PrometheusMetricsV1API) GetNodeRAMUsedRelative(ctx context.Context, start, end time.Time) (domain.NodeRAMInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeRAMUsedRelativeQuery, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[float64](matrix), nil
}

func (p *PrometheusMetricsV1API) GetNodeCPUCores(ctx context.Context, start, end time.Time) (domain.NodeCPUInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeCPUCoresQuery, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[float64](matrix), nil
}

func (p *PrometheusMetricsV1API) GetNodeCPUUsage(ctx context.Context, start, end time.Time) (domain.NodeCPUInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeCPUUsage, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[float64](matrix), nil
}

func (p *PrometheusMetricsV1API) GetNodeCPUUsageRelative(ctx context.Context, start, end time.Time) (domain.NodeCPUInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeCPUUsageRelative, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[float64](matrix), nil
}

func (p *PrometheusMetricsV1API) GetNodeNetworkContainerBytesReceived(ctx context.Context, start, end time.Time) (domain.NodeContainerNetworkInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeNetworkContainerBytesReceived, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[int](matrix), nil
}

func (p *PrometheusMetricsV1API) GetNodeNetworkContainerBytesSend(ctx context.Context, start, end time.Time) (domain.NodeContainerNetworkInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeNetworkContainerBytesSent, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[int](matrix), nil
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

func (p *PrometheusMetricsV1API) GetNodeCount(ctx context.Context, start, end time.Time) (domain.NodeCountRange, error) {
	values, err := p.queryRange(ctx, nodeCountQuery, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	matrix, ok := values.(model.Matrix)
	if !ok {
		return nil, convertMatrixErr
	}

	// count without filtering ("count by") for labels always returns one stream
	sampleStream := matrix[0]

	var result domain.NodeCountRange
	for _, sample := range sampleStream.Values {
		result = append(result, domain.Sample[int]{
			Value: int(sample.Value),
			Time:  sample.Timestamp.Time(),
		})
	}

	return result, nil
}

func (p *PrometheusMetricsV1API) GetNodeNames(ctx context.Context, start, end time.Time) (domain.NodeNameRange, error) {
	values, err := p.queryRange(ctx, nodeNameQuery, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	matrix, ok := values.(model.Matrix)
	if !ok {
		return nil, convertMatrixErr
	}

	var result domain.NodeNameRange

	for _, sampleStream := range matrix {
		value := string(sampleStream.Metric["node"])

		for _, sample := range sampleStream.Values {
			entry := domain.StringSample{
				Value: value,
				Time:  sample.Timestamp.Time(),
			}

			result = append(result, entry)
		}
	}

	return result, nil
}

func (p *PrometheusMetricsV1API) GetNodeStorage(ctx context.Context, start, end time.Time) (domain.NodeStorageInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeStorageQuery, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[float64](matrix), nil
}

func (p *PrometheusMetricsV1API) GetNodeFreeStorage(ctx context.Context, start, end time.Time) (domain.NodeStorageInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeStorageFreeQuery, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[float64](matrix), nil
}

func (p *PrometheusMetricsV1API) GetNodeFreeRelativeStorage(ctx context.Context, start, end time.Time) (domain.NodeStorageInfo, error) {
	matrix, err := p.queryRangeForMatrix(ctx, nodeStorageFreeRelativeQuery, start, end, time.Hour)
	if err != nil {
		return nil, err
	}

	return parseMatrix[float64](matrix), nil
}

func (p *PrometheusMetricsV1API) queryRangeForMatrix(ctx context.Context, query string, start, end time.Time, step time.Duration) (*model.Matrix, error) {
	values, err := p.queryRange(ctx, query, start, end, step)
	if err != nil {
		return nil, err
	}

	matrix, ok := values.(model.Matrix)
	if !ok {
		return nil, convertMatrixErr
	}

	return &matrix, nil
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

func (p *PrometheusMetricsV1API) queryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (model.Value, error) {
	logger := log.FromContext(ctx).WithName("PrometheusMetricsV1API.queryRange")
	r := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}
	values, warnings, err := p.QueryRange(ctx, query, r)
	if err != nil {
		return nil, err
	}

	logWarnings(logger, warnings)

	return values, nil
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

func parseMatrix[n domain.Number](matrix *model.Matrix) []domain.LabeledSamples[n] {
	var result []domain.LabeledSamples[n]

	for _, sampleStream := range *matrix {
		labels := make(map[string]string, len(sampleStream.Metric))
		for k, v := range sampleStream.Metric {
			labels[string(k)] = string(v)
		}
		samples := make([]domain.Sample[n], len(sampleStream.Values))
		entry := domain.LabeledSamples[n]{
			Labels:  labels,
			Samples: samples,
		}

		for i, sample := range sampleStream.Values {
			samples[i] = domain.Sample[n]{
				Time:  sample.Timestamp.Time(),
				Value: n(sample.Value),
			}

		}
		result = append(result, entry)
	}

	return result
}

func logWarnings(logger logr.Logger, warnings []string) {
	for _, warning := range warnings {
		logger.Info(fmt.Sprintf("Warning: %s", warning))
	}
}
