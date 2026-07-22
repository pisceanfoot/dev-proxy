## Why

`rewrite_host` is currently a configurable boolean on upstreams, routes, and host groups. In practice, almost every proxy use case benefits from rewriting the Host header to match the upstream — it is the expected behaviour for a reverse proxy. Keeping it configurable adds YAML boilerplate, complicates the config model with `*bool` pointer fields for inheritance semantics, and creates a foot-gun where users forget to set `rewrite_host: true` and get confusing 404s or TLS errors from upstreams that validate the Host header.

Removing the knob and always rewriting the Host header simplifies the mental model: **dev-proxy always presents the upstream hostname to the backend**.

## What Changes

- Remove `rewrite_host` field from all config structs:
  - `UpstreamConfig` — delete `RewriteHost bool`
  - `RouteConfig` — delete `RewriteHost *bool`
  - `HostGroup` — delete `RewriteHost *bool`
- Remove `rewrite_host` parameter from `proxy.NewReverseProxy`; the proxy now **always** sets `req.Host` to the upstream hostname.
- Remove `RewriteHost bool` from `router.MatchedRoute`.
- Delete `resolveUpstream`’s `rewriteHost` return value and all call-site logic that computed an effective rewrite_host.
- Update example `dev-proxy.yaml` to remove all `rewrite_host` references and comments.
- Update `internal/proxy/proxy_test.go` — remove the `rewriteHost=false` sub-test and always pass `true` semantics.

## Capabilities

### New Capabilities

_(none — this removes a capability)_

### Modified Capabilities

- `proxy`: Host header is now always rewritten to the upstream host.
- `config`: Config file no longer accepts `rewrite_host` anywhere.

### Removed Capabilities

- Per-upstream / per-route / per-host-group opt-out of Host header rewriting.

## Impact

- **Breaking config change**: Any config that contains `rewrite_host:` keys will fail YAML unmarshalling (unknown field). Users must delete those lines.
- **Simpler code**: Eliminates `*bool` pointer gymnastics and inheritance logic in `buildMatchedRoute`.
- **No runtime optionality**: The proxy Director unconditionally sets `req.Host = u.Host`.
- **Test updates**: Proxy tests that verified `rewriteHost=false` behaviour are removed or updated.
