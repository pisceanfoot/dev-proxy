## Why

The proxy currently logs every request unconditionally via `fmt.Printf` with no way to silence it, filter by severity, or get richer routing context when debugging. In production-like dev environments (CI, shared machines) the noise is unwanted; when diagnosing a routing problem the single-line format lacks the detail needed (which host group matched, what path criteria fired, what URL was sent to the upstream). Adding level-controlled logging with `error` as the default makes the tool quiet by default and richly informative on demand.

## What Changes

- Add a `log_level` field to `dev-proxy.yaml` (valid values: `error`, `info`, `debug`; default `error`)
- Create `internal/logger` package with a levelled logger: `Error`, `Info`, `Debug` methods; silently no-ops for levels below the configured threshold
- **error** (default): only `log.Fatalf`-style fatal messages, already present; nothing new printed at runtime
- **info**: startup summary — listen ports, scheme per port, number of host groups, total routes configured; config reload notification
- **debug**: per-request trace — method, request URL, matched host group pattern, matched route criteria (prefix/exact/regex), resolved upstream URL (including any path rewrite), response status code, duration
- Replace all `fmt.Printf`/`fmt.Println` runtime output in `main.go` with levelled logger calls
- Extend `Router.Match` to return a `MatchResult` carrying both the matched host group pattern and the matched route, so debug logging can report routing decisions accurately

## Capabilities

### New Capabilities

- `structured-logging`: Level-controlled logging (`error`/`info`/`debug`) configurable in `dev-proxy.yaml`; each level adds a defined set of output

### Modified Capabilities

_(none — logging is new; existing `fmt.Printf` calls are replaced, not spec-changing)_

## Impact

- **New files:** `internal/logger/logger.go`
- **Modified files:** `internal/config/yaml.go` (add `LogLevel string`; validate values), `internal/router/router.go` (`Match` returns `*MatchResult` instead of `*MatchedRoute`), `cmd/dev-proxy/main.go` (init logger, replace `fmt.Printf`/`Println` with levelled calls, update `rt.Match` callers, enrich debug log with host group + upstream URL), `dev-proxy.yaml` (add `log_level` field)
- **No new dependencies** — stdlib `log`, `fmt`, `os`
