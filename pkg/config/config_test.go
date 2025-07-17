package config

import (
	"fmt"
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
			}
			got, err := GetLogLevel()
			if !tt.wantErr(t, err, fmt.Sprintf("GetLogLevel()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetLogLevel()")
		})
	}
}
