package v1

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			name: "should return error on parsing error",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().Query(testCtx, "kubelet_volume_stats_capacity_bytes{namespace=\"test\", persistentvolumeclaim=\"test-pvc\"}", now).Return(model.Vector{&model.Sample{Value: 1.5}}, nil, nil)

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
				assert.ErrorContains(t, err, "failed to parse string 1.5 to int64")
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
	api := NewPrometheusMetricsV1API(clientMock, 11000)

	// then
	require.NotEmpty(t, api)
}

func TestPrometheusMetricsV1API_GetNodeCount(t *testing.T) {
	end := time.Now()
	start := end.Add(-time.Hour)
	step := time.Minute
	testRange := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}
	testChannel := make(chan *domain.LabeledSample)
	sampleTime := time.Unix(1, 0)
	timestamp := model.Time(sampleTime.UnixMilli())
	testMatrix := model.Matrix{&model.SampleStream{Values: []model.SamplePair{{timestamp, 1}}, Metric: model.Metric{"node": "test-node"}}}

	type fields struct {
		v1API func(t *testing.T) v1API
	}
	type args struct {
		ctx        context.Context
		start      time.Time
		end        time.Time
		step       time.Duration
		resultChan chan *domain.LabeledSample
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        assert.ErrorAssertionFunc
		chanAssertions func(*testing.T, chan *domain.LabeledSample)
	}{
		{
			name: "success",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().QueryRange(testCtx, "count(kube_node_info)", testRange).Return(testMatrix, nil, nil)

					return apiMock
				},
			},
			args: args{
				ctx:        testCtx,
				start:      start,
				end:        end,
				step:       step,
				resultChan: testChannel,
			},
			wantErr: assert.NoError,
			chanAssertions: func(t *testing.T, channel chan *domain.LabeledSample) {
				obj := <-channel

				require.NotNil(t, obj)
				assert.Equal(t, "count", obj.MetricName)
				assert.Equal(t, "test-node", obj.ID)
				assert.Equal(t, sampleTime, obj.Time)
				assert.Equal(t, float64(1), obj.Value)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PrometheusMetricsV1API{
				v1API:      tt.fields.v1API(t),
				maxSamples: 11000,
			}

			group := sync.WaitGroup{}
			group.Add(1)
			go func() {
				if tt.chanAssertions != nil {
					tt.chanAssertions(t, tt.args.resultChan)
				}
				group.Done()
			}()

			tt.wantErr(t, p.GetNodeCount(tt.args.ctx, tt.args.start, tt.args.end, tt.args.step, tt.args.resultChan), fmt.Sprintf("GetNodeCount(%v, %v, %v, %v, %v)", tt.args.ctx, tt.args.start, tt.args.end, tt.args.step, tt.args.resultChan))
			group.Wait()
		})
	}
}

func TestPrometheusMetricsV1API_queryRange(t *testing.T) {
	end := time.Now()
	start := end.Add(-time.Hour)
	step := time.Minute
	testRange := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}
	testChannel := make(chan *domain.LabeledSample)
	sampleTime := time.Unix(1, 0)
	timestamp := model.Time(sampleTime.UnixMilli())
	matrix := model.Matrix{&model.SampleStream{Values: []model.SamplePair{{timestamp, 1}}, Metric: model.Metric{"node": "test-node"}}}

	pageStart := time.Now()
	pageEnd := pageStart.Add(2 * time.Hour)
	pageStep := time.Hour
	mid := pageStart.Add(time.Hour)
	page1Range := v1.Range{
		Start: pageStart,
		End:   mid,
		Step:  pageStep,
	}
	page2Range := v1.Range{
		Start: mid,
		End:   pageEnd,
		Step:  pageStep,
	}
	noPageSampleSize := 1000

	type fields struct {
		v1API func(t *testing.T) v1API
	}
	type args struct {
		ctx            context.Context
		metric         metric
		start          time.Time
		end            time.Time
		step           time.Duration
		resultChan     chan *domain.LabeledSample
		pageSampleSize int
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantErr        assert.ErrorAssertionFunc
		chanAssertions func(*testing.T, chan *domain.LabeledSample)
	}{
		{
			name: "should return error on api error",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().QueryRange(testCtx, "count(kube_node_info)", testRange).Return(nil, nil, assert.AnError)

					return apiMock
				},
			},
			args: args{
				ctx:            testCtx,
				metric:         nodeCountMetric,
				start:          start,
				end:            end,
				step:           step,
				resultChan:     testChannel,
				pageSampleSize: noPageSampleSize,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError) && assert.ErrorContains(t, err, "metric range error")
			},
		},
		{
			name: "should return error on invalid return type",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().QueryRange(testCtx, "count(kube_node_info)", testRange).Return(&model.Scalar{}, nil, nil)

					return apiMock
				},
			},
			args: args{
				ctx:            testCtx,
				metric:         nodeCountMetric,
				start:          start,
				end:            end,
				step:           step,
				resultChan:     testChannel,
				pageSampleSize: noPageSampleSize,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorContains(t, err, "invalid value type: *model.Scalar")
			},
		},
		{
			name: "should return error on get query error",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)

					return apiMock
				},
			},
			args: args{
				ctx:            testCtx,
				metric:         "invalid",
				start:          start,
				end:            end,
				step:           step,
				resultChan:     testChannel,
				pageSampleSize: noPageSampleSize,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorContains(t, err, "no query for metric")
			},
		},
		{
			name: "should use pagination if samples > than max sample size",
			fields: fields{
				v1API: func(t *testing.T) v1API {
					apiMock := newMockV1API(t)
					apiMock.EXPECT().QueryRange(testCtx, "count(kube_node_info)", page1Range).Return(matrix, nil, nil).Once()
					apiMock.EXPECT().QueryRange(testCtx, "count(kube_node_info)", page2Range).Return(matrix, nil, nil).Once()

					return apiMock
				},
			},
			args: args{
				ctx:            testCtx,
				metric:         nodeCountMetric,
				start:          pageStart,
				end:            pageEnd,
				step:           pageStep,
				resultChan:     testChannel,
				pageSampleSize: 1,
			},
			wantErr: assert.NoError,
			chanAssertions: func(t *testing.T, samples chan *domain.LabeledSample) {
				for i := 0; i < 2; i++ {
					<-samples
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PrometheusMetricsV1API{
				v1API: tt.fields.v1API(t),
			}

			group := sync.WaitGroup{}
			group.Add(1)
			go func() {
				if tt.chanAssertions != nil {
					tt.chanAssertions(t, tt.args.resultChan)
				}
				group.Done()
			}()

			tt.wantErr(t, p.queryRange(tt.args.ctx, tt.args.metric, tt.args.start, tt.args.end, tt.args.step, tt.args.resultChan, tt.args.pageSampleSize), fmt.Sprintf("queryRange(%v, %v, %v, %v, %v, %v)", tt.args.ctx, tt.args.metric, tt.args.start, tt.args.end, tt.args.step, tt.args.resultChan))
			group.Wait()
		})
	}
}
