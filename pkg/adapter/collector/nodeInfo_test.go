package collector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

const (
	testHardwareMetricStep = time.Minute * 30
	testUsageMetricStep    = time.Second * 30
)

func TestNewNodeInfoCollector(t *testing.T) {
	// given
	metricsProviderMock := newMockMetricsProvider(t)

	// when
	collector := NewNodeInfoCollector(metricsProviderMock, testUsageMetricStep, testHardwareMetricStep)

	// then
	require.NotNil(t, collector)
	assert.Equal(t, metricsProviderMock, collector.metricsProvider)
	assert.Equal(t, testUsageMetricStep, collector.usageMetricStep)
	assert.Equal(t, testHardwareMetricStep, collector.hardwareMetricStep)
}

func TestNodeInfoCollector_Name(t *testing.T) {
	// given
	sut := &NodeInfoCollector{}

	// when
	name := sut.Name()

	// then
	assert.Equal(t, string(domain.CollectorTypeNodeInfo), name)
}

func TestNodeInfoCollector_Collect(t *testing.T) {

	start := time.Now().Truncate(time.Hour * 1)
	end := time.Now()
	testChan := make(chan<- *domain.LabeledSample)

	type fields struct {
		metricsProvider func(t *testing.T) metricsProvider
	}
	type args struct {
		ctx        context.Context
		namespace  string
		start      time.Time
		end        time.Time
		resultChan chan<- *domain.LabeledSample
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantErr         assert.ErrorAssertionFunc
		shouldCloseChan bool
	}{
		{
			name: "should return error on error getting node names",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(assert.AnError)
					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node count",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(assert.AnError)
					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node storage",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(assert.AnError)
					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node storage free",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node storage free relative",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node cpu cores",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUCores(testCtx, start, end, testHardwareMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node cpu usage",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUCores(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsage(testCtx, start, end, testUsageMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node cpu usage relative",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUCores(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsage(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsageRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node ram",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUCores(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsage(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsageRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAM(testCtx, start, end, testHardwareMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node ram free",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUCores(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsage(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsageRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAM(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAMFree(testCtx, start, end, testUsageMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node ram used relative",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUCores(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsage(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsageRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAM(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAMFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAMUsedRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node container bytes received",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUCores(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsage(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsageRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAM(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAMFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAMUsedRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeNetworkContainerBytesReceived(testCtx, start, end, testUsageMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error getting node container bytes sent",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUCores(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsage(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsageRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAM(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAMFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAMUsedRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeNetworkContainerBytesReceived(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeNetworkContainerBytesSend(testCtx, start, end, testUsageMetricStep, testChan).Return(assert.AnError)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.Error(t, err) && assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should close channel on success",
			fields: fields{
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsProviderMock := newMockMetricsProvider(t)
					metricsProviderMock.EXPECT().GetNodeNames(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCount(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorage(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeStorageFreeRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUCores(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsage(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeCPUUsageRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAM(testCtx, start, end, testHardwareMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAMFree(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeRAMUsedRelative(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeNetworkContainerBytesReceived(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)
					metricsProviderMock.EXPECT().GetNodeNetworkContainerBytesSend(testCtx, start, end, testUsageMetricStep, testChan).Return(nil)

					return metricsProviderMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  "",
				start:      start,
				end:        end,
				resultChan: testChan,
			},
			wantErr:         assert.NoError,
			shouldCloseChan: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nic := &NodeInfoCollector{
				metricsProvider:    tt.fields.metricsProvider(t),
				usageMetricStep:    testUsageMetricStep,
				hardwareMetricStep: testHardwareMetricStep,
			}

			group, _ := errgroup.WithContext(tt.args.ctx)

			if tt.shouldCloseChan {
				timer := time.NewTimer(time.Second * 2)
				group.Go(func() error {
					<-timer.C
					defer func() {
						// recover panic if the channel is closed correctly from the test
						if r := recover(); r != nil {
							tt.args.resultChan <- nil
							return
						}
					}()

					return nil
				})
			}

			tt.wantErr(t, nic.Collect(tt.args.ctx, tt.args.namespace, tt.args.start, tt.args.end, tt.args.resultChan), fmt.Sprintf("Collect(%v, %v, %v, %v, %v)", tt.args.ctx, tt.args.namespace, tt.args.start, tt.args.end, tt.args.resultChan))

			err := group.Wait()
			require.NoError(t, err)
		})
	}
}
