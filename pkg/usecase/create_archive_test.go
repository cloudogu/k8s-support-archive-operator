package usecase

import (
	"context"
	libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

const (
	testArchiveName      = "test-archive"
	testArchiveNamespace = "test-namespace"
	testURL              = "url"
)

var testCtx = context.Background()

func TestCreateArchiveUseCase_HandleArchiveRequest(t *testing.T) {
	type fields struct {
		supportArchivesInterface func(t *testing.T) supportArchiveV1Interface
		supportArchiveRepository func(t *testing.T) supportArchiveRepository
		collectorMapping         func(t *testing.T) CollectorMapping
	}
	type args struct {
		ctx context.Context
		cr  *libapi.SupportArchive
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    time.Duration
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should return false if archive already exists",
			fields: fields{
				collectorMapping: func(t *testing.T) CollectorMapping {
					collectorMapping := CollectorMapping{}
					logRepository := newMockCollectorRepository[domain.LogLine](t)
					logRepository.EXPECT().IsCollected(testCtx, testID).Return(true, nil)

					collectorMapping[domain.CollectorTypeLog] = CollectorAndRepository{Repository: logRepository}
					return collectorMapping
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					repoMock := newMockSupportArchiveRepository(t)
					repoMock.EXPECT().Exists(testCtx, testID).Return(true, nil)
					return repoMock
				},
			},
			args: args{
				ctx: testCtx,
				cr:  testLogCR,
			},
			want: 0,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error query if a collector is completed",
			fields: fields{
				collectorMapping: func(t *testing.T) CollectorMapping {
					collectorMapping := CollectorMapping{}
					logRepository := newMockCollectorRepository[domain.LogLine](t)
					logRepository.EXPECT().IsCollected(testCtx, testID).Return(false, assert.AnError)

					collectorMapping[domain.CollectorTypeLog] = CollectorAndRepository{Repository: logRepository}
					return collectorMapping
				},
			},
			args: args{
				ctx: testCtx,
				cr:  testLogCR,
			},
			want: 0,
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "could not get already executed collectors: failed to determine if collector Logs is already finished")
			},
		},
		{
			name: "should return error on error query if archive already exists",
			fields: fields{
				collectorMapping: func(t *testing.T) CollectorMapping {
					collectorMapping := CollectorMapping{}
					logRepository := newMockCollectorRepository[domain.LogLine](t)
					logRepository.EXPECT().IsCollected(testCtx, testID).Return(true, nil)

					collectorMapping[domain.CollectorTypeLog] = CollectorAndRepository{Repository: logRepository}
					return collectorMapping
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					repoMock := newMockSupportArchiveRepository(t)
					repoMock.EXPECT().Exists(testCtx, testID).Return(false, assert.AnError)
					return repoMock
				},
			},
			args: args{
				ctx: testCtx,
				cr:  testLogCR,
			},
			want: 0,
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "could not check if the support archive exists")
			},
		},
		{
			name: "should execute if it is not completed and return true for retry",
			fields: fields{
				collectorMapping: func(t *testing.T) CollectorMapping {
					collectorMapping := CollectorMapping{}
					logRepository := newMockCollectorRepository[domain.LogLine](t)
					logRepository.EXPECT().IsCollected(testCtx, testID).Return(false, nil)
					logRepository.EXPECT().Create(mock.AnythingOfType("*context.cancelCtx"), testID, mock.AnythingOfType("<-chan *domain.LogLine")).Return(nil)
					logCollector := newMockCollector[domain.LogLine](t)
					logCollector.EXPECT().Collect(mock.AnythingOfType("*context.cancelCtx"), testArchiveNamespace, mock.Anything, mock.Anything, mock.AnythingOfType("chan<- *domain.LogLine")).Return(nil)

					collectorMapping[domain.CollectorTypeLog] = CollectorAndRepository{Repository: logRepository, Collector: logCollector}
					return collectorMapping
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					repoMock := newMockSupportArchiveRepository(t)
					repoMock.EXPECT().Exists(testCtx, testID).Return(false, nil)
					return repoMock
				},
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					interfaceMock := newMockSupportArchiveV1Interface(t)
					clientMock := newMockSupportArchiveInterface(t)
					interfaceMock.EXPECT().SupportArchives(testArchiveNamespace).Return(clientMock)
					clientMock.EXPECT().UpdateStatusWithRetry(testCtx, testLogCR, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).Run(func(ctx context.Context, cr *libapi.SupportArchive, modifyStatusFn func(libapi.SupportArchiveStatus) libapi.SupportArchiveStatus, opts metav1.UpdateOptions) {
						updatedCRStatus := modifyStatusFn(cr.Status)
						for _, cond := range updatedCRStatus.Conditions {
							if cond.Type == "TODO" && cond.Status == ("True") {
								return
							}
						}
						t.FailNow()
					})
					return interfaceMock
				},
			},
			args: args{
				ctx: testCtx,
				cr:  testLogCR,
			},
			want: time.Nanosecond,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error executing next collector",
			fields: fields{
				collectorMapping: func(t *testing.T) CollectorMapping {
					collectorMapping := CollectorMapping{}
					logRepository := newMockCollectorRepository[domain.LogLine](t)
					logRepository.EXPECT().IsCollected(testCtx, testID).Return(false, nil)
					logRepository.EXPECT().Create(mock.AnythingOfType("*context.cancelCtx"), testID, mock.AnythingOfType("<-chan *domain.LogLine")).Return(nil)
					logCollector := newMockCollector[domain.LogLine](t)
					logCollector.EXPECT().Name().Return("Logs")
					logCollector.EXPECT().Collect(mock.AnythingOfType("*context.cancelCtx"), testArchiveNamespace, mock.Anything, mock.Anything, mock.AnythingOfType("chan<- *domain.LogLine")).Return(assert.AnError)

					collectorMapping[domain.CollectorTypeLog] = CollectorAndRepository{Repository: logRepository, Collector: logCollector}
					return collectorMapping
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					repoMock := newMockSupportArchiveRepository(t)
					repoMock.EXPECT().Exists(testCtx, testID).Return(false, nil)
					return repoMock
				},
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					interfaceMock := newMockSupportArchiveV1Interface(t)
					clientMock := newMockSupportArchiveInterface(t)
					interfaceMock.EXPECT().SupportArchives(testArchiveNamespace).Return(clientMock)
					clientMock.EXPECT().UpdateStatusWithRetry(testCtx, testLogCR, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).Run(func(ctx context.Context, cr *libapi.SupportArchive, modifyStatusFn func(libapi.SupportArchiveStatus) libapi.SupportArchiveStatus, opts metav1.UpdateOptions) {
						updatedCRStatus := modifyStatusFn(cr.Status)
						for _, cond := range updatedCRStatus.Conditions {
							if cond.Type == "TODO" && cond.Status == ("False") {
								return
							}
						}
						t.FailNow()
					})
					return interfaceMock
				},
			},
			args: args{
				ctx: testCtx,
				cr:  testLogCR,
			},
			want: 0,
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "could not execute next collector: failed to execute collector Logs: error from error group Logs")
			},
		},
		{
			name: "should create archive and update status",
			fields: fields{
				collectorMapping: func(t *testing.T) CollectorMapping {
					collectorMapping := CollectorMapping{}
					logCollector := newMockCollector[domain.LogLine](t)
					logRepository := newMockCollectorRepository[domain.LogLine](t)
					logRepository.EXPECT().IsCollected(testCtx, testID).Return(true, nil)
					logRepository.EXPECT().IsCollected(mock.AnythingOfType("*context.cancelCtx"), testID).Return(true, nil)
					logRepository.EXPECT().Stream(mock.AnythingOfType("*context.cancelCtx"), testID, mock.AnythingOfType("*domain.Stream")).Return(nil)

					collectorMapping[domain.CollectorTypeLog] = CollectorAndRepository{Repository: logRepository, Collector: logCollector}
					return collectorMapping
				},
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					repoMock := newMockSupportArchiveRepository(t)
					repoMock.EXPECT().Exists(testCtx, testID).Return(false, nil)
					repoMock.EXPECT().Create(mock.AnythingOfType("*context.cancelCtx"), testID, mock.AnythingOfType("map[domain.CollectorType]*domain.Stream")).Return(testURL, nil).Run(func(ctx context.Context, id domain.SupportArchiveID, streams map[domain.CollectorType]*domain.Stream) {
						logStream, ok := streams[domain.CollectorTypeLog]
						require.True(t, ok)
						require.NotNil(t, logStream)
					})
					return repoMock
				},
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					interfaceMock := newMockSupportArchiveV1Interface(t)
					clientMock := newMockSupportArchiveInterface(t)
					interfaceMock.EXPECT().SupportArchives(testArchiveNamespace).Return(clientMock)
					clientMock.EXPECT().UpdateStatusWithRetry(testCtx, testLogCR, mock.Anything, metav1.UpdateOptions{}).Return(nil, nil).Run(func(ctx context.Context, cr *libapi.SupportArchive, modifyStatusFn func(libapi.SupportArchiveStatus) libapi.SupportArchiveStatus, opts metav1.UpdateOptions) {
						updatedCRStatus := modifyStatusFn(cr.Status)
						assert.Equal(t, testURL, updatedCRStatus.DownloadPath)
						foundCondition := false
						for _, conditions := range updatedCRStatus.Conditions {
							if conditions.Type == libapi.ConditionSupportArchiveCreated && conditions.Status == metav1.ConditionTrue {
								foundCondition = true
							}
						}
						require.True(t, foundCondition)
					})
					return interfaceMock
				},
			},
			args: args{
				ctx: testCtx,
				cr:  testLogCR,
			},
			want: 0,
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var crMock supportArchiveV1Interface
			if tt.fields.supportArchivesInterface != nil {
				crMock = tt.fields.supportArchivesInterface(t)
			}

			var repoMock supportArchiveRepository
			if tt.fields.supportArchiveRepository != nil {
				repoMock = tt.fields.supportArchiveRepository(t)
			}

			var collectorMapping CollectorMapping
			if tt.fields.collectorMapping != nil {
				collectorMapping = tt.fields.collectorMapping(t)
			}

			c := &CreateArchiveUseCase{
				supportArchivesInterface: crMock,
				supportArchiveRepository: repoMock,
				collectorMapping:         collectorMapping,
			}
			got, err := c.HandleArchiveRequest(tt.args.ctx, tt.args.cr)
			tt.wantErr(t, err)
			assert.Equalf(t, tt.want, got, "HandleArchiveRequest(%v, %v)", tt.args.ctx, tt.args.cr)
		})
	}
}

func TestNewCreateArchiveUseCase(t *testing.T) {
	// given
	v1Mock := newMockSupportArchiveV1Interface(t)
	mapping := CollectorMapping{}
	repoMock := newMockSupportArchiveRepository(t)

	// when
	useCase := NewCreateArchiveUseCase(v1Mock, mapping, repoMock)

	// then
	require.NotNil(t, useCase)
	assert.Equal(t, v1Mock, useCase.supportArchivesInterface)
	assert.Equal(t, mapping, useCase.collectorMapping)
	assert.Equal(t, repoMock, useCase.supportArchiveRepository)
}

func TestCreateArchiveUseCase_updateFinalStatus(t *testing.T) {
	type fields struct {
		supportArchivesInterface func(t *testing.T) supportArchiveV1Interface
	}
	type args struct {
		ctx context.Context
		cr  *libapi.SupportArchive
		url string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should return error on error update status",
			fields: fields{
				supportArchivesInterface: func(t *testing.T) supportArchiveV1Interface {
					interfaceMock := newMockSupportArchiveV1Interface(t)
					clientMock := newMockSupportArchiveInterface(t)
					interfaceMock.EXPECT().SupportArchives(testArchiveNamespace).Return(clientMock)

					clientMock.EXPECT().UpdateStatusWithRetry(testCtx, testLogCR, mock.Anything, metav1.UpdateOptions{}).Return(nil, assert.AnError)
					return interfaceMock
				},
			},
			args: args{
				ctx: testCtx,
				cr:  testLogCR,
				url: testURL,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to set status for archive test-namespace/test-archive")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CreateArchiveUseCase{
				supportArchivesInterface: tt.fields.supportArchivesInterface(t),
			}
			tt.wantErr(t, c.updateFinalStatus(tt.args.ctx, tt.args.cr, tt.args.url))
		})
	}
}

func Test_streamFromRepository(t *testing.T) {
	type args[DATATYPE any] struct {
		ctx        context.Context
		repository func(t *testing.T) collectorRepository[domain.LogLine]
		id         domain.SupportArchiveID
		stream     *domain.Stream
	}
	type testCase[DATATYPE any] struct {
		name    string
		args    args[DATATYPE]
		wantErr func(t *testing.T, err error)
	}
	tests := []testCase[domain.LogLine]{
		{
			name: "should return error on error query isCollected",
			args: args[domain.LogLine]{
				ctx: testCtx,
				repository: func(t *testing.T) collectorRepository[domain.LogLine] {
					repoMock := newMockCollectorRepository[domain.LogLine](t)
					repoMock.EXPECT().IsCollected(testCtx, testID).Return(false, assert.AnError)

					return repoMock
				},
				id:     testID,
				stream: &domain.Stream{},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error during is collected call for collector")
			},
		},
		{
			name: "should return if repo is not complete",
			args: args[domain.LogLine]{
				ctx: testCtx,
				repository: func(t *testing.T) collectorRepository[domain.LogLine] {
					repoMock := newMockCollectorRepository[domain.LogLine](t)
					repoMock.EXPECT().IsCollected(testCtx, testID).Return(false, nil)

					return repoMock
				},
				id:     testID,
				stream: &domain.Stream{},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "collector is not completed")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := streamFromRepository[domain.LogLine](tt.args.ctx, tt.args.repository(t), tt.args.id, tt.args.stream)
			tt.wantErr(t, err)
		})
	}
}

func TestCollectorMapping_getRequiredCollectorMapping(t *testing.T) {
	t.Run("should return mapping for cr", func(t *testing.T) {
		// given
		cr := &libapi.SupportArchive{}

		logColRepo := CollectorAndRepository{Collector: "logs"}
		volumeColRepo := CollectorAndRepository{Collector: "volume"}
		nodeColRepo := CollectorAndRepository{Collector: "node"}
		secretColRepo := CollectorAndRepository{Collector: "logs"}
		systemStateColRepo := CollectorAndRepository{Collector: "logs"}

		sut := &CollectorMapping{
			domain.CollectorTypeLog:         logColRepo,
			domain.CollectorTypeVolumeInfo:  volumeColRepo,
			domain.CollectorTypeNodeInfo:    nodeColRepo,
			domain.CollectorTypSecret:       secretColRepo,
			domain.CollectorTypeSystemState: systemStateColRepo,
		}

		// when
		mapping := sut.getRequiredCollectorMapping(cr)

		// then
		require.NotNil(t, mapping)
		assert.Equal(t, logColRepo, mapping[domain.CollectorTypeLog])
		assert.Equal(t, volumeColRepo, mapping[domain.CollectorTypeVolumeInfo])
		assert.Equal(t, nodeColRepo, mapping[domain.CollectorTypeNodeInfo])
		assert.Equal(t, secretColRepo, mapping[domain.CollectorTypSecret])
		assert.Equal(t, systemStateColRepo, mapping[domain.CollectorTypeSystemState])
	})

	t.Run("should not add collector to mapping if excluded", func(t *testing.T) {
		// given
		cr := &libapi.SupportArchive{
			Spec: libapi.SupportArchiveSpec{
				ExcludedContents: libapi.ExcludedContents{
					SystemState:   true,
					SensitiveData: true,
					Events:        true,
					Logs:          true,
					VolumeInfo:    true,
					SystemInfo:    true,
				},
			},
		}

		logColRepo := CollectorAndRepository{Collector: "logs"}
		volumeColRepo := CollectorAndRepository{Collector: "volume"}
		nodeColRepo := CollectorAndRepository{Collector: "node"}
		secretColRepo := CollectorAndRepository{Collector: "logs"}
		systemStateColRepo := CollectorAndRepository{Collector: "logs"}

		sut := &CollectorMapping{
			domain.CollectorTypeLog:         logColRepo,
			domain.CollectorTypeVolumeInfo:  volumeColRepo,
			domain.CollectorTypeNodeInfo:    nodeColRepo,
			domain.CollectorTypSecret:       secretColRepo,
			domain.CollectorTypeSystemState: systemStateColRepo,
		}

		// when
		mapping := sut.getRequiredCollectorMapping(cr)

		// then
		require.NotNil(t, mapping)
		assert.Len(t, mapping, 0)
	})
}
