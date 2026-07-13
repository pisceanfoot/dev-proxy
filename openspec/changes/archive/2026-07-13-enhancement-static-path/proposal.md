## Why

The current `static_dir` serving silently falls through to the next handler on any error (stat failure, directory hit, missing file), making it impossible to use static serving as a standalone mode and hiding real problems from operators. Routes configured with `static_dir` but no upstream already use `http.NotFoundHandler` as the fallback, yet errors never produce meaningful HTTP responses.

## What Changes

- When `os.Stat` fails for any reason other than "not found" (e.g. permission denied, I/O error), return **500 Internal Server Error** with a plain-text reason instead of silently falling through.
- When the resolved path is a directory, return an **HTML directory listing** (like nginx `autoindex on`) showing file names, sizes, and last-modified timestamps, rather than falling through.
- When the file does not exist (`os.IsNotExist`), return **404 Not Found** with a plain-text body instead of falling through to the next handler.
- Add a path-containment guard to prevent path traversal outside `static_dir`.

## Capabilities

### New Capabilities

- `static-dir-error-responses`: Proper HTTP error responses (500, 404) from the static file handler when the file cannot be served, replacing the current silent fall-through behaviour.
- `static-dir-listing`: HTML directory listing when a request resolves to a directory inside `static_dir`, mirroring nginx `autoindex on`.

### Modified Capabilities

<!-- No existing spec-level capabilities are changing — static serving has no prior spec. -->

## Impact

- `internal/static/static.go` — core serving logic rewritten to branch on error type, add directory listing, add path containment.
- `cmd/dev-proxy/main.go` — no changes needed; wiring is unaffected.
- Routes that have **both** `static_dir` and `upstream` set: previously, a missing static file would proxy to the upstream. After this change a missing static file returns 404 instead. This is a **BREAKING** behaviour change for mixed-mode routes.
- No new dependencies; directory listing is rendered with `html/template` from the standard library.
