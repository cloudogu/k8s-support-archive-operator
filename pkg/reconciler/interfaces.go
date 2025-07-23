package reconciler

import (
	"context"
	libv1 "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	libclient "github.com/cloudogu/k8s-support-archive-lib/client/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

//nolint:unused
//goland:noinspection GoUnusedType
type controllerManager interface {
	ctrl.Manager
}

//nolint:unused
//goland:noinspection GoUnusedType
type supportArchiveInterface interface {
	libclient.SupportArchiveInterface
}

type supportArchiveV1Interface interface {
	libclient.SupportArchiveV1Interface
}

type createArchiveHandler interface {
	HandleArchiveRequest(ctx context.Context, cr *libv1.SupportArchive) (requeue bool, err error)
}

type deleteArchiveHandler interface {
	Delete(ctx context.Context, cr *libv1.SupportArchive) error
}
