## Context

The `static_dir` feature in dev-proxy lets a route serve files directly from the local filesystem. The implementation lives in `internal/static/static.go` and is wired in `cmd/dev-proxy/main.go` via `static.Serve(route.StaticDir, handler)`.

Current state:
- `os.Stat` failure (any cause) → silently falls through to the next handler.
- Directory hit → silently falls through to the next handler.
- File not found → silently falls through to the next handler.
- File found → served via `http.ServeFile`.

For routes without an upstream, the "next handler" is `http.NotFoundHandler`. This means missing files already produce 404s, but the response has no body and errors (stat failures, permission denied) produce the same 404, hiding the real cause.

## Goals / Non-Goals

**Goals:**
- Return 500 with a human-readable reason when `os.Stat` or file open fails for reasons other than "not found".
- Return 404 with a plain-text body when the requested path does not exist under `static_dir`.
- Return an HTML directory listing (filename, size, last-modified) when the path resolves to a directory.
- Guard against path traversal beyond `static_dir` root (return 403).
- Keep the implementation inside `internal/static/static.go`; no changes to call sites.

**Non-Goals:**
- Sorting or pagination of directory listings.
- Authentication or authorization for static files.
- Serving `index.html` automatically for directory hits (out of scope for this change).
- Changing how routes with both `static_dir` and `upstream` behave beyond what the proposal states.

## Decisions

### 1. Stop falling through on static errors

**Decision**: Once a request is matched to a `static_dir` route, the static handler owns the response entirely. It will never call `next` for errors.

**Rationale**: The fall-through model made sense when static was an optional overlay on top of an upstream. With proper error responses, falling through on 404 would silently hide the static layer from the caller. The only safe semantic is: if `static_dir` is set, the static handler is authoritative.

**Alternative considered**: Keep fall-through for 404 only. Rejected — inconsistent and surprising to operators.

### 2. Path containment via `filepath.Rel`

**Decision**: After `filepath.Join(staticDir, urlPath)`, check that the cleaned path starts with `staticDir` using `filepath.Rel` or `strings.HasPrefix(cleaned, staticDir+string(os.PathSeparator))`. Return 403 if outside.

**Rationale**: `filepath.Join` cleans `..` segments, but a crafted URL could still escape the root on some platforms. An explicit containment check is the safest approach.

**Alternative considered**: Reject URLs containing `..` before joining. Rejected — too fragile; `filepath.Rel` is the canonical Go solution.

### 3. Directory listing with `html/template`

**Decision**: When `os.Stat` succeeds and `IsDir()` is true, render a minimal HTML page listing directory entries using `os.ReadDir` and a `html/template` template embedded in the package.

**Rationale**: No external dependency needed; `html/template` auto-escapes filenames, preventing XSS from malicious filenames on the filesystem.

**Alternative considered**: Plain-text listing. Rejected — HTML is more usable and matches nginx behaviour users expect.

### 4. Error response format

**Decision**: 500 and 404 responses return `text/plain` bodies: `404 Not Found\n` and `500 Internal Server Error: <reason>\n`. Log the detailed error at WARN level using the existing structured logger.

**Rationale**: Simple, grep-able, consistent with how Go's standard library formats error strings.

## Risks / Trade-offs

- **BREAKING: mixed static+upstream routes** — Routes that have both `static_dir` and `upstream` will no longer proxy missing-file requests to the upstream; they'll get 404. Teams relying on this fall-through for SPA routing will need to restructure their config (add a separate route for the upstream, or remove `static_dir`). Mitigation: document clearly in release notes.
- **Directory listing information disclosure** — Listing directory contents may expose unintended files. Mitigation: this is opt-in via `static_dir`; users who serve a directory must expect its contents to be visible.
- **Symlinks** — `os.ReadDir` follows symlinks for stat. A symlink pointing outside `static_dir` will pass the stat step but the containment check will catch it on the resolved path if `os.Lstat` is used. Mitigation: use `filepath.EvalSymlinks` before the containment check.

## Migration Plan

1. Update `internal/static/static.go` with the new logic.
2. No config changes required.
3. Operators with mixed `static_dir` + `upstream` routes must audit their configs before upgrading.
4. Rollback: revert `internal/static/static.go`; no data or schema migrations.

## Open Questions

- Should directory listing be gated behind a config flag (e.g. `static_dir_listing: true`)? For now, assumed always-on when `static_dir` is set — revisit if operators request opt-out.
