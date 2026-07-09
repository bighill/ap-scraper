# AP Scraper — Security & Quality Audit

**Scope:** `server/` Go service, `web/` frontend, `bin/` helper scripts, `readme.md`, `agents.md`.
**Date:** 2026-07-08
**Commit reviewed:** current working tree (uncommitted changes: none except this report).

## Executive summary

The application is a small, well-structured Go service that scrapes AP world-news cards, stores them in SQLite, and serves them over HTTP. The code is generally idiomatic, uses parameterized SQL, and keeps a clean package split. Automated tests pass, including under the race detector.

The audit found **no critical vulnerabilities**. The main issues are documentation/config drift, a dependency-file inaccuracy, and several defense-in-depth hardening opportunities around HTTP fetching, static file serving, and container security. None of these are exploitable in the current single-user, local-use threat model, but they should be addressed before broader deployment.

## Methodology

- Manual review of all Go source, tests, scripts, Dockerfile, and docs.
- Ran `go -C server test ./...` and `go -C server test -race ./...`.
- Ran `go -C server vet ./...`.
- Attempted `staticcheck` (blocked because the installed Staticcheck binary was built with Go 1.24.2; module requires Go 1.25.0).
- Verified `https://apnews.com/robots.txt` allows the scraped paths (`/world-news`, `/article/...`) for generic user-agents.
- Did **not** run a live server or perform dynamic penetration testing.

## Findings

### 1. Documentation/config inconsistency: retention period

- **Severity:** Medium
- **Location:** `readme.md` vs. `server/internal/config/config.go`
- **Details:** `readme.md` states articles older than **5 days** are deleted, but `ArticleRetentionPeriod` in `config.go` is `2 * 24 * time.Hour` (2 days). The scheduler and retention logic both use the 2-day constant.
- **Impact:** Operators reading the README will expect a 5-day archive and may be surprised when articles disappear after 48 hours.
- **Recommendation:** Decide on the intended retention, then update either `config.go` or `readme.md` (and `plan.md` once it exists) so they agree.

### 2. `plan.md` is referenced but missing

- **Severity:** Low
- **Location:** `agents.md` references `plan.md`; file does not exist in the repo.
- **Impact:** Architecture/API/schema policy decisions are undocumented, and agent instructions point to a non-existent file.
- **Recommendation:** Create `plan.md` covering schema policy, API contract, SQLite DSN details, and naming conventions, per `agents.md`.

### 3. `go.mod` lists `gin-gonic/gin` as an indirect dependency

- **Severity:** Low
- **Location:** `server/go.mod`
- **Details:** `server/internal/api/server.go` directly imports `github.com/gin-gonic/gin`, but `go.mod` places it in the indirect `require` block. `go mod tidy` correctly promotes it to a direct dependency and updates the lock file.
- **Impact:** The build currently works because Go resolves the module anyway, but the dependency graph is misleading and can drift over time.
- **Recommendation:** Run `go -C server mod tidy` and commit the resulting `go.mod`/`go.sum` changes.

### 4. Scraped image URLs are not normalized or validated

- **Severity:** Medium
- **Location:** `server/internal/parser/worldnews.go` (`extractImageURL`)
- **Details:** The parser returns `src` or the first `srcset` URL verbatim. If AP ever serves a relative image path, the web UI will resolve it against `localhost:9191` and show a broken image. More importantly, no validation prevents a malicious or compromised source page from returning `javascript:...` or `data:...` image URLs, which can be used for XSS/reflected payload delivery through the UI.
- **Impact:** Broken images in normal operation; possible XSS if the scraped HTML or cache file is attacker-controlled.
- **Recommendation:**
  1. In `canonicalAPArticleURL` style, absolutize image URLs to `https://apnews.com` (or the relevant CDN host) when they are relative.
  2. Reject non-HTTP(S) image schemes before storing or before rendering.

### 5. HTTP fetch has no User-Agent, no body-size limit, and no redirect limit

- **Severity:** Low–Medium
- **Location:** `server/internal/jobs/scrape.go` (`fetchAndWriteCache`)
- **Details:**
  - The scraper does not set a `User-Agent`. AP's `robots.txt` allows the paths, but many sites block or rate-limit requests with no user-agent.
  - `io.ReadAll(resp.Body)` reads the entire response without a byte limit, so an unexpectedly large page could exhaust memory.
  - The default Go client follows redirects; a malicious or misconfigured server could chain redirects, though the default limit is 10.
- **Impact:** Fetch reliability and potential denial-of-service via memory exhaustion.
- **Recommendation:**
  - Set a descriptive `User-Agent` (e.g., `ap-scraper/1.0 (+contact info)`).
  - Wrap `resp.Body` with `http.MaxBytesReader` before `io.ReadAll`.
  - Consider configuring `CheckRedirect` to cap redirects and reject non-HTTPS destinations.

### 6. Cache file is written non-atomically

- **Severity:** Low
- **Location:** `server/internal/jobs/scrape.go`
- **Details:** `os.WriteFile` truncates and writes `world-news.cache.html` in place. If the process crashes mid-write, the cache is left partially written.
- **Impact:** On next `UseCache` run, the parser may fail on truncated HTML.
- **Recommendation:** Write to a temp file in the same directory and `os.Rename` it into place.

### 7. Dockerfile runs as root

- **Severity:** Low–Medium
- **Location:** `server/Dockerfile`
- **Details:** The final stage has no `USER` directive and executes `apnews-server` as root.
- **Impact:** Container escape or compromise grants host root privileges.
- **Recommendation:** Add an unprivileged user/group (`adduser -D -s /bin/false apnews`) and switch to it with `USER apnews` before `ENTRYPOINT`.

### 8. Static file serving uses constructed paths instead of `http.Dir`

- **Severity:** Low
- **Location:** `server/internal/api/server.go`
- **Details:** Routes `/`, `/css.css`, and `/js.js` use `filepath.Join(config.WebUIDir, filename)` with `c.File`. Today `WebUIDir` is a hardcoded constant, so path traversal is not exploitable. If it ever becomes configurable, the current pattern is unsafe.
- **Recommendation:** Serve the directory with `r.StaticFS("/", http.Dir(web))` or use Gin's `StaticFile`/`StaticFS`. Add a `fs.FS` restriction if the static directory should not expose arbitrary files.

### 9. `HideArticle` / `UnhideArticle` do not verify the row exists

- **Severity:** Low
- **Location:** `server/internal/store/articles.go`
- **Details:** Both endpoints return `204 No Content` even when the supplied URL does not match any article.
- **Impact:** The UI removes the item from the DOM and updates counts, which can become misleading if the URL is stale or malformed.
- **Recommendation:** Return `RowsAffected` from the UPDATE and return `404 Not Found` when zero rows were changed.

### 10. State-changing endpoints lack CSRF protection

- **Severity:** Low
- **Location:** `server/internal/api/handlers/articles.go`
- **Details:** `POST /articles/hide` and `POST /articles/unhide` accept JSON bodies but do not use CSRF tokens, `SameSite` cookies, or CORS restrictions.
- **Impact:** Risk is low because the app uses no authentication cookies and the endpoints return minimal data, but cross-site POSTs are still accepted.
- **Recommendation:** Add CORS middleware that restricts origins, or require a `Content-Type: application/json` preflight check (already required for JSON parsing but not enforced as a security control). Consider a simple CSRF token if authentication is added later.

### 11. Custom `parseInt64` does not guard against overflow

- **Severity:** Low
- **Location:** `server/internal/parser/worldnews.go`
- **Details:** The hand-rolled parser wraps on overflow for very long numeric strings.
- **Impact:** AP's timestamps are well within `int64` range, so this is not exploitable in practice, but it is less robust than `strconv.ParseInt`.
- **Recommendation:** Replace `parseInt64` with `strconv.ParseInt` (it will be inlined by the compiler anyway) and handle the error explicitly.

### 12. Schema migration relies on substring matching of the SQLite error message

- **Severity:** Low
- **Location:** `server/internal/store/db.go`
- **Details:** The `ALTER TABLE ADD COLUMN` error is ignored only if `strings.Contains(err.Error(), "duplicate column name")`.
- **Impact:** Any unrelated error whose text happens to contain that substring will be silently swallowed.
- **Recommendation:** Use `sqlite3.Error` type assertion (if the driver exposes it) or query `PRAGMA table_info(articles)` before attempting the migration.

### 13. SQLite WAL with `MaxOpenConns(1)` serializes all access

- **Severity:** Info
- **Location:** `server/internal/store/db.go`
- **Details:** WAL mode is enabled, but only one connection is allowed, so concurrent API and scheduler access is serialized.
- **Impact:** No functional bug for the expected load, but it defeats much of WAL's concurrency benefit.
- **Recommendation:** Increase `MaxOpenConns` (e.g., 10–25) and set `MaxIdleConns`/`ConnMaxLifetime` appropriately.

### 14. `CountArticles` performs three separate queries

- **Severity:** Info
- **Location:** `server/internal/store/articles.go`
- **Details:** Total, visible, and hidden counts are fetched in three round-trips.
- **Recommendation:** Collapse to a single query using `SUM(CASE WHEN is_hidden = 1 THEN 1 ELSE 0 END)` for hidden and `COUNT(*)` for total.

### 15. Typo and stale TODO

- **Severity:** Trivial / Low
- **Location:** `server/internal/jobs/scrape.go` comment `fetchAndndWriteCache`; `todo.md` line "remove cache feature"
- **Details:** Comment typo; `todo.md` asks to remove the cache feature, while `main.go` still configures `CachePath` and `jobs.ScrapeConfig.UseCache` exists.
- **Recommendation:** Fix the typo and either remove the cache feature or clarify the TODO.

## Positive observations

- **Clean SQL injection posture:** All SQL uses parameterized queries (`?` placeholders). No string concatenation of user input into SQL.
- **Good package boundaries:** Only `store` touches SQL; `parser` has no DB access; handlers depend on narrow interfaces, making tests easy to stub.
- **Graceful shutdown:** The HTTP server and scheduler both respect `context.Context` and shut down cleanly on `SIGINT`/`SIGTERM`.
- **Test discipline:** Parser and handler tests avoid SQLite/filesystem; store tests cover DSN string construction, consistent with `agents.md` guidance.
- **Race-clean:** `go test -race ./...` passes.
- **Schema includes `is_hidden`:** The schema already defines the column, so new databases do not need the compatibility `ALTER TABLE` path.

## Recommendations summary

| Priority | Action |
|----------|--------|
| High     | Align retention period in README/config. |
| High     | Normalize/validate scraped image URLs. |
| Medium   | Run `go mod tidy` and commit dependency fix. |
| Medium   | Harden HTTP fetch (User-Agent, MaxBytesReader, redirect cap). |
| Medium   | Run Docker container as non-root user. |
| Low      | Create missing `plan.md`. |
| Low      | Return 404 when hide/unhide affects no row. |
| Low      | Use `http.Dir` / `StaticFS` for static assets. |
| Low      | Replace custom `parseInt64` with `strconv.ParseInt`. |
| Low      | Harden schema migration check. |
| Info     | Increase `MaxOpenConns`; consolidate count query; atomic cache writes. |

## Conclusion

`ap-scraper` is a solid, minimal service with no critical security flaws. The most important fixes are aligning the documented retention period with the code, hardening the HTTP fetch path, and validating/normalizing image URLs. After those, the remaining items are defense-in-depth and maintainability improvements.
