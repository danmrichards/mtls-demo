package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	mhttp "github.com/danmrichards/mtls-demo/internal/http"
	"github.com/johanbrandhorst/certify/issuers/cfssl"
	"github.com/rs/zerolog"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

const requestInterval = 1 * time.Second

var server, caURL, caCert string

func main() {
	flag.StringVar(&server, "server", "0.0.0.0:5000", "the ip:port of the API server to")
	flag.StringVar(&caURL, "ca", "localhost:8888", "the URL of the certificate authority server")
	flag.StringVar(&caCert, "ca-cert", "cert/ca.crt", "path to the root CA certificate")
	flag.Parse()

	l := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger()

	// We're using CFSSL to get TLS certificates.
	cfsslIssuer := &cfssl.Issuer{
		URL: &url.URL{
			Scheme: "http",
			Host:   caURL,
		},
	}

	client, err := mhttp.NewClient("my.server", caCert, cfsslIssuer)
	if err != nil {
		l.Fatal().Err(err).Msg("http client")
	}

	ctx := signals.SetupSignalHandler()

	if err = requestLoop(
		ctx, l, client, requestInterval, fmt.Sprintf("https://%s/hello", server),
	); err != nil {
		l.Fatal().Err(err).Msg("request loop")
	}
}

// requestLoop makes a GET request to the given URL at the given interval.
func requestLoop(ctx context.Context, l zerolog.Logger, client httpDoer, interval time.Duration, url string) error {
	for {
		select {
		case <-time.After(interval):
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return fmt.Errorf("build request: %w", err)
			}

			res, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("do request: %w", err)
			}

			b, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return fmt.Errorf("read response: %w", err)
			}
			res.Body.Close()

			l.Info().Int("status", res.StatusCode).Str("body", string(b)).Msg("")

		case <-ctx.Done():
			return nil
		}
	}
}
