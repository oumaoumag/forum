package server

import (
	"fmt"
	"log"
	"net/http"

	"time"
)

type Server struct {
	config  *ServerConfig
	httpServer *http.Server
	httpsServer *http.Server
}

func New(config *ServerConfig) *Server {
	return &Server{
		config: config,
	}
}

func (s *Server) setupServers() {
		// 	HTTPS Server
		s.httpsServer = &http.Server{
			Addr: s.config.HTTPSAddr,
			Handler: s.config.Handler,
			TLSConfig: createTLSConfig(),
			ReadTimeout: 5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout: 120 * time.Second,
		}

	// HTTP Server (for redirect)
	s.httpServer = &http.Server{
		Addr: s.config.HTTPAddr,
		Handler: s.config.Handler,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 120 * time.Second,
	} 
}

func (s *Server) Start() error {
	s.setupServers()

	// Start Servers
	go func() {
		log.Printf("Starting HTTP server on http://localhost:%s", s.config.HTTPAddr)
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()



	log.Printf("Starting HTTPS server on  http://localhost:%s")
	if err := s.httpsServer.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile); err != http.ErrServerClosed {
		return fmt.Errorf("HTTPS server error: %v", err)
	}
	return nil
}

func (s *Server) Shutdown() error {
	// TODO: Implement graceful  shutdown logic
	return nil 
}