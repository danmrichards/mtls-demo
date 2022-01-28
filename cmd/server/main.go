package main

import (
	"flag"
	"net/http"
	"net/url"
	"os"
	"time"

	mhttp "github.com/danmrichards/mtls-demo/internal/http"
	"github.com/johanbrandhorst/certify/issuers/cfssl"
	"github.com/rs/zerolog"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const shutdownTimeout = 5 * time.Second

var bind, caURL, caCert string

func main() {
	flag.StringVar(&bind, "bind", "0.0.0.0:5000", "the ip:port to bind the API server to")
	flag.StringVar(&caURL, "ca", "localhost:8888", "the URL of the certificate authority server")
	flag.StringVar(&caCert, "ca-cert", "cert/ca.crt", "path to the root CA certificate")
	flag.Parse()

	l := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger()

	r := http.NewServeMux()
	r.HandleFunc("/hello", hello(l))

	ctx := signals.SetupSignalHandler()

	// We're using CFSSL to get TLS certificates.
	cfsslIssuer := &cfssl.Issuer{
		URL: &url.URL{
			Scheme: "http",
			Host:   caURL,
		},
	}

	srv, err := mhttp.NewServer(bind, caCert, shutdownTimeout, r, cfsslIssuer)
	if err != nil {
		l.Fatal().Err(err).Msg("could not start server")
	}

	l.Info().Str("bind", bind).Msg("starting API server")

	if err = srv.Serve(ctx); err != nil {
		l.Fatal().Err(err).Msg("could not start API server")
	}
}

func hello(l zerolog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l.Info().Msg("request received")

		w.Write([]byte("hello world"))
	}
}
