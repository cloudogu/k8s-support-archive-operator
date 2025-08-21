package domain

import (
	libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"
)

type CollectorType string

const (
	CollectorTypeLog       CollectorType = "Logs"
	CollectorTypVolumeInfo CollectorType = "VolumeInfo"
	CollectorTypSecret     CollectorType = "Resources/Secrets"
)

func (c CollectorType) GetConditionType() string {
	switch c {
	case CollectorTypeLog:
		return "TODO"
	case CollectorTypVolumeInfo:
		return libapi.ConditionVolumeInfoFetched
	case CollectorTypSecret:
		return libapi.ConditionSecretsFetched
	default:
		return ""
	}
}

type CollectorUnionDataType interface {
	PodLog | VolumeInfo | SecretYaml
}
