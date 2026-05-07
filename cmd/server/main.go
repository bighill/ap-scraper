package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"ap-scraper/internal/api"
	"ap-scraper/internal/config"
	"ap-scraper/internal/jobs"
	"ap-scraper/internal/scheduler"
	"ap-scraper/internal/store"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := ensureParentDir(config.DBPath); err != nil {
		return err
	}
	if err := ensureParentDir(config.CachePath); err != nil {
		return err
	}

	st, err := store.Open(ctx, config.DBPath)
	if err != nil {
		return err
	}
	defer st.Close()

	scrapeCfg := jobs.ScrapeConfig{
		WorldNewsURL: config.WorldNewsURL,
		CachePath:    config.CachePath,
		UseCache:     false,
		FetchTimeout: config.FetchTimeout,
		Retention:    config.ArticleRetentionPeriod,
	}

	sched := scheduler.New(st, config.ScrapeInterval, scrapeCfg)
	srv := api.New(st, config.HTTPAddr)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return sched.Run(ctx) })
	g.Go(func() error { return srv.Run(ctx) })

	log.Printf("listening on %s (GET /, /css.css, /js.js, /articles); scrape every %v", config.HTTPAddr, config.ScrapeInterval)
	return g.Wait()
}

func ensureParentDir(path string) error {
	parent := filepath.Dir(path)
	if parent == "." || parent == "" {
		return nil
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	return nil
}
