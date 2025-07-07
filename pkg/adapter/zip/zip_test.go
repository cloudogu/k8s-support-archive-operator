package zip

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreator_NewWriter(t *testing.T) {
	t.Run("should create", func(t *testing.T) {
		// when
		actual := NewCreator().NewWriter(nil)
		// then
		require.NotNil(t, actual)
	})
}

func TestNewCreator(t *testing.T) {
	t.Run("should create", func(t *testing.T) {
		// when
		actual := NewCreator()
		// then
		require.NotNil(t, actual)
	})
}
