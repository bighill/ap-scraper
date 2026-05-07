package store

import (
	"context"
	"database/sql"
	"fmt"

	"ap-scraper/internal/model"
)

// QueryAll returns all articles ordered by posted time.
func (s *Store) QueryAll(ctx context.Context) ([]model.Article, error) {
	const q = `
SELECT url, title, image_url, blurb, posted_at, updated_at, scraped_at
FROM articles
ORDER BY posted_at DESC;
`

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
INSERT INTO articles (url, title, image_url, blurb, posted_at, updated_at, scraped_at)
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

func scanArticles(rows *sql.Rows) ([]model.Article, error) {
	articles := make([]model.Article, 0)
	for rows.Next() {
		var item model.Article
		if err := rows.Scan(
			&item.URL,
			&item.Title,
			&item.ImageURL,
			&item.Blurb,
			&item.PostedAt,
			&item.UpdatedAt,
			&item.ScrapedAt,
		); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}

		articles = append(articles, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate articles: %w", err)
	}

	return articles, nil
}
