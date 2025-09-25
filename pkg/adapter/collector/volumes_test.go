package collector

import (
	"context"
	"testing"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testNamespace = "test"
	testPvcName   = "test-pvc"
)

var (
	testCtx = context.Background()
)

func TestVolumesCollector_Collect(t *testing.T) {
	pvc := v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPvcName,
			Namespace: testNamespace,
		},
		Status: v1.PersistentVolumeClaimStatus{
			Phase: v1.ClaimBound,
		},
	}
	pvcList := &v1.PersistentVolumeClaimList{Items: []v1.PersistentVolumeClaim{pvc}}
	now := time.Now()

	type fields struct {
		coreV1Interface func(t *testing.T) coreV1Interface
		metricsProvider func(t *testing.T) metricsProvider
	}
	type args struct {
		ctx        context.Context
		namespace  string
		start      time.Time
		end        time.Time
		resultChan chan *domain.VolumeInfo
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  func(t *testing.T, err error)
		wantData *domain.VolumeInfo
	}{
		{
			name: "should return error on error listing pvcs",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					clientMock := newMockPvcInterface(t)
					clientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(nil, assert.AnError)

					interfaceMock := newMockCoreV1Interface(t)
					interfaceMock.EXPECT().PersistentVolumeClaims(testNamespace).Return(clientMock)

					return interfaceMock
				},
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsMock := newMockMetricsProvider(t)
					return metricsMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  testNamespace,
				start:      time.Time{},
				end:        time.Time{},
				resultChan: make(chan *domain.VolumeInfo, 1),
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error listing pvcs")
			},
		},
		{
			name: "should just close the channel if no pvcs are fetched",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					clientMock := newMockPvcInterface(t)
					clientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(&v1.PersistentVolumeClaimList{Items: []v1.PersistentVolumeClaim{}}, nil)

					interfaceMock := newMockCoreV1Interface(t)
					interfaceMock.EXPECT().PersistentVolumeClaims(testNamespace).Return(clientMock)

					return interfaceMock
				},
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsMock := newMockMetricsProvider(t)
					return metricsMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  testNamespace,
				start:      time.Time{},
				end:        time.Time{},
				resultChan: make(chan *domain.VolumeInfo),
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error getting capacity bytes",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					clientMock := newMockPvcInterface(t)
					clientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(pvcList, nil)

					interfaceMock := newMockCoreV1Interface(t)
					interfaceMock.EXPECT().PersistentVolumeClaims(testNamespace).Return(clientMock)

					return interfaceMock
				},
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsMock := newMockMetricsProvider(t)
					metricsMock.EXPECT().GetCapacityBytesForPVC(testCtx, testNamespace, testPvcName, now).Return(0, assert.AnError)
					return metricsMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  testNamespace,
				start:      time.Time{},
				end:        now,
				resultChan: make(chan *domain.VolumeInfo),
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error getting output item for pvc test-pvc: failed to get capacity bytes")
			},
		},
		{
			name: "should return error on error getting used bytes",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					clientMock := newMockPvcInterface(t)
					clientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(pvcList, nil)

					interfaceMock := newMockCoreV1Interface(t)
					interfaceMock.EXPECT().PersistentVolumeClaims(testNamespace).Return(clientMock)

					return interfaceMock
				},
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsMock := newMockMetricsProvider(t)
					metricsMock.EXPECT().GetCapacityBytesForPVC(testCtx, testNamespace, testPvcName, now).Return(1, nil)
					metricsMock.EXPECT().GetUsedBytesForPVC(testCtx, testNamespace, testPvcName, now).Return(0, assert.AnError)
					return metricsMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  testNamespace,
				start:      time.Time{},
				end:        now,
				resultChan: make(chan *domain.VolumeInfo),
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error getting output item for pvc test-pvc: failed to get used bytes")
			},
		},
		{
			name: "should write to channel and close",
			fields: fields{
				coreV1Interface: func(t *testing.T) coreV1Interface {
					clientMock := newMockPvcInterface(t)
					clientMock.EXPECT().List(testCtx, metav1.ListOptions{}).Return(pvcList, nil)

					interfaceMock := newMockCoreV1Interface(t)
					interfaceMock.EXPECT().PersistentVolumeClaims(testNamespace).Return(clientMock)

					return interfaceMock
				},
				metricsProvider: func(t *testing.T) metricsProvider {
					metricsMock := newMockMetricsProvider(t)
					metricsMock.EXPECT().GetCapacityBytesForPVC(testCtx, testNamespace, testPvcName, now).Return(2, nil)
					metricsMock.EXPECT().GetUsedBytesForPVC(testCtx, testNamespace, testPvcName, now).Return(1, nil)
					return metricsMock
				},
			},
			args: args{
				ctx:        testCtx,
				namespace:  testNamespace,
				start:      time.Time{},
				end:        now,
				resultChan: make(chan *domain.VolumeInfo),
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			wantData: &domain.VolumeInfo{
				Name:      "persistentVolumeClaims",
				Timestamp: now,
				Items: []domain.VolumeInfoItem{
					{
						Name:            testPvcName,
						Capacity:        2,
						Used:            1,
						PercentageUsage: "50.00",
						Phase:           "Bound",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vc := &VolumesCollector{
				coreV1Interface: tt.fields.coreV1Interface(t),
				metricsProvider: tt.fields.metricsProvider(t),
			}

			group, _ := errgroup.WithContext(tt.args.ctx)
			group.Go(func() error {
				err := vc.Collect(tt.args.ctx, tt.args.namespace, tt.args.start, tt.args.end, tt.args.resultChan)
				return err
			})

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

			v, open := <-tt.args.resultChan
			if v == nil && open {
				t.Fatal("test timed out")
			}

			if v != nil && open {
				assert.Equal(t, tt.wantData, v)
			}

			err := group.Wait()
			tt.wantErr(t, err)
		})
	}
}

func TestVolumesCollector_Name(t *testing.T) {
	// given
	collector := VolumesCollector{}

	//when
	name := collector.Name()

	// then
	assert.Equal(t, "VolumeInfo", name)
}

func TestNewVolumesCollector(t *testing.T) {
	// given
	corev1Mock := newMockCoreV1Interface(t)
	providerMock := newMockMetricsProvider(t)

	// when
	collector := NewVolumesCollector(corev1Mock, providerMock)

	// then
	require.NotNil(t, collector)
	assert.Equal(t, corev1Mock, collector.coreV1Interface)
	assert.Equal(t, providerMock, collector.metricsProvider)
}
