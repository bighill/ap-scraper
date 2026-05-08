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
	if s.runScrape == nil {
		t.Fatal("runScrape not initialized")
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
