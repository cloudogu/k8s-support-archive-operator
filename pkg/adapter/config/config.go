package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Masterminds/semver/v3"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	StageDevelopment                           = "development"
	StageProduction                            = "production"
	StageEnvVar                                = "STAGE"
	namespaceEnvVar                            = "NAMESPACE"
	archiveVolumeDownloadServiceNameEnvVar     = "ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME"
	archiveVolumeDownloadServiceProtocolEnvVar = "ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL"
	archiveVolumeDownloadServicePortEnvVar     = "ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT"
	supportArchiveSyncIntervalEnvVar           = "SUPPORT_ARCHIVE_SYNC_INTERVAL"
	garbageCollectionIntervalEnvVar            = "GARBAGE_COLLECTION_INTERVAL"
	garbageCollectionNumberToKeepEnvVar        = "GARBAGE_COLLECTION_NUMBER_TO_KEEP"
	logLevelEnvVar                             = "LOG_LEVEL"
	errGetEnvVarFmt                            = "failed to get env var [%s]: %w"
	errParseEnvVarFmt                          = "failed to parse env var [%s]: %w"
	metricsServiceNameEnvVar                   = "METRICS_SERVICE_NAME"
	metricsServicePortEnvVar                   = "METRICS_SERVICE_PORT"
	metricsServiceProtocolEnvVar               = "METRICS_SERVICE_PROTOCOL"
	nodeInfoUsageMetricStepEnvVar              = "NODE_INFO_USAGE_METRIC_STEP"
	nodeInfoHardwareMetricStepEnvVar           = "NODE_INFO_HARDWARE_METRIC_STEP"
	metricsMaxSamplesEnvVar                    = "METRICS_MAX_SAMPLES"
	systemStateLabelSelectorsEnvVar            = "SYSTEM_STATE_LABEL_SELECTORS"
	systemStateGvkExclusionsEnvVar             = "SYSTEM_STATE_GVK_EXCLUSIONS"
)

var log = ctrl.Log.WithName("config")
var Stage = StageProduction

// OperatorConfig contains all configurable values for the dogu operator.
type OperatorConfig struct {
	// Version contains the current version of the operator
	Version *semver.Version
	// Namespace specifies the namespace that the operator is deployed to.
	Namespace string
	// ArchiveVolumeDownloadServiceName defines the service name for exposed support archives from the share volume.
	ArchiveVolumeDownloadServiceName string
	// ArchiveVolumeDownloadServiceProtocol defines the used protocol e.g. http or https.
	ArchiveVolumeDownloadServiceProtocol string
	// ArchiveVolumeDownloadServicePort defines the used port for the download service.
	ArchiveVolumeDownloadServicePort string
	// SupportArchiveSyncInterval defines the interval in which to resolve the difference between support archive CRs and the archives on disk.
	SupportArchiveSyncInterval time.Duration
	// GarbageCollectionInterval defines the interval between the cleaning of old support archive CRs.
	GarbageCollectionInterval time.Duration
	// GarbageCollectionNumberToKeep defines the number of latest support archive CRs to keep when cleaning them.
	GarbageCollectionNumberToKeep int
	// MetricsServiceName defines the service name for metrics service.
	MetricsServiceName string
	// MetricsServicePort defines the service port for metrics service.
	MetricsServicePort string
	// MetricsServiceProtocol defines the service protocol for metrics service.
	MetricsServiceProtocol string
	// NodeInfoUsageMetricStep defines the step width used for usage metrics (cpu/ram/network/storage free).
	NodeInfoUsageMetricStep time.Duration
	// NodeInfoHardwareMetricStep defines the step width used for hardware metrics (names, count, cores, capacities).
	NodeInfoHardwareMetricStep time.Duration
	// MetricsMaxSamples defines the maximum number of samples the metrics server can serve in a single request.
	MetricsMaxSamples int
	// SystemStateLabelSelectors defines a slice of label selectors as string in YAML format.
	SystemStateLabelSelectors string
	// SystemStateGvkExclusions defines a slice of group version kind structs as string in YAML format.
	SystemStateGvkExclusions string
}

func IsStageDevelopment() bool {
	return Stage == StageDevelopment
}

// NewOperatorConfig creates a new operator config by reading values from the environment variables
func NewOperatorConfig(version string) (*OperatorConfig, error) {
	configureStage()

	parsedVersion, err := semver.NewVersion(version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}
	log.Info(fmt.Sprintf("Version: [%s]", version))

	namespace, err := getNamespace()
	if err != nil {
		return nil, fmt.Errorf("failed to read namespace: %w", err)
	}
	log.Info(fmt.Sprintf("Deploying the k8s dogu operator in namespace %s", namespace))

	archiveVolumeDownloadServiceName, err := getArchiveVolumeDownloadServiceName()
	if err != nil {
		return nil, fmt.Errorf("failed to get archive volume download service name: %w", err)
	}
	log.Info(fmt.Sprintf("Archive volume download service name: %s", archiveVolumeDownloadServiceName))

	archiveVolumeDownloadServiceProtocol, err := getArchiveVolumeDownloadServiceProtocol()
	if err != nil {
		return nil, fmt.Errorf("failed to get archive volume download service protocol: %w", err)
	}
	log.Info(fmt.Sprintf("Archive volume download service protocol: %s", archiveVolumeDownloadServiceProtocol))

	archiveVolumeDownloadServicePort, err := getArchiveVolumeDownloadServicePort()
	if err != nil {
		return nil, fmt.Errorf("failed to get archive volume download service port: %w", err)
	}
	log.Info(fmt.Sprintf("Archive volume download service port: %s", archiveVolumeDownloadServicePort))

	supportArchiveSyncInterval, err := getDurationEnvVar(supportArchiveSyncIntervalEnvVar)
	if err != nil {
		return nil, fmt.Errorf("failed to get support archive sync interval: %w", err)
	}
	log.Info(fmt.Sprintf("Support archive sync interval: %s", supportArchiveSyncInterval))

	garbageCollectionInterval, err := getDurationEnvVar(garbageCollectionIntervalEnvVar)
	if err != nil {
		return nil, fmt.Errorf("failed to get garbage collection interval: %w", err)
	}
	log.Info(fmt.Sprintf("Garbage collection interval: %s", garbageCollectionInterval))

	garbageCollectionNumberToKeep, err := getIntEnvVar(garbageCollectionNumberToKeepEnvVar)
	if err != nil {
		return nil, fmt.Errorf("failed to get garbage collection number to keep: %w", err)
	}
	log.Info(fmt.Sprintf("Garbage collection number to keep: %d", garbageCollectionNumberToKeep))

	metricsServiceName, err := getEnvVar(metricsServiceNameEnvVar)
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Metrics service name: %s", metricsServiceName))

	metricsServicePort, err := getEnvVar(metricsServicePortEnvVar)
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Metrics service port: %s", metricsServicePort))

	metricsServiceProtocol, err := getEnvVar(metricsServiceProtocolEnvVar)
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("Metrics service protocol: %s", metricsServiceProtocol))

	nodeInfoUsageMetricStep, err := getDurationEnvVar(nodeInfoUsageMetricStepEnvVar)
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("NodeInfo usage metric step: %s", nodeInfoUsageMetricStep))

	nodeInfoHardwareMetricStep, err := getDurationEnvVar(nodeInfoHardwareMetricStepEnvVar)
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("NodeInfo hardware metric step: %s", nodeInfoHardwareMetricStep))

	metricsMaxSamples, err := getIntEnvVar(metricsMaxSamplesEnvVar)
	if err != nil {
		return nil, fmt.Errorf("failed to get maximum number of metrics samples: %w", err)
	}
	log.Info(fmt.Sprintf("Maximum number of metrics samples: %d", metricsMaxSamples))

	systemStateLabelsSelectors, err := getEnvVar(systemStateLabelSelectorsEnvVar)
	if err != nil {
		return nil, fmt.Errorf("failed to get system state label selectors: %w", err)
	}
	log.Info(fmt.Sprintf("System state label selectors: %s", systemStateLabelsSelectors))
	systemStateGvkExclusions, err := getEnvVar(systemStateGvkExclusionsEnvVar)
	if err != nil {
		return nil, fmt.Errorf("failed to get system state gvks to exclude: %w", err)
	}
	log.Info(fmt.Sprintf("System state excluded gvks: %s", systemStateGvkExclusions))

	return &OperatorConfig{
		Version:                              parsedVersion,
		Namespace:                            namespace,
		ArchiveVolumeDownloadServiceName:     archiveVolumeDownloadServiceName,
		ArchiveVolumeDownloadServiceProtocol: archiveVolumeDownloadServiceProtocol,
		ArchiveVolumeDownloadServicePort:     archiveVolumeDownloadServicePort,
		SupportArchiveSyncInterval:           supportArchiveSyncInterval,
		GarbageCollectionInterval:            garbageCollectionInterval,
		GarbageCollectionNumberToKeep:        garbageCollectionNumberToKeep,
		// prometheus is optional?
		MetricsServiceName:         metricsServiceName,
		MetricsServicePort:         metricsServicePort,
		MetricsServiceProtocol:     metricsServiceProtocol,
		NodeInfoUsageMetricStep:    nodeInfoUsageMetricStep,
		NodeInfoHardwareMetricStep: nodeInfoHardwareMetricStep,
		MetricsMaxSamples:          metricsMaxSamples,
		SystemStateLabelSelectors:  systemStateLabelsSelectors,
		SystemStateGvkExclusions:   systemStateGvkExclusions,
	}, nil
}

func configureStage() {
	var err error
	Stage, err = getEnvVar(StageEnvVar)
	if err != nil {
		log.Error(err, "Error reading stage environment variable. Use stage production")
	}

	if IsStageDevelopment() {
		log.Info("Starting in development mode! This is not recommended for production!")
	}
}

func GetLogLevel() (string, error) {
	logLevel, err := getEnvVar(logLevelEnvVar)
	if err != nil {
		return "", fmt.Errorf(errGetEnvVarFmt, logLevelEnvVar, err)
	}

	return logLevel, nil
}

func getNamespace() (string, error) {
	namespace, err := getEnvVar(namespaceEnvVar)
	if err != nil {
		return "", fmt.Errorf(errGetEnvVarFmt, namespaceEnvVar, err)
	}

	return namespace, nil
}

func getArchiveVolumeDownloadServiceName() (string, error) {
	envVar, err := getEnvVar(archiveVolumeDownloadServiceNameEnvVar)
	if err != nil {
		return "", fmt.Errorf(errGetEnvVarFmt, archiveVolumeDownloadServiceNameEnvVar, err)
	}

	return envVar, nil
}

func getArchiveVolumeDownloadServiceProtocol() (string, error) {
	envVar, err := getEnvVar(archiveVolumeDownloadServiceProtocolEnvVar)
	if err != nil {
		return "", fmt.Errorf(errGetEnvVarFmt, archiveVolumeDownloadServiceProtocolEnvVar, err)
	}

	return envVar, nil
}

func getArchiveVolumeDownloadServicePort() (string, error) {
	envVar, err := getEnvVar(archiveVolumeDownloadServicePortEnvVar)
	if err != nil {
		return "", fmt.Errorf(errGetEnvVarFmt, archiveVolumeDownloadServicePortEnvVar, err)
	}

	_, err = strconv.Atoi(envVar)
	if err != nil {
		return "", fmt.Errorf(errParseEnvVarFmt, archiveVolumeDownloadServicePortEnvVar, err)
	}

	return envVar, nil
}

func getDurationEnvVar(name string) (time.Duration, error) {
	envVar, err := getEnvVar(name)
	if err != nil {
		return 0, fmt.Errorf(errGetEnvVarFmt, name, err)
	}

	duration, err := time.ParseDuration(envVar)
	if err != nil {
		return 0, fmt.Errorf(errParseEnvVarFmt, name, err)
	}

	return duration, nil
}

func getIntEnvVar(name string) (int, error) {
	envVar, err := getEnvVar(name)
	if err != nil {
		return 0, fmt.Errorf(errGetEnvVarFmt, name, err)
	}

	intVal, err := strconv.Atoi(envVar)
	if err != nil {
		return 0, fmt.Errorf(errParseEnvVarFmt, name, err)
	}

	return intVal, nil
}

func getEnvVar(name string) (string, error) {
	env, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("environment variable %s must be set", name)
	}
	return env, nil
}
