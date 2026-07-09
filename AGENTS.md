# Agent notes

## Repository boundaries

Do not write scratch files, downloads, or outputs under `/tmp` or anywhere outside this repository. Keep fetches, build artifacts, and temporary data inside this project directory (for example under `server/data/`).

## Long-running processes

Do not start or leave running long-lived processes (for example `go -C server run .`, HTTP dev servers, watchers, or background jobs that keep listening). The user runs servers and daemons. Prefer bounded checks: `go -C server test ./...`, `go -C server build ./...`, or other commands that exit on their own.

NEVER run the actual application. If you expect the app to be running but it is not, inform the user instead of starting it yourself. Do not clean up by starting the app first—only stop processes you personally started, and always verify with `ps aux | grep ...` (or similar) that nothing is left behind.

## Git

When on the `main` branch, do not run `git commit` unless the user explicitly requests it. Other git commands (status, diff, log, branch, add, etc.) are fine.

## Project shape

- **Language / module:** Go 1.25+, module `ap-scraper`.
- **Entry point:** `server/main.go` — single long-running binary: HTTP API + background scheduler (`golang.org/x/sync/errgroup`).
- **No CLI:** The old `cmd/apnews` flow is gone; scraping is driven by the scheduler (and `server/internal/jobs`).

| Area | Package / path | Notes |
|------|----------------|--------|
| SQLite | `server/internal/store` | Only package with SQL. `Open` applies schema; no `migrations/` history. |
| Scraping | `server/internal/jobs` | Orchestrates fetch, `parser`, store upsert + retention. |
| Scheduler | `server/internal/scheduler` | Default interval 2 hours; config in `server/internal/config`. |
| HTTP | `server/internal/api`, `server/internal/api/handlers` | `GET /articles` returns all articles as JSON. |
| HTML parsing | `server/internal/parser` | No database access. |
| Types | `server/internal/model` | `Article` and JSON tags. |
| Static config | `server/internal/config` | Constants only for now (paths, addr, durations). |

## Testing expectations

- Prefer **unit tests that do not open SQLite or read the filesystem** (inline HTML in parser tests; `articleLister` stub for handlers; DSN string checks in `server/internal/store`).
- Do not add tests that require a real DB file unless explicitly requested.
- Run `go -C server test ./...` before finishing substantive changes.

## Docs

- **`readme.md`** — user-facing behavior, layout table, run instructions.

Keep agent work consistent with `readme.md`; avoid contradicting the “no migration ledger” and “static config until env vars are added” policies.
