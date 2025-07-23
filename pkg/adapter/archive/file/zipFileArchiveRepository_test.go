package file

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/config"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/fs"
	"os"
	"testing"
	"time"
)

const (
	testArchivesPath  = "test-archives"
	testNamespace     = "ecosystem"
	testName          = "archive-123"
	testArchivePath   = testArchivesPath + "/" + testNamespace + "/" + testName + ".zip"
	testNamespacePath = testArchivesPath + "/" + testNamespace
	testArchiveURL    = "https://servicename.ecosystem.svc.cluster.local:8080/ecosystem/archive-123.zip"
	testServiceName   = "servicename"
	testProtocol      = "https"
	testPort          = "8080"
)

var (
	testCtx = context.Background()
	testID  = domain.SupportArchiveID{
		Namespace: testNamespace,
		Name:      testName,
	}
)

func TestZipFileArchiveRepository_Exists(t *testing.T) {
	type fields struct {
		filesystem  func(t *testing.T) volumeFs
		archivePath string
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
		wantErr bool
	}{
		{
			name: "should return true if file exists",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Stat(testArchivePath).Return(nil, nil)
					return fsMock
				},
				archivePath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "should return false if file does not exist",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Stat(testArchivePath).Return(nil, fs.ErrNotExist)
					return fsMock
				},
				archivePath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "should return error on stat error",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Stat(testArchivePath).Return(nil, assert.AnError)
					return fsMock
				},
				archivePath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &ZipFileArchiveRepository{
				filesystem:   tt.fields.filesystem(t),
				archivesPath: tt.fields.archivePath,
			}
			got, err := z.Exists(tt.args.ctx, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Exists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestZipFileArchiveRepository_Delete(t *testing.T) {
	type fields struct {
		filesystem  func(t *testing.T) volumeFs
		archivePath string
	}
	type args struct {
		ctx context.Context
		id  domain.SupportArchiveID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "should delete archive and empty dir with one existent file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Remove(testArchivePath).Return(nil)
					fsMock.EXPECT().ReadDir(testNamespacePath).Return(nil, nil)
					fsMock.EXPECT().Remove(testNamespacePath).Return(nil)

					return fsMock
				},
				archivePath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should not delete archive namespace if other files exist",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Remove(testArchivePath).Return(nil)
					fsMock.EXPECT().ReadDir(testNamespacePath).Return([]os.DirEntry{testEntry{}}, nil)

					return fsMock
				},
				archivePath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return error on error removing archive",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Remove(testArchivePath).Return(assert.AnError)
					fsMock.EXPECT().ReadDir(testNamespacePath).Return([]os.DirEntry{testEntry{}}, nil)

					return fsMock
				},
				archivePath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: assert.Error,
		},
		{
			name: "should return error on error reading namespace dir",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Remove(testArchivePath).Return(nil)
					fsMock.EXPECT().ReadDir(testNamespacePath).Return([]os.DirEntry{testEntry{}}, assert.AnError)

					return fsMock
				},
				archivePath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: assert.Error,
		},
		{
			name: "should return error on error removing namespace dir",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Remove(testArchivePath).Return(nil)
					fsMock.EXPECT().ReadDir(testNamespacePath).Return(nil, nil)
					fsMock.EXPECT().Remove(testNamespacePath).Return(assert.AnError)

					return fsMock
				},
				archivePath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z := &ZipFileArchiveRepository{
				filesystem:   tt.fields.filesystem(t),
				archivesPath: tt.fields.archivePath,
			}
			tt.wantErr(t, z.Delete(tt.args.ctx, tt.args.id), fmt.Sprintf("Delete(%v, %v)", tt.args.ctx, tt.args.id))
		})
	}
}

type testEntry struct {
	name string
}

func (t testEntry) Name() string {
	return t.name
}

func (t testEntry) IsDir() bool {
	panic("implement me")
}

func (t testEntry) Type() fs.FileMode {
	panic("implement me")
}

func (t testEntry) Info() (fs.FileInfo, error) {
	panic("implement me")
}

func TestZipFileArchiveRepository_Create(t *testing.T) {
	casWriter := newMockClosableRWFile(t)
	casReader := NewMockReader(t)
	ldapWriter := newMockClosableRWFile(t)
	ldapReader := newMockClosableRWFile(t)

	type fields struct {
		filesystem                           func(t *testing.T) volumeFs
		zipCreator                           func(t *testing.T) zipCreator
		archivesPath                         string
		archiveVolumeDownloadServiceName     string
		archiveVolumeDownloadServicePort     string
		archiveVolumeDownloadServiceProtocol string
	}
	type args struct {
		ctx     context.Context
		id      domain.SupportArchiveID
		streams map[domain.CollectorType]domain.Stream
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should return error on create namespace dir",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testNamespacePath, os.FileMode(0755)).Return(assert.AnError)
					return fsMock
				},
				archivesPath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to create zip archive directory")
			},
		},
		{
			name: "should return error on open zip file",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testNamespacePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().OpenFile(testArchivePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0644)).Return(nil, assert.AnError)
					return fsMock
				},
				archivesPath: testArchivesPath,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to open file")
			},
		},
		{
			name: "should copy data from stream to zip archive",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testNamespacePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().OpenFile(testArchivePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0644)).Return(fileMock, nil)

					fsMock.EXPECT().Copy(casWriter, casReader).Return(0, nil)
					fsMock.EXPECT().Copy(ldapWriter, ldapReader).Return(0, nil)

					return fsMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					zipMock := NewMockZipper(t)
					zipMock.EXPECT().Close().Return(nil)

					zipMock.EXPECT().Create("Logs/cas.log").Return(casWriter, nil)
					zipMock.EXPECT().Create("Logs/ldap.log").Return(ldapWriter, nil)

					return func(w io.Writer) Zipper {
						return zipMock
					}
				},
				archivesPath:                         testArchivesPath,
				archiveVolumeDownloadServicePort:     testPort,
				archiveVolumeDownloadServiceName:     testServiceName,
				archiveVolumeDownloadServiceProtocol: testProtocol,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				streams: map[domain.CollectorType]domain.Stream{
					domain.CollectorTypeLog: getTestStream(casReader, ldapReader, true),
				},
			},
			want: testArchiveURL,
		},
		{
			name: "should return error creating the zip file writer",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testNamespacePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().OpenFile(testArchivePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0644)).Return(fileMock, nil)

					fsMock.EXPECT().Remove(testArchivePath).Return(nil)

					return fsMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					zipMock := NewMockZipper(t)
					zipMock.EXPECT().Close().Return(nil)
					zipMock.EXPECT().Create("Logs/cas.log").Return(casWriter, assert.AnError)

					return func(w io.Writer) Zipper {
						return zipMock
					}
				},
				archivesPath:                         testArchivesPath,
				archiveVolumeDownloadServicePort:     testPort,
				archiveVolumeDownloadServiceName:     testServiceName,
				archiveVolumeDownloadServiceProtocol: testProtocol,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				streams: map[domain.CollectorType]domain.Stream{
					domain.CollectorTypeLog: getTestStream(casReader, ldapReader, false),
				},
			},
			wantErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error streaming data: failed to create zip writer for file /cas.log")
			},
		},
		{
			name: "should return error on error copy data",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testNamespacePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().OpenFile(testArchivePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0644)).Return(fileMock, nil)

					fsMock.EXPECT().Copy(casWriter, casReader).Return(0, assert.AnError)

					fsMock.EXPECT().Remove(testArchivePath).Return(nil)

					return fsMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					zipMock := NewMockZipper(t)
					zipMock.EXPECT().Close().Return(nil)

					zipMock.EXPECT().Create("Logs/cas.log").Return(casWriter, nil)

					return func(w io.Writer) Zipper {
						return zipMock
					}
				},
				archivesPath:                         testArchivesPath,
				archiveVolumeDownloadServicePort:     testPort,
				archiveVolumeDownloadServiceName:     testServiceName,
				archiveVolumeDownloadServiceProtocol: testProtocol,
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				streams: map[domain.CollectorType]domain.Stream{
					domain.CollectorTypeLog: getTestStream(casReader, ldapReader, false),
				},
			},
			wantErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error streaming data: failed to copy file /cas.log")
			},
		},
		{
			name: "should return error on canceled context",
			fields: fields{
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testNamespacePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().OpenFile(testArchivePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(0644)).Return(fileMock, nil)

					fsMock.EXPECT().Remove(testArchivePath).Return(nil)

					return fsMock
				},
				zipCreator: func(t *testing.T) zipCreator {
					zipMock := NewMockZipper(t)
					zipMock.EXPECT().Close().Return(nil)

					return func(w io.Writer) Zipper {
						return zipMock
					}
				},
				archivesPath:                         testArchivesPath,
				archiveVolumeDownloadServicePort:     testPort,
				archiveVolumeDownloadServiceName:     testServiceName,
				archiveVolumeDownloadServiceProtocol: testProtocol,
			},
			args: args{
				ctx: getDeadlineContext(testCtx),
				id:  testID,
				streams: map[domain.CollectorType]domain.Stream{
					domain.CollectorTypeLog: getEmptyStream(),
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "context deadline exceeded")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var creator zipCreator
			if tt.fields.zipCreator != nil {
				creator = tt.fields.zipCreator(t)
			}

			var filesystem volumeFs
			if tt.fields.filesystem != nil {
				filesystem = tt.fields.filesystem(t)
			}

			z := &ZipFileArchiveRepository{
				filesystem:                           filesystem,
				zipCreator:                           creator,
				archivesPath:                         tt.fields.archivesPath,
				archiveVolumeDownloadServiceName:     tt.fields.archiveVolumeDownloadServiceName,
				archiveVolumeDownloadServicePort:     tt.fields.archiveVolumeDownloadServicePort,
				archiveVolumeDownloadServiceProtocol: tt.fields.archiveVolumeDownloadServiceProtocol,
			}
			got, err := z.Create(tt.args.ctx, tt.args.id, tt.args.streams)
			if err != nil {
				tt.wantErr(t, err)
				return
			}
			assert.Equalf(t, tt.want, got, "Create(%v, %v, %v)", tt.args.ctx, tt.args.id, tt.args.streams)
		})
	}
}

func getTestStream(casReader io.Reader, ldapReader io.Reader, closeStream bool) domain.Stream {
	stream := domain.Stream{
		Data: make(chan domain.StreamData),
	}

	go func() {
		stream.Data <- domain.StreamData{
			ID:     "/cas.log",
			Reader: casReader,
		}
		stream.Data <- domain.StreamData{
			ID:     "/ldap.log",
			Reader: ldapReader,
		}

		if closeStream {
			close(stream.Data)
		}
	}()

	return stream
}

func getEmptyStream() domain.Stream {
	stream := domain.Stream{
		Data: make(chan domain.StreamData),
	}

	return stream
}

func getDeadlineContext(ctx context.Context) context.Context {
	deadline, _ := context.WithDeadline(ctx, time.Now().Add(time.Second))

	return deadline
}

func TestNewZipFileArchiveRepository(t *testing.T) {
	// given
	zipMock := NewMockZipper(t)
	creatorFunc := func(w io.Writer) Zipper {
		return zipMock
	}

	c := &config.OperatorConfig{
		ArchiveVolumeDownloadServiceName:     testServiceName,
		ArchiveVolumeDownloadServiceProtocol: testProtocol,
		ArchiveVolumeDownloadServicePort:     testPort,
	}

	// when
	repository := NewZipFileArchiveRepository(testArchivesPath, creatorFunc, c)

	// then
	require.NotNil(t, repository)
	assert.Equal(t, testArchivesPath, repository.archivesPath)
	assert.Equal(t, testPort, repository.archiveVolumeDownloadServicePort)
	assert.Equal(t, testProtocol, repository.archiveVolumeDownloadServiceProtocol)
	assert.Equal(t, testServiceName, repository.archiveVolumeDownloadServiceName)
	assert.Equal(t, zipMock, repository.zipCreator(nil))
}
