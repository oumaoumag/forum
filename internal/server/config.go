package server

import (
	"crypto/tls"
	"net/http"
)

type ServerConfig struct {
	HTTPAddr string
	HTTPSAddr string
	CertFile string
	KeyFile  string
	Handler http.Handler
}

func NewDefaultConfig(handler http.Handler) *ServerConfig {
	return &ServerConfig{
		HTTPAddr: ":80",
		HTTPSAddr: ":443",
		CertFile: "server.crt",
		KeyFile: "server.key",
		Handler: handler,

	}
}

// TLS Configuration
func createTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
		PreferServerCipherSuites: true,
	}
}
