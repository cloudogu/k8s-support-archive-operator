package collector

import (
	"context"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"time"
)

type coreV1Interface interface {
	corev1.CoreV1Interface
}

type metricsProvider interface {
	GetCapacityBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error)
	GetUsedBytesForPVC(ctx context.Context, namespace, pvcName string, ts time.Time) (int64, error)
}

//nolint:unused
//goland:noinspection GoUnusedType
type pvcInterface interface {
	corev1.PersistentVolumeClaimInterface
}
