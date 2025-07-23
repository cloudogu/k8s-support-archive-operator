package v1

import (
	"context"
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const (
	testNamespace = "test"
	testPvcName   = "test-pvc"
)

var (
	testCtx = context.Background()
)

func TestPrometheusMetricsV1API_GetUsedBytesForPVC(t *testing.T) {
	now := time.Now()
	vectorValue := model.Vector{&model.Sample{Value: 1}}

	type fields struct {
		v1API func(t *testing.T) v1API
	}
	type args struct {
		ctx       context.Context
		namespace string
		pvcName   string
		ts        time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "success",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().Query(testCtx, "kubelet_volume_stats_used_bytes{namespace=\"test\", persistentvolumeclaim=\"test-pvc\"}", now).Return(vectorValue, nil, nil)

					return apiMock
				},
			},
			args: args{
				ctx:       testCtx,
				namespace: testNamespace,
				pvcName:   testPvcName,
				ts:        now,
			},
			want: 1,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PrometheusMetricsV1API{
				v1API: tt.fields.v1API(t),
			}
			got, err := p.GetUsedBytesForPVC(tt.args.ctx, tt.args.namespace, tt.args.pvcName, tt.args.ts)
			tt.wantErr(t, err)
			if got != tt.want {
				t.Errorf("GetUsesBytesForPVC() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrometheusMetricsV1API_GetCapacityBytesForPVC(t *testing.T) {
	now := time.Now()
	vectorValue := model.Vector{&model.Sample{Value: 1}}
	emptyVectorValue := model.Vector{}

	type fields struct {
		v1API func(t *testing.T) v1API
	}
	type args struct {
		ctx       context.Context
		namespace string
		pvcName   string
		ts        time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "success",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().Query(testCtx, "kubelet_volume_stats_capacity_bytes{namespace=\"test\", persistentvolumeclaim=\"test-pvc\"}", now).Return(vectorValue, nil, nil)

					return apiMock
				},
			},
			args: args{
				ctx:       testCtx,
				namespace: testNamespace,
				pvcName:   testPvcName,
				ts:        now,
			},
			want: 1,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return 0 on empty value",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().Query(testCtx, "kubelet_volume_stats_capacity_bytes{namespace=\"test\", persistentvolumeclaim=\"test-pvc\"}", now).Return(emptyVectorValue, nil, nil)

					return apiMock
				},
			},
			args: args{
				ctx:       testCtx,
				namespace: testNamespace,
				pvcName:   testPvcName,
				ts:        now,
			},
			want: 0,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error calling query",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().Query(testCtx, "kubelet_volume_stats_capacity_bytes{namespace=\"test\", persistentvolumeclaim=\"test-pvc\"}", now).Return(nil, nil, assert.AnError)

					return apiMock
				},
			},
			args: args{
				ctx:       testCtx,
				namespace: testNamespace,
				pvcName:   testPvcName,
				ts:        now,
			},
			want: 0,
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PrometheusMetricsV1API{
				v1API: tt.fields.v1API(t),
			}
			got, err := p.GetCapacityBytesForPVC(tt.args.ctx, tt.args.namespace, tt.args.pvcName, tt.args.ts)
			tt.wantErr(t, err)
			if got != tt.want {
				t.Errorf("GetCapacityBytesForPVC() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrometheusMetricsV1API_query(t *testing.T) {
	testQuery := "testQuery"
	now := time.Now()

	type fields struct {
		v1API func(t *testing.T) v1API
	}
	type args struct {
		ctx   context.Context
		query string
		ts    time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return error on unsupported matrix type",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().Query(testCtx, testQuery, now).Return(model.Matrix{}, nil, nil)

					return apiMock
				},
			},
			args: args{
				ctx:   testCtx,
				query: testQuery,
				ts:    now,
			},
			want:    "",
			wantErr: assert.Error,
		},
		{
			name: "should return error on unsupported scalar type",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().Query(testCtx, testQuery, now).Return(&model.Scalar{}, nil, nil)

					return apiMock
				},
			},
			args: args{
				ctx:   testCtx,
				query: testQuery,
				ts:    now,
			},
			want:    "",
			wantErr: assert.Error,
		},
		{
			name: "should return error on unsupported string type",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().Query(testCtx, testQuery, now).Return(&model.String{}, nil, nil)

					return apiMock
				},
			},
			args: args{
				ctx:   testCtx,
				query: testQuery,
				ts:    now,
			},
			want:    "",
			wantErr: assert.Error,
		},
		{
			name: "should return error on nil",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().Query(testCtx, testQuery, now).Return(nil, nil, nil)

					return apiMock
				},
			},
			args: args{
				ctx:   testCtx,
				query: testQuery,
				ts:    now,
			},
			want:    "",
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PrometheusMetricsV1API{
				v1API: tt.fields.v1API(t),
			}
			got, err := p.query(tt.args.ctx, tt.args.query, tt.args.ts)
			if !tt.wantErr(t, err, fmt.Sprintf("query(%v, %v, %v)", tt.args.ctx, tt.args.query, tt.args.ts)) {
				return
			}
			assert.Equalf(t, tt.want, got, "query(%v, %v, %v)", tt.args.ctx, tt.args.query, tt.args.ts)
		})
	}
}

func TestNewPrometheusMetricsV1API(t *testing.T) {
	// given
	clientMock := newMockClient(t)

	// when
	api := NewPrometheusMetricsV1API(clientMock)

	// then
	require.NotNil(t, api)
}
