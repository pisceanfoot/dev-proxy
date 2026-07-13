## 1. Extend config model

- [x] 1.1 In `internal/config/yaml.go`, add `Upstream string \`yaml:"upstream"\`` field to the `HostGroup` struct
- [x] 1.2 Update `validateRoute` signature to `validateRoute(idx int, r RouteConfig, hostUpstream string, upstreams map[string]UpstreamConfig) error`; when `r.Upstream == ""`, use `hostUpstream` as the effective upstream for validation
- [x] 1.3 In `validate`, update the host-group route loop to pass `hg.Upstream` as `hostUpstream` to `validateRoute`; update the flat-routes loop to pass `""` as `hostUpstream`
- [x] 1.4 In `validate`, add validation of the host-group `upstream` field itself: if non-empty, apply the same inline-URL / named-reference checks used for route upstreams; emit a fatal error identifying the host group index on failure
- [x] 1.5 In `validate`, replace the implicit "missing upstream is ok" behaviour with an explicit check: after resolving the effective upstream, if it is still empty, return a fatal error identifying the host group index and route index

## 2. Update route compilation

- [x] 2.1 In `cmd/dev-proxy/main.go`, update `buildMatchedRoute` signature to accept a `hostUpstream string` parameter
- [x] 2.2 Inside `buildMatchedRoute`, compute `effectiveUpstream`: if `rc.Upstream != ""` use it, else use `hostUpstream`; construct a local copy of `rc` with `Upstream` set to `effectiveUpstream` before calling `resolveUpstream`
- [x] 2.3 Update the host-group route loop in `buildHostGroups` to pass `hg.Upstream` as the `hostUpstream` argument to `buildMatchedRoute`
- [x] 2.4 Update the flat-routes loop in `buildHostGroups` to pass `""` as `hostUpstream`

## 3. Update example config

- [x] 3.1 In `dev-proxy.yaml`, update the `api.local` host group to use `upstream: api-v1` at the host level and remove the redundant `upstream: api-v1` from routes that were already using it; leave the route that uses `api-v2` as-is to demonstrate the override

## 4. Verify

- [x] 4.1 Run `go build ./...` to confirm clean compile
- [x] 4.2 Run `go vet ./...` to check for static analysis issues
