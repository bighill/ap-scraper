package scheduler

import (
	"testing"
	"time"

	"ap-scraper/internal/jobs"
)

func TestNew_nonNil(t *testing.T) {
	t.Parallel()

	s := New(nil, time.Minute, jobs.ScrapeConfig{})
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.interval != time.Minute {
		t.Fatalf("interval %v", s.interval)
	}
}
