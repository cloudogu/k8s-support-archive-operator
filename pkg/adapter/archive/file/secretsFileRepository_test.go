package file

import (
	"context"
	"os"
	"testing"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	testSecretCollectorDirName   = "Resources/core/secrets"
	testSecretWorkDirArchivePath = testWorkPath + "/" + testNamespace + "/" + testName + "/" + testSecretCollectorDirName
	testSecretWorkFile           = testSecretWorkDirArchivePath + "/secret.yaml"
)

func TestNewSecretsFileRepository(t *testing.T) {
	// given
	fsMock := newMockSecretFs(t)

	// when
	repository := NewSecretsFileRepository(testWorkPath, fsMock)

	// then
	assert.NotNil(t, repository)
	assert.Equal(t, testWorkPath, repository.workPath)
	assert.Equal(t, fsMock, repository.filesystem)
	assert.NotEmpty(t, repository.baseFileRepo)
}

func TestSecretFileRepository_createCoreSecret(t *testing.T) {
	type fields struct {
		workPath   string
		filesystem func(t *testing.T) secretFs
	}
	type args struct {
		ctx  context.Context
		id   domain.SupportArchiveID
		data *domain.SecretYaml
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
				filesystem: func(t *testing.T) secretFs {
					fsMock := newMockSecretFs(t)
					fsMock.EXPECT().MkdirAll(testSecretWorkDirArchivePath, os.FileMode(0755)).Return(assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx:  testCtx,
				id:   testID,
				data: &domain.SecretYaml{},
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
				filesystem: func(t *testing.T) secretFs {
					fsMock := newMockSecretFs(t)
					fsMock.EXPECT().MkdirAll(testSecretWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().WriteFile(testSecretWorkFile, mock.Anything, os.FileMode(0644)).Return(assert.AnError)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.SecretYaml{
					Metadata: domain.SecretYamlMetaData{Name: "secret"},
				},
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
				filesystem: func(t *testing.T) secretFs {
					fsMock := newMockSecretFs(t)
					out, err := yaml.Marshal(&domain.SecretYaml{
						Metadata: domain.SecretYamlMetaData{Name: "secret"},
					})
					require.NoError(t, err)
					fsMock.EXPECT().MkdirAll(testSecretWorkDirArchivePath, os.FileMode(0755)).Return(nil)
					fsMock.EXPECT().WriteFile(testSecretWorkFile, out, os.FileMode(0644)).Return(nil)

					return fsMock
				},
			},
			args: args{
				ctx: testCtx,
				id:  testID,
				data: &domain.SecretYaml{
					Metadata: domain.SecretYamlMetaData{Name: "secret"},
				},
			},
			wantErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SecretsFileRepository{
				workPath:   tt.fields.workPath,
				filesystem: tt.fields.filesystem(t),
			}
			tt.wantErr(t, s.createCoreSecret(tt.args.ctx, tt.args.id, tt.args.data))
		})
	}
}
