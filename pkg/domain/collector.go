package domain

type CollectorType string

const (
	CollectorTypeLog   CollectorType = "Logs"
	CollectorTypVolume CollectorType = "Volumes"
)

type CollectorUnionDataType interface {
	PodLog | VolumeMetrics
}
