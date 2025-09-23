package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	archiveEventsDirName = "Events"
)

type EventFileRepository struct {
	*SingleLogFileRepository
}

func NewEventFileRepository(workPath string, fs volumeFs) *LogFileRepository {
	return &LogFileRepository{
		NewSingleLogFileRepository(workPath, archiveEventsDirName, fs),
	}
}
