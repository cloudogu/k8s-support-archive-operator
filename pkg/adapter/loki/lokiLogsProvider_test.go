package loki

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLokiLogsProvider(t *testing.T) {

	t.Run("should call REST API once because duration is less than the 30d1h limit", func(t *testing.T) {
		var callCount int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount += 1
		}))
		defer server.Close()

		lokiLogsPrv := NewLokiLogsProvider(server.Client())
		startTime := time.Now()
		endTime := startTime.AddDate(0, 0, 30)
		_, _ = lokiLogsPrv.getValuesOfLabel(context.TODO(), startTime, endTime, "aKind")

		assert.Equal(t, 1, callCount)

	})
}
