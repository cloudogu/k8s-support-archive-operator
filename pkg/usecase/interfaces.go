package usecase

import (
	"context"
	libclient "github.com/cloudogu/k8s-support-archive-lib/client/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"time"
)

type collector[DATATYPE domain.CollectorUnionDataType] interface {
	Collect(ctx context.Context, namespace string, startTime, endTime time.Time, resultChan chan<- *DATATYPE) error
	Name() string
}

type baseCollectorRepository interface {
	Delete(ctx context.Context, id domain.SupportArchiveID) error
	FinishCollection(ctx context.Context, id domain.SupportArchiveID) error
	IsCollected(ctx context.Context, id domain.SupportArchiveID) (bool, error)
}

type collectorRepository[DATATYPE domain.CollectorUnionDataType] interface {
	baseCollectorRepository
	Create(ctx context.Context, id domain.SupportArchiveID, data <-chan *DATATYPE) error
	// Stream streams data to the given stream
	// It returns a func to finalize the stream which has to be called by the useCase to free up resources and avoid memory exhaustion.
	// The repository itself cannot do this because it cannot recognize when the data is fully read.
	// The func may be nil.
	Stream(ctx context.Context, id domain.SupportArchiveID, stream domain.Stream) (func() error, error)
}

type supportArchiveRepository interface {
	Create(ctx context.Context, id domain.SupportArchiveID, streams map[domain.CollectorType]domain.Stream) (url string, err error)
	Delete(ctx context.Context, id domain.SupportArchiveID) error
	Exists(ctx context.Context, id domain.SupportArchiveID) (bool, error)
}

type supportArchiveV1Interface interface {
	libclient.SupportArchiveV1Interface
}

//nolint:unused
//goland:noinspection GoUnusedType
type supportArchiveInterface interface {
	libclient.SupportArchiveInterface
}
