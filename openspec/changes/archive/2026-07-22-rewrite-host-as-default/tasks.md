## 1. Update config model

- [x] 1.1 In `internal/config/yaml.go`, remove `RewriteHost bool \`yaml:"rewrite_host"\`` from `UpstreamConfig`
- [x] 1.2 In `internal/config/yaml.go`, remove `RewriteHost *bool \`yaml:"rewrite_host"\`` from `RouteConfig`
- [x] 1.3 In `internal/config/yaml.go`, remove `RewriteHost *bool \`yaml:"rewrite_host"\`` from `HostGroup`
- [x] 1.4 In `internal/config/yaml.go`, delete the comment blocks that explain `*bool` inheritance semantics for `RewriteHost`

## 2. Update proxy package

- [x] 2.1 In `internal/proxy/proxy.go`, remove the `rewriteHost bool` parameter from `NewReverseProxy`
- [x] 2.2 In `internal/proxy/proxy.go`, change the Director to unconditionally set `req.Host = u.Host` (remove the `if rewriteHost && u.Host != ""` guard)
- [x] 2.3 In `internal/proxy/proxy_test.go`, update `TestNewReverseProxy_InvalidURL` call site (remove second argument)
- [x] 2.4 In `internal/proxy/proxy_test.go`, update `TestNewReverseProxy_ValidUpstream` call site
- [x] 2.5 In `internal/proxy/proxy_test.go`, rewrite `TestNewReverseProxy_RewriteHost` — remove the `rewriteHost=false` sub-test; the `rewriteHost=true` sub-test becomes the only behaviour and should be renamed/merged
- [x] 2.6 In `internal/proxy/proxy_test.go`, update `TestNewReverseProxy_InsecureHTTPS` call site
- [x] 2.7 In `internal/proxy/proxy_test.go`, update `TestNewReverseProxy_PathRewriting` call sites
- [x] 2.8 In `internal/proxy/proxy_test.go`, update `TestServeHTTP` and `TestServeHTTP_WithPathPrefix` call sites

## 3. Update router package

- [x] 3.1 In `internal/router/router.go`, remove `RewriteHost bool` from `MatchedRoute` struct

## 4. Update main entrypoint

- [x] 4.1 In `cmd/dev-proxy/main.go`, update `resolveUpstream` signature to drop the `rewriteHost bool` return value
- [x] 4.2 In `cmd/dev-proxy/main.go`, simplify `resolveUpstream` body — inline-upstream routes no longer need `*bool` dereference logic
- [x] 4.3 In `cmd/dev-proxy/main.go`, update `buildMatchedRoute` signature to remove `hostRewriteHost *bool` parameter
- [x] 4.4 In `cmd/dev-proxy/main.go`, delete the effective-rewrite_host resolution block inside `buildMatchedRoute`
- [x] 4.5 In `cmd/dev-proxy/main.go`, update the `router.MatchedRoute` construction to stop assigning `RewriteHost`
- [x] 4.6 In `cmd/dev-proxy/main.go`, update `buildHostGroups` calls to `buildMatchedRoute` — remove the `hg.RewriteHost` and `nil` arguments
- [x] 4.7 In `cmd/dev-proxy/main.go`, update `buildHandler` call to `proxy.NewReverseProxy` — remove the `route.RewriteHost` argument
- [x] 4.8 In `cmd/dev-proxy/main.go`, update the catch-all fallback route in `buildHostGroups` to remove `RewriteHost: true`

## 5. Update example config

- [x] 5.1 In `dev-proxy.yaml`, remove any comments referencing `rewrite_host` inheritance

## 6. Verify

- [x] 6.1 Run `go build ./...` to confirm clean compile
- [x] 6.2 Run `go vet ./...` to check for static analysis issues
- [x] 6.3 Run `go test ./...` to confirm all tests pass
