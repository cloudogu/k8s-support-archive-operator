package file

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"gopkg.in/yaml.v3"
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

func (v *SecretsFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.SecretYaml) error {
	return create(ctx, id, dataStream, v.createCoreSecret, v.Delete, v.FinishCollection)
}

// createCoreSecret writes the content from data to the volumeInfo file.
// If the CoreSecret file exists, it overrides the existing file.
func (v *SecretsFileRepository) createCoreSecret(ctx context.Context, id domain.SupportArchiveID, data *domain.SecretYaml) error {
	logger := log.FromContext(ctx).WithName("SecretsFileRepository.createCoreSecret")

	/*
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
	*/

	filePath := filepath.Join(v.workPath, id.Namespace, id.Name, archiveSecretsInfoDirName, fmt.Sprintf("%s%s", data.Metadata.Name, ".yaml"))
	err := v.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling secrets file: %w", err)
	}

	err = v.filesystem.WriteFile(filePath, out, 0644)
	if err != nil {
		return fmt.Errorf("error creating secrets file: %w", err)
	}

	/*
		secret := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret",
				Namespace: "ecosystem",
			},
			Type: v1.SecretTypeOpaque,
			Data: map[string][]byte{
				"username": []byte("admin"),
				"password": []byte("secret"),
			},
		}

		yamlBytes, err := yaml.Marshal(secret)
		if err != nil {
			return err
		}
		filePath := filepath.Join(v.workPath, id.Namespace, id.Name, archiveSecretsInfoDirName, fmt.Sprintf("%s%s", "secret", ".yaml"))
		err = v.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(filePath), err)
		}
		if err := os.WriteFile(filePath, yamlBytes, 0644); err != nil {
			return err
		}
	*/

	logger.Info("created secrets file")

	return nil
}
