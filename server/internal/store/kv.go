package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

const (
	lastScrapeAtKey = "last_scrape_at"
	showImagesKey   = "show_images"
)

// LastScrapeAt returns the last successful scrape time.
// The boolean is false when no timestamp has been recorded yet.
func (s *Store) LastScrapeAt(ctx context.Context) (time.Time, bool, error) {
	const q = `SELECT value FROM kv WHERE key = ?;`

	var raw string
	if err := s.conn.QueryRowContext(ctx, q, lastScrapeAtKey).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, false, nil
		}
		return time.Time{}, false, fmt.Errorf("query last_scrape_at: %w", err)
	}

	ms, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse last_scrape_at %q: %w", raw, err)
	}
	return time.UnixMilli(ms).UTC(), true, nil
}

// SetLastScrapeAt upserts the last successful scrape time in UTC millis.
func (s *Store) SetLastScrapeAt(ctx context.Context, ts time.Time) error {
	const q = `
INSERT INTO kv (key, value)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;
`
	_, err := s.conn.ExecContext(ctx, q, lastScrapeAtKey, strconv.FormatInt(ts.UTC().UnixMilli(), 10))
	if err != nil {
		return fmt.Errorf("upsert last_scrape_at: %w", err)
	}
	return nil
}

// ShowImages returns the app-level image visibility setting.
// It returns true when the key is missing, preserving the default behavior.
func (s *Store) ShowImages(ctx context.Context) (bool, error) {
	const q = `SELECT value FROM kv WHERE key = ?;`

	var raw string
	if err := s.conn.QueryRowContext(ctx, q, showImagesKey).Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, fmt.Errorf("query show_images: %w", err)
	}

	return raw == "1", nil
}

// SetShowImages upserts the app-level image visibility setting.
// The value is stored as "1" for true and "0" for false.
func (s *Store) SetShowImages(ctx context.Context, show bool) error {
	const q = `
INSERT INTO kv (key, value)
VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value;
`
	var raw string
	if show {
		raw = "1"
	} else {
		raw = "0"
	}
	_, err := s.conn.ExecContext(ctx, q, showImagesKey, raw)
	if err != nil {
		return fmt.Errorf("upsert show_images: %w", err)
	}
	return nil
}
