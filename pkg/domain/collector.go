package domain

type CollectorType string

const (
	CollectorTypeLog       CollectorType = "Logs"
	CollectorTypVolumeInfo CollectorType = "VolumeInfo"
)

type CollectorUnionDataType interface {
	PodLog | VolumeInfo
}
