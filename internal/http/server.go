package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/johanbrandhorst/certify"
)

const serverName = "my.server"

// Server is a basic HTTP server.
type Server struct {
	ln              net.Listener
	svr             *http.Server
	shutdownTimeout time.Duration
}

// NewServer returns a new Server.
func NewServer(bind, caCert string, shutdownTimeout time.Duration, handler http.Handler, certIssuer certify.Issuer) (*Server, error) {
	// Load the root CA certificate, needed to verify the client certificate
	// (signed by this CA).
	cert, err := ioutil.ReadFile(caCert)
	if err != nil {
		return nil, fmt.Errorf("open root CA cert: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(cert)

	c := &certify.Certify{
		// Used when request client-side certificates and
		// added to SANs or IPSANs depending on format.
		CommonName: serverName,
		Issuer:     certIssuer,
		// It is recommended to use a cache.
		Cache: certify.NewMemCache(),
		// It is recommended to set RenewBefore.
		// Refresh cached certificates when < 24H left before expiry.
		RenewBefore: 24 * time.Hour,
	}

	return &Server{
		svr: &http.Server{
			Addr:    bind,
			Handler: handler,
			TLSConfig: &tls.Config{
				ClientAuth:     tls.RequireAndVerifyClientCert,
				ClientCAs:      caCertPool,
				GetCertificate: c.GetCertificate,
				MinVersion:     tls.VersionTLS12,
			},
		},
		shutdownTimeout: shutdownTimeout,
	}, nil
}

// Serve binds the http to its bind address and starts serving requests.
func (s *Server) Serve(ctx context.Context) (err error) {
	ch := make(chan error)
	defer close(ch)

	s.ln, err = tls.Listen("tcp", s.svr.Addr, s.svr.TLSConfig)
	if err != nil {
		return err
	}

	// Start serving.
	go s.serve(ch)

	// Wait on our context to be cancelled.
	<-ctx.Done()

	// Shutdown http within 5 seconds
	ctxShutdown, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err = suppressServerClosed(s.svr.Shutdown(ctxShutdown)); err == nil {
		return suppressServerClosed(<-ch)
	}

	if chErr := suppressServerClosed(<-ch); chErr != nil {
		return multierror.Append(err, chErr)
	}

	return err
}

func suppressServerClosed(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) serve(ch chan error) {
	ch <- s.svr.Serve(s.ln)
}
