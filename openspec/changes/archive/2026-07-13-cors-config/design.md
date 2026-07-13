## Context

CORS support in dev-proxy is handled by `internal/cors/cors.go` and wired in `cmd/dev-proxy/main.go`. The `cors_allow_origin` field exists only on `RouteConfig`; there is no host-level CORS setting in `HostGroup`. The middleware already handles all HTTP methods that carry an `Origin` header (preflight OPTIONS gets a short-circuit 204; all other methods have headers injected and are forwarded). The gap is:

1. No host-level `cors_allow_origin` field, so operators must repeat the value on every route.
2. No inheritance/override chain: host → route.

## Goals / Non-Goals

**Goals:**
- Add `cors_allow_origin` to `HostGroup` config.
- Implement route-level override: if a route has a non-empty `cors_allow_origin`, use it; otherwise fall back to the host group's value.
- Ensure CORS headers are present on every response that includes an `Origin` request header, across all HTTP methods (GET, POST, PUT, DELETE, OPTIONS — both preflight and non-preflight).
- Add tests covering host-level inheritance and route-level override.

**Non-Goals:**
- New CORS config fields beyond `cors_allow_origin` (e.g. `cors_allow_headers`, `cors_allow_credentials`).
- Per-method CORS configuration.
- Wildcard sub-domain origin matching.
- Changes to the `Access-Control-Allow-Methods` or `Access-Control-Allow-Headers` hardcoded values.

## Decisions

### 1. Where to store host-level CORS in the config struct

**Decision:** Add `CORSAllowOrigin string \`yaml:"cors_allow_origin"\`` to `HostGroup` in `internal/config/yaml.go`, mirroring the existing field on `RouteConfig`.

**Rationale:** Keeps the config shape symmetric. Operators already know the field name from route-level usage; no new vocabulary needed.

**Alternative considered:** A nested `cors:` block on `HostGroup`. Rejected — over-engineered for a single string field; would require a new config struct and a larger migration surface.

### 2. Inheritance resolution point

**Decision:** Resolve the effective `cors_allow_origin` in `buildMatchedRoute` inside `cmd/dev-proxy/main.go`. The function already receives both the `HostGroup` and the `RouteConfig`. The rule is: use the route value if non-empty; otherwise use the host value.

```
effective = route.CORSAllowOrigin
if effective == "" {
    effective = host.CORSAllowOrigin
}
```

**Rationale:** All other per-route config resolution (e.g. `RewriteHost` inheritance) happens in `buildMatchedRoute`. Keeping the same pattern avoids spreading config logic across packages.

**Alternative considered:** Resolve in `HostGroup.resolveRoute()` or a new helper. Rejected — no such helper exists yet; introducing it would be a larger refactor than the problem warrants.

### 3. Ensure headers on all methods

**Decision:** Audit `cors.Middleware` to confirm `addHeaders` is called for every request with an `Origin` header. Current code: OPTIONS+`Access-Control-Request-Method` → preflight (204, headers included); all other origins → `addHeaders` + forward. This already covers GET, POST, PUT, DELETE, and plain OPTIONS. No code change is needed — only a test to lock in the behaviour.

**Rationale:** The current implementation is already correct. The test prevents future regressions.

## Risks / Trade-offs

- **Operator confusion on precedence** — If both host and route set `cors_allow_origin`, only the route value is used. This is standard override semantics but should be documented.  Mitigation: update the sample config comment.
- **Flat-mode routes (no host group)** — The flat `Routes` list has no host group, so host-level CORS does not apply. This is expected and consistent with how other host-level fields work. Mitigation: no action needed; document clearly.

## Migration Plan

1. Add field to `HostGroup` — backward compatible (empty by default).
2. Update `buildMatchedRoute` to pass the host value through.
3. Add tests.
4. No config file migration required; existing configs continue to work unchanged.
5. Rollback: revert `yaml.go` and `main.go`; no data changes.
