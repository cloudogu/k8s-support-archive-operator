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
	"time"
)

const (
	testWorkPath           = "/work"
	testWorkDirArchivePath = testWorkPath + "/" + testNamespace + "/" + testName + "/logs"
	testWorkCasLog         = testWorkDirArchivePath + "/cas.log"
)

func TestLogFileRepository_Delete(t *testing.T) {
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
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().RemoveAll(testWorkDirArchivePath).Return(nil)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return error on error remove directory",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().RemoveAll(testWorkDirArchivePath).Return(assert.AnError)

					return fsMock
				},
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
			l := &LogFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, l.Delete(tt.args.ctx, tt.args.id), fmt.Sprintf("Delete(%v, %v)", tt.args.ctx, tt.args.id))
		})
	}
}

func TestLogFileRepository_Stream(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		ctx    context.Context
		id     domain.SupportArchiveID
		stream domain.Stream
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should return error on error reading directory",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().ReadDir(testWorkDirArchivePath).Return(nil, assert.AnError)
					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to read directory")
			},
		},
		{
			name: "should write file to stream and close on end",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().ReadDir(testWorkDirArchivePath).Return([]os.DirEntry{testEntry{"cas.log"}}, nil)
					fsMock.EXPECT().Open(testWorkCasLog).Return(fileMock, nil)
					return fsMock
				},
			},
			args: args{
				ctx:    testCtx,
				id:     testID,
				stream: getReadStream(t),
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error open file",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().ReadDir(testWorkDirArchivePath).Return([]os.DirEntry{testEntry{"cas.log"}}, nil)
					fsMock.EXPECT().Open(testWorkCasLog).Return(nil, assert.AnError)
					return fsMock
				},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LogFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, l.Stream(tt.args.ctx, tt.args.id, tt.args.stream))
		})
	}
}

func getReadStream(t *testing.T) domain.Stream {
	data := make(chan domain.StreamData)
	stream := domain.Stream{Data: data}

	go func() {
		for {
			select {
			case d, ok := <-stream.Data:
				if ok {
					assert.Equal(t, "cas.log", d.ID)
				} else {
					return
				}
			}
		}
	}()

	return stream
}

func TestNewLogFileRepository(t *testing.T) {
	// given
	fsMock := newMockVolumeFs(t)
	baseRepo := NewBaseFileRepository(testWorkPath, fsMock)

	// when
	repository := NewLogFileRepository(testWorkPath, fsMock, baseRepo)

	// then
	require.NotNil(t, repository)
	assert.Equal(t, testWorkPath, repository.workPath)
	assert.Equal(t, fsMock, repository.filesystem)
	assert.Equal(t, baseRepo, repository.baseFileRepository)
}

func TestLogFileRepository_Create(t *testing.T) {
	type fields struct {
		workPath           string
		filesystem         func(t *testing.T) volumeFs
		baseFileRepository *baseFileRepository
	}
	type args struct {
		ctx        context.Context
		id         domain.SupportArchiveID
		dataStream <-chan *domain.PodLog
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "success and finish collection",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileMock.EXPECT().Write([]byte("logline1")).Return(0, nil)
					fileMock.EXPECT().Write([]byte("logline2")).Return(0, nil)

					stateFileMock := newMockClosableRWFile(t)
					stateFileMock.EXPECT().Close().Return(nil)
					stateFileMock.EXPECT().Write([]byte("done")).Return(0, nil)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().Create(testWorkCasLog).Return(fileMock, nil)
					fsMock.EXPECT().MkdirAll(testWorkDirArchivePath, fs.ModePerm).Return(nil)
					fsMock.EXPECT().Create(testStateFilePath).Return(stateFileMock, nil)

					return fsMock
				},
			},
			args: args{
				ctx:        testCtx,
				id:         testID,
				dataStream: getSuccessStream(),
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsMock := tt.fields.filesystem(t)
			l := &LogFileRepository{
				workPath:           tt.fields.workPath,
				filesystem:         fsMock,
				baseFileRepository: NewBaseFileRepository(tt.fields.workPath, fsMock),
			}
			tt.wantErr(t, l.Create(tt.args.ctx, tt.args.id, tt.args.dataStream))
		})
	}
}

func getSuccessStream() chan *domain.PodLog {
	channel := make(chan *domain.PodLog)

	go func() {
		channel <- &domain.PodLog{
			PodName:   "cas",
			StartTime: time.Now(),
			EndTime:   time.Now(),
			Entries:   []string{"logline1", "logline2"},
		}

		close(channel)
	}()

	return channel
}

func TestLogFileRepository_createPodLog(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		ctx  context.Context
		id   domain.SupportArchiveID
		data *domain.PodLog
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should return error on error creating dir",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testWorkDirArchivePath, os.FileMode(0755)).Return(assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.PodLog{
					PodName: "cas",
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to create directory")
			},
		},
		{
			name: "should return error on error creating file",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().Create(testWorkCasLog).Return(nil, assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.PodLog{
					PodName: "cas",
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to create file")
			},
		},
		{
			name: "should return error on error writing file",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileMock.EXPECT().Write([]byte("logline1")).Return(0, assert.AnError)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().Create(testWorkCasLog).Return(fileMock, nil)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.PodLog{
					PodName: "cas",
					Entries: []string{"logline1", "logline2"},
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to write to file")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &LogFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, l.createPodLog(tt.args.ctx, tt.args.id, tt.args.data))
		})
	}
}
