package domain

import libapi "github.com/cloudogu/k8s-support-archive-lib/api/v1"

type CollectorType string

const (
	CollectorTypeLog       CollectorType = "Logs"
	CollectorTypVolumeInfo CollectorType = "VolumeInfo"
	CollectorTypeNodeInfo  CollectorType = "NodeInfo"
)

func (c CollectorType) GetConditionType() string {
	switch c {
	case CollectorTypeLog:
		return "TODO"
	case CollectorTypVolumeInfo:
		return libapi.ConditionVolumeInfoFetched
	case CollectorTypeNodeInfo:
		return libapi.ConditionNodeInfoFetched
	default:
		return ""
	}
}

type CollectorUnionDataType interface {
	PodLog | VolumeInfo | NodeInfo
}
