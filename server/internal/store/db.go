package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// schema defines the SQLite tables used by the service.
const schema = `
CREATE TABLE IF NOT EXISTS articles (
	url TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	image_url TEXT,
	blurb TEXT,
	posted_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	scraped_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS kv (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// Store wraps the SQLite database connection.
type Store struct {
	conn *sql.DB
}

// sqliteDSN builds a modernc.org/sqlite connection string.
// WAL and busy_timeout must be set via repeated _pragma query keys, e.g.
// _pragma=journal_mode(WAL)&_pragma=busy_timeout(5000) — not bare _journal_mode / _busy_timeout.
// See: https://pkg.go.dev/modernc.org/sqlite (DSN query parameters).
func sqliteDSN(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve db path: %w", err)
	}
	abs = filepath.ToSlash(abs)
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	return "file:" + abs + "?" + q.Encode(), nil
}

// Open initializes a SQLite connection and ensures schema exists.
func Open(ctx context.Context, path string) (*Store, error) {
	dsn, err := sqliteDSN(path)
	if err != nil {
		return nil, err
	}

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	conn.SetMaxOpenConns(1)

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	if _, err := conn.ExecContext(ctx, schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Store{conn: conn}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.conn.Close()
}
