package file

import (
	"context"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/filesystem"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"io"
)

type volumeFs interface {
	filesystem.Filesystem
}

type closableRWFile interface {
	filesystem.ClosableRWFile
}

type Zipper interface {
	Close() error
	Create(name string) (io.Writer, error)
}

//nolint:unused
//goland:noinspection GoUnusedType
type Reader interface {
	io.Reader
}

type baseFileRepo interface {
	IsCollected(ctx context.Context, id domain.SupportArchiveID, collectorDir string) (bool, error)
	FinishCollection(ctx context.Context, id domain.SupportArchiveID, collectorDir string) error
	Delete(ctx context.Context, id domain.SupportArchiveID, collectorDir string) error
	stream(ctx context.Context, id domain.SupportArchiveID, directory string, stream *domain.Stream) (func() error, error)
}
