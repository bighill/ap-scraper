package store

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

const lastScrapeAtKey = "last_scrape_at"

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
