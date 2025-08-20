package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLabeledSample_GetHeader(t *testing.T) {
	sut := LabeledSample{}

	header := sut.GetHeader()

	assert.Equal(t, []string{"label", "value", "time"}, header)
}

func TestLabeledSample_GetRow(t *testing.T) {
	type fields struct {
		MetricName string
		ID         string
		Value      float64
		Time       time.Time
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		{
			name: "example 1",
			fields: fields{
				MetricName: "cpuUsage",
				ID:         "ces-main",
				Value:      0.194234,
				Time:       time.Unix(1755693772, 0),
			},
			want: []string{"ces-main", "0.19", "2025-08-20T14:42:52+02:00"},
		},
		{
			name: "example 2",
			fields: fields{
				MetricName: "cpuUsage",
				ID:         "ces-worker-0",
				Value:      0.225234234,
				Time:       time.Unix(1755693828, 0),
			},
			want: []string{"ces-worker-0", "0.23", "2025-08-20T14:43:48+02:00"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ls := &LabeledSample{
				MetricName: tt.fields.MetricName,
				ID:         tt.fields.ID,
				Value:      tt.fields.Value,
				Time:       tt.fields.Time,
			}
			assert.Equalf(t, tt.want, ls.GetRow(), "GetRow()")
		})
	}
}
