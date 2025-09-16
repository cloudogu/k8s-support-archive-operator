package domain

import libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"

type CollectorType string

const (
	CollectorTypeLog        CollectorType = "Logs"
	CollectorTypeVolumeInfo CollectorType = "VolumeInfo"
	CollectorTypeNodeInfo   CollectorType = "NodeInfo"
	CollectorTypeSecret     CollectorType = "Resources/Secrets"
	CollectorTypeEvents     CollectorType = "Events"
)

func (c CollectorType) GetConditionType() string {
	switch c {
	case CollectorTypeLog:
		return libapi.ConditionLogsFetched
	case CollectorTypeVolumeInfo:
		return libapi.ConditionVolumeInfoFetched
	case CollectorTypeNodeInfo:
		return libapi.ConditionNodeInfoFetched
	case CollectorTypeSecret:
		return libapi.ConditionSecretsFetched
	case CollectorTypeEvents:
		return libapi.ConditionEventsFetched
	default:
		return ""
	}
}

type CollectorUnionDataType interface {
	LogLine | VolumeInfo | LabeledSample | SecretYaml | EventSet
}
