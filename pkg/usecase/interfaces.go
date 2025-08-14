package usecase

import (
	"context"
	"time"

	libclient "github.com/cloudogu/k8s-support-archive-lib/client/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
)

type collector[DATATYPE domain.CollectorUnionDataType] interface {
	Collect(ctx context.Context, namespace string, startTime, endTime time.Time, resultChan chan<- *DATATYPE) error
	Name() string
}

type baseCollectorRepository interface {
	Delete(ctx context.Context, id domain.SupportArchiveID) error
	IsCollected(ctx context.Context, id domain.SupportArchiveID) (bool, error)
	Stream(ctx context.Context, id domain.SupportArchiveID, stream *domain.Stream) error
}

type collectorRepository[DATATYPE domain.CollectorUnionDataType] interface {
	baseCollectorRepository
	Create(ctx context.Context, id domain.SupportArchiveID, data <-chan *DATATYPE) error
}

type supportArchiveRepository interface {
	// Create builds the support archive for the provided streams.
	// The stream itself contains a constructor with a Close Func.
	// The func must be called by the repository after reading the stream or when an error occurs to avoid resource exhaustion.
	Create(ctx context.Context, id domain.SupportArchiveID, streams map[domain.CollectorType]*domain.Stream) (url string, err error)
	Delete(ctx context.Context, id domain.SupportArchiveID) error
	Exists(ctx context.Context, id domain.SupportArchiveID) (bool, error)
	List(ctx context.Context) ([]domain.SupportArchiveID, error)
}

type supportArchiveV1Interface interface {
	libclient.SupportArchiveV1Interface
}

type deleteArchiveHandler interface {
	Delete(ctx context.Context, id domain.SupportArchiveID) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type supportArchiveInterface interface {
	libclient.SupportArchiveInterface
}
