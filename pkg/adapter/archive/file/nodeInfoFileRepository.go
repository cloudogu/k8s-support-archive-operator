package file

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"os"
	"path/filepath"
)

const (
	archiveNodeInfoDirName = "NodeInfo"
)

type metricForID struct {
	string
	domain.SupportArchiveID
}

type NodeInfoRepository struct {
	baseFileRepo
	workPath      string
	filesystem    volumeFs
	nodeInfoFiles map[metricForID]*os.File
	writers       map[metricForID]*csv.Writer
}

func NewNodeInfoFileRepository(workPath string, fs volumeFs) *NodeInfoRepository {
	return &NodeInfoRepository{
		workPath:      workPath,
		filesystem:    fs,
		baseFileRepo:  NewBaseFileRepository(workPath, archiveNodeInfoDirName, fs),
		nodeInfoFiles: make(map[metricForID]*os.File),
		writers:       make(map[metricForID]*csv.Writer),
	}
}

func (v *NodeInfoRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.LabeledSample) error {
	return create(ctx, id, dataStream, v.createNodeInfo, v.Delete, v.finishCollection, v.close)
}

// createNodeInfo writes the content from data to the volumeInfo file.
// If the volumeInfo file exists, it overrides the existing file.
func (v *NodeInfoRepository) createNodeInfo(_ context.Context, id domain.SupportArchiveID, data *domain.LabeledSample) error {
	idMetric := metricForID{
		data.Name,
		id,
	}

	if v.nodeInfoFiles[idMetric] == nil {
		filePath := fmt.Sprintf("%s.csv", filepath.Join(v.workPath, id.Namespace, id.Name, archiveNodeInfoDirName, data.Name))
		err := v.filesystem.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			return fmt.Errorf("error creating directory for volume node info file: %w", err)
		}
		v.nodeInfoFiles[idMetric], err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}

		v.writers[idMetric] = csv.NewWriter(v.nodeInfoFiles[idMetric])
		err = v.writers[idMetric].Write(data.GetHeader())
		if err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}

	row := data.GetRow()
	err := v.writers[idMetric].Write(row)
	if err != nil {
		return fmt.Errorf("failed to write row: %w", err)
	}

	return nil
}

func (v *NodeInfoRepository) finishCollection(ctx context.Context, id domain.SupportArchiveID) error {
	var multiErr []error
	err := v.close(ctx, id)
	if err != nil {
		multiErr = append(multiErr, err)
	}

	err = v.baseFileRepo.finishCollection(ctx, id)
	if err != nil {
		multiErr = append(multiErr, err)
	}

	return errors.Join(multiErr...)
}

func (v *NodeInfoRepository) close(_ context.Context, id domain.SupportArchiveID) error {
	for key, val := range v.writers {
		if key.SupportArchiveID == id {
			val.Flush()
			val = nil
			delete(v.writers, key)
		}
	}

	var multiErr []error
	for key, val := range v.nodeInfoFiles {
		if key.SupportArchiveID == id {
			closeErr := val.Close()
			if closeErr != nil {
				multiErr = append(multiErr, closeErr)
			}
			val = nil
			delete(v.nodeInfoFiles, key)
		}
	}

	return nil
}
