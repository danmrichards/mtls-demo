package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	// Test the check handler
	req, err := http.NewRequestWithContext(context.Background(), "GET", "/check", nil)
	assert.NoError(t, err)

	// Run our request through our muxer
	rr := httptest.NewRecorder()
	s := NewServer("127.0.0.1:5678", time.Second, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK")) //nolint:errcheck
	}))
	s.svr.Handler.ServeHTTP(rr, req)

	// Assert OK
	assert.Equal(t, http.StatusOK, rr.Code)

	// Use a pre-cancelled context so Serve immediately shuts down.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.Nil(t, s.Serve(ctx))
}
