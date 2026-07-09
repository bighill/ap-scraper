package jobs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"ap-scraper/internal/parser"
	"ap-scraper/internal/store"
)

// ScrapeConfig controls a single scrape run.
type ScrapeConfig struct {
	WorldNewsURL string
	CachePath    string
	UseCache     bool
	FetchTimeout time.Duration
	Retention    time.Duration
}

// RunScrape fetches (or reads cache), parses HTML, upserts articles, and applies retention.
func RunScrape(ctx context.Context, st *store.Store, cfg ScrapeConfig) error {
	mode := "refresh-cache"
	var html []byte
	if cfg.UseCache {
		mode = "use-cache"
		cached, err := os.ReadFile(cfg.CachePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("use-cache: cache file missing: %s", cfg.CachePath)
			}
			return fmt.Errorf("read cache file: %w", err)
		}
		html = cached
	} else {
		fetched, err := fetchAndWriteCache(ctx, cfg.WorldNewsURL, cfg.CachePath, cfg.FetchTimeout)
		if err != nil {
			return err
		}
		html = fetched
	}

	articles, err := parser.ParseWorldNewsHTML(html)
	if err != nil {
		return err
	}

	scrapedAt := time.Now().UTC().UnixMilli()
	for i := range articles {
		articles[i].ScrapedAt = scrapedAt
	}

	if err := st.UpsertArticles(ctx, articles); err != nil {
		return err
	}

	retentionThreshold := time.Now().UTC().Add(-cfg.Retention).UnixMilli()
	deleted, err := st.DeleteOlderThanPostedAt(ctx, retentionThreshold)
	if err != nil {
		return err
	}

	log.Printf(
		"scrape: ingested %d articles (mode=%s cache_path=%s html_bytes=%d deleted_old=%d)",
		len(articles),
		mode,
		cfg.CachePath,
		len(html),
		deleted,
	)
	return nil
}

// fetchAndWriteCache fetches the HTML content of a URL and writes it to a file.
func fetchAndWriteCache(ctx context.Context, pageURL, cachePath string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch world-news page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch world-news page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if err := writeCacheFile(cachePath, body); err != nil {
		return nil, fmt.Errorf("write cache file: %w", err)
	}

	return body, nil
}

// writeCacheFile writes data to path atomically by creating a temp file and renaming it.
func writeCacheFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".world-news.cache.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp cache file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp cache file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp cache file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("chmod temp cache file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp cache file: %w", err)
	}
	cleanup = false
	return nil
}
