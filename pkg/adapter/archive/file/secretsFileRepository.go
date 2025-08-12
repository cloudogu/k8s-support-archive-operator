package file

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	archiveSecretsInfoDirName = "core/secrets"
)

type SecretsFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
}

func NewSecretsFileRepository(workPath string, fs volumeFs) *SecretsFileRepository {
	return &SecretsFileRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, archiveSecretsInfoDirName, fs),
	}
}

func (v *SecretsFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *v1.SecretList) error {
	return create(ctx, id, dataStream, v.createCoreSecret, v.Delete, v.FinishCollection)
}

// createCoreSecret writes the content from data to the volumeInfo file.
// If the CoreSecret file exists, it overrides the existing file.
func (v *SecretsFileRepository) createCoreSecret(ctx context.Context, id domain.SupportArchiveID, data *v1.SecretList) error {
	logger := log.FromContext(ctx).WithName("SecretsFileRepository.createCoreSecret")

	for _, secret := range data.Items {
		filePath := filepath.Join(v.workPath, id.Namespace, id.Name, archiveSecretsInfoDirName, fmt.Sprintf("%s%s", secret.Name, ".yaml"))
		err := v.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
		}
		out, err := yaml.Marshal(secret)
		if err != nil {
			return fmt.Errorf("error marshalling secrets file: %w", err)
		}

		err = v.filesystem.WriteFile(filePath, out, 0644)
		if err != nil {
			return fmt.Errorf("error creating secrets file: %w", err)
		}
	}

	logger.Info("created secrets file")

	return nil
}
