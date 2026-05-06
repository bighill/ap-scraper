package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseWorldNewsHTML_FromSnapshot(t *testing.T) {
	t.Parallel()

	snapshotPath := filepath.Join("..", "..", "data", "world-news.cache.html")
	html, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read snapshot %s: %v", snapshotPath, err)
	}

	items, err := ParseWorldNewsHTML(html)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least 1 story, got 0")
	}

	seen := make(map[string]struct{}, len(items))
	for i, it := range items {
		if it.URL == "" {
			t.Fatalf("item[%d] url is empty", i)
		}
		if !strings.HasPrefix(it.URL, "https://apnews.com/article/") {
			t.Fatalf("item[%d] url not canonical article: %q", i, it.URL)
		}
		if it.Title == "" {
			t.Fatalf("item[%d] title is empty", i)
		}
		if it.PostedAt <= 0 {
			t.Fatalf("item[%d] posted_at invalid: %d", i, it.PostedAt)
		}
		if it.UpdatedAt <= 0 {
			t.Fatalf("item[%d] updated_at invalid: %d", i, it.UpdatedAt)
		}
		if it.ScrapedAt != 0 {
			t.Fatalf("item[%d] scraped_at should be 0 in parser output, got %d", i, it.ScrapedAt)
		}
		if _, ok := seen[it.URL]; ok {
			t.Fatalf("duplicate url in parse output: %q", it.URL)
		}
		seen[it.URL] = struct{}{}
	}
}

