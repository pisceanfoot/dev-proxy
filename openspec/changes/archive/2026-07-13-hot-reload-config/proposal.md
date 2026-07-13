## Why

The proxy already watches `dev-proxy.yaml` for changes and rebuilds routes on every save — but the reload is incomplete in two ways. First, `log_level` changes are ignored (the logger level is set once at startup and never updated). Second, when server-level config (`listen_ports`, `tls`, `redirect_http`) changes, the proxy silently applies nothing, leaving the developer with no indication that a restart is required. Additionally, `rt` (the router pointer) is reassigned in the reload callback without synchronization, which is a data race under concurrent requests. Watcher also still uses `fmt.Printf` internally despite the logger package existing.

## What Changes

- On config reload: re-apply `log_level` immediately (no restart needed)
- On config reload: compare the new `server` block against the snapshot captured at startup; if any server field changed (`listen_ports`, `tls.*`, `redirect_http`), emit a clear warning — `[info] server config changed (listen_ports); restart required to apply` — but continue running on the original ports
- On config reload error: log the parse/validation error clearly and keep running on the previous config (already works, but now surfaced via `logger.Error` instead of `fmt.Printf`)
- Fix the data race: protect the `*router.Router` pointer with `sync/atomic.Pointer[router.Router]` so concurrent in-flight requests always see a consistent router
- Update `internal/watcher/watcher.go` to delegate log output to a caller-supplied `onError func(error)` and `onReload func()` callback pair instead of using `fmt.Printf` directly

## Capabilities

### New Capabilities

- `hot-reload-config`: Define which config sections are live-reloadable (`routes`, `hosts`, `upstreams`, `log_level`) vs. restart-required (`server.*`), detect and warn on restart-required changes, keep old config on reload error

### Modified Capabilities

- `structured-logging`: Watcher internal messages now flow through `logger.Error`/`logger.Info` (behaviour change: messages were previously unconditional `fmt.Printf`, now level-gated)

## Impact

- **Modified files:** `cmd/dev-proxy/main.go` (snapshot server config at startup; enrich reload callback to re-apply log_level, detect server changes, use `atomic.Pointer` for rt), `internal/watcher/watcher.go` (replace `fmt.Printf` with caller-supplied callbacks), `internal/config/yaml.go` (add `serverConfigEqual` helper or expose comparable fields)
- **No new dependencies** — `sync/atomic` is stdlib (Go 1.19+, project uses 1.23.3)
