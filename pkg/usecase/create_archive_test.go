package usecase

import (
	"context"
	libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"slices"
	"testing"
)

const (
	testArchiveName      = "test-archive"
	testArchiveNamespace = "test-namespace"
)

var testCtx = context.Background()

func TestCreateArchiveUseCase_HandleArchiveRequest(t *testing.T) {
	supportArchiveCr := &libapi.SupportArchive{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testArchiveName,
			Namespace: testArchiveNamespace,
		},
	}

	type fields struct {
		supportArchivesInterface func(t *testing.T) supportArchiveV1Interface
		stateHandler             func(t *testing.T) stateHandler
		targetCollectors         func(t *testing.T) []collector
	}
	type args struct {
		ctx context.Context
		cr  *libapi.SupportArchive
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should return error and retry on error reading actual state",
			fields: fields{
				stateHandler: func(t *testing.T) stateHandler {
					stateHandlerMock := newMockStateHandler(t)
					stateHandlerMock.EXPECT().Read(testCtx, testArchiveName, testArchiveNamespace).Return(nil, false, assert.AnError)
					return stateHandlerMock
				},
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					return nil
				},
				targetCollectors: func(t *testing.T) []collector {
					return nil
				},
			},
			args: args{
				ctx: testCtx,
				cr:  supportArchiveCr,
			},
			want: true,
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to read state")
			},
		},
		{
			name: "should not requeue if state is done",
			fields: fields{
				stateHandler: func(t *testing.T) stateHandler {
					stateHandlerMock := newMockStateHandler(t)
					stateHandlerMock.EXPECT().Read(testCtx, testArchiveName, testArchiveNamespace).Return([]string{"Logs"}, true, nil)

					return stateHandlerMock
				},
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					return nil
				},
				targetCollectors: func(t *testing.T) []collector {
					return nil
				},
			},
			args: args{
				ctx: testCtx,
				cr:  supportArchiveCr,
			},
			want: false,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should finalize if no collector remaining",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					v1Mock := newMockSupportArchiveV1Interface(t)
					client := newMockSupportArchiveInterface(t)
					v1Mock.EXPECT().SupportArchives(testArchiveNamespace).Return(client)

					client.EXPECT().UpdateStatusWithRetry(testCtx, supportArchiveCr, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).Run(func(ctx context.Context, cr *libapi.SupportArchive, modifyStatusFn func(libapi.SupportArchiveStatus) libapi.SupportArchiveStatus, opts metav1.UpdateOptions) {
						status := modifyStatusFn(cr.Status)
						assert.Equal(t, "Created", string(status.Phase))
						assert.Equal(t, "ns/name.zip", status.DownloadPath)
						assert.True(t, slices.ContainsFunc(status.Conditions, func(condition metav1.Condition) bool {
							nullTime := metav1.Time{}
							if condition.Status == metav1.ConditionTrue && libapi.StatusPhase(condition.Type) == "Created" && condition.LastTransitionTime != nullTime && condition.Reason == "AllCollectorsExecuted" && condition.Message == "It is available for download under following url: ns/name.zip" {
								return true
							}
							return false
						}))
					})

					return v1Mock
				},
				stateHandler: func(t *testing.T) stateHandler {
					stateHandlerMock := newMockStateHandler(t)
					stateHandlerMock.EXPECT().Read(testCtx, testArchiveName, testArchiveNamespace).Return([]string{"Logs"}, false, nil)
					stateHandlerMock.EXPECT().GetDownloadURL(testCtx, testArchiveName, testArchiveNamespace).Return("ns/name.zip")
					stateHandlerMock.EXPECT().Finalize(testCtx, testArchiveName, testArchiveNamespace).Return(nil)
					return stateHandlerMock
				},
				targetCollectors: func(t *testing.T) []collector {
					collectorMock := newMockArchiveDataCollector(t)
					collectorMock.EXPECT().Name().Return("Logs")
					return []collector{collectorMock}
				},
			},
			args: args{
				ctx: testCtx,
				cr:  supportArchiveCr,
			},
			want: false,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should requeue and return error on error finalize",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					return nil
				},
				stateHandler: func(t *testing.T) stateHandler {
					stateHandlerMock := newMockStateHandler(t)
					stateHandlerMock.EXPECT().Read(testCtx, testArchiveName, testArchiveNamespace).Return([]string{"Logs"}, false, nil)
					stateHandlerMock.EXPECT().Finalize(testCtx, testArchiveName, testArchiveNamespace).Return(assert.AnError)
					return stateHandlerMock
				},
				targetCollectors: func(t *testing.T) []collector {
					collectorMock := newMockArchiveDataCollector(t)
					collectorMock.EXPECT().Name().Return("Logs")
					return []collector{collectorMock}
				},
			},
			args: args{
				ctx: testCtx,
				cr:  supportArchiveCr,
			},
			want: true,
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to finalize state")
			},
		},
		{
			name: "should run next collector",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					return nil
				},
				stateHandler: func(t *testing.T) stateHandler {
					stateHandlerMock := newMockStateHandler(t)
					stateHandlerMock.EXPECT().Read(testCtx, testArchiveName, testArchiveNamespace).Return([]string{}, false, nil)
					stateHandlerMock.EXPECT().WriteState(testCtx, testArchiveName, testArchiveNamespace, "Logs").Return(nil)
					return stateHandlerMock
				},
				targetCollectors: func(t *testing.T) []collector {
					collectorMock := newMockArchiveDataCollector(t)
					collectorMock.EXPECT().Name().Return("Logs")
					collectorMock.EXPECT().Collect(testCtx, testArchiveName, testArchiveNamespace, mock.Anything).Return(nil)
					return []collector{collectorMock}
				},
			},
			args: args{
				ctx: testCtx,
				cr:  supportArchiveCr,
			},
			want: true,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return true and error on error writing state",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					return nil
				},
				stateHandler: func(t *testing.T) stateHandler {
					stateHandlerMock := newMockStateHandler(t)
					stateHandlerMock.EXPECT().Read(testCtx, testArchiveName, testArchiveNamespace).Return([]string{}, false, nil)
					stateHandlerMock.EXPECT().WriteState(testCtx, testArchiveName, testArchiveNamespace, "Logs").Return(assert.AnError)
					return stateHandlerMock
				},
				targetCollectors: func(t *testing.T) []collector {
					collectorMock := newMockArchiveDataCollector(t)
					collectorMock.EXPECT().Name().Return("Logs")
					collectorMock.EXPECT().Collect(testCtx, testArchiveName, testArchiveNamespace, mock.Anything).Return(nil)
					return []collector{collectorMock}
				},
			},
			args: args{
				ctx: testCtx,
				cr:  supportArchiveCr,
			},
			want: true,
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to write state Logs for test-namespace/test-archive")
			},
		},
		{
			name: "should return error and requeue on error execute collector",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					return nil
				},
				stateHandler: func(t *testing.T) stateHandler {
					stateHandlerMock := newMockStateHandler(t)
					stateHandlerMock.EXPECT().Read(testCtx, testArchiveName, testArchiveNamespace).Return([]string{}, false, nil)
					return stateHandlerMock
				},
				targetCollectors: func(t *testing.T) []collector {
					collectorMock := newMockArchiveDataCollector(t)
					collectorMock.EXPECT().Name().Return("Logs")
					collectorMock.EXPECT().Collect(testCtx, testArchiveName, testArchiveNamespace, mock.Anything).Return(assert.AnError)
					return []collector{collectorMock}
				},
			},
			args: args{
				ctx: testCtx,
				cr:  supportArchiveCr,
			},
			want: true,
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to execute collector Logs")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := CreateArchiveUseCase{
				supportArchivesInterface: tt.fields.supportArchivesInterface(t),
				stateHandler:             tt.fields.stateHandler(t),
				targetCollectors:         tt.fields.targetCollectors(t),
			}
			got, err := c.HandleArchiveRequest(tt.args.ctx, tt.args.cr)
			if tt.wantErr != nil {
				tt.wantErr(t, err)
			}
			if got != tt.want {
				t.Errorf("HandleArchiveRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewCreateArchiveUseCase(t *testing.T) {
	t.Run("should not return empty", func(t *testing.T) {
		interfaceMock := newMockSupportArchiveV1Interface(t)
		stateHandlerMock := newMockStateHandler(t)
		actual := NewCreateArchiveUseCase(interfaceMock, stateHandlerMock)

		require.NotNil(t, actual)
		assert.Equal(t, actual.supportArchivesInterface, interfaceMock)
		assert.Equal(t, actual.stateHandler, stateHandlerMock)
	})
}
