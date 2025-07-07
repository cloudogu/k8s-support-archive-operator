package state

import (
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/filesystem"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/adapter/zip"
)

type volumeFs interface {
	filesystem.Filesystem
}

type closableRWFile interface {
	filesystem.ClosableRWFile
}

//nolint:unused
//goland:noinspection GoUnusedType
type zipper interface {
	zip.Zipper
}

type zipCreator interface {
	zip.ZipCreator
}
