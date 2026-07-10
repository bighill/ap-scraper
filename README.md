# AP Scraper

Go service that scrapes AP world news articles from [apnews.com/world-news](https://apnews.com/world-news), normalizes metadata, stores them in local SQLite, and serves them over HTTP. A built-in scheduler runs a live scrape on a fixed interval and follows each successful listing scrape with a content-scrape pass.

## Behavior

- Source page: `https://apnews.com/world-news`
- Parses `div.PagePromo` promo cards and keeps article URLs matching `https://apnews.com/article/...`
- Captures per article (all ms epoch where applicable): `id`, `url`, `title`, `image_url`, `blurb`, `posted_at`, `updated_at`, `scraped_at`, `content`, `content_scraped_at`, `is_hidden`
- Deduplicates by canonical URL within each parse; upserts by `url` into SQLite
- Retention after each scrape: delete rows where `posted_at` is older than **2 days** (UTC)
- **Scheduler:** checks `kv.last_scrape_at` on startup and each tick; runs only when the last scrape is older than **2 hours**
  - After a successful listing scrape, it fetches article pages one-by-one (20s timeout, 2s delay) and stores the body text
  - Each article URL is content-scraped at most once; once `content_scraped_at` is set it is never revisited
- **HTTP:**
  - `GET /articles` returns stored articles as JSON summaries (visible only by default, newest `posted_at` first); pass `?hidden=1` to list hidden articles
  - `GET /articles?full=1` includes the `content` field for each article
  - `GET /articles/:id` returns a single article by database id, including `content`
  - `GET /articles/count`, `POST /articles/hide`, `POST /articles/unhide`
  - `GET /settings/images`, `POST /settings/images` — app-level image-visibility toggle used by the web UI

Configuration is **static** in [`server/internal/config/config.go`](server/internal/config/config.go) (paths, listen address, intervals). Environment variables can be added later without changing this layout.

## Layout

| Path | Role |
|------|------|
| `server/main.go` | Process entry: signal handling, open store, run scheduler + HTTP API (`golang.org/x/sync/errgroup`) |
| `server/internal/store` | SQLite only: DSN/pragmas, schema on open, queries |
| `server/internal/jobs` | `RunScrape`: fetch HTML, parse, upsert, retention; `RunContentScrape`: per-article body fetch |
| `server/internal/scheduler` | Periodic scrape (2-hour default), content pass after each listing scrape |
| `server/internal/api` | `http.Server`, graceful shutdown; `/articles` endpoints |
| `server/internal/parser` | HTML → `[]model.Article` and article page body text |
| `server/internal/model` | `Article` struct |
| `server/data` | Runtime SQLite DB |
| `web` | Static frontend served by the server (list view + inline article detail) |

There is **no** CLI binary and **no** versioned SQL migration directory; DDL lives next to `store.Open`.

## Run

```bash
go -C server run .
```

- Default listen address: `:9191` (see `server/internal/config/config.go`)
- Example: `curl -s http://localhost:9191/articles | head`
- Single article: `curl -s http://localhost:9191/articles/1`

## Paths and storage

Keep runtime data **inside this repo** (e.g. `server/data/`), not under `/tmp` or other paths outside the project.

- Database: `server/data/apnews.db` (SQLite WAL + `busy_timeout` via modernc DSN — see `server/internal/store/db.go`)
- Tables: `articles`, `kv`
- The `articles` table uses an autoincrement `id` column. If you have an older database created before this change, delete `server/data/apnews.db` and let the server recreate the schema.

## Development

- Tests: `go -C server test ./...` or `./bin/test.sh`
- **Unit tests** avoid touching SQLite and the filesystem: parser tests use inline HTML; handler tests use stubs; store tests cover DSN string construction only. Integration-style tests against a real DB are not required for routine changes.

## Helper scripts

- `bin/dev.sh` — run `air` hot-reload for `server/main.go`
- `bin/reload-docker-prod.sh` — rebuild Docker image, replace running prod container, mount `web/` and `server/data/`
- `bin/purge-data.sh` — delete `server/data/*.db*` runtime database files
- `bin/test.sh` — `go -C server test ./...`

## Docker (prod-style local run)

`./bin/reload-docker-prod.sh` builds the app image (`ap-scraper:prod`), stops/removes any existing `ap-scraper` container, and starts a fresh one with:

- Port mapping: `9191:9191`
- Volume mounts:
  - `./web -> /app/web`
  - `./server/data -> /app/server/data`

## Constraints

- Respect [robots.txt](https://apnews.com/robots.txt), rate limits, and AP terms of use.
- Intended for personal or otherwise permitted use.
