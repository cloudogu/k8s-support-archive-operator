package file

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testWorkPath              = "/work"
	testCollectorDirName      = "collectorDir"
	testLogCollectorDirName   = "Logs"
	testWorkDirArchivePath    = testWorkPath + "/" + testNamespace + "/" + testName + "/" + testCollectorDirName
	testLogWorkDirArchivePath = testWorkPath + "/" + testNamespace + "/" + testName + "/" + testLogCollectorDirName
	testWorkLog               = testLogWorkDirArchivePath + "/logs.log"
)

func TestLogFileRepository_createPodLog(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T, fileMock closableRWFile) volumeFs
		logFiles   func(id domain.SupportArchiveID, fileMock closableRWFile) map[domain.SupportArchiveID]closableRWFile
	}
	type args struct {
		ctx  context.Context
		id   domain.SupportArchiveID
		data *domain.LogLine
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		fileMock func(t *testing.T) closableRWFile
		wantErr  func(t *testing.T, err error)
	}{
		{
			name: "should return error on error creating dir",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T, _ closableRWFile) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testLogWorkDirArchivePath, os.FileMode(0755)).Return(assert.AnError)

					return fsMock
				},
				logFiles: func(_ domain.SupportArchiveID, _ closableRWFile) map[domain.SupportArchiveID]closableRWFile {
					return make(map[domain.SupportArchiveID]closableRWFile)
				},
			},
			fileMock: func(t *testing.T) closableRWFile {
				return nil
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "logline",
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
				filesystem: func(t *testing.T, _ closableRWFile) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testLogWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().OpenFile(testWorkLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0666)).Return(nil, assert.AnError)

					return fsMock
				},
				logFiles: func(_ domain.SupportArchiveID, _ closableRWFile) map[domain.SupportArchiveID]closableRWFile {
					return make(map[domain.SupportArchiveID]closableRWFile)
				},
			},
			fileMock: func(t *testing.T) closableRWFile {
				return nil
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "logline",
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to create log file")
			},
		},
		{
			name: "should return error on error writing header",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T, fileMock closableRWFile) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testLogWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().OpenFile(testWorkLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0666)).Return(fileMock, nil)

					return fsMock
				},
				logFiles: func(_ domain.SupportArchiveID, _ closableRWFile) map[domain.SupportArchiveID]closableRWFile {
					return make(map[domain.SupportArchiveID]closableRWFile)
				},
			},
			fileMock: func(t *testing.T) closableRWFile {
				fileMock := newMockClosableRWFile(t)
				fileMock.EXPECT().Write([]byte("LOGS\n")).Return(0, assert.AnError)
				return fileMock
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "logline",
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to write header to log file")
			},
		},
		{
			name: "should return error on error writing value",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T, fileMock closableRWFile) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testLogWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().OpenFile(testWorkLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0666)).Return(fileMock, nil)

					return fsMock
				},
				logFiles: func(_ domain.SupportArchiveID, _ closableRWFile) map[domain.SupportArchiveID]closableRWFile {
					return make(map[domain.SupportArchiveID]closableRWFile)
				},
			},
			fileMock: func(t *testing.T) closableRWFile {
				fileMock := newMockClosableRWFile(t)
				fileMock.EXPECT().Write([]byte("LOGS\n")).Return(0, nil)
				fileMock.EXPECT().Write([]byte("logline\n")).Return(0, assert.AnError)

				return fileMock
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "logline",
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to write data to log file")
			},
		},
		{
			name: "should return nil on initial success",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T, fileMock closableRWFile) volumeFs {
					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testLogWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().OpenFile(testWorkLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.FileMode(0666)).Return(fileMock, nil)

					return fsMock
				},
				logFiles: func(_ domain.SupportArchiveID, _ closableRWFile) map[domain.SupportArchiveID]closableRWFile {
					return make(map[domain.SupportArchiveID]closableRWFile)
				},
			},
			fileMock: func(t *testing.T) closableRWFile {
				fileMock := newMockClosableRWFile(t)
				fileMock.EXPECT().Write([]byte("LOGS\n")).Return(0, nil)
				fileMock.EXPECT().Write([]byte("logline\n")).Return(0, nil)
				return fileMock
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "logline",
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "should return nil on append success",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T, fileMock closableRWFile) volumeFs {
					return nil
				},
				logFiles: func(id domain.SupportArchiveID, fileMock closableRWFile) map[domain.SupportArchiveID]closableRWFile {
					return map[domain.SupportArchiveID]closableRWFile{id: fileMock}
				},
			},
			fileMock: func(t *testing.T) closableRWFile {
				fileMock := newMockClosableRWFile(t)
				fileMock.EXPECT().Write([]byte("logline\n")).Return(0, nil)
				return fileMock
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "logline",
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileMock := tt.fileMock(t)
			l := &LogFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: tt.fields.filesystem(t, fileMock),
				logFiles:   tt.fields.logFiles(tt.args.id, fileMock),
			}
			tt.wantErr(t, l.createPodLog(tt.args.ctx, tt.args.id, tt.args.data))
		})
	}
}

func TestNewLogFileRepository(t *testing.T) {
	// given
	fsMock := newMockVolumeFs(t)

	// when
	repository := NewLogFileRepository(testWorkPath, fsMock)

	// then
	assert.NotNil(t, repository)
	assert.Equal(t, testWorkPath, repository.workPath)
	assert.Equal(t, fsMock, repository.filesystem)
	assert.NotEmpty(t, repository.baseFileRepo)
	assert.NotNil(t, repository.logFiles)
}

func TestLogFileRepository_close(t *testing.T) {
	type fields struct {
		baseFileRepo baseFileRepo
		workPath     string
		filesystem   volumeFs
		logFiles     func(t *testing.T) map[domain.SupportArchiveID]closableRWFile
	}
	type args struct {
		in0 context.Context
		id  domain.SupportArchiveID
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "should return nil if map is nil",
			wantErr: assert.NoError,
		},
		{
			name: "should return nil if map is not nil but does not contains closable file",
			fields: fields{
				logFiles: func(t *testing.T) map[domain.SupportArchiveID]closableRWFile {
					return map[domain.SupportArchiveID]closableRWFile{testID: nil}
				},
			},
			args: args{
				id: testID,
			},
			wantErr: assert.NoError,
		},
		{
			name: "should return error on close error",
			fields: fields{
				logFiles: func(t *testing.T) map[domain.SupportArchiveID]closableRWFile {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(assert.AnError)
					return map[domain.SupportArchiveID]closableRWFile{testID: fileMock}
				},
			},
			args: args{
				id: testID,
			},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "failed to close log file")
			},
		},
		{
			name: "should return nil on successful close",
			fields: fields{
				logFiles: func(t *testing.T) map[domain.SupportArchiveID]closableRWFile {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					return map[domain.SupportArchiveID]closableRWFile{testID: fileMock}
				},
			},
			args: args{
				id: testID,
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		var files map[domain.SupportArchiveID]closableRWFile
		if tt.fields.logFiles != nil {
			files = tt.fields.logFiles(t)
		}
		t.Run(tt.name, func(t *testing.T) {
			l := &LogFileRepository{
				baseFileRepo: tt.fields.baseFileRepo,
				workPath:     tt.fields.workPath,
				filesystem:   tt.fields.filesystem,
				logFiles:     files,
			}
			tt.wantErr(t, l.close(tt.args.in0, tt.args.id), fmt.Sprintf("close(%v, %v)", tt.args.in0, tt.args.id))
		})
	}
}
