package config

import (
	"fmt"
	"github.com/Masterminds/semver/v3"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
)

const (
	StageDevelopment                           = "development"
	StageProduction                            = "production"
	StageEnvVar                                = "STAGE"
	namespaceEnvVar                            = "NAMESPACE"
	archiveVolumeDownloadServiceNameEnvVar     = "ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME"
	archiveVolumeDownloadServiceProtocolEnvVar = "ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL"
	archiveVolumeDownloadServicePortEnvVar     = "ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT"
	logLevelEnvVar   = "LOG_LEVEL"
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

	return &OperatorConfig{
		Version:                              parsedVersion,
		Namespace:                            namespace,
		ArchiveVolumeDownloadServiceName:     archiveVolumeDownloadServiceName,
		ArchiveVolumeDownloadServiceProtocol: archiveVolumeDownloadServiceProtocol,
		ArchiveVolumeDownloadServicePort:     archiveVolumeDownloadServicePort,
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
		return "", fmt.Errorf("failed to get env var [%s]: %w", logLevelEnvVar, err)
	}

	return logLevel, nil
}

func getNamespace() (string, error) {
	namespace, err := getEnvVar(namespaceEnvVar)
	if err != nil {
		return "", fmt.Errorf("failed to get env var [%s]: %w", namespaceEnvVar, err)
	}

	return namespace, nil
}

func getArchiveVolumeDownloadServiceName() (string, error) {
	envVar, err := getEnvVar(archiveVolumeDownloadServiceNameEnvVar)
	if err != nil {
		return "", fmt.Errorf("failed to get env var [%s]: %w", archiveVolumeDownloadServiceNameEnvVar, err)
	}

	return envVar, nil
}

func getArchiveVolumeDownloadServiceProtocol() (string, error) {
	envVar, err := getEnvVar(archiveVolumeDownloadServiceProtocolEnvVar)
	if err != nil {
		return "", fmt.Errorf("failed to get env var [%s]: %w", archiveVolumeDownloadServiceProtocolEnvVar, err)
	}

	return envVar, nil
}

func getArchiveVolumeDownloadServicePort() (string, error) {
	envVar, err := getEnvVar(archiveVolumeDownloadServicePortEnvVar)
	if err != nil {
		return "", fmt.Errorf("failed to get env var [%s]: %w", archiveVolumeDownloadServicePortEnvVar, err)
	}

	_, err = strconv.Atoi(envVar)
	if err != nil {
		return "", fmt.Errorf("failed to parse env var [%s]: %w", archiveVolumeDownloadServicePortEnvVar, err)
	}

	return envVar, nil
}

func getEnvVar(name string) (string, error) {
	env, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("environment variable %s must be set", name)
	}
	return env, nil
}
