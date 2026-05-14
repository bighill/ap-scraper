# Plan: Hide Articles

Feature: mark individual articles as hidden, toggle between visible and hidden views, show hidden count.

## Branch
`feature/hide-articles`

## 1. Model

Add to `server/internal/model/article.go`:

```go
IsHidden bool `json:"is_hidden"`
```

## 2. Schema

Update `server/internal/store/db.go` schema `articles` table:

```sql
CREATE TABLE IF NOT EXISTS articles (
    url TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    image_url TEXT,
    blurb TEXT,
    posted_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    scraped_at INTEGER NOT NULL,
    is_hidden INTEGER NOT NULL DEFAULT 0
);
```

SQLite has no native boolean; use `INTEGER` (`0` = visible, `1` = hidden).  
`DEFAULT 0` makes existing rows visible on open. No migration ledger needed — the project applies schema on startup via `store.Open`.

## 3. Store methods (`server/internal/store/articles.go`)

### QueryVisible / QueryHidden
Replace `QueryAll` with two filtered queries, or change its signature to accept a `hidden bool` parameter.  
**Decision:** change `QueryAll(ctx context.Context, hidden bool)` so existing caller `AllArticles` is updated in one place, and test stubs change minimally.

```go
type articleLister interface {
    QueryAll(context.Context, bool) ([]model.Article, error)
}
```

- SQL for visible: `WHERE is_hidden = 0`  
- SQL for hidden: `WHERE is_hidden = 1`  
- Update `scanArticles` to scan `is_hidden` into `item.IsHidden`.

### HideArticle
```go
func (s *Store) HideArticle(ctx context.Context, url string) error
```
SQL: `UPDATE articles SET is_hidden = 1 WHERE url = ?`

### UnhideArticle
```go
func (s *Store) UnhideArticle(ctx context.Context, url string) error
```
SQL: `UPDATE articles SET is_hidden = 0 WHERE url = ?`

### CountHidden
```go
func (s *Store) CountHidden(ctx context.Context) (int, error)
```
SQL: `SELECT COUNT(*) FROM articles WHERE is_hidden = 1`

## 4. API (`server/internal/api/`)

### Endpoints

| Method | Path | Body | Response | Purpose |
|--------|------|------|----------|---------|
| GET | `/articles?hidden=1` | — | `[]Article` | list hidden articles |
| GET | `/articles` (no query) | — | `[]Article` | list visible articles (breaking change from "all", acceptable because only consumer is the built-in UI) |
| POST | `/articles/hide` | `{"url":"..."}` | `204 No Content` | hide one article |
| POST | `/articles/unhide` | `{"url":"..."}` | `204 No Content` | unhide one article |
| GET | `/articles/count` | — | `{"total":N,"visible":N,"hidden":N}` | counts for header badge |

**Rationale for filtering:** the frontend always needs either visible-only or hidden-only; returning both sets and filtering client-side wastes bytes and complicates the count UI.

### Handlers

Update `server/internal/api/handlers/articles.go`:

- `AllArticles` → rename/refactor to `ListArticles`.
- Read query param `hidden` (`r.URL.Query().Get("hidden")`). If `"1"` or `"true"`, pass `true` to `QueryAll`; else `false`.
- Add `HideArticle(st hidingStore) http.HandlerFunc`
- Add `UnhideArticle(st hidingStore) http.HandlerFunc`
- Add `ArticleCounts(st countingStore) http.HandlerFunc`

### Routes (`server/internal/api/server.go`)

Register with `gin`:

```go
r.GET("/articles", gin.WrapF(handlers.ListArticles(st)))
r.GET("/articles/count", gin.WrapF(handlers.ArticleCounts(st)))
r.POST("/articles/hide", gin.WrapF(handlers.HideArticle(st)))
r.POST("/articles/unhide", gin.WrapF(handlers.UnhideArticle(st)))
```

## 5. Frontend (`web/`)

### index.html
Add above `<ul id="list">`:

```html
<div id="controls" class="controls" hidden>
  <span id="count-badge">0 hidden</span>
  <button id="toggle-view" type="button">Show hidden</button>
</div>
```

### js.js
- Keep `statusEl`, add `controlsEl`, `countBadgeEl`, `toggleViewEl`.
- Track `viewMode` (`'visible'` or `'hidden'`).
- `load(mode)`:
  - fetch `/articles` or `/articles?hidden=1`
  - fetch `/articles/count` to update badge
- `render(articles, mode)`:
  - for each article, build the card as before
  - append a `<button class="hide-btn">Hide</button>` or `<button class="unhide-btn">Unhide</button>` depending on `mode`
  - wire click handler:
    - visible mode → `POST /articles/hide` → on `204`, remove `li` from DOM and reload counts
    - hidden mode → `POST /articles/unhide` → on `204`, remove `li` from DOM and reload counts
- `#toggle-view` click:
  - if currently visible → switch to hidden view, button text becomes "Show main"
  - else → switch to visible view, button text becomes "Show hidden"
  - call `load(currentMode)`

### css.css
Add minimal styles:

```css
.controls {
  display: flex;
  gap: 1rem;
  align-items: center;
  margin-bottom: 1rem;
}

.hide-btn, .unhide-btn {
  margin-top: 0.6rem;
  font-size: 0.85rem;
  cursor: pointer;
}
```

## 6. Upsert behaviour

When the scraper upserts an existing article that was previously hidden, the current `ON CONFLICT` update list **does not** touch `is_hidden`.  
Therefore a re-scraped article stays in whatever state the user set.  
This is intentional and desired — hide state is user-managed, not data-managed.

## 7. Testing approach

Follow repo policy: **unit tests that do not open SQLite or read the filesystem.**

- `server/internal/model`: no new test needed for a field addition unless JSON tag logic is tested.
- `server/internal/store`: DSN tests already exist; no new DSN changes. Skip integration tests against real DB.
- `server/internal/api/handlers/articles_test.go`:
  - Update `articleLister` stub to accept `bool` parameter.
  - Add stub tests for `ListArticles` with `hidden=true`.
  - Add stub tests for `HideArticle`, `UnhideArticle`, `ArticleCounts` using minimal in-memory stub store.
- `server/internal/parser`: unchanged.
- Run `go -C server test ./...` before finishing.

## 8. Rollout / backwards compatibility

- Only consumer is the single-page UI served from root; changing `GET /articles` from "all" to "visible-only" is safe for this project.
- Existing DB rows acquire `is_hidden = 0` automatically because of `DEFAULT 0`.
- No CLI or external API clients to migrate.

## Checklist

- [ ] `model.Article` gains `IsHidden bool`
- [ ] Schema adds `is_hidden INTEGER NOT NULL DEFAULT 0`
- [ ] `store.QueryAll` signature updated; `scanArticles` scans `is_hidden`
- [ ] `store.HideArticle`, `store.UnhideArticle`, `store.CountHidden` implemented
- [ ] Handler `ListArticles` reads `?hidden=` query; `HideArticle`, `UnhideArticle`, `ArticleCounts` added
- [ ] Routes wired in `api/server.go`
- [ ] `web/index.html` controls added
- [ ] `web/js.js` view switching, hide/unhide AJAX, DOM removal, count badge
- [ ] `web/css.css` minimal button/control styles
- [ ] Handler stubs updated for new interfaces
- [ ] `go -C server test ./...` passes
