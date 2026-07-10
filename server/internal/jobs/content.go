package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"ap-scraper/internal/model"
	"ap-scraper/internal/parser"
)

// contentStore is the store surface needed by RunContentScrape.
type contentStore interface {
	QueryArticlesMissingContent(context.Context) ([]model.Article, error)
	UpdateArticleContent(ctx context.Context, id int64, content string, scrapedAt int64) error
}

// ContentScrapeConfig controls a single content scrape pass.
type ContentScrapeConfig struct {
	FetchTimeout        time.Duration
	DelayBetweenFetches time.Duration
	Fetch               func(ctx context.Context, url string, timeout time.Duration) ([]byte, error)
}

// RunContentScrape fetches article pages for every article that has never had its
// content scraped, parses the body, and stores it. Once an article has content
// (empty or not), it is never fetched again.
//
// Errors for individual articles are logged; the pass continues with the rest.
func RunContentScrape(ctx context.Context, st contentStore, cfg ContentScrapeConfig) error {
	articles, err := st.QueryArticlesMissingContent(ctx)
	if err != nil {
		return fmt.Errorf("load articles missing content: %w", err)
	}

	fetch := cfg.Fetch
	if fetch == nil {
		fetch = FetchHTML
	}

	var fetched, succeeded int
	for _, article := range articles {
		content, err := fetchAndParse(ctx, fetch, article.URL, cfg.FetchTimeout)
		fetched++
		if err == nil && content != "" {
			succeeded++
		}

		scrapedAt := time.Now().UTC().UnixMilli()
		if err != nil {
			log.Printf("content scrape: article %d failed: %v", article.ID, err)
		}

		if uErr := st.UpdateArticleContent(ctx, article.ID, content, scrapedAt); uErr != nil {
			log.Printf("content scrape: update article %d failed: %v", article.ID, uErr)
		}

		if cfg.DelayBetweenFetches > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(cfg.DelayBetweenFetches):
			}
		}
	}

	log.Printf("content scrape: %d articles, %d fetched, %d with content", len(articles), fetched, succeeded)
	return nil
}

func fetchAndParse(ctx context.Context, fetch func(context.Context, string, time.Duration) ([]byte, error), pageURL string, timeout time.Duration) (string, error) {
	html, err := fetch(ctx, pageURL, timeout)
	if err != nil {
		return "", err
	}

	content, err := parser.ParseArticleHTML(html)
	if err != nil {
		return "", err
	}
	return content, nil
}
