package http

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	tsRes = []byte("hello world")
)

const (
	testServerName = "example.com"
)

type helloHandler struct {
	url string
}

func (h *helloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-URL", h.url)
	if r.Host != testServerName {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(tsRes)
}

func testServer() *httptest.Server {
	h := &helloHandler{}
	ts := httptest.NewTLSServer(h)
	h.url = ts.URL
	return ts
}

func TestNewClient(t *testing.T) {
	ts1 := testServer()
	defer ts1.Close()

	ts2 := testServer()
	defer ts2.Close()

	c, err := NewClient(testServerName)
	assert.NoError(t, err)

	// Replace the TLS config on the SNI client with the one from the test TLS
	// client. This client has been specifically configured to be able to talk
	// to the test TLS server using the standard lib self-signed certificate.
	tsTLSConfig := ts1.Client().Transport.(*http.Transport).TLSClientConfig
	tsTLSConfig.ServerName = testServerName
	c.Transport.(*http.Transport).TLSClientConfig = tsTLSConfig

	tests := []struct {
		name string
		url  string
	}{
		{name: "first-server", url: ts1.URL},
		{name: "second-server", url: ts2.URL},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tc.url+"/hello", nil)
			assert.NoError(t, err, "new request")

			res, err := c.Do(req)
			assert.NoError(t, err, "do request")
			assert.Equal(t, http.StatusOK, res.StatusCode)

			// Validate we got a response from the correct server.
			assert.Equal(t, res.Header.Get("X-URL"), tc.url)

			b, err := ioutil.ReadAll(res.Body)
			assert.NoError(t, err, "decode response")
			assert.Equal(t, tsRes, b)
		})
	}
}
