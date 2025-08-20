package usecase

import (
	"context"
	libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var (
	testID = domain.SupportArchiveID{
		Namespace: testArchiveNamespace,
		Name:      testArchiveName,
	}

	testLogCR = &libapi.SupportArchive{ObjectMeta: metav1.ObjectMeta{Namespace: testArchiveNamespace, Name: testArchiveName}, Spec: libapi.SupportArchiveSpec{ExcludedContents: libapi.ExcludedContents{VolumeInfo: true, SystemState: true, SensitiveData: true, Events: true, SystemInfo: true}}}
)

func TestDeleteArchiveUseCase_Delete(t *testing.T) {
	type fields struct {
		supportArchiveRepository func(t *testing.T) supportArchiveRepository
		collectorMapping         func(t *testing.T) CollectorMapping
	}
	type args struct {
		ctx context.Context
		id  domain.SupportArchiveID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should delete all collector data und archives for cr",
			fields: fields{
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					repoMock := newMockSupportArchiveRepository(t)
					repoMock.EXPECT().Delete(testCtx, testID).Return(nil)

					return repoMock
				},
				collectorMapping: func(t *testing.T) CollectorMapping {
					mapping := CollectorMapping{}

					logRepoMock := newMockBaseCollectorRepository(t)
					logRepoMock.EXPECT().Delete(testCtx, testID).Return(nil)
					volumeRepoMock := newMockBaseCollectorRepository(t)
					volumeRepoMock.EXPECT().Delete(testCtx, testID).Return(nil)

					mapping[domain.CollectorTypeLog] = CollectorAndRepository{
						Repository: logRepoMock,
					}
					mapping[domain.CollectorTypeVolumeInfo] = CollectorAndRepository{
						Repository: volumeRepoMock,
					}

					return mapping
				},
			},
			args: args{
				ctx: context.Background(),
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return multi error",
			fields: fields{
				supportArchiveRepository: func(t *testing.T) supportArchiveRepository {
					repoMock := newMockSupportArchiveRepository(t)
					repoMock.EXPECT().Delete(testCtx, testID).Return(assert.AnError)

					return repoMock
				},
				collectorMapping: func(t *testing.T) CollectorMapping {
					mapping := CollectorMapping{}

					logRepoMock := newMockBaseCollectorRepository(t)
					logRepoMock.EXPECT().Delete(testCtx, testID).Return(assert.AnError)

					mapping[domain.CollectorTypeLog] = CollectorAndRepository{
						Repository: logRepoMock,
					}

					return mapping
				},
			},
			args: args{
				ctx: context.Background(),
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "failed to delete Logs collector repository")
				assert.ErrorContains(t, err, "failed to delete support archive")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DeleteArchiveUseCase{
				supportArchiveRepository: tt.fields.supportArchiveRepository(t),
				collectorMapping:         tt.fields.collectorMapping(t),
			}
			tt.wantErr(t, d.Delete(tt.args.ctx, tt.args.id))
		})
	}
}

func TestNewDeleteArchiveUseCase(t *testing.T) {
	// given
	repoMock := newMockSupportArchiveRepository(t)
	mapping := CollectorMapping{}

	// when
	result := NewDeleteArchiveUseCase(mapping, repoMock)

	// then
	require.NotNil(t, result)
	assert.Equal(t, repoMock, result.supportArchiveRepository)
	assert.Equal(t, mapping, result.collectorMapping)
}
