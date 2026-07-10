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
	if WorldNewsURL == "" {
		t.Fatal("WorldNewsURL empty")
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
	if ContentFetchTimeout <= 0 {
		t.Fatalf("ContentFetchTimeout %v", ContentFetchTimeout)
	}
	if ContentFetchDelay <= 0 {
		t.Fatalf("ContentFetchDelay %v", ContentFetchDelay)
	}
	if ArticleRetentionPeriod <= 0 {
		t.Fatalf("ArticleRetentionPeriod %v", ArticleRetentionPeriod)
	}
	// Documented default: 2 hours.
	if ScrapeInterval != 2*time.Hour {
		t.Fatalf("ScrapeInterval = %v, want 2h", ScrapeInterval)
	}
}
