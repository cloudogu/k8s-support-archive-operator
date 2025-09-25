package kubernetes

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-support-archive-lib/api/v1"
	libclient "github.com/cloudogu/k8s-support-archive-lib/client/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"

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
	HandleArchiveRequest(ctx context.Context, cr *v1.SupportArchive) (requeueAfter time.Duration, err error)
}

type deleteArchiveHandler interface {
	Delete(ctx context.Context, id domain.SupportArchiveID) error
}
