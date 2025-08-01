package prometheus

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestGetClient(t *testing.T) {
	// when
	client, err := GetClient("address", "token")

	// then
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestTokenTransport_RoundTrip(t *testing.T) {
	t.Run("should add a token transport to the client", func(t *testing.T) {
		// given
		token := "token"
		testRequest := &http.Request{
			Header: http.Header{},
		}
		tripperMock := newMockRoundTripper(t)
		tripperMock.EXPECT().RoundTrip(testRequest).Return(nil, nil)
		transport := TokenTransport{
			roundTripper: tripperMock,
			token:        token,
		}

		// when
		_, err := transport.RoundTrip(testRequest)

		// then
		require.NoError(t, err)
		assert.Equal(t, "token", testRequest.Header.Get("Authorization"))
	})
}
