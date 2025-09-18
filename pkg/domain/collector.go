package domain

import libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"

type CollectorType string

const (
	CollectorTypeLog         CollectorType = "Logs"
	CollectorTypeVolumeInfo  CollectorType = "VolumeInfo"
	CollectorTypeNodeInfo    CollectorType = "NodeInfo"
	CollectorTypSecret       CollectorType = "Resources/Secrets"
	CollectorTypeSystemState CollectorType = "Resources/SystemState"
)

func (c CollectorType) GetConditionType() string {
	switch c {
	case CollectorTypeLog:
		return "TODO"
	case CollectorTypeVolumeInfo:
		return libapi.ConditionVolumeInfoFetched
	case CollectorTypeNodeInfo:
		return libapi.ConditionNodeInfoFetched
	case CollectorTypSecret:
		return libapi.ConditionSecretsFetched
	case CollectorTypeSystemState:
		return libapi.ConditionSystemStateFetched
	default:
		return ""
	}
}

type CollectorUnionDataType interface {
	PodLog | VolumeInfo | LabeledSample | SecretYaml | UnstructuredResource
}
