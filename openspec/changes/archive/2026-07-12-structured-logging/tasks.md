## 1. Create internal/logger package

- [x] 1.1 Create `internal/logger/logger.go` with `Level` type and constants `LevelError`, `LevelInfo`, `LevelDebug`
- [x] 1.2 Add `Logger` struct with `level Level` field and `Info`, `Debug`, `Error` methods that write to stderr prefixed with `[info]`, `[debug]`, `[error]` — no-op when level is below threshold
- [x] 1.3 Add package-level `Default *Logger` (initialised to `LevelError`) and package-level `Info`, `Debug`, `Error` functions delegating to `Default`
- [x] 1.4 Add `SetLevel(l Level)` and `ParseLevel(s string) (Level, error)` functions; `ParseLevel` accepts `"error"`, `"info"`, `"debug"` case-insensitively

## 2. Add log_level to config

- [x] 2.1 Add `LogLevel string` field to `Config` in `internal/config/yaml.go` with `yaml:"log_level"` tag
- [x] 2.2 Add validation in `validate`: if `LogLevel` is non-empty and not one of `error`/`info`/`debug` (case-insensitive), return a fatal error listing valid options; default to `"error"` when empty

## 3. Extend Router.Match to return MatchResult

- [x] 3.1 Add `MatchResult` struct to `internal/router/router.go` with fields `HostGroupPattern string` and `Route *MatchedRoute`
- [x] 3.2 Change `Router.Match(*http.Request) *MatchedRoute` to `Router.Match(*http.Request) *MatchResult`; populate `HostGroupPattern` from the matched group's `Match` field; return `nil` when nothing matches
- [x] 3.3 Update both callers of `rt.Match` in `cmd/dev-proxy/main.go` (in `buildHandler` and any other site) to use `result.Route` instead of the direct return value

## 4. Wire logger into main.go

- [x] 4.1 After `config.Load()`, call `logger.ParseLevel(cfg.LogLevel)` and `logger.SetLevel(...)`; `log.Fatalf` on parse error
- [x] 4.2 Replace the `fmt.Printf("[dev-proxy] Listening on ...")` lines in `startServers` goroutines with `logger.Info`
- [x] 4.3 Replace `fmt.Println("[dev-proxy] Routes updated")` in the watcher callback with `logger.Info`
- [x] 4.4 Remove `printBanner` (or convert it to `logger.Info` calls) and replace with structured info lines: one `logger.Info` per listen port showing scheme + address, then one `logger.Info` with host group count and total route count
- [x] 4.5 Replace `fmt.Printf("[dev-proxy] Watcher error: ...")` and `fmt.Printf("[dev-proxy] Error shutting down ...")` with `logger.Error`

## 5. Replace loggingMiddleware with debug-aware request logger

- [x] 5.1 Add helper `computeUpstreamURL(route *router.MatchedRoute, reqPath string) string` in `main.go`: when `route.UpstreamPath != ""` strip `route.PathPrefix` from `reqPath`, prepend `route.UpstreamPath`; otherwise return `route.Upstream + reqPath`
- [x] 5.2 Add helper `routeCriterion(route *router.MatchedRoute) string` returning the most-specific active criterion as `prefix:<v>`, `exact:<v>`, `regex:<v>`, or `*`
- [x] 5.3 Rewrite `loggingMiddleware` to accept `hostGroupPattern string` alongside `route` and emit a `logger.Debug` line in the format: `method=GET path=/api/v1/users host=api.local group=api.local route=prefix:/api/v1 upstream=http://localhost:3001 upstream_url=... status=200 duration=3ms`
- [x] 5.4 Update the `buildHandler` call site to extract `result.HostGroupPattern` from `rt.Match` and pass it to the updated `loggingMiddleware`; for no-match 504 responses emit a debug line with `group=- route=- upstream=- status=504`

## 6. Update fixture and verify

- [x] 6.1 Add `log_level: debug` to `dev-proxy.yaml` (commented out by default, with `log_level: error` as the active setting)
- [x] 6.2 Run `go build ./...` to confirm clean compile
- [x] 6.3 Run `go vet ./...` to check for static analysis issues
