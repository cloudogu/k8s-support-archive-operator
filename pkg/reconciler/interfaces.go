package reconciler

import (
	"context"
	libv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	"github.com/cloudogu/k8s-support-archive-lib/client"
	"k8s.io/client-go/kubernetes"
)

type EcosystemClientSet interface {
	kubernetes.Interface
	client.SupportArchiveEcosystemInterface
}

type archiveHandler interface {
	HandleArchiveRequest(ctx context.Context, cr *libv1.SupportArchive) (requeue bool, err error)
}

type archiveCleaner interface {
	Clean(ctx context.Context, supportArchiveName, namespace string) error
}
