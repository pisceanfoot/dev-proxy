## Context

The codebase currently supports an optional `rewrite_host` toggle at three levels:

1. **Named upstreams** (`UpstreamConfig.RewriteHost bool`)
2. **Routes** (`RouteConfig.RewriteHost *bool`)
3. **Host groups** (`HostGroup.RewriteHost *bool`)

Route and host-group levels use `*bool` so that omission (inherit) is distinguishable from explicit `false`. This was added in the `rewrite-host-in-host` change to reduce repetition. However, the overwhelmingly common case is `true`; making it optional creates friction and the zero-value default (`false`) is usually the *wrong* choice for a reverse proxy.

The proxy already sets `req.Host = u.Host` when `rewriteHost` is true. The only case where `rewriteHost=false` is useful is when the backend must see the original incoming Host header (rare for a dev-proxy use case and better handled by a more complex proxy like nginx).

## Goals / Non-Goals

**Goals:**
- Delete `RewriteHost` from `UpstreamConfig`, `RouteConfig`, and `HostGroup`.
- Make `proxy.NewReverseProxy` unconditionally rewrite the Host header.
- Remove `RewriteHost` from `router.MatchedRoute`.
- Strip all `rewrite_host` logic from `cmd/dev-proxy/main.go` (`resolveUpstream`, `buildMatchedRoute`, `buildHostGroups`).
- Update tests and example config.
- Ensure `go build ./...` and `go test ./...` pass.

**Non-Goals:**
- Preserving backward compatibility for configs containing `rewrite_host`. This is an intentional breaking simplification.
- Adding a new global toggle to disable host rewriting.
- Changing any other proxy behaviour (path rewriting, TLS, CORS, etc.).

## Decisions

### Decision 1: Always rewrite host — no global fallback switch

**Chosen:** Remove the field entirely and always rewrite. `NewReverseProxy` drops the `rewriteHost` parameter and the Director unconditionally sets `req.Host = u.Host`.

**Alternative considered:** Keep a server-level `rewrite_host: false` escape hatch. Rejected — it reintroduces the same complexity we are trying to remove. If a user truly needs the original Host header, dev-proxy is the wrong tool.

### Decision 2: Let YAML unknown-field error serve as the migration signal

**Chosen:** Delete the struct fields entirely. Configs that still contain `rewrite_host:` will fail to load with a clear YAML unmarshal error.

**Alternative considered:** Keep the fields, mark them deprecated, and ignore them. Rejected — silent deprecation lingers forever; a hard break forces a clean config and is acceptable for a dev tool.

## Risks / Trade-offs

- **Breaking change for existing configs**: Any `rewrite_host` keys must be deleted. Mitigation: the error message from `yaml.Unmarshal` will name the unknown field, making the fix obvious.
- **Loss of rare use case**: Users relying on `rewrite_host: false` to preserve the original Host header lose that capability. Mitigation: this is a dev-proxy, not a production load balancer; the trade-off is justified by the massive simplification.

## Migration Plan

1. Delete every `rewrite_host:` line from `dev-proxy.yaml`.
2. Delete every `rewrite_host:` line from any other config files.
3. Restart dev-proxy.

## Open Questions

_(none)_
