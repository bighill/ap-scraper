package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"ap-scraper/internal/model"
)

// schema defines the SQLite table for world news stories.
const schema = `
CREATE TABLE IF NOT EXISTS world_news_stories (
	url TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	image_url TEXT,
	blurb TEXT,
	posted_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	scraped_at INTEGER NOT NULL
);
`

// Store wraps the SQLite database connection.
type Store struct {
	conn *sql.DB
}

// Open initializes a SQLite connection and ensures schema exists.
func Open(ctx context.Context, path string) (*Store, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

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

// QueryRecent returns recent stories ordered by posted time.
func (s *Store) QueryRecent(ctx context.Context, limit int) ([]model.Story, error) {
	// q selects recent rows with a caller-provided limit.
	const q = `
SELECT url, title, image_url, blurb, posted_at, updated_at, scraped_at
FROM world_news_stories
ORDER BY posted_at DESC
LIMIT ?;
`

	rows, err := s.conn.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent: %w", err)
	}
	defer rows.Close()

	return scanStories(rows)
}

// QueryAll returns all stories ordered by posted time.
func (s *Store) QueryAll(ctx context.Context) ([]model.Story, error) {
	// q selects all rows without a limit.
	const q = `
SELECT url, title, image_url, blurb, posted_at, updated_at, scraped_at
FROM world_news_stories
ORDER BY posted_at DESC;
`

	rows, err := s.conn.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query all: %w", err)
	}
	defer rows.Close()

	return scanStories(rows)
}

// UpsertStories inserts or updates stories by URL.
func (s *Store) UpsertStories(ctx context.Context, stories []model.Story) error {
	if len(stories) == 0 {
		return nil
	}

	// q writes all story fields and updates on URL conflicts.
	const q = `
INSERT INTO world_news_stories (url, title, image_url, blurb, posted_at, updated_at, scraped_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(url) DO UPDATE SET
	title = excluded.title,
	image_url = excluded.image_url,
	blurb = excluded.blurb,
	posted_at = excluded.posted_at,
	updated_at = excluded.updated_at,
	scraped_at = excluded.scraped_at;
`

	tx, err := s.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upsert tx: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("prepare upsert statement: %w", err)
	}
	defer stmt.Close()

	for _, story := range stories {
		if _, err := stmt.ExecContext(
			ctx,
			story.URL,
			story.Title,
			story.ImageURL,
			story.Blurb,
			story.PostedAt,
			story.UpdatedAt,
			story.ScrapedAt,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("upsert story %q: %w", story.URL, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert tx: %w", err)
	}
	return nil
}

// DeleteOlderThanPostedAt deletes stories with posted_at before threshold.
func (s *Store) DeleteOlderThanPostedAt(ctx context.Context, threshold int64) (int64, error) {
	const q = `
DELETE FROM world_news_stories
WHERE posted_at < ?;
`

	res, err := s.conn.ExecContext(ctx, q, threshold)
	if err != nil {
		return 0, fmt.Errorf("delete old stories: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted rows count: %w", err)
	}
	return rows, nil
}

// scanStories converts SQL rows into story models.
func scanStories(rows *sql.Rows) ([]model.Story, error) {
	stories := make([]model.Story, 0)
	for rows.Next() {
		var item model.Story
		if err := rows.Scan(
			&item.URL,
			&item.Title,
			&item.ImageURL,
			&item.Blurb,
			&item.PostedAt,
			&item.UpdatedAt,
			&item.ScrapedAt,
		); err != nil {
			return nil, fmt.Errorf("scan story: %w", err)
		}

		stories = append(stories, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stories: %w", err)
	}

	return stories, nil
}
