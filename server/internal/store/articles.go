package store

import (
	"context"
	"database/sql"
	"fmt"

	"ap-scraper/internal/model"
)

// QueryAll returns articles ordered by posted time, optionally filtered by hidden status.
func (s *Store) QueryAll(ctx context.Context, hidden bool) ([]model.Article, error) {
	var q string
	if hidden {
		q = `
SELECT url, title, image_url, blurb, posted_at, updated_at, scraped_at, is_hidden
FROM articles
WHERE is_hidden = 1
ORDER BY posted_at DESC;
`
	} else {
		q = `
SELECT url, title, image_url, blurb, posted_at, updated_at, scraped_at, is_hidden
FROM articles
WHERE is_hidden = 0
ORDER BY posted_at DESC;
`
	}

	rows, err := s.conn.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query all: %w", err)
	}
	defer rows.Close()

	return scanArticles(rows)
}

// UpsertArticles inserts or updates articles by URL.
func (s *Store) UpsertArticles(ctx context.Context, articles []model.Article) error {
	if len(articles) == 0 {
		return nil
	}

	const q = `
INSERT INTO articles (url, title, image_url, blurb, posted_at, updated_at, scraped_at, is_hidden)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
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

	for _, article := range articles {
		if _, err := stmt.ExecContext(
			ctx,
			article.URL,
			article.Title,
			article.ImageURL,
			article.Blurb,
			article.PostedAt,
			article.UpdatedAt,
			article.ScrapedAt,
			0,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("upsert article %q: %w", article.URL, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert tx: %w", err)
	}
	return nil
}

// DeleteOlderThanPostedAt deletes articles with posted_at before threshold.
func (s *Store) DeleteOlderThanPostedAt(ctx context.Context, threshold int64) (int64, error) {
	const q = `
DELETE FROM articles
WHERE posted_at < ?;
`

	res, err := s.conn.ExecContext(ctx, q, threshold)
	if err != nil {
		return 0, fmt.Errorf("delete old articles: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read deleted rows count: %w", err)
	}
	return rows, nil
}

// HideArticle marks an article as hidden.
func (s *Store) HideArticle(ctx context.Context, url string) error {
	const q = `UPDATE articles SET is_hidden = 1 WHERE url = ?;`
	_, err := s.conn.ExecContext(ctx, q, url)
	if err != nil {
		return fmt.Errorf("hide article %q: %w", url, err)
	}
	return nil
}

// UnhideArticle marks an article as visible.
func (s *Store) UnhideArticle(ctx context.Context, url string) error {
	const q = `UPDATE articles SET is_hidden = 0 WHERE url = ?;`
	_, err := s.conn.ExecContext(ctx, q, url)
	if err != nil {
		return fmt.Errorf("unhide article %q: %w", url, err)
	}
	return nil
}

// CountArticles returns total, visible, and hidden counts.
func (s *Store) CountArticles(ctx context.Context) (total, visible, hidden int, err error) {
	if err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles;`).Scan(&total); err != nil {
		return 0, 0, 0, fmt.Errorf("count total articles: %w", err)
	}
	if err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles WHERE is_hidden = 0;`).Scan(&visible); err != nil {
		return 0, 0, 0, fmt.Errorf("count visible articles: %w", err)
	}
	if err := s.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM articles WHERE is_hidden = 1;`).Scan(&hidden); err != nil {
		return 0, 0, 0, fmt.Errorf("count hidden articles: %w", err)
	}
	return total, visible, hidden, nil
}

func scanArticles(rows *sql.Rows) ([]model.Article, error) {
	articles := make([]model.Article, 0)
	for rows.Next() {
		var item model.Article
		var hidden int64
		if err := rows.Scan(
			&item.URL,
			&item.Title,
			&item.ImageURL,
			&item.Blurb,
			&item.PostedAt,
			&item.UpdatedAt,
			&item.ScrapedAt,
			&hidden,
		); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		item.IsHidden = hidden != 0
		articles = append(articles, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate articles: %w", err)
	}

	return articles, nil
}
