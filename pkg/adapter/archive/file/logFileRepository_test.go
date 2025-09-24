package file

const (
	testWorkPath              = "/work"
	testCollectorDirName      = "collectorDir"
	testLogCollectorDirName   = "Logs"
	testWorkDirArchivePath    = testWorkPath + "/" + testNamespace + "/" + testName + "/" + testCollectorDirName
	testLogWorkDirArchivePath = testWorkPath + "/" + testNamespace + "/" + testName + "/" + testLogCollectorDirName
	testWorkCasLog            = testLogWorkDirArchivePath + "/cas.log"
)

//TODO Fixed in log story
/*func TestLogFileRepository_createPodLog(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T) volumeFs
	}
	type args struct {
		ctx  context.Context
		id   domain.SupportArchiveID
		data *domain.LogLine
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
					fsMock.EXPECT().MkdirAll(testLogWorkDirArchivePath, os.FileMode(0755)).Return(assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "log line",
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
					fsMock.EXPECT().MkdirAll(testLogWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().Create(testWorkCasLog).Return(nil, assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "log line",
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
					fsMock.EXPECT().MkdirAll(testLogWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().Create(testWorkCasLog).Return(fileMock, nil)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "log line",
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.ErrorIs(t, err, assert.AnError)
				assert.ErrorContains(t, err, "failed to write to file")
			},
		},
		{
			name: "should return nil on success",
			fields: fields{
				workPath: testWorkPath,
				filesystem: func(t *testing.T) volumeFs {
					fileMock := newMockClosableRWFile(t)
					fileMock.EXPECT().Close().Return(nil)
					fileMock.EXPECT().Write([]byte("logline1")).Return(0, nil)
					fileMock.EXPECT().Write([]byte("logline2")).Return(0, nil)

					fsMock := newMockVolumeFs(t)
					fsMock.EXPECT().MkdirAll(testLogWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().Create(testWorkCasLog).Return(fileMock, nil)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.LogLine{
					Value: "log line",
				},
			},
			wantErr: func(t *testing.T, err error) {
				require.NoError(t, err)
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
}
*/
