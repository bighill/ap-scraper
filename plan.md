# Architecture plan

This document records the design decisions and conventions for `ap-scraper`. Keep it in sync with the code and `readme.md`.

## Layout

- `server/main.go` is the only binary entry point.
- `server/internal/store` is the only package that contains SQL.
- `server/internal/parser` has no database or filesystem access.
- `server/internal/jobs` orchestrates fetch/cache, parse, store upsert, and retention.
- `server/internal/scheduler` runs periodic scrapes using `kv.last_scrape_at`.
- `server/internal/api` and `server/internal/api/handlers` serve HTTP.
- `server/internal/config` holds static constants only; environment variables are added later if needed.

## Schema policy

- SQLite is used with the `modernc.org/sqlite` driver.
- `store.Open` applies the schema on startup.
- There is **no** versioned migration directory/ledger.
- One-off compatibility changes (e.g., adding `is_hidden` to older databases) may be applied inside `store.Open`, guarded by checks that avoid failing when the column already exists.

## SQLite DSN details

- Use the `file:` scheme with absolute paths.
- WAL and `busy_timeout` are set via repeated `_pragma` query keys:
  - `_pragma=journal_mode(WAL)`
  - `_pragma=busy_timeout(5000)`
- See `server/internal/store/db.go` for the DSN builder.

## API contract

- `GET /articles` returns all non-hidden articles as JSON, newest `posted_at` first.
- `GET /articles?hidden=1` returns hidden articles.
- `GET /articles/count` returns `{ total, visible, hidden }`.
- `POST /articles/hide` and `POST /articles/unhide` accept `{ "url": "..." }` and return `204 No Content` on success or `404 Not Found` if the URL does not exist.
- Static frontend files are served from `../web` relative to the server working directory.

## Naming

- Package name matches directory name (`store`, `parser`, etc.).
- Exported identifiers use Go convention; narrow interfaces in handlers are named by behavior (`articleLister`, `articleHider`, `articleCounter`).

## Testing policy

- Prefer unit tests that do not open SQLite or read the filesystem.
- Parser tests use inline HTML.
- Handler tests use interface stubs.
- Store tests cover DSN construction.
- Integration tests requiring a real DB or cache file are added only when explicitly requested.
