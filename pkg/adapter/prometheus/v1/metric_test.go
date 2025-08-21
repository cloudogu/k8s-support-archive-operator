package v1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_metric_getQuery(t *testing.T) {
	tests := []struct {
		name    string
		q       metric
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "node count metric",
			q:       "count",
			want:    "count(kube_node_info)",
			wantErr: assert.NoError,
		},
		{
			name:    "node name metric",
			q:       "name",
			want:    "count(kube_node_info) by (node)",
			wantErr: assert.NoError,
		},
		{
			name:    "node storage metric",
			q:       "storageTotalBytes",
			want:    "node_filesystem_size_bytes{mountpoint=\"/\",fstype!=\"rootfs\"}",
			wantErr: assert.NoError,
		},
		{
			name:    "node storage available metric",
			q:       "storageAvailableBytes",
			want:    "node_filesystem_avail_bytes{mountpoint=\"/\",fstype!=\"rootfs\"}",
			wantErr: assert.NoError,
		},
		{
			name:    "node storage used relative metric",
			q:       "storageUsedRelative",
			want:    "100 - ((node_filesystem_avail_bytes{mountpoint=\"/\",fstype!=\"rootfs\"} * 100) / node_filesystem_size_bytes{mountpoint=\"/\",fstype!=\"rootfs\"})",
			wantErr: assert.NoError,
		},
		{
			name:    "node ram metric",
			q:       "ramTotalBytes",
			want:    "machine_memory_bytes",
			wantErr: assert.NoError,
		},
		{
			name:    "node ram available metric",
			q:       "ramAvailableBytes",
			want:    "avg_over_time(node_memory_MemFree_bytes[5m]) + avg_over_time(node_memory_Cached_bytes[10m]) + avg_over_time(node_memory_Buffers_bytes[5m])",
			wantErr: assert.NoError,
		},
		{
			name:    "node ram used relative metric",
			q:       "ramUsedRelative",
			want:    "100 * (1- ((avg_over_time(node_memory_MemFree_bytes[5m]) + avg_over_time(node_memory_Cached_bytes[10m]) + avg_over_time(node_memory_Buffers_bytes[5m])) / avg_over_time(node_memory_MemTotal_bytes[5m])))",
			wantErr: assert.NoError,
		},
		{
			name:    "node cpu cores metric",
			q:       "cpuCores",
			want:    "machine_cpu_cores",
			wantErr: assert.NoError,
		},
		{
			name:    "node cpu usage cores metric",
			q:       "cpuUsageCores",
			want:    "sum(rate (container_cpu_usage_seconds_total{id=~\"/.*\"}[2m])) by (node)",
			wantErr: assert.NoError,
		},
		{
			name:    "node cpu usage relative metric",
			q:       "cpuUsageRelative",
			want:    "100 * avg(1 - rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) by (node)",
			wantErr: assert.NoError,
		},
		{
			name:    "node network container bytes received metric",
			q:       "containerNetworkRxBytesRate",
			want:    "sum (rate (container_network_receive_bytes_total[2m])) by (node)",
			wantErr: assert.NoError,
		},
		{
			name:    "node network container bytes sent metric",
			q:       "containerNetworkTxBytesRate",
			want:    "sum (rate (container_network_transmit_bytes_total[2m])) by (node)",
			wantErr: assert.NoError,
		},
		{
			name: "unknown metric",
			q:    "unknown",
			want: "",
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "no query for metric \"unknown\"", i)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.q.getQuery()
			if !tt.wantErr(t, err, fmt.Sprintf("getQuery()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "getQuery()")
		})
	}
}
