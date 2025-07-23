package file

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"testing"
)

const (
	testStateFilePath = testWorkDirArchivePath + "/.done"
)

func Test_baseFileRepository_FinishCollection(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T) volumeFs
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
			name: "success",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileMock.EXPECT().Write([]byte("done")).Return(0, nil)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testWorkDirArchivePath, fs.ModePerm).Return(nil)
					fsMock.EXPECT().Create(testStateFilePath).Return(fileMock, nil)

					return fsMock
				},
				workPath: testWorkPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error write state file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileMock.EXPECT().Write([]byte("done")).Return(0, assert.AnError)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testWorkDirArchivePath, fs.ModePerm).Return(nil)
					fsMock.EXPECT().Create(testStateFilePath).Return(fileMock, nil)

					return fsMock
				},
				workPath: testWorkPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to write to file")
			},
		},
		{
			name: "should return error on error create state file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testWorkDirArchivePath, fs.ModePerm).Return(nil)
					fsMock.EXPECT().Create(testStateFilePath).Return(nil, assert.AnError)

					return fsMock
				},
				workPath: testWorkPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to create file")
			},
		},
		{
			name: "should return error on error create dirs",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testWorkDirArchivePath, fs.ModePerm).Return(assert.AnError)

					return fsMock
				},
				workPath: testWorkPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to create directory")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &baseFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, l.FinishCollection(tt.args.ctx, tt.args.id))
		})
	}
}

func Test_baseFileRepository_IsCollected(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		ctx context.Context
		id  domain.SupportArchiveID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should return true if state file exists",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Stat(testStateFilePath).Return(nil, nil)
					return fsMock
				},
				workPath: testWorkPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			want:    true,
			wantErr: assert.NoError,
		},
		{
			name: "should return false if state file does not exist",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Stat(testStateFilePath).Return(nil, os.ErrNotExist)
					return fsMock
				},
				workPath: testWorkPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			want:    false,
			wantErr: assert.NoError,
		},
		{
			name: "should return error on error stat file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Stat(testStateFilePath).Return(nil, assert.AnError)
					return fsMock
				},
				workPath: testWorkPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			want:    false,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &baseFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: tt.fields.filesystem(t),
			}
			got, err := l.IsCollected(tt.args.ctx, tt.args.id)
			if !tt.wantErr(t, err, fmt.Sprintf("IsCollected(%v, %v)", tt.args.ctx, tt.args.id)) {
				return
			}
			assert.Equalf(t, tt.want, got, "IsCollected(%v, %v)", tt.args.ctx, tt.args.id)
		})
	}
}

func TestNewBaseFileRepository(t *testing.T) {
	// given
	fsMock := newMockVolumeFs(t)

	// when
	repository := NewBaseFileRepository(testWorkPath, fsMock)

	// then
	assert.NotNil(t, repository)
	assert.Equal(t, testWorkPath, repository.workPath)
	assert.Equal(t, fsMock, repository.filesystem)
}
