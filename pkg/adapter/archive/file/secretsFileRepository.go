package file

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	archiveSecretsInfoDirName = "core/secrets"
)

type SecretsFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem secretFs
}

func NewSecretsFileRepository(workPath string, fs secretFs) *SecretsFileRepository {
	return &SecretsFileRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, archiveSecretsInfoDirName, fs),
	}
}

func (v *SecretsFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.SecretYaml) error {
	return create(ctx, id, dataStream, v.createCoreSecret, v.Delete, v.finishCollection, nil)
}

// createCoreSecret writes the content from data to the volumeInfo file.
// If the CoreSecret file exists, it overrides the existing file.
func (v *SecretsFileRepository) createCoreSecret(ctx context.Context, id domain.SupportArchiveID, data *domain.SecretYaml) error {
	logger := log.FromContext(ctx).WithName("SecretsFileRepository.createCoreSecret")

	filePath := filepath.Join(v.workPath, id.Namespace, id.Name, archiveSecretsInfoDirName, fmt.Sprintf("%s%s", data.Metadata.Name, ".yaml"))

	err := createYAMLFile(v.filesystem, filePath, data)
	if err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("created file for secret: %s", data.Metadata.Name))

	return nil
}
