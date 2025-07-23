package file

import (
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/filesystem"
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
