package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestNewOperatorConfig(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_NAME", "service")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PROTOCOL", "http")
		t.Setenv("ARCHIVE_VOLUME_DOWNLOAD_SERVICE_PORT", "8080")

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

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
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
		assert.ErrorContains(t, err, "failed to parse version: Invalid Semantic Version")
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
