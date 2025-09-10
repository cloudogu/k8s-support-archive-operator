package file

import (
	"context"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"os"
	"testing"
)

const (
	testSystemStateCollectorDirName   = "Resources/SystemState"
	testSystemStateWorkDirArchivePath = testWorkPath + "/" + testNamespace + "/" + testName + "/" + testSystemStateCollectorDirName + "/apps/v1"
	testSystemStateWorkFile           = testSystemStateWorkDirArchivePath + "/deployment.yaml"
)

func TestNewSystemStateFileRepository(t *testing.T) {
	// given
	fsMock := newMockVolumeFs(t)

	// when
	repository := NewSystemStateFileRepository(testWorkPath, fsMock)

	// then
	assert.NotNil(t, repository)
	assert.Equal(t, testWorkPath, repository.workPath)
	assert.Equal(t, fsMock, repository.filesystem)
	assert.NotEmpty(t, repository.baseFileRepo)
}

func TestSystemStateFileRepository_createSystemStateInfo(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		ctx  context.Context
		id   domain.SupportArchiveID
		data *domain.UnstructuredResource
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
					fsMock.EXPECT().MkdirAll(testSystemStateWorkDirArchivePath, os.FileMode(0755)).Return(assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx:  testCtx,
				id:   testID,
				data: &domain.UnstructuredResource{Name: "deployment", Path: "apps/v1"},
			},
			wantErr: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error creating directory for file")
			},
		},
		{
			name: "should return error on error writing file",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testSystemStateWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().WriteFile(testSystemStateWorkFile, mock.Anything, os.FileMode(0644)).Return(assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx:  testCtx,
				id:   testID,
				data: &domain.UnstructuredResource{Name: "deployment", Path: "apps/v1"},
			},
			wantErr: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error creating file")
			},
		},
		{
			name: "should return nil on success",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testSystemStateWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().WriteFile(testSystemStateWorkFile, mock.Anything, os.FileMode(0644)).Return(nil)

					return fsMock
				},
			},
			args: args{
				ctx:  testCtx,
				id:   testID,
				data: &domain.UnstructuredResource{Name: "deployment", Path: "apps/v1"},
			},
			wantErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &SystemStateFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, v.createSystemState(tt.args.ctx, tt.args.id, tt.args.data))
		})
	}
}
