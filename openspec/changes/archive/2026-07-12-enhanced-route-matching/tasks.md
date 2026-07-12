## 1. Extend config and MatchedRoute structs

- [x] 1.1 Add `PathExact string` and `PathRegex string` fields to `RouteConfig` in `internal/config/yaml.go` with `yaml:"path_exact"` and `yaml:"path_regex"` tags
- [x] 1.2 Add `PathExact string` and `PathRegex *regexp.Regexp` fields to `MatchedRoute` in `internal/router/router.go`

## 2. Compile regex patterns at startup

- [x] 2.1 In `buildRoutes` in `cmd/dev-proxy/main.go`, for each `RouteConfig` with a non-empty `PathRegex`, call `regexp.Compile`; on error, `log.Fatalf` with the route index and the offending pattern
- [x] 2.2 Populate `mr.PathExact` and `mr.PathRegex` when mapping `RouteConfig` → `MatchedRoute` in `buildRoutes`

## 3. Rewrite Router.Match with AND-combination logic

- [x] 3.1 Replace the existing `Match` body with a loop that evaluates all configured criteria per route using AND logic — all non-empty criteria must pass
- [x] 3.2 Implement host glob check: `path.Match(rt.HostMatch, req.Host)` when `rt.HostMatch != ""`; skip on mismatch
- [x] 3.3 Implement path exact check: `req.URL.Path == rt.PathExact` when `rt.PathExact != ""`; skip on mismatch
- [x] 3.4 Implement path prefix check: `strings.HasPrefix(req.URL.Path, rt.PathPrefix)` when `rt.PathPrefix != ""`; skip on mismatch (existing behaviour, retained)
- [x] 3.5 Implement path regex check: `rt.PathRegex.MatchString(req.URL.Path)` when `rt.PathRegex != nil`; skip on mismatch

## 4. Replace no-match response with 504

- [x] 4.1 Rename `HandleNotFound` to `HandleNoMatch` and change status from 404 to 504 with body `no route matched: host=<host> path=<path>` using the request's Host and URL.Path
- [x] 4.2 Update all callers of `HandleNotFound` in `cmd/dev-proxy/main.go` to call `HandleNoMatch` with the `*http.Request` argument

## 5. Verify and validate

- [x] 5.1 Run `go build ./...` to confirm the project compiles cleanly
- [x] 5.2 Run `go vet ./...` to check for static analysis issues
- [x] 5.3 Update `dev-proxy.yaml` test fixture to demonstrate all three path modes and a wildcard host match
