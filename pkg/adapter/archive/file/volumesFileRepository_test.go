package file

import (
	"context"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

const (
	testVolumeCollectorDirName   = "VolumeInfo"
	testVolumeWorkDirArchivePath = testWorkPath + "/" + testNamespace + "/" + testName + "/" + testVolumeCollectorDirName
	testVolumeWorkFile           = testVolumeWorkDirArchivePath + "/pvcs.yaml"
)

func TestNewVolumesFileRepository(t *testing.T) {
	// given
	fsMock := newMockVolumeFs(t)
	baseRepoMock := newMockBaseFileRepo(t)

	// when
	repository := NewVolumesFileRepository(testWorkPath, fsMock, baseRepoMock)

	// then
	assert.NotNil(t, repository)
	assert.Equal(t, testWorkPath, repository.workPath)
	assert.Equal(t, fsMock, repository.filesystem)
	assert.Equal(t, baseRepoMock, repository.baseFileRepo)
}

func TestVolumesRepository_Stream(t *testing.T) {
	t.Run("should delegate to base repo", func(t *testing.T) {
		// given
		baseRepoMock := newMockBaseFileRepo(t)
		testStream := &domain.Stream{}
		baseRepoMock.EXPECT().stream(testCtx, testID, testVolumeCollectorDirName, testStream).Return(nil, nil)

		sut := &VolumesFileRepository{baseFileRepo: baseRepoMock}

		// when
		_, err := sut.Stream(testCtx, testID, testStream)

		// then
		require.NoError(t, err)
	})
}

func TestVolumesRepository_Delete(t *testing.T) {
	t.Run("should delegate to base repo", func(t *testing.T) {
		// given
		baseRepoMock := newMockBaseFileRepo(t)
		baseRepoMock.EXPECT().Delete(testCtx, testID, testVolumeCollectorDirName).Return(nil)

		sut := &VolumesFileRepository{baseFileRepo: baseRepoMock}

		// when
		err := sut.Delete(testCtx, testID)

		// then
		require.NoError(t, err)
	})
}

func TestVolumesRepository_FinishCollection(t *testing.T) {
	t.Run("should delegate to base repo", func(t *testing.T) {
		// given
		baseRepoMock := newMockBaseFileRepo(t)
		baseRepoMock.EXPECT().FinishCollection(testCtx, testID, testVolumeCollectorDirName).Return(nil)

		sut := &VolumesFileRepository{baseFileRepo: baseRepoMock}

		// when
		err := sut.FinishCollection(testCtx, testID)

		// then
		require.NoError(t, err)
	})
}

func TestVolumesRepository_IsCollected(t *testing.T) {
	t.Run("should delegate to base repo", func(t *testing.T) {
		// given
		baseRepoMock := newMockBaseFileRepo(t)
		baseRepoMock.EXPECT().IsCollected(testCtx, testID, testVolumeCollectorDirName).Return(false, nil)

		sut := &VolumesFileRepository{baseFileRepo: baseRepoMock}

		// when
		collected, err := sut.IsCollected(testCtx, testID)

		// then
		require.NoError(t, err)
		assert.False(t, collected)
	})
}

func TestVolumesFileRepository_createVolumeInfo(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		ctx  context.Context
		id   domain.SupportArchiveID
		data *domain.VolumeInfo
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should return error on error creating directory",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testVolumeWorkDirArchivePath, os.FileMode(0755)).Return(assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx:  testCtx,
				id:   testID,
				data: &domain.VolumeInfo{Name: "pvcs"},
			},
			wantErr: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error creating directory for volume metrics file")
			},
		},
		{
			name: "should return error on error writing file",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testVolumeWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().WriteFile(testVolumeWorkFile, mock.Anything, os.FileMode(0644)).Return(assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx:  testCtx,
				id:   testID,
				data: &domain.VolumeInfo{Name: "pvcs"},
			},
			wantErr: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error creating volume metrics file")
			},
		},
		{
			name: "should return nil on success",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testVolumeWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().WriteFile(testVolumeWorkFile, mock.Anything, os.FileMode(0644)).Return(nil)

					return fsMock
				},
			},
			args: args{
				ctx:  testCtx,
				id:   testID,
				data: &domain.VolumeInfo{Name: "pvcs"},
			},
			wantErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &VolumesFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, v.createVolumeInfo(tt.args.ctx, tt.args.id, tt.args.data))
		})
	}
}
