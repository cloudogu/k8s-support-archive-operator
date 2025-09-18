package file

import (
	"context"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

const (
	archiveLogDirName = "Logs"
)

type LogFileRepository struct {
	baseFileRepo
	workPath   string
	filesystem volumeFs
}

func NewLogFileRepository(workPath string, fs volumeFs) *LogFileRepository {
	return &LogFileRepository{
		workPath:     workPath,
		filesystem:   fs,
		baseFileRepo: NewBaseFileRepository(workPath, archiveLogDirName, fs),
	}
}

func (l *LogFileRepository) Create(ctx context.Context, id domain.SupportArchiveID, dataStream <-chan *domain.LogLine) error {
	return create(ctx, id, dataStream, l.createPodLog, l.Delete, l.finishCollection, nil)
}

func (l *LogFileRepository) createPodLog(ctx context.Context, id domain.SupportArchiveID, data *domain.LogLine) error {
	// TODO

	return nil
}
