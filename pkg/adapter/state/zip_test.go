package state

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/fs"
	"os"
	"testing"
)

var (
	testCtx = context.Background()
)

const (
	testCollectorName = "Logs"
	testArchiveName   = "archive"
	testNamespace     = "namespace"
	testZipFilePath   = "logs/example.log"
	testZipPath       = "/data/support-archives/namespace/archive.zip"
	testZipDir        = "/data/support-archives/namespace"
	testStateFilePath = "/data/work/namespace/archive.json"
	testStateDir      = "/data/work/namespace"
)

func TestNewArchiver(t *testing.T) {
	filesystemMock := newMockVolumeFs(t)
	zipperCreatorMock := newMockZipCreator(t)
	actual := NewArchiver(filesystemMock, zipperCreatorMock)

	require.NotNil(t, actual)
	assert.Equal(t, filesystemMock, actual.filesystem)
	assert.Equal(t, zipperCreatorMock, actual.zipCreator)
}

func TestZipArchiver_Write(t *testing.T) {
	type fields struct {
		filesystem func(t *testing.T, zipFile closableRWFile) volumeFs
		zipCreator func(t *testing.T, zipFile closableRWFile, zipper zipper) zipCreator
		zipper     func(t *testing.T) zipper
		zipFile    func(t *testing.T) closableRWFile
	}
	type args struct {
		ctx           context.Context
		collectorName string
		name          string
		namespace     string
		zipFilePath   string
		writer        func(w io.Writer) error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			fields: fields{
				filesystem: func(t *testing.T, zipFile closableRWFile) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Stat(testZipPath).Return(nil, fs.ErrNotExist)
					filesystemMock.EXPECT().MkdirAll(testZipDir, os.FileMode(0755)).Return(nil)
					filesystemMock.EXPECT().Create(testZipPath).Return(nil, nil)
					filesystemMock.EXPECT().OpenFile(testZipPath, os.O_RDWR|os.O_APPEND, os.FileMode(0644)).Return(zipFile, nil)

					filesystemMock.EXPECT().Stat(testStateFilePath).Return(nil, nil).Times(3)
					filesystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_APPEND, os.FileMode(0644)).Return(nil, nil)

					actualState := []byte("{\"executedCollectors\": [\"OtherCollector\"]}")
					filesystemMock.EXPECT().ReadAll(nil).Return(actualState, nil)
					expectedState := []byte("{\"executedCollectors\":[\"OtherCollector\",\"Logs\"]}")
					filesystemMock.EXPECT().WriteFile(testStateFilePath, expectedState, os.FileMode(0644)).Return(nil)

					return filesystemMock
				},
				zipCreator: func(t *testing.T, zipFile closableRWFile, zipper zipper) zipCreator {
					zipperCreatorMock := newMockZipCreator(t)
					zipperCreatorMock.EXPECT().NewWriter(zipFile).Return(zipper)
					return zipperCreatorMock
				},
				zipFile: func(t *testing.T) closableRWFile {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					return fileMock
				},
				zipper: func(t *testing.T) zipper {
					zipperMock := newMockZipper(t)
					zipperMock.EXPECT().Create(testZipFilePath).Return(nil, nil)
					zipperMock.EXPECT().Close().Return(nil)

					return zipperMock
				},
			},
			args: args{
				ctx:           testCtx,
				collectorName: testCollectorName,
				name:          testArchiveName,
				namespace:     testNamespace,
				zipFilePath:   testZipFilePath,
				writer: func(w io.Writer) error {
					return nil
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zipperMock := tt.fields.zipper(t)
			zipFileMock := tt.fields.zipFile(t)

			a := &ZipArchiver{
				filesystem: tt.fields.filesystem(t, zipFileMock),
				zipCreator: tt.fields.zipCreator(t, zipFileMock, zipperMock),
			}
			tt.wantErr(t, a.Write(tt.args.ctx, tt.args.collectorName, tt.args.name, tt.args.namespace, tt.args.zipFilePath, tt.args.writer), fmt.Sprintf("Write(%v, %v, %v, %v, %v)", tt.args.ctx, tt.args.collectorName, tt.args.name, tt.args.namespace, tt.args.zipFilePath))
		})
	}
}

func TestZipArchiver_GetDownloadURL(t *testing.T) {
	archiver := &ZipArchiver{}

	url := archiver.GetDownloadURL(testCtx, testArchiveName, testNamespace)

	assert.Equal(t, "https://k8s-support-operator-webserver.namespace.svc.cluster.local/namespace/archive.zip", url, testArchiveName)
}

func TestZipArchiver_Read(t *testing.T) {
	type fields struct {
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		ctx       context.Context
		name      string
		namespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success with non existent state file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Stat(testStateFilePath).Return(nil, fs.ErrNotExist)

					return filesystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
			},
			want:    nil,
			wantErr: assert.NoError,
		},
		{
			name: "success with existent state file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Stat(testStateFilePath).Return(nil, nil).Times(2)
					fileMock := newMockClosableRWFile(t)
					filesystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_APPEND, os.FileMode(0644)).Return(fileMock, nil)
					data := []byte("{\"executedCollectors\":[\"Logs\",\"Resources\"]}")
					filesystemMock.EXPECT().ReadAll(fileMock).Return(data, nil)

					return filesystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
			},
			want:    []string{"Logs", "Resources"},
			wantErr: assert.NoError,
		},
		{
			name: "should return error on error stat state file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Stat(testStateFilePath).Return(nil, assert.AnError)

					return filesystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to stat state file")

				return false
			},
		},
		{
			name: "should return error on error reading state file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Stat(testStateFilePath).Return(nil, nil).Times(2)
					fileMock := newMockClosableRWFile(t)
					filesystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_APPEND, os.FileMode(0644)).Return(fileMock, nil)
					filesystemMock.EXPECT().ReadAll(fileMock).Return(nil, assert.AnError)

					return filesystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to read state file")

				return false
			},
		},
		{
			name: "should return error on error unmarshal state file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Stat(testStateFilePath).Return(nil, nil).Times(2)
					fileMock := newMockClosableRWFile(t)
					filesystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_APPEND, os.FileMode(0644)).Return(fileMock, nil)
					invalidData := []byte("{\"executedCollectors\":\"string instead of slice\"}")
					filesystemMock.EXPECT().ReadAll(fileMock).Return(invalidData, nil)

					return filesystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "failed to unmarshal state file")

				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ZipArchiver{
				filesystem: tt.fields.filesystem(t),
			}
			got, err := a.Read(tt.args.ctx, tt.args.name, tt.args.namespace)
			if !tt.wantErr(t, err, fmt.Sprintf("Read(%v, %v)", tt.args.name, tt.args.namespace)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Read(%v, %v)", tt.args.name, tt.args.namespace)
		})
	}
}

type testDirInfo struct{}

func (t testDirInfo) Name() string {
	return "dummy"
}

func (t testDirInfo) IsDir() bool {
	return false
}

func (t testDirInfo) Type() fs.FileMode {
	return fs.FileMode(0000)
}

func (t testDirInfo) Info() (fs.FileInfo, error) {
	return nil, nil
}

func TestZipArchiver_Clean(t *testing.T) {

	type fields struct {
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		ctx       context.Context
		name      string
		namespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should cleanup archive, state and empty dirs",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Remove(testZipPath).Return(nil)
					filesystemMock.EXPECT().ReadDir(testZipDir).Return([]fs.DirEntry{}, nil)
					filesystemMock.EXPECT().Remove(testZipDir).Return(nil)

					filesystemMock.EXPECT().Remove(testStateFilePath).Return(nil)
					filesystemMock.EXPECT().ReadDir(testStateDir).Return([]fs.DirEntry{}, nil)
					filesystemMock.EXPECT().Remove(testStateDir).Return(nil)

					return filesystemMock
				},
			},
			args: args{
				ctx:       testCtx,
				name:      testArchiveName,
				namespace: testNamespace,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should not remove dirs if not empty",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Remove(testZipPath).Return(nil)
					filesystemMock.EXPECT().ReadDir(testZipDir).Return([]fs.DirEntry{testDirInfo{}}, nil)

					filesystemMock.EXPECT().Remove(testStateFilePath).Return(nil)
					filesystemMock.EXPECT().ReadDir(testStateDir).Return([]fs.DirEntry{testDirInfo{}}, nil)

					return filesystemMock
				},
			},
			args: args{
				ctx:       testCtx,
				name:      testArchiveName,
				namespace: testNamespace,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return multiple errors",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Remove(testZipPath).Return(assert.AnError)
					filesystemMock.EXPECT().ReadDir(testZipDir).Return(nil, assert.AnError)

					filesystemMock.EXPECT().Remove(testStateFilePath).Return(assert.AnError)
					filesystemMock.EXPECT().ReadDir(testStateDir).Return([]fs.DirEntry{}, nil)
					filesystemMock.EXPECT().Remove(testStateDir).Return(assert.AnError)

					return filesystemMock
				},
			},
			args: args{
				ctx:       testCtx,
				name:      testArchiveName,
				namespace: testNamespace,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "failed to remove archive")
				assert.ErrorContains(t, err, "failed to remove state file")
				assert.ErrorContains(t, err, "error reading dir")
				assert.ErrorContains(t, err, "error removing empty dir")

				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ZipArchiver{
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, a.Clean(tt.args.ctx, tt.args.name, tt.args.namespace), fmt.Sprintf("Clean(%v, %v, %v)", tt.args.ctx, tt.args.name, tt.args.namespace))
		})
	}
}
