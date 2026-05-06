package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"ap-scraper/internal/config"
	"ap-scraper/internal/db"
	"ap-scraper/internal/parser"
)

// main runs the CLI and exits on error.
func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run dispatches top-level CLI commands.
func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usageError("missing command")
	}

	switch args[0] {
	case "scrape":
		return runScrape(ctx, args[1:])
	case "query":
		return runQuery(ctx, args[1:])
	default:
		return usageError(fmt.Sprintf("unknown command %q", args[0]))
	}
}

// runScrape handles the scrape command flags and setup.
func runScrape(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("scrape", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	useCache := fs.Bool("use-cache", false, "parse cached HTML instead of refreshing cache from live fetch")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return usageError("scrape does not accept positional arguments")
	}
	if err := ensureDBPath(config.DBPath); err != nil {
		return err
	}
	if err := ensureDBPath(config.CachePath); err != nil {
		return err
	}

	mode := "refresh-cache"
	var html []byte
	if *useCache {
		mode = "use-cache"
		cached, err := os.ReadFile(config.CachePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("--use-cache requested but cache file is missing: %s", config.CachePath)
			}
			return fmt.Errorf("read cache file: %w", err)
		}
		html = cached
	} else {
		fetched, err := fetchAndWriteCache(ctx, config.WorldNewsURL, config.CachePath)
		if err != nil {
			return err
		}
		html = fetched
	}

	store, err := db.Open(ctx, config.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()

	stories, err := parser.ParseWorldNewsHTML(html)
	if err != nil {
		return err
	}

	scrapedAt := time.Now().UTC().UnixMilli()
	for i := range stories {
		stories[i].ScrapedAt = scrapedAt
	}

	if err := store.UpsertStories(ctx, stories); err != nil {
		return err
	}

	retentionThreshold := time.Now().UTC().Add(-5 * 24 * time.Hour).UnixMilli()
	deleted, err := store.DeleteOlderThanPostedAt(ctx, retentionThreshold)
	if err != nil {
		return err
	}

	fmt.Printf(
		"scrape ingested %d stories (mode=%s cache_path=%s html_bytes=%d deleted_old=%d)\n",
		len(stories),
		mode,
		config.CachePath,
		len(html),
		deleted,
	)
	return nil
}

// runQuery handles query and query-all command behavior.
func runQuery(ctx context.Context, args []string) error {
	isAll := false
	if len(args) > 0 && args[0] == "all" {
		isAll = true
		args = args[1:]
	}

	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	limit := fs.Int("limit", 25, "number of rows to return")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return usageError("unexpected positional arguments for query")
	}
	if !isAll && *limit <= 0 {
		return usageError("--limit must be > 0")
	}
	if err := ensureDBPath(config.DBPath); err != nil {
		return err
	}

	store, err := db.Open(ctx, config.DBPath)
	if err != nil {
		return err
	}
	defer store.Close()

	var data any
	if isAll {
		items, err := store.QueryAll(ctx)
		if err != nil {
			return err
		}
		data = items
	} else {
		items, err := store.QueryRecent(ctx, *limit)
		if err != nil {
			return err
		}
		data = items
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// usageError returns a formatted usage error message.
func usageError(msg string) error {
	usage := `usage:
  apnews scrape [--use-cache]
  apnews query [--limit N]
  apnews query all
`
	return errors.New(msg + "\n" + usage)
}

// ensureDBPath creates the parent directory for the database file.
func ensureDBPath(dbPath string) error {
	parent := filepath.Dir(dbPath)
	if parent == "." || parent == "" {
		return nil
	}

	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}

	return nil
}

// fetchAndWriteCache downloads HTML and writes it to cache path.
func fetchAndWriteCache(ctx context.Context, url string, cachePath string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch world-news page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch world-news page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if err := os.WriteFile(cachePath, body, 0o644); err != nil {
		return nil, fmt.Errorf("write cache file: %w", err)
	}

	return body, nil
}
