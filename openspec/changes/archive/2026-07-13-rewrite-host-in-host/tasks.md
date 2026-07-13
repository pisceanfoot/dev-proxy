## 1. Update config model

- [x] 1.1 In `internal/config/yaml.go`, change `RouteConfig.RewriteHost` from `bool` to `*bool \`yaml:"rewrite_host"\``
- [x] 1.2 In `internal/config/yaml.go`, add `RewriteHost *bool \`yaml:"rewrite_host"\`` field to the `HostGroup` struct (alongside the existing `Upstream` field)

## 2. Update upstream resolution

- [x] 2.1 In `cmd/dev-proxy/main.go`, update `resolveUpstream` to dereference `rc.RewriteHost *bool` for inline-URL routes: if `rc.RewriteHost != nil`, use `*rc.RewriteHost`; else use `false`
- [x] 2.2 In `cmd/dev-proxy/main.go`, update `buildMatchedRoute` signature to accept a `hostRewriteHost *bool` parameter (alongside the existing `hostUpstream string`)
- [x] 2.3 Inside `buildMatchedRoute`, compute `effectiveRewriteHost bool`: if `rc.RewriteHost != nil` use `*rc.RewriteHost`; else if `hostRewriteHost != nil` use `*hostRewriteHost`; else use `false` — then set `effectiveRC.RewriteHost = &effectiveRewriteHost`
- [x] 2.4 Update the host-group route loop in `buildHostGroups` to pass `hg.RewriteHost` as the `hostRewriteHost` argument to `buildMatchedRoute`
- [x] 2.5 Update the flat-routes loop in `buildHostGroups` to pass `nil` as `hostRewriteHost`

## 3. Update example config

- [x] 3.1 In `dev-proxy.yaml`, add `rewrite_host: true` at the host level on the `api.local` group and remove the now-redundant per-route `rewrite_host` fields; add a comment explaining the inheritance rule

## 4. Verify

- [x] 4.1 Run `go build ./...` to confirm clean compile
- [x] 4.2 Run `go vet ./...` to check for static analysis issues
