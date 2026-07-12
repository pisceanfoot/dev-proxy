## Context

The current `internal/router/router.go` evaluates routes with two simple checks: exact `HostMatch` string equality and `strings.HasPrefix` for path. This covers the basic case but fails for common dev scenarios — matching all subdomains of a local TLD, routing to different upstreams based on exact vs. prefix vs. regex path patterns. The `RouteConfig` struct in `internal/config/yaml.go` exposes only `path_prefix` and `host_match` to users.

When no route matches, `HandleNotFound` writes a 404. The change replaces this with 504 per the new spec.

## Goals / Non-Goals

**Goals:**
- Extend `MatchedRoute` with `PathExact`, `PathRegex` (pre-compiled `*regexp.Regexp`) alongside existing `PathPrefix`
- Extend `host_match` to support glob wildcards via `path.Match`
- AND-combination: all non-empty criteria on a route must pass
- Compile regex patterns at startup in `buildRoutes`; fatal error on invalid pattern
- Replace 404 no-match with 504 + `no route matched: host=<h> path=<p>` body

**Non-Goals:**
- OR logic between criteria on a single route
- Named capture groups or path parameter extraction from regex
- Priority scoring or specificity ordering (order is declaration order only)
- Negative match / exclusion patterns

## Decisions

### D1: Three path match modes are independent fields, not an enum

**Choice:** `PathPrefix`, `PathExact`, and `PathRegex` are three separate fields on `MatchedRoute` (and `RouteConfig`). A route may set any combination; all set ones must pass.

```yaml
routes:
  - path_exact: "/health"          # only exact
    upstream: http://localhost:9000

  - host_match: "api.*.local"
    path_regex: "^/v[0-9]+/"       # host glob + regex
    upstream: http://localhost:3001

  - path_prefix: "/"               # catch-all
    upstream: http://localhost:3000
```

**Rationale:** An enum mode (`match_mode: prefix|exact|regex`) would require extra validation to prevent contradictions. Separate fields are simpler to validate and more composable — a route can combine host glob with regex path, for example. The AND semantics mean redundant fields (e.g., both `path_exact` and `path_prefix` set) still work correctly (both must pass, which implies exact must also be a prefix of itself).

**Alternatives considered:**
- Single `match_mode` enum + single `match_value` — rejected: loses composability, harder to read in YAML
- Separate `match` sub-object — rejected: extra nesting for small gain

### D2: Host glob uses `path.Match` from stdlib

**Choice:** Host matching uses Go's `path.Match(pattern, host)`. `*` matches any sequence not containing `/`; `?` matches exactly one character. Since hostnames never contain `/`, `*` effectively matches any substring within a label or across labels.

```go
import "path"

matched, err := path.Match(rt.HostMatch, req.Host)
```

**Rationale:** `path.Match` is stdlib, zero dependencies, handles `*` and `?` correctly for hostnames. It does not treat `.` as a separator — `*.example.com` matches `api.example.com` but also `api.v2.example.com` (because `*` matches `api.v2`). This is intentional: if the user wants to limit to one subdomain level they should use `?.example.com` or a regex route.

**Alternatives considered:**
- Manual split-and-compare per label — rejected: reimplements what `path.Match` already does
- `filepath.Match` — same as `path.Match` for hostnames; either works, `path` import is cleaner semantically

### D3: Regex compiled once at startup in `buildRoutes`

**Choice:** In `main.go::buildRoutes`, when a `RouteConfig.PathRegex` string is non-empty, call `regexp.MustCompile` — actually `regexp.Compile` with a fatal-on-error pattern. The compiled `*regexp.Regexp` is stored in `MatchedRoute.PathRegex`. No compilation happens per-request.

```go
// in buildRoutes():
if rc.PathRegex != "" {
    re, err := regexp.Compile(rc.PathRegex)
    if err != nil {
        log.Fatalf("[dev-proxy] route %d: invalid path_regex %q: %v", i, rc.PathRegex, err)
    }
    mr.PathRegex = re
}
```

**Rationale:** Compiling on every request is wasteful. Failing fast at startup with a clear message is better than silently skipping or panicking on first request.

### D4: No-match returns 504 with structured plain-text body

**Choice:** Replace `HandleNotFound` (404) with `HandleNoMatch` that writes status 504 and body `no route matched: host=<host> path=<path>`.

**Rationale:** 504 (Gateway Timeout) signals a gateway-level failure — the proxy could not find a backend to forward to. For a dev proxy, "no backend configured for this request" maps reasonably to 504 rather than 404 (which implies a resource is genuinely absent). The plain-text body helps developers immediately see which host/path was unmatched.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| `path.Match` `*` matches across dot-labels (e.g., `*.local` matches `a.b.local`) | Document clearly; users wanting single-label matching should use `?` or `path_regex` |
| Routes with both `path_exact` and `path_prefix` set may confuse authors | Both criteria must pass; effectively the exact path must also be a prefix of itself, which it always is — so `path_prefix` is redundant but not harmful |
| 504 is an unusual no-match status | Matches the specification; documented in spec with example body |
| Regex patterns can be arbitrarily slow (ReDoS) | Dev-only tool; acceptable risk. Document that patterns should be simple |
