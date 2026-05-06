package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"ap-scraper/internal/api/handlers"
	"ap-scraper/internal/store"
)

// Server is the HTTP API.
type Server struct {
	srv *http.Server
}

// New configures routes and returns a Server for the given listen address.
func New(st *store.Store, addr string) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /articles", handlers.AllArticles(st))
	return &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Run serves until ctx is cancelled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		err := s.srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}
