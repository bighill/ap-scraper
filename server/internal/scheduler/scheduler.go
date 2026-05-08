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
	store     *store.Store
	interval  time.Duration
	scrape    jobs.ScrapeConfig
	nowFn     func() time.Time
	runScrape func(context.Context, *store.Store, jobs.ScrapeConfig) error
}

// New returns a scheduler that uses the given scrape settings.
func New(st *store.Store, interval time.Duration, scrape jobs.ScrapeConfig) *Scheduler {
	return &Scheduler{
		store:     st,
		interval:  interval,
		scrape:    scrape,
		nowFn:     time.Now,
		runScrape: jobs.RunScrape,
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
	}
}

func shouldRunScrape(now, lastScrapeAt time.Time, hasLastScrape bool, interval time.Duration) bool {
	if !hasLastScrape {
		return true
	}
	return now.Sub(lastScrapeAt) > interval
}
