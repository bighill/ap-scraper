package config

import (
	"strings"
	"testing"
	"time"
)

func TestConfig_pathsNonEmpty(t *testing.T) {
	t.Parallel()

	if DBPath == "" {
		t.Fatal("DBPath empty")
	}
	if CachePath == "" {
		t.Fatal("CachePath empty")
	}
}

func TestConfig_worldNewsURL(t *testing.T) {
	t.Parallel()

	if !strings.HasPrefix(WorldNewsURL, "https://") {
		t.Fatalf("WorldNewsURL should be https: %q", WorldNewsURL)
	}
	if !strings.Contains(WorldNewsURL, "apnews.com") {
		t.Fatalf("unexpected WorldNewsURL: %q", WorldNewsURL)
	}
}

func TestConfig_durationsPositive(t *testing.T) {
	t.Parallel()

	if ScrapeInterval <= 0 {
		t.Fatalf("ScrapeInterval %v", ScrapeInterval)
	}
	if FetchTimeout <= 0 {
		t.Fatalf("FetchTimeout %v", FetchTimeout)
	}
	if ArticleRetentionPeriod <= 0 {
		t.Fatalf("ArticleRetentionPeriod %v", ArticleRetentionPeriod)
	}
	// Documented default: 77 minutes.
	if ScrapeInterval != 77*time.Minute {
		t.Fatalf("ScrapeInterval = %v, want 77m", ScrapeInterval)
	}
}
