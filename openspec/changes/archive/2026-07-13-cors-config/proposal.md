## Why

`cors_allow_origin` is currently only configurable at the route level, forcing every route in a host group to repeat the same value. There is also no guarantee that CORS response headers are sent on all HTTP methods (OPTIONS, GET, POST, PUT, DELETE, etc.) rather than only on preflight requests.

## What Changes

- Add `cors_allow_origin` field to `HostGroup` config so a single CORS origin can be set for all routes in that host.
- Route-level `cors_allow_origin` overrides the host-level value, allowing per-route exceptions.
- CORS response headers (`Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers`) are sent on every request that carries an `Origin` header, regardless of HTTP method — including GET, POST, PUT, DELETE, and OPTIONS (both preflight and simple).
- Requests without an `Origin` header continue to receive no CORS headers (standard browser behaviour).

## Capabilities

### New Capabilities

- `cors-host-level-config`: Ability to set `cors_allow_origin` on a host group, with route-level override semantics.

### Modified Capabilities

<!-- No existing specs for cors behaviour exist in openspec/specs/ — no delta needed. -->

## Impact

- `internal/config/yaml.go` — add `CORSAllowOrigin string` field to `HostGroup`.
- `cmd/dev-proxy/main.go` — `buildMatchedRoute` must inherit the host-level CORS value when the route has none, and use the route value when it does.
- `internal/cors/cors.go` — verify (and fix if needed) that `addHeaders` is called for every HTTP method with an `Origin` header, not only for preflight OPTIONS.
- `internal/router/router.go` — no struct changes required; `CORSConfig` is already method-agnostic.
- Config documentation / sample `dev-proxy.yaml` — illustrate host-level `cors_allow_origin`.
- No external dependencies added.
