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
SELECT id, url, title, image_url, blurb, content, posted_at, updated_at, scraped_at, content_scraped_at, is_hidden
FROM articles
WHERE is_hidden = 1
ORDER BY posted_at DESC;
`
	} else {
		q = `
SELECT id, url, title, image_url, blurb, content, posted_at, updated_at, scraped_at, content_scraped_at, is_hidden
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

// QueryArticlesMissingContent returns articles whose content has never been fetched.
func (s *Store) QueryArticlesMissingContent(ctx context.Context) ([]model.Article, error) {
	const q = `
SELECT id, url, title, image_url, blurb, content, posted_at, updated_at, scraped_at, content_scraped_at, is_hidden
FROM articles
WHERE content IS NULL AND content_scraped_at IS NULL
ORDER BY posted_at DESC;
`
	rows, err := s.conn.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query missing content: %w", err)
	}
	defer rows.Close()

	return scanArticles(rows)
}

// QueryOne returns a single article by its database id.
func (s *Store) QueryOne(ctx context.Context, id int64) (model.Article, error) {
	const q = `
SELECT id, url, title, image_url, blurb, content, posted_at, updated_at, scraped_at, content_scraped_at, is_hidden
FROM articles
WHERE id = ?;
`
	rows, err := s.conn.QueryContext(ctx, q, id)
	if err != nil {
		return model.Article{}, fmt.Errorf("query one: %w", err)
	}
	defer rows.Close()

	items, err := scanArticles(rows)
	if err != nil {
		return model.Article{}, err
	}
	if len(items) == 0 {
		return model.Article{}, sql.ErrNoRows
	}
	return items[0], nil
}

// UpsertArticles inserts or updates articles by URL.
// Existing content and content_scraped_at values are never overwritten.
func (s *Store) UpsertArticles(ctx context.Context, articles []model.Article) error {
	if len(articles) == 0 {
		return nil
	}

	const q = `
INSERT INTO articles (url, title, image_url, blurb, content, posted_at, updated_at, scraped_at, content_scraped_at, is_hidden)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
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
			sql.NullString{String: article.Content, Valid: article.Content != ""},
			article.PostedAt,
			article.UpdatedAt,
			article.ScrapedAt,
			sql.NullInt64{Valid: false},
			0,
		); err != nil {
			return fmt.Errorf("upsert article %q: %w", article.URL, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit upsert tx: %w", err)
	}
	return nil
}

// UpdateArticleContent stores the parsed body for one article and marks it as content-scraped.
func (s *Store) UpdateArticleContent(ctx context.Context, id int64, content string, scrapedAt int64) error {
	const q = `
UPDATE articles
SET content = ?, content_scraped_at = ?
WHERE id = ?;
`
	_, err := s.conn.ExecContext(ctx, q, content, scrapedAt, id)
	if err != nil {
		return fmt.Errorf("update article content %d: %w", id, err)
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

// HideArticle marks an article as hidden and reports whether a row was changed.
func (s *Store) HideArticle(ctx context.Context, url string) (bool, error) {
	const q = `UPDATE articles SET is_hidden = 1 WHERE url = ?;`
	res, err := s.conn.ExecContext(ctx, q, url)
	if err != nil {
		return false, fmt.Errorf("hide article %q: %w", url, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read hide rows affected: %w", err)
	}
	return n > 0, nil
}

// UnhideArticle marks an article as visible and reports whether a row was changed.
func (s *Store) UnhideArticle(ctx context.Context, url string) (bool, error) {
	const q = `UPDATE articles SET is_hidden = 0 WHERE url = ?;`
	res, err := s.conn.ExecContext(ctx, q, url)
	if err != nil {
		return false, fmt.Errorf("unhide article %q: %w", url, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read unhide rows affected: %w", err)
	}
	return n > 0, nil
}

// CountArticles returns total, visible, and hidden counts.
func (s *Store) CountArticles(ctx context.Context) (total, visible, hidden int, err error) {
	const q = `
SELECT
	COUNT(*) AS total,
	SUM(CASE WHEN is_hidden = 0 THEN 1 ELSE 0 END) AS visible,
	SUM(CASE WHEN is_hidden = 1 THEN 1 ELSE 0 END) AS hidden
FROM articles;
`
	if err := s.conn.QueryRowContext(ctx, q).Scan(&total,
		&visible,
		&hidden,
	); err != nil {
		return 0, 0, 0, fmt.Errorf("count articles: %w", err)
	}
	return total, visible, hidden, nil
}

func scanArticles(rows *sql.Rows) ([]model.Article, error) {
	articles := make([]model.Article, 0)
	for rows.Next() {
		var item model.Article
		var hidden int64
		var imageURL, blurb, content sql.NullString
		var contentScrapedAt sql.NullInt64
		if err := rows.Scan(
			&item.ID,
			&item.URL,
			&item.Title,
			&imageURL,
			&blurb,
			&content,
			&item.PostedAt,
			&item.UpdatedAt,
			&item.ScrapedAt,
			&contentScrapedAt,
			&hidden,
		); err != nil {
			return nil, fmt.Errorf("scan article: %w", err)
		}
		item.ImageURL = imageURL.String
		item.Blurb = blurb.String
		item.Content = content.String
		item.IsHidden = hidden != 0
		if contentScrapedAt.Valid {
			item.ContentScrapedAt = contentScrapedAt.Int64
		}
		articles = append(articles, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate articles: %w", err)
	}

	return articles, nil
}
