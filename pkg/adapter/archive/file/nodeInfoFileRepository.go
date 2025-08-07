package file

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	archiveNodeInfoDirName = "NodeInfo"
)

type NodeInfoRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
}

func NewNodeInfoFileRepository(workPath string, fs volumeFs) *NodeInfoRepository {
	return &NodeInfoRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, archiveVolumeInfoDirName, fs),
	}
}

func (v *NodeInfoRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.NodeInfo) error {
	return create(ctx, id, dataStream, v.createNodeInfo, v.Delete, v.FinishCollection)
}

// createNodeInfo writes the content from data to the volumeInfo file.
// If the volumeInfo file exists, it overrides the existing file.
func (v *NodeInfoRepository) createNodeInfo(ctx context.Context, id domain.SupportArchiveID, data *domain.NodeInfo) error {
	logger := log.FromContext(ctx).WithName("NodeInfoRepository.createNodeInfo")
	filePath := fmt.Sprintf("%s.json", filepath.Join(v.workPath, id.Namespace, id.Name, archiveNodeInfoDirName, "data"))
	err := v.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
	if err != nil {
		return fmt.Errorf("error creating directory for volume node info file: %w", err)
	}

	out, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshalling volume node info file: %w", err)
	}

	err = v.filesystem.WriteFile(filePath, out, 0644)
	if err != nil {
		return fmt.Errorf("error creating node info file: %w", err)
	}

	logger.Info("created volume node info file")

	return nil
}
