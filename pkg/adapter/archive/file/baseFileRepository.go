package file

import (
	"context"
	"fmt"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	stateFileName = ".done"
)

type baseFileRepository struct {
	workPath   string
	filesystem volumeFs
}

func NewBaseFileRepository(workPath string, filesystem volumeFs) *baseFileRepository {
	return &baseFileRepository{
		workPath:   workPath,
		filesystem: filesystem,
	}
}

func (l *baseFileRepository) FinishCollection(ctx context.Context, id domain.SupportArchiveID) error {
	logger := log.FromContext(ctx).WithName("LogFileRepository.FinishCollection")
	stateFilePath := getStateFilePath(l.workPath, id)

	err := l.filesystem.MkdirAll(filepath.Dir(stateFilePath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	create, err := l.filesystem.Create(stateFilePath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", stateFilePath, err)
	}
	defer func() {
		createErr := create.Close()
		if err != nil {
			logger.Error(createErr, fmt.Sprintf("failed to close file %s", id))
		}
	}()
	_, err = create.Write([]byte("done"))
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", stateFilePath, err)
	}

	return nil
}

func (l *baseFileRepository) IsCollected(_ context.Context, id domain.SupportArchiveID) (bool, error) {
	stateFilePath := getStateFilePath(l.workPath, id)
	_, err := l.filesystem.Stat(stateFilePath)

	if err != nil && os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check if file %s exists: %w", stateFilePath, err)
	}
	return true, nil
}

func getStateFilePath(workPath string, id domain.SupportArchiveID) string {
	return filepath.Join(workPath, id.Namespace, id.Name, stateFileName)
}
