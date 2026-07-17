## 1. Config Layer

- [x] 1.1 Add `URLRewriteConfig` struct to `internal/config/yaml.go` with `Match string` and `Replace string` fields (yaml tags `match`, `replace`)
- [x] 1.2 Add `URLRewrite *URLRewriteConfig` field to `RouteConfig` (yaml tag `url_rewrite`)
- [x] 1.3 Add validation in `validateRoute`: reject route if both `url_rewrite` and `upstream_path` are set
- [x] 1.4 Add validation: if `url_rewrite` is present, both `match` and `replace` must be non-empty
- [x] 1.5 Compile `url_rewrite.match` regex in validation (or a startup compile pass); fatal error with route index and pattern on invalid regex

## 2. Router Layer

- [x] 2.1 Add `URLRewriteRegex *regexp.Regexp` and `URLRewriteReplace string` fields to `router.MatchedRoute`
- [x] 2.2 Populate these fields when building `MatchedRoute` from config (in `main.go` or wherever `MatchedRoute` is assembled from `RouteConfig`)

## 3. Proxy Layer

- [x] 3.1 Update `proxy.NewReverseProxy` signature to accept `urlRewriteRegex *regexp.Regexp` and `urlRewriteReplace string`
- [x] 3.2 In the proxy director, after `originalDirector(req)`, apply `urlRewriteRegex.ReplaceAllString(req.URL.Path, urlRewriteReplace)` when `urlRewriteRegex != nil`
- [x] 3.3 Clear `req.URL.RawPath` after rewrite (consistent with existing `upstream_path` handling)

## 4. Integration / Wiring

- [x] 4.1 Update call sites in `main.go` (or proxy dispatch logic) that construct `httputil.ReverseProxy` to pass through the new `URLRewriteRegex` and `URLRewriteReplace` fields from `MatchedRoute`

## 5. Tests

- [x] 5.1 Unit test in `internal/config/yaml_test.go` (or equivalent): valid `url_rewrite` block loads correctly
- [x] 5.2 Unit test: `url_rewrite` + `upstream_path` on same route produces validation error
- [x] 5.3 Unit test: missing `match` or `replace` field produces validation error
- [x] 5.4 Unit test: invalid regex in `match` produces fatal/validation error
- [x] 5.5 Integration test in `cmd/dev-proxy/main_test.go`: capture group rewrite — incoming `/api/users` with `match: "^/api/(.*)"` and `replace: "/v2/$1"` → upstream receives `/v2/users`
- [x] 5.6 Integration test: pattern with no match leaves path unchanged
- [x] 5.7 Integration test: `url_rewrite` does not interfere with `path_prefix` route selection

## 6. Documentation

- [x] 6.1 Add `url_rewrite` example to `README.md` (or existing config reference section) showing regex capture group usage
