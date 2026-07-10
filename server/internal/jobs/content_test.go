package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"ap-scraper/internal/model"
)

type stubContentStore struct {
	missing []model.Article
	updated []contentUpdate
	err     error
}

type contentUpdate struct {
	id        int64
	content   string
	scrapedAt int64
}

func (s *stubContentStore) QueryArticlesMissingContent(ctx context.Context) ([]model.Article, error) {
	return s.missing, s.err
}

func (s *stubContentStore) UpdateArticleContent(ctx context.Context, id int64, content string, scrapedAt int64) error {
	s.updated = append(s.updated, contentUpdate{id: id, content: content, scrapedAt: scrapedAt})
	return nil
}

func TestRunContentScrape_success(t *testing.T) {
	t.Parallel()

	store := &stubContentStore{
		missing: []model.Article{
			{ID: 1, URL: "https://apnews.com/article/one"},
			{ID: 2, URL: "https://apnews.com/article/two"},
		},
	}

	cfg := ContentScrapeConfig{
		FetchTimeout:        time.Second,
		DelayBetweenFetches: 0,
		Fetch: func(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
			return []byte(fmt.Sprintf(`<html><article><p>Body of %s</p></article></html>`, url)), nil
		},
	}

	if err := RunContentScrape(context.Background(), store, cfg); err != nil {
		t.Fatal(err)
	}

	if len(store.updated) != 2 {
		t.Fatalf("updated %d articles, want 2", len(store.updated))
	}
	if store.updated[0].id != 1 || !strings.Contains(store.updated[0].content, "Body of https://apnews.com/article/one") {
		t.Fatalf("first update bad: %+v", store.updated[0])
	}
	if store.updated[1].id != 2 || !strings.Contains(store.updated[1].content, "Body of https://apnews.com/article/two") {
		t.Fatalf("second update bad: %+v", store.updated[1])
	}
}

func TestRunContentScrape_emptyQueue(t *testing.T) {
	t.Parallel()

	store := &stubContentStore{missing: []model.Article{}}
	called := false
	cfg := ContentScrapeConfig{
		Fetch: func(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
			called = true
			return nil, errors.New("should not be called")
		},
	}

	if err := RunContentScrape(context.Background(), store, cfg); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("fetch should not be called with empty queue")
	}
}

func TestRunContentScrape_fetchErrorMarksScraped(t *testing.T) {
	t.Parallel()

	store := &stubContentStore{
		missing: []model.Article{{ID: 7, URL: "https://apnews.com/article/bad"}},
	}
	cfg := ContentScrapeConfig{
		FetchTimeout: time.Second,
		Fetch: func(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
			return nil, errors.New("network down")
		},
	}

	if err := RunContentScrape(context.Background(), store, cfg); err != nil {
		t.Fatal(err)
	}

	if len(store.updated) != 1 {
		t.Fatalf("updated %d articles, want 1", len(store.updated))
	}
	if store.updated[0].content != "" {
		t.Fatalf("expected empty content on error, got %q", store.updated[0].content)
	}
	if store.updated[0].scrapedAt == 0 {
		t.Fatal("expected scraped_at to be set")
	}
}

func TestRunContentScrape_queryError(t *testing.T) {
	t.Parallel()

	store := &stubContentStore{err: errors.New("db fail")}
	cfg := ContentScrapeConfig{}

	err := RunContentScrape(context.Background(), store, cfg)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunContentScrape_parseReturnsEmpty(t *testing.T) {
	t.Parallel()

	store := &stubContentStore{
		missing: []model.Article{{ID: 3, URL: "https://apnews.com/article/empty"}},
	}
	cfg := ContentScrapeConfig{
		Fetch: func(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
			return []byte(`<html><body><div class="other">No article body</div></body></html>`), nil
		},
	}

	if err := RunContentScrape(context.Background(), store, cfg); err != nil {
		t.Fatal(err)
	}

	if len(store.updated) != 1 || store.updated[0].content != "" || store.updated[0].scrapedAt == 0 {
		t.Fatalf("unexpected update: %+v", store.updated[0])
	}
}
