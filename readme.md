# AP Scraper

Go service that scrapes AP world news articles from [apnews.com/world-news](https://apnews.com/world-news), normalizes metadata, stores them in local SQLite, and serves them over HTTP. A built-in scheduler runs a live scrape on a fixed interval.

## Behavior

- Source page: `https://apnews.com/world-news`
- Parses `div.PagePromo` promo cards and keeps article URLs matching `https://apnews.com/article/...`
- Captures per article: `url`, `title`, `image_url`, `blurb`, `posted_at`, `updated_at`, `scraped_at` (ms epoch)
- Deduplicates by canonical URL within each parse; upserts by `url` into SQLite
- Retention after each scrape: delete rows where `posted_at` is older than **2 days** (UTC)
- **Scheduler:** checks `kv.last_scrape_at` on startup and each tick; runs only when the last scrape is older than **2 hours**
- **HTTP:** `GET /articles` returns **all** stored articles as JSON (newest `posted_at` first). No pagination or limit query parameter.

Configuration is **static** in [`server/internal/config/config.go`](server/internal/config/config.go) (paths, listen address, intervals). Environment variables can be added later without changing this layout.

## Layout

| Path | Role |
|------|------|
| `server/main.go` | Process entry: signal handling, open store, run scheduler + HTTP API (`golang.org/x/sync/errgroup`) |
| `server/internal/store` | SQLite only: DSN/pragmas, schema on open, queries |
| `server/internal/jobs` | `RunScrape`: fetch HTML, parse, upsert, retention (no SQL here) |
| `server/internal/scheduler` | Periodic scrape (2-hour default) |
| `server/internal/api` | `http.Server`, graceful shutdown; `GET /articles` |
| `server/internal/parser` | HTML → `[]model.Article` |
| `server/internal/model` | `Article` struct |
| `server/data` | Runtime SQLite DB |
| `web` | Static frontend served by the server |

There is **no** CLI binary and **no** versioned SQL migration directory; DDL lives next to `store.Open`.

## Run

```bash
go -C server run .
```

- Default listen address: `:9191` (see `server/internal/config/config.go`)
- Example: `curl -s http://localhost:9191/articles | head`

## Paths and storage

Keep runtime data **inside this repo** (e.g. `server/data/`), not under `/tmp` or other paths outside the project.

- Database: `server/data/apnews.db` (SQLite WAL + `busy_timeout` via modernc DSN — see `server/internal/store/db.go`)
- Tables: `articles`, `kv`

## Development

- Tests: `go -C server test ./...` or `./bin/test.sh`
- **Unit tests** avoid touching SQLite and the filesystem: parser tests use inline HTML; handler tests use stubs; store tests cover DSN string construction only. Integration-style tests against a real DB are not required for routine changes.

## Helper scripts

- `bin/dev.sh` — run `air` hot-reload for `server/main.go`
- `bin/reload-docker-prod.sh` — rebuild Docker image, replace running prod container, mount `web/` and `server/data/`
- `bin/test.sh` — `go -C server test ./...`

## Docker (prod-style local run)

`./bin/reload-docker-prod.sh` builds the app image, stops/removes any existing `ap-scraper-prod` container, and starts a fresh one with:

- Port mapping: `9191:9191`
- Volume mounts:
  - `./web -> /app/web`
  - `./server/data -> /app/server/data`

## Constraints

- Respect [robots.txt](https://apnews.com/robots.txt), rate limits, and AP terms of use.
- Intended for personal or otherwise permitted use.
