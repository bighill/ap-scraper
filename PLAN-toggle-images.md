# Plan: Toggle Images

## Goal

Add an app-level **"show images"** boolean that is persisted server-side in the existing `kv` table and immediately visible to every connected client. When images are disabled, the web UI hides article thumbnails without altering the stored article rows.

## Defaults

- **Default value:** `true` (images shown). This keeps current behavior for fresh databases and avoids surprising users on first load.
- The key is absent from `kv` until a user explicitly toggles it; the default applies when the key is missing.

## Server-side storage

Re-use the existing `kv` table (`key TEXT PRIMARY KEY, value TEXT NOT NULL`).

- Key name: `show_images`
- Stored value: `"1"` for true, `"0"` for false.

Add store methods alongside the existing `LastScrapeAt` helpers in `server/internal/store/kv.go`:

- `ShowImages(ctx context.Context) (bool, error)` — returns `true` when the key is missing.
- `SetShowImages(ctx context.Context, show bool) error` — upserts `"1"` / `"0"`.

No schema change is required.

## HTTP API

Add two routes in `server/internal/api/server.go`:

- `GET /settings/images`
  - Returns `200 OK` with JSON body: `{ "show_images": <bool> }`.
  - Always reflects the current server-side value (or default if absent).
- `POST /settings/images`
  - Accepts JSON body: `{ "show_images": <bool> }`.
  - Persists the value and returns `204 No Content` on success.
  - Returns `400 Bad Request` for malformed JSON or non-boolean values.

Implement handlers in `server/internal/api/handlers/` using small store interfaces for testability, matching the existing `articleLister` / `articleHider` / `articleCounter` pattern.

Suggested interface:

```go
type imageSettingSetter interface {
    SetShowImages(context.Context, bool) error
}
type imageSettingGetter interface {
    ShowImages(context.Context) (bool, error)
}
```

## Web client

Changes in `web/index.html` and `web/js.js`:

1. Add a toggle control in the controls bar, e.g. a checkbox labeled **"Images"** or a button **"Hide images / Show images"**.
2. On page load, fetch `/settings/images` and set the control state to match `show_images`.
3. Apply the setting to rendering:
   - If `show_images` is `false`, skip creating `<img class="article-thumb">` elements entirely (do not just hide with CSS).
4. When the user toggles the control, `POST /settings/images` with the new value.
   - On success, keep the new state and re-render the current article list.
   - On failure, revert the control to the previous state and show a transient error in the status area.

The setting is global: one client’s change is seen by all clients on their next load/render.

## Testing

Keep tests unit-style and avoid touching SQLite or the filesystem where possible.

- **Store:** Add tests that verify the SQL strings and default behavior by using a small `*sql.DB` opened with `:memory:` only if the project already does so. Given the current rule of avoiding real DB tests, prefer a thin in-memory stub or verify the constant key name and string normalization via a small exported helper. If an in-memory test is acceptable for KV helpers, keep it isolated to `kv_test.go` and do not add a real file-based DB dependency.
- **Handlers:** Use stub implementations of `imageSettingGetter` / `imageSettingSetter`, similar to `articles_test.go`.
  - Test `GET /settings/images` returns the expected JSON.
  - Test `POST /settings/images` with valid and invalid bodies.
- **Server:** No new tests required unless route registration is verified; existing pattern is minimal.
- **Frontend:** Manual verification is sufficient for the first pass; add a small DOM-based test only if the project later adopts a JS test runner.

Run `go -C server test ./...` before finishing.

## Files to modify

| File | Change |
|------|--------|
| `server/internal/store/kv.go` | Add `ShowImages` / `SetShowImages` methods |
| `server/internal/api/handlers/` | Add `settings.go` with `GetShowImages` and `SetShowImages` handlers |
| `server/internal/api/server.go` | Register `GET /settings/images` and `POST /settings/images` |
| `web/index.html` | Add the images toggle control |
| `web/js.js` | Fetch, apply, and update the setting; conditionally render thumbnails |

## Out of scope / later

- Per-user or per-browser settings (cookies/localStorage).
- Other settings endpoints; this plan is scoped to the single `show_images` toggle.
- Changing the article list API; `GET /articles` continues to return `image_url` unchanged.

## Progress

- [x] Store: add `ShowImages` / `SetShowImages` methods to `server/internal/store/kv.go`
- [x] Handlers: add `GET /settings/images` and `POST /settings/images` handlers with tests
- [x] Server: register settings routes in `server/internal/api/server.go`
- [ ] Web UI: add "Images" checkbox and conditional thumbnail rendering

## Decisions

1. **Control style:** checkbox labeled **"Images"**.
2. **Apply timing:** the setting is fetched on page load and applied during rendering. It is also re-applied when the user toggles the checkbox locally. There is no polling or push for live cross-tab sync; refreshing the page picks up server-side changes.
