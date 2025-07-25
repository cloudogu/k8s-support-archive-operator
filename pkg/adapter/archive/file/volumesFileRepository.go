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
	archiveVolumeInfoDirName = "VolumeInfo"
)

type VolumesFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
}

func NewVolumesFileRepository(workPath string, fs volumeFs, repository baseFileRepo) *VolumesFileRepository {
	return &VolumesFileRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: repository,
	}
}

func (v *VolumesFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.VolumeInfo) error {
	return create(ctx, id, dataStream, v.createVolumeInfo, v.Delete, v.FinishCollection)
}

// createVolumeInfo writes the content from data to the volumeInfo file.
// If the volumeInfo file exists, it overrides the existing file.
func (v *VolumesFileRepository) createVolumeInfo(ctx context.Context, id domain.SupportArchiveID, data *domain.VolumeInfo) error {
	logger := log.FromContext(ctx).WithName("VolumesFileRepository.createVolumeInfo")
	filePath := fmt.Sprintf("%s.yaml", filepath.Join(v.workPath, id.Namespace, id.Name, archiveVolumeInfoDirName, data.Name))
	err := v.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("error creating directory for volume metrics file: %w", err)
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling volume metrics file: %w", err)
	}

	err = v.filesystem.WriteFile(filePath, out, 0644)
	if err != nil {
		return fmt.Errorf("error creating volume metrics file: %w", err)
	}

	logger.Info("created volume metrics file")

	return nil
}

func (v *VolumesFileRepository) Stream(ctx context.Context, id domain.SupportArchiveID, stream *domain.Stream) (func() error, error) {
	return v.baseFileRepo.stream(ctx, id, archiveVolumeInfoDirName, stream)
}

func (v *VolumesFileRepository) Delete(ctx context.Context, id domain.SupportArchiveID) error {
	return v.baseFileRepo.Delete(ctx, id, archiveVolumeInfoDirName)
}

func (v *VolumesFileRepository) FinishCollection(ctx context.Context, id domain.SupportArchiveID) error {
	return v.baseFileRepo.FinishCollection(ctx, id, archiveVolumeInfoDirName)
}

func (v *VolumesFileRepository) IsCollected(ctx context.Context, id domain.SupportArchiveID) (bool, error) {
	return v.baseFileRepo.IsCollected(ctx, id, archiveVolumeInfoDirName)
}
