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
	archiveVolumesDirName = "volumes"
)

type VolumesFileRepository struct {
	*baseFileRepository
	workPath   string
	filesystem volumeFs
}

func NewVolumesFileRepository(workPath string, fs volumeFs, repository *baseFileRepository) *VolumesFileRepository {
	return &VolumesFileRepository{
		workPath:           workPath,
		filesystem:         fs,
		baseFileRepository: repository,
	}
}

func (v *VolumesFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.VolumeMetrics) error {
	return create(ctx, id, dataStream, v.createVolumeMetricsFile, v.Delete, v.FinishCollection)
}

// createVolumeMetricsFile writes the content from data to the metrics file.
// If the metrics file exists, it overrides the existing file.
func (v *VolumesFileRepository) createVolumeMetricsFile(ctx context.Context, id domain.SupportArchiveID, data *domain.VolumeMetrics) error {
	logger := log.FromContext(ctx).WithName("VolumesFileRepository.createVolumeMetricsFile")
	filePath := fmt.Sprintf("%s.yaml", filepath.Join(v.workPath, id.Namespace, id.Name, archiveVolumesDirName, data.Name))
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

func (v *VolumesFileRepository) Stream(ctx context.Context, id domain.SupportArchiveID, stream domain.Stream) (func() error, error) {
	return v.baseFileRepository.stream(ctx, id, archiveVolumesDirName, stream)
}

func (v *VolumesFileRepository) Delete(ctx context.Context, id domain.SupportArchiveID) error {
	return v.baseFileRepository.Delete(ctx, id, archiveVolumesDirName)
}

func (v *VolumesFileRepository) FinishCollection(ctx context.Context, id domain.SupportArchiveID) error {
	return v.baseFileRepository.FinishCollection(ctx, id, archiveVolumesDirName)
}

func (v *VolumesFileRepository) IsCollected(ctx context.Context, id domain.SupportArchiveID) (bool, error) {
	return v.baseFileRepository.IsCollected(ctx, id, archiveVolumesDirName)
}
