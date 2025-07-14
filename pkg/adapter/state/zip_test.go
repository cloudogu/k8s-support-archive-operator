package state

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	zipperMock := NewMockZipper(t)
	fn := func(w io.Writer) Zipper {
		return zipperMock
	}
	actual := NewArchiver(filesystemMock, fn, config.OperatorConfig{ArchiveVolumeDownloadServiceProtocol: "http", ArchiveVolumeDownloadServiceName: "name", ArchiveVolumeDownloadServicePort: "8080"})

	require.NotNil(t, actual)
	assert.Equal(t, filesystemMock, actual.filesystem)
	assert.Equal(t, zipperMock, actual.zipCreator(nil))
	assert.Equal(t, "http", actual.volumeDownloadServiceProtocol)
	assert.Equal(t, "name", actual.volumeDownloadServiceName)
	assert.Equal(t, "8080", actual.volumeDownloadServicePort)
}

func TestZipArchiver_GetDownloadURL(t *testing.T) {
	archiver := &ZipArchiver{volumeDownloadServiceProtocol: "http", volumeDownloadServicePort: "123", volumeDownloadServiceName: "test"}

	url := archiver.GetDownloadURL(testCtx, testArchiveName, testNamespace)

	assert.Equal(t, "http://test.namespace.svc.cluster.local:123/namespace/archive.zip", url, testArchiveName)
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
		name     string
		fields   fields
		args     args
		want     []string
		wantDone bool
		wantErr  assert.ErrorAssertionFunc
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
					filesystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
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
			name: "success with done state file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Stat(testStateFilePath).Return(nil, nil).Times(2)
					fileMock := newMockClosableRWFile(t)
					filesystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
					data := []byte("{\"done\":true,\"executedCollectors\":[\"Logs\",\"Resources\"]}")
					filesystemMock.EXPECT().ReadAll(fileMock).Return(data, nil)

					return filesystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
			},
			want:     []string{"Logs", "Resources"},
			wantDone: true,
			wantErr:  assert.NoError,
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
					filesystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
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
					filesystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
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
			got, done, err := a.Read(tt.args.ctx, tt.args.name, tt.args.namespace)
			if !tt.wantErr(t, err, fmt.Sprintf("Read(%v, %v)", tt.args.name, tt.args.namespace)) {
				return
			}
			assert.Equalf(t, tt.wantDone, done, "Read(%v, %v)", tt.args.name, tt.args.namespace)
			assert.Equalf(t, tt.want, got, "Read(%v, %v)", tt.args.name, tt.args.namespace)
		})
	}
}

type testDirInfo struct {
	dir bool
}

func (t testDirInfo) Name() string {
	return "dummy"
}

func (t testDirInfo) IsDir() bool {
	return t.dir
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

func TestZipArchiver_Write(t *testing.T) {
	type fields struct {
		filesystem func(t *testing.T) volumeFs
		zipCreator func(t *testing.T) zipCreator
	}
	type args struct {
		name        string
		namespace   string
		zipFilePath string
		writer      func(w io.Writer) error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should write file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().MkdirAll("/data/work/namespace/archive/logs", os.FileMode(0755)).Return(nil)
					filesystemMock.EXPECT().Create("/data/work/namespace/archive/logs/example.log").Return(nil, nil)

					return filesystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return nil
				},
			},
			args: args{
				name:        testArchiveName,
				namespace:   testNamespace,
				zipFilePath: testZipFilePath,
				writer: func(w io.Writer) error {
					return nil
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error creating dirs",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().MkdirAll("/data/work/namespace/archive/logs", os.FileMode(0755)).Return(assert.AnError)

					return filesystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return nil
				},
			},
			args: args{
				name:        testArchiveName,
				namespace:   testNamespace,
				zipFilePath: testZipFilePath,
				writer: func(w io.Writer) error {
					return nil
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "failed to create directory")
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error creating file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().MkdirAll("/data/work/namespace/archive/logs", os.FileMode(0755)).Return(nil)
					filesystemMock.EXPECT().Create("/data/work/namespace/archive/logs/example.log").Return(nil, assert.AnError)

					return filesystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return nil
				},
			},
			args: args{
				name:        testArchiveName,
				namespace:   testNamespace,
				zipFilePath: testZipFilePath,
				writer: func(w io.Writer) error {
					return nil
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "failed to create file")
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on write error",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().MkdirAll("/data/work/namespace/archive/logs", os.FileMode(0755)).Return(nil)
					filesystemMock.EXPECT().Create("/data/work/namespace/archive/logs/example.log").Return(nil, nil)

					return filesystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return nil
				},
			},
			args: args{
				name:        testArchiveName,
				namespace:   testNamespace,
				zipFilePath: testZipFilePath,
				writer: func(w io.Writer) error {
					return assert.AnError
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "failed to write file")
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ZipArchiver{
				filesystem: tt.fields.filesystem(t),
				zipCreator: tt.fields.zipCreator(t),
			}
			tt.wantErr(t, a.Write(nil, "", tt.args.name, tt.args.namespace, tt.args.zipFilePath, tt.args.writer))
		})
	}
}

func TestZipArchiver_WriteState(t *testing.T) {
	type fields struct {
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		in0       context.Context
		name      string
		namespace string
		stateName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should add to empty state",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testStateFilePath).Return(nil, fs.ErrNotExist).Times(2)
					fileSystemMock.EXPECT().MkdirAll(testStateDir, os.FileMode(0755)).Return(nil)
					fileSystemMock.EXPECT().Create(testStateFilePath).Return(nil, nil)
					fileSystemMock.EXPECT().WriteFile(testStateFilePath, []byte("{\"done\":false,\"executedCollectors\":[\"Logs\"]}"), os.FileMode(0644)).Return(nil)
					return fileSystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
				stateName: "Logs",
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should add to existing state",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testStateFilePath).Return(nil, nil).Times(3)
					fileMock := newMockClosableRWFile(t)
					fileSystemMock.EXPECT().ReadAll(fileMock).Return([]byte("{\"done\":false,\"executedCollectors\":[\"Logs\"]}"), nil)
					fileSystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
					fileSystemMock.EXPECT().WriteFile(testStateFilePath, []byte("{\"done\":false,\"executedCollectors\":[\"Logs\",\"Resources\"]}"), os.FileMode(0644)).Return(nil)
					return fileSystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
				stateName: "Resources",
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on parse error",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testStateFilePath).Return(nil, assert.AnError)
					return fileSystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
				stateName: "Logs",
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on write error",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testStateFilePath).Return(nil, fs.ErrNotExist).Times(2)
					fileSystemMock.EXPECT().MkdirAll(testStateDir, os.FileMode(0755)).Return(nil)
					fileSystemMock.EXPECT().Create(testStateFilePath).Return(nil, nil)
					fileSystemMock.EXPECT().WriteFile(testStateFilePath, []byte("{\"done\":false,\"executedCollectors\":[\"Logs\"]}"), os.FileMode(0644)).Return(assert.AnError)
					return fileSystemMock
				},
			},
			args: args{
				name:      testArchiveName,
				namespace: testNamespace,
				stateName: "Logs",
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ZipArchiver{
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, a.WriteState(tt.args.in0, tt.args.name, tt.args.namespace, tt.args.stateName))
		})
	}
}

func TestZipArchiver_Finalize(t *testing.T) {
	type fields struct {
		filesystem func(t *testing.T) volumeFs
		zipCreator func(t *testing.T) zipCreator
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
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "success",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testZipPath).Return(nil, nil).Return(nil, nil).Times(1)
					fileSystemMock.EXPECT().OpenFile(testZipPath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
					fileSystemMock.EXPECT().Stat(testStateFilePath).Return(nil, nil).Return(nil, nil).Times(3)
					fileSystemMock.EXPECT().OpenFile(testStateFilePath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
					fileSystemMock.EXPECT().WalkDir("/data/work/namespace/archive", mock.Anything).Return(nil)
					fileSystemMock.EXPECT().RemoveAll("/data/work/namespace/archive").Return(nil)
					fileSystemMock.EXPECT().ReadAll(fileMock).Return([]byte("{\"done\":false,\"executedCollectors\":[\"Logs\"]}"), nil)
					fileSystemMock.EXPECT().WriteFile(testStateFilePath, []byte("{\"done\":true,\"executedCollectors\":[\"Logs\"]}"), os.FileMode(0644)).Return(nil)

					return fileSystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return func(w io.Writer) Zipper {
						zipper := NewMockZipper(t)
						zipper.EXPECT().Close().Return(nil)
						return zipper
					}
				},
			},
			args: args{
				ctx:       context.Background(),
				name:      testArchiveName,
				namespace: testNamespace,
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error create zip archive",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testZipPath).Return(nil, nil).Return(nil, nil).Times(1)
					fileSystemMock.EXPECT().OpenFile(testZipPath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(nil, assert.AnError)

					return fileSystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return func(w io.Writer) Zipper {
						zipper := NewMockZipper(t)
						zipper.EXPECT().Close().Return(nil)
						return zipper
					}
				},
			},
			args: args{
				ctx:       context.Background(),
				name:      testArchiveName,
				namespace: testNamespace,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error walking files",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testZipPath).Return(nil, nil).Return(nil, nil).Times(1)
					fileSystemMock.EXPECT().OpenFile(testZipPath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
					fileSystemMock.EXPECT().WalkDir("/data/work/namespace/archive", mock.Anything).Return(assert.AnError)

					return fileSystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return func(w io.Writer) Zipper {
						zipper := NewMockZipper(t)
						zipper.EXPECT().Close().Return(nil)
						return zipper
					}
				},
			},
			args: args{
				ctx:       context.Background(),
				name:      testArchiveName,
				namespace: testNamespace,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to copy files to zip archive")
			},
		},
		{
			name: "should return error on error removing state files",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testZipPath).Return(nil, nil).Return(nil, nil).Times(1)
					fileSystemMock.EXPECT().OpenFile(testZipPath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
					fileSystemMock.EXPECT().WalkDir("/data/work/namespace/archive", mock.Anything).Return(nil)
					fileSystemMock.EXPECT().RemoveAll("/data/work/namespace/archive").Return(assert.AnError)

					return fileSystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return func(w io.Writer) Zipper {
						zipper := NewMockZipper(t)
						zipper.EXPECT().Close().Return(nil)
						return zipper
					}
				},
			},
			args: args{
				ctx:       context.Background(),
				name:      testArchiveName,
				namespace: testNamespace,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to remove state files")
			},
		},
		{
			name: "should return error on error parsing actual state",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testZipPath).Return(nil, nil).Return(nil, nil)
					fileSystemMock.EXPECT().OpenFile(testZipPath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
					fileSystemMock.EXPECT().WalkDir("/data/work/namespace/archive", mock.Anything).Return(nil)
					fileSystemMock.EXPECT().RemoveAll("/data/work/namespace/archive").Return(nil)
					fileSystemMock.EXPECT().Stat(testStateFilePath).Return(nil, nil).Return(nil, assert.AnError)

					return fileSystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return func(w io.Writer) Zipper {
						zipper := NewMockZipper(t)
						zipper.EXPECT().Close().Return(nil)
						return zipper
					}
				},
			},
			args: args{
				ctx:       context.Background(),
				name:      testArchiveName,
				namespace: testNamespace,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return error on error writing actual state",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileSystemMock := newMockVolumeFs(t)
					fileSystemMock.EXPECT().Stat(testZipPath).Return(nil, nil).Return(nil, nil)
					fileSystemMock.EXPECT().OpenFile(testZipPath, os.O_RDWR|os.O_CREATE, os.FileMode(0644)).Return(fileMock, nil)
					fileSystemMock.EXPECT().WalkDir("/data/work/namespace/archive", mock.Anything).Return(nil)
					fileSystemMock.EXPECT().RemoveAll("/data/work/namespace/archive").Return(nil)
					fileSystemMock.EXPECT().Stat(testStateFilePath).Return(nil, fs.ErrNotExist)
					fileSystemMock.EXPECT().MkdirAll(testStateDir, os.FileMode(0755)).Return(assert.AnError)

					return fileSystemMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					return func(w io.Writer) Zipper {
						zipper := NewMockZipper(t)
						zipper.EXPECT().Close().Return(nil)
						return zipper
					}
				},
			},
			args: args{
				ctx:       context.Background(),
				name:      testArchiveName,
				namespace: testNamespace,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ZipArchiver{
				filesystem: tt.fields.filesystem(t),
				zipCreator: tt.fields.zipCreator(t),
			}
			tt.wantErr(t, a.Finalize(tt.args.ctx, tt.args.name, tt.args.namespace))
		})
	}
}

func TestZipArchiver_CopyFileToArchive(t *testing.T) {
	type fields struct {
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		zipper          func(t *testing.T) Zipper
		stateArchiveDir string
		path            string
		d               fs.DirEntry
		err             error
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
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Open("/data/work/namespace/archive/logs/example.log").Return(nil, nil)
					filesystemMock.EXPECT().Copy(mock.Anything, mock.Anything).Return(0, nil)

					return filesystemMock
				},
			},
			args: args{
				zipper: func(t *testing.T) Zipper {
					zipperMock := NewMockZipper(t)
					zipperMock.EXPECT().Create(testZipFilePath).Return(nil, nil)

					return zipperMock
				},
				stateArchiveDir: "/data/work/namespace/archive",
				path:            "/data/work/namespace/archive/logs/example.log",
				d:               testDirInfo{},
				err:             nil,
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on walk error",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					return nil
				},
			},
			args: args{
				zipper: func(t *testing.T) Zipper {
					return nil
				},
				stateArchiveDir: "/data/work/namespace/archive",
				path:            "/data/work/namespace/archive/logs/example.log",
				d:               testDirInfo{},
				err:             assert.AnError,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should return nil on dir",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					return nil
				},
			},
			args: args{
				zipper: func(t *testing.T) Zipper {
					return nil
				},
				stateArchiveDir: "/data/work/namespace/archive",
				path:            "/data/work/namespace/archive/logs/example.log",
				d:               testDirInfo{dir: true},
				err:             nil,
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error creating zip file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					return nil
				},
			},
			args: args{
				zipper: func(t *testing.T) Zipper {
					zipperMock := NewMockZipper(t)
					zipperMock.EXPECT().Create(testZipFilePath).Return(nil, assert.AnError)

					return zipperMock
				},
				stateArchiveDir: "/data/work/namespace/archive",
				path:            "/data/work/namespace/archive/logs/example.log",
				d:               testDirInfo{},
				err:             nil,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to create zip writer for file")
			},
		},
		{
			name: "should return error on error open source file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Open("/data/work/namespace/archive/logs/example.log").Return(nil, assert.AnError)

					return filesystemMock
				},
			},
			args: args{
				zipper: func(t *testing.T) Zipper {
					zipperMock := NewMockZipper(t)
					zipperMock.EXPECT().Create(testZipFilePath).Return(nil, nil)

					return zipperMock
				},
				stateArchiveDir: "/data/work/namespace/archive",
				path:            "/data/work/namespace/archive/logs/example.log",
				d:               testDirInfo{},
				err:             nil,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to open file")
			},
		},
		{
			name: "should return error on error copy file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					filesystemMock := newMockVolumeFs(t)
					filesystemMock.EXPECT().Open("/data/work/namespace/archive/logs/example.log").Return(nil, nil)
					filesystemMock.EXPECT().Copy(mock.Anything, mock.Anything).Return(0, assert.AnError)

					return filesystemMock
				},
			},
			args: args{
				zipper: func(t *testing.T) Zipper {
					zipperMock := NewMockZipper(t)
					zipperMock.EXPECT().Create(testZipFilePath).Return(nil, nil)

					return zipperMock
				},
				stateArchiveDir: "/data/work/namespace/archive",
				path:            "/data/work/namespace/archive/logs/example.log",
				d:               testDirInfo{},
				err:             nil,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to copy file")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &ZipArchiver{
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, a.copyFileToArchive(tt.args.zipper(t), tt.args.stateArchiveDir, tt.args.path, tt.args.d, tt.args.err))
		})
	}
}
