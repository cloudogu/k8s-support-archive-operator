package file

import (
	"context"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"io/fs"
	"os"
	"testing"
	"time"
)

const (
	testStateFilePath = testWorkDirArchivePath + "/.done"
)

func Test_baseFileRepository_FinishCollection(t *testing.T) {
	type fields struct {
		workPath     string
		collectorDir string
		filesystem   func(t *testing.T) volumeFs
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
				workPath:     testWorkPath,
				collectorDir: testCollectorDirName,
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
				workPath:     testWorkPath,
				collectorDir: testCollectorDirName,
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
				workPath:     testWorkPath,
				collectorDir: testCollectorDirName,
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
				workPath:     testWorkPath,
				collectorDir: testCollectorDirName,
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
				workPath:     tt.fields.workPath,
				collectorDir: tt.fields.collectorDir,
				filesystem:   tt.fields.filesystem(t),
			}
			tt.wantErr(t, l.FinishCollection(tt.args.ctx, tt.args.id))
		})
	}
}

func Test_baseFileRepository_IsCollected(t *testing.T) {
	type fields struct {
		workPath     string
		collectorDir string
		filesystem   func(t *testing.T) volumeFs
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
				workPath:     testWorkPath,
				collectorDir: testCollectorDirName,
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
				workPath:     testWorkPath,
				collectorDir: testCollectorDirName,
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
				workPath:     testWorkPath,
				collectorDir: testCollectorDirName,
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
				workPath:     tt.fields.workPath,
				collectorDir: tt.fields.collectorDir,
				filesystem:   tt.fields.filesystem(t),
			}
			got, err := l.IsCollected(tt.args.ctx, tt.args.id)
			if !tt.wantErr(t, err, fmt.Sprintf("IsCollected(%v, %v)", tt.args.ctx, tt.args.id)) {
				return
			}
			assert.Equalf(t, tt.want, got, "IsCollected(%v, %v)", tt.args.ctx, tt.args.id)
		})
	}
}

func Test_baseFileRepository_Delete(t *testing.T) {
	type fields struct {
		workPath     string
		collectorDir string
		filesystem   func(t *testing.T) volumeFs
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
				workPath:     testWorkPath,
				collectorDir: testCollectorDirName,
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
				workPath:     testWorkPath,
				collectorDir: testCollectorDirName,
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
			l := &baseFileRepository{
				workPath:     tt.fields.workPath,
				collectorDir: tt.fields.collectorDir,
				filesystem:   tt.fields.filesystem(t),
			}
			tt.wantErr(t, l.Delete(tt.args.ctx, tt.args.id), fmt.Sprintf("Delete(%v, %v)", tt.args.ctx, tt.args.id))
		})
	}
}

func TestNewBaseFileRepository(t *testing.T) {
	// given
	fsMock := newMockVolumeFs(t)

	// when
	repository := NewBaseFileRepository(testWorkPath, testCollectorDirName, fsMock)

	// then
	assert.NotNil(t, repository)
	assert.Equal(t, testWorkPath, repository.workPath)
	assert.Equal(t, testCollectorDirName, repository.collectorDir)
	assert.Equal(t, fsMock, repository.filesystem)
}

func Test_create(t *testing.T) {
	type args[DATATYPE domain.CollectorUnionDataType] struct {
		ctx        context.Context
		id         domain.SupportArchiveID
		dataStream <-chan *DATATYPE
		createFn   createFn[domain.PodLog]
		deleteFn   deleteFn
		finishFn   finishFn
	}
	type testCase[DATATYPE domain.CollectorUnionDataType] struct {
		name    string
		args    args[DATATYPE]
		wantErr func(t *testing.T, err error)
	}
	tests := []testCase[domain.PodLog]{
		{
			name: "should call create and finish if channel is closed",
			args: args[domain.PodLog]{
				ctx:        testCtx,
				id:         testID,
				dataStream: getSuccessStream(),
				createFn: func(ctx context.Context, id domain.SupportArchiveID, d *domain.PodLog) error {
					return nil
				},
				finishFn: func(ctx context.Context, id domain.SupportArchiveID) error {
					return nil
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return error on error finish collection",
			args: args[domain.PodLog]{
				ctx:        testCtx,
				id:         testID,
				dataStream: getSuccessStream(),
				createFn: func(ctx context.Context, id domain.SupportArchiveID, d *domain.PodLog) error {
					return nil
				},
				finishFn: func(ctx context.Context, id domain.SupportArchiveID) error {
					return assert.AnError
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error finishing collection")
			},
		},
		{
			name: "should return error on error create data",
			args: args[domain.PodLog]{
				ctx:        testCtx,
				id:         testID,
				dataStream: getSuccessStream(),
				createFn: func(ctx context.Context, id domain.SupportArchiveID, d *domain.PodLog) error {
					return assert.AnError
				},
				deleteFn: func(ctx context.Context, id domain.SupportArchiveID) error {
					return nil
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "error creating element from data stream")
			},
		},
		{
			name: "should return join error on cleanup error",
			args: args[domain.PodLog]{
				ctx:        testCtx,
				id:         testID,
				dataStream: getSuccessStream(),
				createFn: func(ctx context.Context, id domain.SupportArchiveID, d *domain.PodLog) error {
					return assert.AnError
				},
				deleteFn: func(ctx context.Context, id domain.SupportArchiveID) error {
					return assert.AnError
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to clean up data after error")
				assert.ErrorContains(t, err, "error creating element from data stream")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(t, create(tt.args.ctx, tt.args.id, tt.args.dataStream, tt.args.createFn, tt.args.deleteFn, tt.args.finishFn))
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

func Test_baseFileRepository_Stream(t *testing.T) {
	type fields struct {
		workPath   string
		directory  string
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		ctx    context.Context
		id     domain.SupportArchiveID
		stream *domain.Stream
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		want         func() error
		wantErr      func(t *testing.T, err error)
		waitForClose bool
	}{
		{
			name: "should return error on error walking work directory",
			fields: fields{
				workPath:  testWorkPath,
				directory: testCollectorDirName,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().WalkDir(testWorkDirArchivePath, mock.Anything).Return(assert.AnError)
					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
			},
		},
		{
			name: "should close the stream on success",
			fields: fields{
				workPath:  testWorkPath,
				directory: testCollectorDirName,
				filesystem: func(t *testing.T) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().WalkDir(testWorkDirArchivePath, mock.Anything).Return(nil)
					return fsMock
				},
			},
			args: args{
				ctx:    testCtx,
				id:     testID,
				stream: getEmptyStream(),
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
			waitForClose: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &baseFileRepository{
				workPath:     tt.fields.workPath,
				collectorDir: tt.fields.directory,
				filesystem:   tt.fields.filesystem(t),
			}

			group, _ := errgroup.WithContext(tt.args.ctx)
			var got func() error
			group.Go(func() error {
				var err error
				got, err = l.Stream(tt.args.ctx, tt.args.id, tt.args.stream)
				return err
			})

			if tt.waitForClose {
				timer := time.NewTimer(time.Second * 2)
				go func() {
					<-timer.C
					defer func() {
						println("defer")
						// recover panic if the channel is closed correctly from the test
						if r := recover(); r != nil {
							println("recovered from panic")
							tt.args.stream.Data <- domain.StreamData{ID: "timeout"}
							return
						}
					}()

					return
				}()

				v, open := <-tt.args.stream.Data
				if v.ID == "timeout" && open {
					t.Fatal("test timed out")
				}
			}

			err := group.Wait()
			tt.wantErr(t, err)

			if err != nil && got != nil {
				require.NoError(t, got())
			}

			// Should always return finalizer func if no error happens.
			if err == nil && got == nil {
				t.FailNow()
			}
		})
	}
}

func Test_getFileFinalizerFunc(t *testing.T) {
	type args struct {
		filesToClose func(t *testing.T) []closableRWFile
	}
	tests := []struct {
		name    string
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should return function which closes all files",
			args: args{
				filesToClose: func(t *testing.T) []closableRWFile {
					file1 := newMockClosableRWFile(t)
					file1.EXPECT().Close().Return(nil)
					file2 := newMockClosableRWFile(t)
					file2.EXPECT().Close().Return(nil)

					return []closableRWFile{file1, file2}
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return multiple errors",
			args: args{
				filesToClose: func(t *testing.T) []closableRWFile {
					file1 := newMockClosableRWFile(t)
					file1.EXPECT().Close().Return(errors.New("error1"))
					file2 := newMockClosableRWFile(t)
					file2.EXPECT().Close().Return(errors.New("error2"))

					return []closableRWFile{file1, file2}
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorContains(t, err, "error1")
				assert.ErrorContains(t, err, "error2")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finalizerFunc := getFileFinalizerFunc(tt.args.filesToClose(t))
			err := finalizerFunc()
			tt.wantErr(t, err)
		})
	}
}

func Test_baseFileRepository_streamFile(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T) (volumeFs, closableRWFile)
	}
	type args struct {
		ctx      context.Context
		path     string
		filename string
		stream   *domain.Stream
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr func(t *testing.T, err error)
	}{
		{
			name: "should return error on error open file",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) (volumeFs, closableRWFile) {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().Open(testWorkCasLog).Return(nil, assert.AnError)
					return fsMock, nil
				},
			},
			args: args{
				ctx:      testCtx,
				path:     testWorkCasLog,
				filename: "cas.log",
				stream:   &domain.Stream{},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to open file")
			},
		},
		{
			name: "should write file to stream",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) (volumeFs, closableRWFile) {
					fsMock := newMockVolumeFs(t)
					fileMock := newMockClosableRWFile(t)
					fsMock.EXPECT().Open(testWorkCasLog).Return(fileMock, nil)
					return fsMock, fileMock
				},
			},
			args: args{
				ctx:      testCtx,
				path:     testWorkCasLog,
				filename: "cas.log",
				stream:   getReadStream(),
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filesystem, expectedWant := tt.fields.filesystem(t)
			l := &baseFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: filesystem,
			}
			want, err := l.streamFile(tt.args.ctx, tt.args.path, tt.args.filename, tt.args.stream)
			tt.wantErr(t, err)
			assert.Equal(t, expectedWant, want)
		})
	}
}

func getReadStream() *domain.Stream {
	stream := domain.Stream{}
	data := make(chan domain.StreamData)
	stream.Data = data

	go func() {
		<-data
	}()

	return &stream
}
