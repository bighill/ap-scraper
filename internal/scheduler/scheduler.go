package scheduler

import (
	"context"
	"log"
	"time"

	"ap-scraper/internal/jobs"
	"ap-scraper/internal/store"
)

// Scheduler runs periodic scrape jobs.
type Scheduler struct {
	store    *store.Store
	interval time.Duration
	scrape   jobs.ScrapeConfig
}

// New returns a scheduler that uses the given scrape settings.
func New(st *store.Store, interval time.Duration, scrape jobs.ScrapeConfig) *Scheduler {
	return &Scheduler{store: st, interval: interval, scrape: scrape}
}

// Run blocks until ctx is cancelled. It runs an initial scrape, then one per interval.
// Scrape errors are logged; they do not stop the scheduler.
func (s *Scheduler) Run(ctx context.Context) error {
	t := time.NewTicker(s.interval)
	defer t.Stop()

	run := func() {
		if err := jobs.RunScrape(ctx, s.store, s.scrape); err != nil {
			log.Printf("scheduler: scrape failed: %v", err)
		}
	}

	run()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			run()
		}
	}
}
