package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/johanbrandhorst/certify"
)

const clientName = "my.client"

// Client wraps http.Client while providing convenience methods and default
// values for making HTTP requests where the network host differs from the TLS
// Server Name. The client also presents a TLS certificate when making requests
// using the "Certify" library.
type Client struct {
	*http.Client

	serverName string
}

// NewClient returns a new http.Client configured to be able to make TLS
// connections to hosts with SSL certificates valid for serverName.
//
// If making HTTPS requests this client can ONLY be used to make requests
// to hosts with TLS certificates valid for serverName. Requests to any other
// hosts will fail during the TLS handshake process.
func NewClient(serverName, caCert string, certIssuer certify.Issuer) (*Client, error) {
	// Use the default transport as a starting point to make use of it's
	// configured timeout values.
	dt := http.DefaultTransport.(*http.Transport)

	// Load the root CA certificate, needed to verify the server certificate
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
		CommonName: clientName,
		Issuer:     certIssuer,
		// It is recommended to use a cache.
		Cache: certify.NewMemCache(),
		// It is recommended to set RenewBefore.
		// Refresh cached certificates when < 24H left before expiry.
		RenewBefore: 24 * time.Hour,
	}

	tr := &http.Transport{
		MaxIdleConns:          dt.MaxIdleConns,
		IdleConnTimeout:       dt.IdleConnTimeout,
		TLSHandshakeTimeout:   dt.TLSHandshakeTimeout,
		ExpectContinueTimeout: dt.ExpectContinueTimeout,

		// When communicating with different hosts with different IP addresses,
		// that share the same SSL certificate, we need to consider
		// SNI (Server Name Indication). This is used during the TLS handshake
		// process to verify the SSL certificate used by the server.
		//
		// Due to the way in which Go implements SNI we have to manually set
		// the server name on the TLS config. We also need to set the same
		// server name as the Host value on the HTTP request itself.
		//
		// See: https://github.com/golang/go/issues/22704
		TLSClientConfig: &tls.Config{
			ServerName:           serverName,
			RootCAs:              caCertPool,
			GetClientCertificate: c.GetClientCertificate,
		},
	}

	return &Client{
		Client:     &http.Client{Transport: tr},
		serverName: serverName,
	}, nil
}

// Do sends an HTTP request and returns an HTTP response, following policy (such
// as redirects, cookies, auth) as configured on the client.
//
// This method wraps the standard lib net/http client Do method. Before passing
// the request upstream the method sets the Host value of the request to the
// configured server name on the client, used for SNI.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	req.Host = c.serverName

	return c.Client.Do(req)
}
