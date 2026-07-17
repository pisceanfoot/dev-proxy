## Why

Routes currently support `upstream_path` for a static path override, but there is no way to transform the incoming request path dynamically before forwarding it upstream. Developers working with versioned APIs, legacy path remapping, or path normalization need regex-based rewriting so that matched segments can be captured and substituted rather than replaced wholesale.

## What Changes

- Add a `url_rewrite` block to `RouteConfig` with two fields:
  - `match`: a Go `regexp` pattern to match against the incoming request path
  - `replace`: a replacement string that may reference capture groups (e.g. `$1`, `${name}`)
- At startup, compile the `match` pattern; invalid patterns are a fatal error
- When a route is selected and `url_rewrite` is configured, rewrite the request path using `regexp.ReplaceAllString(path, replace)` before forwarding to the upstream
- The rewritten path applies to the upstream request; the original path is unaffected in logs / access records unless the logger is updated
- `url_rewrite` and `upstream_path` are mutually exclusive on a single route — specifying both is a validation error

## Capabilities

### New Capabilities
- `url-rewrite`: Regex-based URL path rewriting on a route — match the incoming path with a regexp and replace it (including capture group references) before the request is forwarded upstream

### Modified Capabilities
- `enhanced-route-matching`: The route forwarding step now optionally applies a path rewrite transform after a route is matched, extending the existing routing pipeline

## Impact

- `internal/config/yaml.go`: new `URLRewriteConfig` struct; `RouteConfig` gains `URLRewrite` field; validation rejects invalid regex and `url_rewrite` + `upstream_path` co-existence
- `internal/router/router.go`: path rewrite logic applied when forwarding a matched route
- `cmd/dev-proxy/main_test.go` / integration tests: new test cases covering rewrite scenarios
- No breaking changes to existing config — `url_rewrite` is optional
