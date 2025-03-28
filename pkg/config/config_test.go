package config

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewOperatorConfig(t *testing.T) {
	t.Run("should succeed", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("NAMESPACE", "ecosystem")

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.NoError(t, err)
		require.NotNil(t, operatorConfig)
	})
	t.Run("should succeed with namespace set", func(t *testing.T) {
		// given
		version := "0.0.0"
		t.Setenv("NAMESPACE", "ecosystem")
		t.Setenv("STAGE", "development")

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
	t.Run("fail to read namespace", func(t *testing.T) {
		// given
		version := "0.0.0"

		// when
		operatorConfig, err := NewOperatorConfig(version)

		// then
		require.Error(t, err)
		require.Nil(t, operatorConfig)
		assert.ErrorContains(t, err, "failed to read namespace: failed to get env var")
	})
}
