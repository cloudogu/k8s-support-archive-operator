package collector

import (
	"context"
	"github.com/cloudogu/k8s-support-archive-operator/pkg/domain"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"time"
)

type coreV1Interface interface {
	corev1.CoreV1Interface
}

type metricsProvider interface {
	GetCapacityBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error)
	GetUsedBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error)
	GetNodeCount(ctx context.Context, start, end time.Time) (domain.NodeCountRange, error)
	GetNodeNames(ctx context.Context, start, end time.Time) (domain.NodeNameRange, error)
	GetNodeStorage(ctx context.Context, start, end time.Time) (domain.NodeStorageInfo, error)
	GetNodeFreeStorage(ctx context.Context, start, end time.Time) (domain.NodeStorageInfo, error)
	GetNodeFreeRelativeStorage(ctx context.Context, start, end time.Time) (domain.NodeStorageInfo, error)
	GetNodeRAM(ctx context.Context, start, end time.Time) (domain.NodeRAMInfo, error)
	GetNodeRAMFree(ctx context.Context, start, end time.Time) (domain.NodeRAMInfo, error)
	GetNodeRAMUsedRelative(ctx context.Context, start, end time.Time) (domain.NodeRAMInfo, error)
	GetNodeCPUCores(ctx context.Context, start, end time.Time) (domain.NodeCPUInfo, error)
	GetNodeCPUUsage(ctx context.Context, start, end time.Time) (domain.NodeCPUInfo, error)
	GetNodeCPUUsageRelative(ctx context.Context, start, end time.Time) (domain.NodeCPUInfo, error)
	GetNodeNetworkContainerBytesReceived(ctx context.Context, start, end time.Time) (domain.NodeContainerNetworkInfo, error)
	GetNodeNetworkContainerBytesSend(ctx context.Context, start, end time.Time) (domain.NodeContainerNetworkInfo, error)
}

//nolint:unused
//goland:noinspection GoUnusedType
type pvcInterface interface {
	corev1.PersistentVolumeClaimInterface
}
