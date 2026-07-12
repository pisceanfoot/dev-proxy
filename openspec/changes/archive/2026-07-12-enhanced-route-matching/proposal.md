## Why

The current router only supports exact host matching and path prefix matching, which is too rigid for realistic development scenarios. Teams proxying multiple services often need to match routes by hostname pattern (e.g., `*.local`), exact paths, or regex patterns — and the current single-mode prefix check cannot express these rules. When no route matches, the proxy returns a generic 404 rather than a clear gateway-level error.

## What Changes

- **BREAKING**: `HostMatch` in route config now interprets glob wildcards (`*` matches any substring, `?` matches a single character); previously it was exact-only
- Route config gains two new path-matching fields: `path_exact` (exact equality) and `path_regex` (compiled regular expression)
- Existing `path_prefix` field remains, now one of three path match modes
- All configured match criteria on a route must pass (AND logic); unset criteria always pass
- Routes continue to be evaluated in declaration order; first full match wins
- No-match response changes from 504 (Gateway Timeout) with a plain-text error body
- Regex patterns are compiled at startup; invalid patterns produce a fatal error with the offending pattern

## Capabilities

### New Capabilities

- `enhanced-route-matching`: Multi-mode route matching — host glob, path exact, path prefix, path regex — with AND-combination semantics and 504 on no match

### Modified Capabilities

_(none — `proxy-routing` has no main spec yet; this supersedes its archived definition)_

## Impact

- **Modified files:** `internal/router/router.go` (matching logic, `MatchedRoute` struct, `HandleNotFound` → `HandleNoMatch` returning 504), `internal/config/yaml.go` (`RouteConfig` gains `path_exact`, `path_regex` fields; `host_match` now documented as glob), `cmd/dev-proxy/main.go` (`buildRoutes` compiles regex at startup, maps new fields to `MatchedRoute`)
- **New dependency:** none — `path.Match` (stdlib) for glob host matching; `regexp` (stdlib) for path regex
- Regex compilation errors at startup are fatal with the offending pattern and route index shown
