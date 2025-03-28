package reconciler

import (
	"github.com/cloudogu/k8s-support-archive-lib/client"
	"k8s.io/client-go/kubernetes"
)

type EcosystemClientSet interface {
	kubernetes.Interface
	client.SupportArchiveEcosystemInterface
}
