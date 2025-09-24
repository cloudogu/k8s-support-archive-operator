package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setTestEnvVars(t *testing.T) {
	t.Setenv("NAMESPACE", "ecosystem")
	t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME", "service")
	t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL", "http")
	t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT", "8080")
	t.Setenv("METRICS_SERVICE_NAME", "metrics")
	t.Setenv("METRICS_SERVICE_PORT", "8081")
	t.Setenv("METRICS_SERVICE_PROTOCOL", "http")
	t.Setenv("SUPPORT_ARCHIVE_SYNC_INTERVAL", "1m")
	t.Setenv("GARBAGE_COLLECTION_INTERVAL", "5m")
	t.Setenv("GARBAGE_COLLECTION_NUMBER_TO_KEEP", "5")
	t.Setenv("NODE_INFO_USAGE_METRIC_STEP", "30s")
	t.Setenv("NODE_INFO_HARDWARE_METRIC_STEP", "30m")
	t.Setenv("METRICS_MAX_SAMPLES", "11000")
	t.Setenv("LOG_GATEWAY_URL", "loki")
	t.Setenv("LOG_GATEWAY_USERNAME", "lokiU")
	t.Setenv("LOG_GATEWAY_PASSWORD", "lokiP")
	t.Setenv("LOG_MAX_QUERY_RESULT_COUNT", "2000")
	t.Setenv("LOG_MAX_QUERY_TIME_WINDOW", "24h")
	t.Setenv("LOG_EVENT_SOURCE_NAME", "loki.kubernetes_events")
	t.Setenv("SYSTEM_STATE_LABEL_SELECTORS", "app: ces")
	t.Setenv("SYSTEM_STATE_GVK_EXCLUSIONS", "- group: apps\n  kind: Deployment\n  version: v1")
}

func TestNewOperatorConfig(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		version := "0.0.0"
		setTestEnvVars(t)

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
		assert.Equal(t, "ecosystem", operatorConfig.Namespace)
		assert.Equal(t, "service", operatorConfig.ArchiveVolumeDownloadServiceName)
		assert.Equal(t, "8080", operatorConfig.ArchiveVolumeDownloadServicePort)
		assert.Equal(t, "http", operatorConfig.ArchiveVolumeDownloadServiceProtocol)
		assert.Equal(t, "metrics", operatorConfig.MetricsServiceName)
		assert.Equal(t, "8081", operatorConfig.MetricsServicePort)
		assert.Equal(t, "http", operatorConfig.MetricsServiceProtocol)
		assert.Equal(t, time.Minute, operatorConfig.SupportArchiveSyncInterval)
		assert.Equal(t, time.Minute*5, operatorConfig.GarbageCollectionInterval)
		assert.Equal(t, time.Second*30, operatorConfig.NodeInfoUsageMetricStep)
		assert.Equal(t, time.Minute*30, operatorConfig.NodeInfoHardwareMetricStep)
		assert.Equal(t, 11000, operatorConfig.MetricsMaxSamples)
		assert.Equal(t, "loki", operatorConfig.LogGatewayConfig.Url)
		assert.Equal(t, "lokiU", operatorConfig.LogGatewayConfig.Username)
		assert.Equal(t, "lokiP", operatorConfig.LogGatewayConfig.Password)
		assert.Equal(t, 2000, operatorConfig.LogsMaxQueryResultCount)
		assert.Equal(t, time.Hour*24, operatorConfig.LogsMaxQueryTimeWindow)
		assert.Equal(t, "loki.kubernetes_events", operatorConfig.LogsEventSourceName)
	})
	t.Run("should succeed with stage set", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("STAGE", "development")
		setTestEnvVars(t)

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
		assert.Equal(t, "development", Stage)
	})
	t.Run("should fail to parse sync interval", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME", "service")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL", "http")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT", "8080")
		t.Setenv("SUPPORT_ARCHIVE_SYNC_INTERVAL", "not a time.Duration")

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.Error(t, err)
		require.Nil(t, operatorConfig)
		assert.ErrorContains(t, err, "failed to get support archive sync interval: failed to parse env var [SUPPORT_ARCHIVE_SYNC_INTERVAL]")
	})
	t.Run("should fail to parse garbage collection interval", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME", "service")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL", "http")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT", "8080")
		t.Setenv("SUPPORT_ARCHIVE_SYNC_INTERVAL", "1m")
		t.Setenv("GARBAGE_COLLECTION_INTERVAL", "not a time.Duration")

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.Error(t, err)
		require.Nil(t, operatorConfig)
		assert.ErrorContains(t, err, "failed to get garbage collection interval: failed to parse env var [GARBAGE_COLLECTION_INTERVAL]")
	})
	t.Run("should fail to parse garbage collection number to keep", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME", "service")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL", "http")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT", "8080")
		t.Setenv("SUPPORT_ARCHIVE_SYNC_INTERVAL", "1m")
		t.Setenv("GARBAGE_COLLECTION_INTERVAL", "5m")
		t.Setenv("GARBAGE_COLLECTION_NUMBER_TO_KEEP", "not a number")

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.Error(t, err)
		require.Nil(t, operatorConfig)
		assert.ErrorContains(t, err, "failed to get garbage collection number to keep: failed to parse env var [GARBAGE_COLLECTION_NUMBER_TO_KEEP]")
	})

	t.Run("should fail to parse node info usage metric step", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME", "service")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL", "http")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT", "8080")
		t.Setenv("METRICS_SERVICE_NAME", "metrics")
		t.Setenv("METRICS_SERVICE_PORT", "8081")
		t.Setenv("METRICS_SERVICE_PROTOCOL", "http")
		t.Setenv("SUPPORT_ARCHIVE_SYNC_INTERVAL", "1m")
		t.Setenv("GARBAGE_COLLECTION_INTERVAL", "5m")
		t.Setenv("GARBAGE_COLLECTION_NUMBER_TO_KEEP", "5")
		t.Setenv("NODE_INFO_USAGE_METRIC_STEP", "not a duration")
		t.Setenv("NODE_INFO_HARDWARE_METRIC_STEP", "30m")

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.Error(t, err)
		require.Nil(t, operatorConfig)
		assert.ErrorContains(t, err, "failed to parse env var [NODE_INFO_USAGE_METRIC_STEP]")
	})

	t.Run("should fail to parse node info hardware metric step", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME", "service")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL", "http")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT", "8080")
		t.Setenv("METRICS_SERVICE_NAME", "metrics")
		t.Setenv("METRICS_SERVICE_PORT", "8081")
		t.Setenv("METRICS_SERVICE_PROTOCOL", "http")
		t.Setenv("SUPPORT_ARCHIVE_SYNC_INTERVAL", "1m")
		t.Setenv("GARBAGE_COLLECTION_INTERVAL", "5m")
		t.Setenv("GARBAGE_COLLECTION_NUMBER_TO_KEEP", "5")
		t.Setenv("NODE_INFO_USAGE_METRIC_STEP", "30s")
		t.Setenv("NODE_INFO_HARDWARE_METRIC_STEP", "not a duration")

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.Error(t, err)
		require.Nil(t, operatorConfig)
		assert.ErrorContains(t, err, "failed to parse env var [NODE_INFO_HARDWARE_METRIC_STEP]")
	})
	t.Run("should fail to parse metrics max samples", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME", "service")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL", "http")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT", "8080")
		t.Setenv("METRICS_SERVICE_NAME", "metrics")
		t.Setenv("METRICS_SERVICE_PORT", "8081")
		t.Setenv("METRICS_SERVICE_PROTOCOL", "http")
		t.Setenv("SUPPORT_ARCHIVE_SYNC_INTERVAL", "1m")
		t.Setenv("GARBAGE_COLLECTION_INTERVAL", "5m")
		t.Setenv("GARBAGE_COLLECTION_NUMBER_TO_KEEP", "5")
		t.Setenv("NODE_INFO_USAGE_METRIC_STEP", "30s")
		t.Setenv("NODE_INFO_HARDWARE_METRIC_STEP", "30m")
		t.Setenv("METRICS_MAX_SAMPLES", "not a number")

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.Error(t, err)
		require.Nil(t, operatorConfig)
		assert.ErrorContains(t, err, "failed to get maximum number of metrics samples")
	})

	t.Run("fail to parse version", func(t *testing.T) {
		// given
		version := "0.0."
		t.Setenv("NAMESPACE", "ecosystem")

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.Error(t, err)
		require.Nil(t, operatorConfig)
		assert.ErrorContains(t, err, "failed to parse version: invalid semantic version")
	})
	t.Run("fail to read namespace because of non existent env var", func(t *testing.T) {
		// given
		version := "0.0.0"
		require.NoError(t, os.Unsetenv("NAMESPACE"))

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.Error(t, err)
		require.Nil(t, operatorConfig)
		assert.ErrorContains(t, err, "failed to read namespace: failed to get env var")
	})
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "test log level not set",
			wantErr: assert.Error,
		},
		{
			name:    "test log level set to debug",
			want:    "debug",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.want != "" {
				t.Setenv(logLevelEnvVar, tt.want)
			} else {
				require.NoError(t, os.Unsetenv(logLevelEnvVar))
			}
			got, err := GetLogLevel()
			if !tt.wantErr(t, err, fmt.Sprintf("GetLogLevel()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetLogLevel()")
		})
	}
}

func Test_configureLokiGateway(t *testing.T) {
	tests := []struct {
		name      string
		want      LogGatewayConfig
		wantErr   assert.ErrorAssertionFunc
		envSetter func(t *testing.T)
	}{
		{
			name: "should return error on error reading gateway url",
			want: LogGatewayConfig{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "environment variable LOG_GATEWAY_URL must be set")
			},
		},
		{
			name: "should return error on error reading gateway username",
			want: LogGatewayConfig{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "environment variable LOG_GATEWAY_USERNAME must be set")
			},
			envSetter: func(t *testing.T) {
				t.Setenv(logGatewayUrlEnvironmentVariable, "loki")
			},
		},
		{
			name: "should return error on error reading gateway password",
			want: LogGatewayConfig{},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return assert.ErrorContains(t, err, "environment variable LOG_GATEWAY_PASSWORD must be set")
			},
			envSetter: func(t *testing.T) {
				t.Setenv(logGatewayUrlEnvironmentVariable, "loki")
				t.Setenv(logGatewayUsernameEnvironmentVariable, "lokiU")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envSetter != nil {
				tt.envSetter(t)
			}

			config := &OperatorConfig{}
			err := getLogConfig(config)
			if !tt.wantErr(t, err, fmt.Sprintf("getLogConfig()")) {
				return
			}
			assert.Equalf(t, tt.want, config.LogGatewayConfig, "getLogConfig()")
		})
	}
}
