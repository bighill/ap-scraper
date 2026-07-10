package scheduler

import (
	"context"
	"errors"
	"testing"
	"time"

	"ap-scraper/internal/jobs"
	"ap-scraper/internal/model"
)

type fakeStore struct {
	lastScrapeAt    time.Time
	hasLastScrapeAt bool
	setCalled       bool
	missing         []model.Article
}

func (f *fakeStore) LastScrapeAt(ctx context.Context) (time.Time, bool, error) {
	return f.lastScrapeAt, f.hasLastScrapeAt, nil
}

func (f *fakeStore) SetLastScrapeAt(ctx context.Context, t time.Time) error {
	f.setCalled = true
	f.lastScrapeAt = t
	f.hasLastScrapeAt = true
	return nil
}

func (f *fakeStore) QueryArticlesMissingContent(ctx context.Context) ([]model.Article, error) {
	return f.missing, nil
}

func (f *fakeStore) UpdateArticleContent(ctx context.Context, id int64, content string, scrapedAt int64) error {
	return nil
}

func TestNew_nonNil(t *testing.T) {
	t.Parallel()

	s := New(nil, time.Minute, jobs.ScrapeConfig{}, jobs.ContentScrapeConfig{})
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.interval != time.Minute {
		t.Fatalf("interval %v", s.interval)
	}
	if s.runScrape == nil {
		t.Fatal("runScrape not initialized")
	}
	if s.runContentScrape == nil {
		t.Fatal("runContentScrape not initialized")
	}
	if s.nowFn == nil {
		t.Fatal("nowFn not initialized")
	}
}

func TestShouldRunScrape(t *testing.T) {
	t.Parallel()

	now := time.UnixMilli(1_000_000).UTC()
	interval := 10 * time.Minute

	tests := []struct {
		name     string
		last     time.Time
		hasLast  bool
		expected bool
	}{
		{
			name:     "missing last scrape runs",
			hasLast:  false,
			expected: true,
		},
		{
			name:     "exactly at interval skips",
			last:     now.Add(-interval),
			hasLast:  true,
			expected: false,
		},
		{
			name:     "within interval skips",
			last:     now.Add(-interval + time.Second),
			hasLast:  true,
			expected: false,
		},
		{
			name:     "after interval runs",
			last:     now.Add(-interval - time.Second),
			hasLast:  true,
			expected: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldRunScrape(now, tc.last, tc.hasLast, interval)
			if got != tc.expected {
				t.Fatalf("shouldRunScrape() = %v, expected %v", got, tc.expected)
			}
		})
	}
}

func TestScheduler_runsContentScrapeAfterScrape(t *testing.T) {
	t.Parallel()

	fs := &fakeStore{}
	scrapeCalled := false
	contentCalled := false

	s := New(fs, time.Hour, jobs.ScrapeConfig{}, jobs.ContentScrapeConfig{})
	s.nowFn = func() time.Time { return time.UnixMilli(1_000_000).UTC() }
	s.runScrape = func(ctx context.Context, st contentStore, cfg jobs.ScrapeConfig) error {
		scrapeCalled = true
		return nil
	}
	s.runContentScrape = func(ctx context.Context, st contentStore, cfg jobs.ContentScrapeConfig) error {
		contentCalled = true
		return nil
	}

	s.maybeRun(context.Background())

	if !scrapeCalled {
		t.Fatal("listing scrape was not called")
	}
	if !contentCalled {
		t.Fatal("content scrape was not called after listing scrape")
	}
	if !fs.setCalled {
		t.Fatal("SetLastScrapeAt was not called")
	}
}

func TestScheduler_contentScrapeSkippedOnListingError(t *testing.T) {
	t.Parallel()

	fs := &fakeStore{}
	contentCalled := false

	s := New(fs, time.Hour, jobs.ScrapeConfig{}, jobs.ContentScrapeConfig{})
	s.nowFn = func() time.Time { return time.UnixMilli(1_000_000).UTC() }
	s.runScrape = func(ctx context.Context, st contentStore, cfg jobs.ScrapeConfig) error {
		return errors.New("scrape failed")
	}
	s.runContentScrape = func(ctx context.Context, st contentStore, cfg jobs.ContentScrapeConfig) error {
		contentCalled = true
		return nil
	}

	s.maybeRun(context.Background())

	if contentCalled {
		t.Fatal("content scrape should not run after failed listing scrape")
	}
}
