package usecase

import (
	"context"
	libclient "github.com/cloudogu/k8s-support-archive-lib/client/v1"
	col "github.com/cloudogu/k8s-support-archive-operator/pkg/collector"
)

type stateHandler interface {
	col.StateWriter
	Read(ctx context.Context, name, namespace string) ([]string, error)
	GetDownloadURL(ctx context.Context, name, namespace string) string
}

type archiveDataCollector interface {
	Collect(ctx context.Context, name, namespace string, stateWriter col.StateWriter) error
	Name() string
}

type supportArchiveV1Interface interface {
	libclient.SupportArchiveV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type supportArchiveInterface interface {
	libclient.SupportArchiveInterface
}
