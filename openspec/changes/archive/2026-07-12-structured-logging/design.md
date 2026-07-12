## Context

The current proxy writes output exclusively via `fmt.Printf` / `fmt.Println` — no levels, no filtering, every request always printed. The request log line in `loggingMiddleware` shows method, path, host, upstream, status, and duration, but not the matched host group or the full upstream URL after path rewriting. `Router.Match` returns `*MatchedRoute` only, so the caller cannot know which host group produced the match.

## Goals / Non-Goals

**Goals:**
- `internal/logger` package: `Error`, `Info`, `Debug` methods, level threshold, no-op below threshold
- `log_level` config field with `error` default; fatal on unrecognised value
- `info`: startup summary (ports, host group count, route count) + reload notification
- `debug`: full per-request trace including host group pattern, route criterion, upstream URL after path rewriting, status, duration
- Replace all `fmt.Printf`/`Println` runtime calls in `main.go` with levelled calls
- Extend `Router.Match` to expose host group pattern via `MatchResult`

**Non-Goals:**
- Structured/JSON log output (plain text only)
- Log file output (stdout/stderr only)
- Per-route log level overrides
- Log sampling or rate limiting
- Colours or ANSI formatting

## Decisions

### D1: Package-level singleton logger, not dependency-injected

**Choice:** `internal/logger` exposes a package-level `Default *Logger` and top-level functions `Info`, `Debug`, `Error` that delegate to it. `main.go` calls `logger.SetLevel(level)` at startup.

```go
// internal/logger/logger.go
var Default = &Logger{level: LevelError}

func Info(format string, args ...any)  { Default.Info(format, args...) }
func Debug(format string, args ...any) { Default.Debug(format, args...) }
func Error(format string, args ...any) { Default.Error(format, args...) }
```

**Rationale:** Passing a logger through every function signature adds noise to a tool that doesn't need production-grade injection. The logger level is set once at startup and never changes per-request. A singleton matches the simplicity of the rest of the codebase.

**Alternatives considered:**
- Dependency injection via function parameters — rejected: adds boilerplate to every function; no benefit for a single-binary dev tool
- `slog` (Go 1.21+) — rejected: more than needed; custom levels and format would require just as much code

### D2: Router.Match returns MatchResult carrying host group pattern

**Choice:** Change `Router.Match(*http.Request) *MatchedRoute` to `Router.Match(*http.Request) *MatchResult` where:

```go
type MatchResult struct {
    HostGroupPattern string   // the matched HostGroup's Match field, e.g. "api.local"
    Route            *MatchedRoute
}
```

Return `nil` when nothing matches (existing nil-check semantics preserved). Callers change `route := rt.Match(r)` to `result := rt.Match(r); route := result.Route`.

**Rationale:** The debug log requires the host group pattern ("which host group handled this request"). The only place this information exists is inside `Router.Match` when iterating groups. Returning it in a result struct avoids a second lookup and keeps the routing logic in one place.

**Alternatives considered:**
- Store matched group pattern in `MatchedRoute` — rejected: `MatchedRoute` is a config-derived struct; mixing routing decision metadata into it blurs ownership
- Second method `MatchGroup` — rejected: requires two traversals of host groups

### D3: Debug upstream URL computed from route fields, not transport hook

**Choice:** The debug log computes the upstream URL as `upstreamBase + rewrittenPath` from `route.Upstream`, `route.PathPrefix`, and `route.UpstreamPath` using the same logic as the proxy Director. No RoundTripper wrapping required.

```go
func computeUpstreamURL(route *router.MatchedRoute, reqPath string) string {
    if route.UpstreamPath != "" {
        suffix := strings.TrimPrefix(reqPath, route.PathPrefix)
        if !strings.HasPrefix(suffix, "/") { suffix = "/" + suffix }
        return route.Upstream + route.UpstreamPath + suffix
    }
    return route.Upstream + reqPath
}
```

**Rationale:** Hooking the transport to capture actual outbound requests adds complexity (custom RoundTripper, goroutine safety). Since the path rewriting formula is deterministic and defined in design, computing it in the log middleware gives equivalent information with zero overhead.

### D4: Route criterion shown as `<type>:<value>` in debug line

**Choice:** The debug line encodes the active match criterion as a compact tag:
- `prefix:/api` — PathPrefix was set and matched
- `exact:/health` — PathExact was set and matched
- `regex:^/v[0-9]+` — PathRegex was set and matched
- `*` — no path criterion (catch-all)

When multiple criteria are set (AND logic), show the most specific: regex > exact > prefix > `*`.

**Rationale:** A single short tag fits on one log line. The routing decision is clear without needing a full JSON payload. "Most specific" ordering matches developer intuition.

### D5: Log format

```
[info] listening on http://localhost:8080
[info] listening on https://localhost:8443
[info] 2 host groups, 5 routes configured
[debug] GET /api/v1/users host=api.local group=api.local route=prefix:/api/v1 upstream=http://localhost:3001 upstream_url=http://localhost:3001/v1/users status=200 duration=3ms
[debug] GET /missing host=app.local group=- route=- upstream=- status=504 duration=0s
```

**Rationale:** Key=value pairs are grep-friendly and human-readable without requiring a log viewer. Brackets prefix distinguishes levels. No timestamp (dev tool, logs are read live).

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| Computed upstream URL diverges from actual transport URL if Director logic changes | Both use the same `strings.TrimPrefix` + prepend formula; keep in sync via the `computeUpstreamURL` helper in main.go |
| Changing `Router.Match` signature breaks callers | Two call sites in main.go; update both in the same task |
| Package-level logger not safe for concurrent `SetLevel` calls | `SetLevel` is called once at startup before goroutines launch; no mutex needed |
