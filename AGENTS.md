# Agent notes

## Repository boundaries

Do not write scratch files, downloads, or outputs under `/tmp` or anywhere outside this repository. Keep fetches, caches, build artifacts, and temporary data inside this project directory (for example under `data/`).

## Long-running processes

Do not start or leave running long-lived processes (for example `go run ./cmd/server`, HTTP dev servers, watchers, or background jobs that keep listening). The user runs servers and daemons. Prefer bounded checks: `go test ./...`, `go build`, or other commands that exit on their own.

## Project shape

- **Language / module:** Go 1.25+, module `ap-scraper`.
- **Entry point:** `cmd/server` — single long-running binary: HTTP API + background scheduler (`golang.org/x/sync/errgroup`).
- **No CLI:** The old `cmd/apnews` flow is gone; scraping is driven by the scheduler (and `internal/jobs`).

| Area | Package / path | Notes |
|------|----------------|--------|
| SQLite | `internal/store` | Only package with SQL. `Open` applies schema; no `migrations/` history — see `plan.md`. |
| Scraping | `internal/jobs` | Orchestrates fetch/cache, `parser`, store upsert + retention. |
| Scheduler | `internal/scheduler` | Default interval 77 minutes; config in `internal/config`. |
| HTTP | `internal/api`, `internal/api/handlers` | `GET /articles` returns all articles as JSON. |
| HTML parsing | `internal/parser` | No database access. |
| Types | `internal/model` | `Article` and JSON tags. |
| Static config | `internal/config` | Constants only for now (paths, addr, durations). |

## Testing expectations

- Prefer **unit tests that do not open SQLite or read the filesystem** (inline HTML in parser tests; `articleLister` stub for handlers; DSN string checks in `internal/store`).
- Do not add tests that require `data/world-news.cache.html` or a real DB file unless explicitly requested.
- Run `go test ./...` before finishing substantive changes.

## Docs

- **`readme.md`** — user-facing behavior, layout table, run instructions.
- **`plan.md`** — architecture decisions (API contract, schema policy, SQLite DSN details, naming).

Keep agent work consistent with both; avoid contradicting the “no migration ledger” and “static config until env vars are added” policies.
