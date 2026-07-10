package jobs

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"ap-scraper/internal/parser"
	"ap-scraper/internal/store"
)

// ScrapeConfig controls a single scrape run.
type ScrapeConfig struct {
	WorldNewsURL string
	FetchTimeout time.Duration
	Retention    time.Duration
}

// RunScrape fetches the world-news page, parses HTML, upserts articles, and applies retention.
func RunScrape(ctx context.Context, st *store.Store, cfg ScrapeConfig) error {
	html, err := FetchHTML(ctx, cfg.WorldNewsURL, cfg.FetchTimeout)
	if err != nil {
		return err
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
		"scrape: ingested %d articles (html_bytes=%d deleted_old=%d)",
		len(articles),
		len(html),
		deleted,
	)
	return nil
}

// FetchHTML fetches the HTML content of a URL and returns it.
func FetchHTML(ctx context.Context, pageURL string, timeout time.Duration) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return body, nil
}
