package domain

import (
	libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"
	v1 "k8s.io/api/core/v1"
)

type CollectorType string

const (
	CollectorTypeLog       CollectorType = "Logs"
	CollectorTypVolumeInfo CollectorType = "VolumeInfo"
	CollectorTypSecret     CollectorType = "k8s/core/secrets"
)

func (c CollectorType) GetConditionType() string {
	switch c {
	case CollectorTypeLog:
		return "TODO"
	case CollectorTypVolumeInfo:
		return libapi.ConditionVolumeInfoFetched
	default:
		return ""
	}
}

type CollectorUnionDataType interface {
	PodLog | VolumeInfo | v1.SecretList
}
