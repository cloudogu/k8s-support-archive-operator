package collector

import (
	"context"
	"time"

	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type coreV1Interface interface {
	corev1.CoreV1Interface
}

type metricsProvider interface {
	GetCapacityBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error)
	GetUsedBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error)
	GetNodeCount(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeNames(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeStorage(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeStorageFree(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeStorageFreeRelative(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeRAM(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeRAMFree(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeRAMUsedRelative(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeCPUCores(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeCPUUsage(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeCPUUsageRelative(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeNetworkContainerBytesReceived(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
	GetNodeNetworkContainerBytesSend(ctx context.Context, start, end time.Time, steps time.Duration, resultChan chan<- *domain.LabeledSample) error
}

//nolint:unused
//goland:noinspection GoUnusedType
type pvcInterface interface {
	corev1.PersistentVolumeClaimInterface
}

//nolint:unused
//goland:noinspection GoUnusedType
type secretInterface interface {
	corev1.SecretInterface
}

type LogsProvider interface {
	FindValuesOfLabel(ctx context.Context, startTimeInNanoSec, endTimeInNanoSec int64, label string) ([]string, error)
	FindLogs(ctx context.Context, startTimeInNanoSec, endTimeInNanoSec int64, namespace string, kind string) ([]string, error)
}
