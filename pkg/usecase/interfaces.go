package usecase

import (
	"context"
	libclient "github.com/cloudogu/k8s-support-archive-lib/client/v1"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	"time"
)

type collector[DATATYPE any] interface {
	Collect(ctx context.Context, startTime, endTime time.Time, resultChan chan<- *DATATYPE) error
	Name() string
}

type baseCollectorRepository interface {
	Delete(ctx context.Context, id domain.SupportArchiveID) error
	FinishCollection(ctx context.Context, id domain.SupportArchiveID) error
	IsCollected(ctx context.Context, id domain.SupportArchiveID) (bool, error)
}

type collectorRepository[DATATYPE any] interface {
	baseCollectorRepository
	Create(ctx context.Context, id domain.SupportArchiveID, data <-chan *DATATYPE) error
	Stream(ctx context.Context, id domain.SupportArchiveID, stream domain.Stream) error
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
