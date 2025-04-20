package apiserver

import (
	"context"
	"net/http"
)

// Server implements manager.Runnable for the HTTP API server
type Server struct {
	httpServer *http.Server
}

// NewServer creates a new API server
func NewServer(httpServer *http.Server) *Server {
	return &Server{
		httpServer: httpServer,
	}
}

// Start implements manager.Runnable
func (s *Server) Start(ctx context.Context) error {
	errCh := make(chan error)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		// Graceful shutdown
		return s.httpServer.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}
