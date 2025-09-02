package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOperatorConfig(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		version := "0.0.0"
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

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
	})
	t.Run("should succeed with stage set", func(t *testing.T) {
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
		t.Setenv("METRICS_MAX_SAMPLES", "11000")

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
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
