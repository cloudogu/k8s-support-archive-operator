package usecase

import (
	"context"
	libclient "github.com/cloudogu/k8s-support-archive-lib/client/v1"
	col "github.com/cloudogu/k8s-support-archive-operator/pkg/collector"
)

type stateHandler interface {
	col.StateWriter
	Read(name, namespace string) ([]string, error)
	GetDownloadURL(name, namespace string) string
}

type collector interface {
	Collect(ctx context.Context, name, namespace string, stateWriter col.StateWriter) error
	Name() string
}

type supportArchiveInterface interface {
	libclient.SupportArchiveV1Interface
}
