## 1. Extend config structs

- [x] 1.1 Add `UpstreamConfig` struct to `internal/config/yaml.go` with fields `URL string`, `RewriteHost bool`, `Insecure bool` and yaml tags
- [x] 1.2 Add `HostGroup` struct to `internal/config/yaml.go` with fields `Match string` and `Routes []RouteConfig` and yaml tags
- [x] 1.3 Add `UpstreamPath string` field to `RouteConfig` with `yaml:"upstream_path"` tag
- [x] 1.4 Add `Upstreams map[string]UpstreamConfig` and `Hosts []HostGroup` fields to `Config` with yaml tags

## 2. Extend config validation

- [x] 2.1 Validate each `upstreams` entry: name non-empty, URL non-empty and parseable — fatal error on failure
- [x] 2.2 Validate each `hosts` entry: `match` non-empty, `routes` non-empty — fatal error on failure
- [x] 2.3 Validate route `upstream` references in both flat routes and host group routes: if no `://`, look up in `upstreams` map — fatal error if not found
- [x] 2.4 Log a startup warning when both `hosts` and `routes` are defined

## 3. Add upstream_path rewriting to proxy package

- [x] 3.1 Extend `proxy.NewReverseProxy` signature to accept `routePrefix string` and `upstreamPath string`
- [x] 3.2 In the Director closure, when `upstreamPath != ""`: strip `routePrefix` from `req.URL.Path`, prepend `upstreamPath`, clear `req.URL.RawPath`
- [x] 3.3 Update all existing callers of `NewReverseProxy` in `cmd/dev-proxy/main.go` to pass empty strings for the new parameters (no behaviour change)

## 4. Extend MatchedRoute and resolve upstreams in buildRoutes

- [x] 4.1 Add `UpstreamPath string` field to `router.MatchedRoute`
- [x] 4.2 In `buildRoutes` in `cmd/dev-proxy/main.go`, implement `resolveUpstream`: detect `://` in `rc.Upstream`; if absent, look up name in `cfg.Upstreams` and use its URL/RewriteHost/Insecure; log.Fatalf on unknown name
- [x] 4.3 Populate `mr.UpstreamPath` from `rc.UpstreamPath` when mapping `RouteConfig` → `MatchedRoute`
- [x] 4.4 Pass `mr.PathPrefix` and `mr.UpstreamPath` to `proxy.NewReverseProxy` in the handler builder

## 5. Implement host-group routing in Router

- [x] 5.1 Add a `HostGroup` struct to `internal/router/router.go` with fields `Match string` and `Routes []MatchedRoute`
- [x] 5.2 Add `hostGroups []HostGroup` field to `Router`; update `New` to accept both flat routes and host groups; if host groups are present, store them; otherwise populate a single implicit group matching `*`
- [x] 5.3 Rewrite `Router.Match` to two-phase logic: iterate `hostGroups` in order, apply `path.Match(group.Match, req.Host)` for host phase; on first host match, iterate that group's routes for path phase; return nil if neither phase matches
- [x] 5.4 Update `buildRoutes` and `main.go` to build `[]router.HostGroup` from `cfg.Hosts` (when present) and pass them to `router.New`

## 6. Verify and update fixture

- [x] 6.1 Update `dev-proxy.yaml` to demonstrate `upstreams` map, `hosts` groups, a named upstream reference, and an `upstream_path` override
- [x] 6.2 Run `go build ./...` to confirm clean compile
- [x] 6.3 Run `go vet ./...` to check for static analysis issues
