## Why

`rewrite_host` on routes has the same repetition problem as `upstream` did: when every route behind a host group should rewrite the Host header (the common case for local backends), every route must repeat `rewrite_host: true` individually. A single host-level default would eliminate the boilerplate.

## What Changes

- Add an optional `rewrite_host` field to each entry in the `hosts` list.
- Routes with inline upstreams that omit their own `rewrite_host` inherit the host-level default.
- Route-level `rewrite_host` explicitly overrides the host-level value — including overriding `true` back to `false`.
- To distinguish "not set / inherit" from "explicitly false", `RouteConfig.RewriteHost` changes type from `bool` to `*bool`. Nil means inherit; a pointer value is explicit.
- Routes using named upstreams are unaffected — their `rewrite_host` comes from the upstream definition, as before.

## Capabilities

### New Capabilities

_(none — this extends an existing capability)_

### Modified Capabilities

- `host-groups`: Add optional `rewrite_host` field on host entries; define inheritance and override semantics for inline-URL routes.

## Impact

- `config` package: `HostGroup` gains `RewriteHost *bool`; `RouteConfig.RewriteHost` changes from `bool` to `*bool`.
- `cmd/dev-proxy/main.go`: `buildMatchedRoute` resolves effective `rewriteHost` using host fallback before calling `resolveUpstream`.
- `resolveUpstream`: no signature change — it continues to receive a `bool`; the caller resolves the pointer and passes the concrete value.
- Fully backward-compatible: configs that omit `rewrite_host` get nil → resolved to `false`, same behaviour as the current `bool` zero value.
- No changes to the router, proxy, watcher, or TLS packages.
