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
	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(web, "index.html"))
	})
	r.GET("/css.css", func(c *gin.Context) {
		c.File(filepath.Join(web, "css.css"))
	})
	r.GET("/js.js", func(c *gin.Context) {
		c.File(filepath.Join(web, "js.js"))
	})
	r.GET("/articles", gin.WrapF(handlers.ListArticles(st)))
	r.GET("/articles/count", gin.WrapF(handlers.ArticleCounts(st)))
	r.POST("/articles/hide", gin.WrapF(handlers.HideArticle(st)))
	r.POST("/articles/unhide", gin.WrapF(handlers.UnhideArticle(st)))

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
