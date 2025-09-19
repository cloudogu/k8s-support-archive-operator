package file

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	archiveSystemStateDirName = "Resources/SystemState"
)

type SystemStateFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
}

func NewSystemStateFileRepository(workPath string, fs volumeFs) *SystemStateFileRepository {
	return &SystemStateFileRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, archiveSystemStateDirName, fs),
	}
}

func (v *SystemStateFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.UnstructuredResource) error {
	return create(ctx, id, dataStream, v.createSystemState, v.Delete, v.finishCollection, nil)
}

// createSystemState writes the content from data to a system state file.
// If the system state file exists, it overrides the existing file.
func (v *SystemStateFileRepository) createSystemState(ctx context.Context, id domain.SupportArchiveID, data *domain.UnstructuredResource) error {
	logger := log.FromContext(ctx).WithName("SystemStateFileRepository.createVolumeInfo")
	filePath := fmt.Sprintf("%s.yaml", filepath.Join(v.workPath, id.Namespace, id.Name, archiveSystemStateDirName, data.Path, data.Name))

	err := createYAMLFile(v.filesystem, filePath, data)
	if err != nil {
		return err
	}

	logger.Info("created volume metrics file")

	return nil
}

func createYAMLFile(filesystem volumeFs, filePath string, data interface{}) error {
	err := filesystem.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("error creating directory for file: %w", err)
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling file: %w", err)
	}

	err = filesystem.WriteFile(filePath, out, 0644)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}

	return nil
}
