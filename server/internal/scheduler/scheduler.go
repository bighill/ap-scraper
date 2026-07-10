package scheduler

import (
	"context"
	"log"
	"time"

	"ap-scraper/internal/jobs"
	"ap-scraper/internal/model"
	"ap-scraper/internal/store"
)

// scrapeStore is the subset of Store needed to drive the scheduler itself.
type scrapeStore interface {
	LastScrapeAt(context.Context) (time.Time, bool, error)
	SetLastScrapeAt(context.Context, time.Time) error
}

// contentStore is the subset of Store needed to run the content scrape pass.
type contentStore interface {
	scrapeStore
	QueryArticlesMissingContent(context.Context) ([]model.Article, error)
	UpdateArticleContent(ctx context.Context, id int64, content string, scrapedAt int64) error
}

// Scheduler runs periodic scrape jobs.
type Scheduler struct {
	store            contentStore
	interval         time.Duration
	scrape           jobs.ScrapeConfig
	content          jobs.ContentScrapeConfig
	nowFn            func() time.Time
	runScrape        func(context.Context, contentStore, jobs.ScrapeConfig) error
	runContentScrape func(context.Context, contentStore, jobs.ContentScrapeConfig) error
}

// New returns a scheduler that uses the given scrape settings.
func New(st contentStore, interval time.Duration, scrape jobs.ScrapeConfig, content jobs.ContentScrapeConfig) *Scheduler {
	return &Scheduler{
		store:    st,
		interval: interval,
		scrape:   scrape,
		content:  content,
		nowFn:    time.Now,
		runScrape: func(ctx context.Context, s contentStore, cfg jobs.ScrapeConfig) error {
			return jobs.RunScrape(ctx, s.(*store.Store), cfg)
		},
		runContentScrape: func(ctx context.Context, s contentStore, cfg jobs.ContentScrapeConfig) error {
			return jobs.RunContentScrape(ctx, s, cfg)
		},
	}
}

// Run blocks until ctx is cancelled. It checks if scrape is due on startup and each interval tick.
// Scrape errors are logged; they do not stop the scheduler.
func (s *Scheduler) Run(ctx context.Context) error {
	t := time.NewTicker(s.interval)
	defer t.Stop()

	s.maybeRun(ctx)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			s.maybeRun(ctx)
		}
	}
}

func (s *Scheduler) maybeRun(ctx context.Context) {
	lastScrapeAt, ok, err := s.store.LastScrapeAt(ctx)
	if err != nil {
		log.Printf("scheduler: read last scrape timestamp failed: %v", err)
		return
	}

	now := s.nowFn().UTC()
	if !shouldRunScrape(now, lastScrapeAt, ok, s.interval) {
		log.Printf("scheduler: skipping scrape; last_scrape_at=%s interval=%s", lastScrapeAt.Format(time.RFC3339), s.interval)
		return
	}

	if err := s.runScrape(ctx, s.store, s.scrape); err != nil {
		log.Printf("scheduler: scrape failed: %v", err)
		return
	}
	if err := s.store.SetLastScrapeAt(ctx, now); err != nil {
		log.Printf("scheduler: persist last scrape timestamp failed: %v", err)
		return
	}

	if err := s.runContentScrape(ctx, s.store, s.content); err != nil {
		log.Printf("scheduler: content scrape failed: %v", err)
	}
}

func shouldRunScrape(now, lastScrapeAt time.Time, hasLastScrape bool, interval time.Duration) bool {
	if !hasLastScrape {
		return true
	}
	return now.Sub(lastScrapeAt) > interval
}
