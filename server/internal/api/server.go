package api

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"ap-scraper/internal/api/handlers"
	"ap-scraper/internal/config"
	"ap-scraper/internal/store"

	"github.com/gin-gonic/gin"
)

// Server is the HTTP API and static web UI.
type Server struct {
	srv *http.Server
}

// New configures routes and returns a Server for the given listen address.
func New(st *store.Store, addr string) *Server {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())

	web := filepath.Clean(config.WebUIDir)

	r.GET("/articles", gin.WrapF(handlers.ListArticles(st)))
	r.GET("/articles/:id", handlers.GetArticle(st))
	r.GET("/articles/count", gin.WrapF(handlers.ArticleCounts(st)))
	r.POST("/articles/hide", gin.WrapF(handlers.HideArticle(st)))
	r.POST("/articles/unhide", gin.WrapF(handlers.UnhideArticle(st)))

	r.GET("/settings/images", gin.WrapF(handlers.GetShowImages(st)))
	r.POST("/settings/images", gin.WrapF(handlers.SetShowImages(st)))

	// Serve the web UI from the root without registering a catch-all
	// wildcard, which would conflict with the /articles API routes.
	fileServer := http.FileServer(http.Dir(web))
	r.NoRoute(func(c *gin.Context) {
		fileServer.ServeHTTP(c.Writer, c.Request)
	})

	return &Server{
		srv: &http.Server{
			Addr:              addr,
			Handler:           r,
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
