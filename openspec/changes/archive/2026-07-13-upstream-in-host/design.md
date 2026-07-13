## Context

`dev-proxy.yaml` supports a `hosts` list where each entry groups routes under a host-match pattern. Currently every route must declare its own `upstream` field. In practice, most host groups proxy all their routes to the same backend — the repetition is noisy and error-prone.

The existing route-resolution path in `main.go` already handles named upstream references and inline URLs via `resolveUpstream`. The `config.HostGroup` struct and the `validateRoute` helper need small additions; no changes are needed to the router, proxy, or watcher packages.

## Goals / Non-Goals

**Goals:**
- Add an optional `upstream` field to `config.HostGroup` (config parse + YAML tag).
- Routes within a host group with no `upstream` of their own inherit the host's `upstream`.
- Route-level `upstream` always wins over host-level `upstream`.
- Validate the host-level upstream at startup using the same rules as route-level upstreams.
- Fatal error if a route ends up with no resolved upstream after inheritance.
- Full backward compatibility — existing configs that specify `upstream` on every route are unaffected.

**Non-Goals:**
- Inheritance for other route fields (e.g., `rewrite_host`, `cors_allow_origin`, `insecure`) — only `upstream`.
- Flat `routes` (non-host-grouped) — no default upstream at the top-level config.
- Runtime override of host upstream without config reload.

## Decisions

### Decision 1: Inherit only `upstream`, not `rewrite_host` / `insecure`

**Chosen:** Inherit only `upstream` (the reference string). The resolved `rewrite_host` and `insecure` values come from the named upstream's definition when a name is used, or from the route's own fields when an inline URL is used — exactly the same as today.

**Alternative considered:** Inherit all connection settings from the host level. Rejected — it introduces a more complex precedence chain that is harder to reason about and harder to document.

### Decision 2: Apply inheritance in `buildMatchedRoute`, not in `validateRoute`

**Chosen:** Pass an `effectiveUpstream` string (= `route.Upstream` if non-empty, else `hostUpstream`) into the existing `resolveUpstream` / `buildMatchedRoute` logic. Validation receives the same effective value.

**Alternative considered:** Mutate `RouteConfig.Upstream` in place during validation. Rejected — it blurs the line between validation (read-only) and compilation (transforms).

**Concretely:**
- `config.HostGroup` gains `Upstream string \`yaml:"upstream"\``.
- `validateRoute` gains a second string parameter `hostUpstream string`; it uses `hostUpstream` when `r.Upstream == ""`.
- `buildMatchedRoute` gains a third string parameter `hostUpstream string`; it constructs `effectiveRC` with `Upstream` resolved before calling `resolveUpstream`.

### Decision 3: No new struct; reuse `RouteConfig` with effective upstream

Rather than creating a new `EffectiveRouteConfig` type, the caller constructs a shallow copy of `RouteConfig` with `Upstream` filled in before passing to `resolveUpstream`. This keeps `resolveUpstream` unchanged.

## Risks / Trade-offs

- **Config ambiguity risk**: A user may not realize a route is inheriting its upstream silently. Mitigation: the debug log already prints `upstream=<value>` per request, making the effective upstream visible.
- **Validation message clarity**: Fatal errors for missing upstreams now need to mention whether the host default was checked. Mitigation: the error message should identify the host group index and route index explicitly.
- **Flat routes unaffected**: The new field only applies inside `hosts` entries; flat `routes` have no host context to inherit from. This asymmetry is intentional but should be documented in the YAML comment.

## Migration Plan

No migration required. The new `upstream` field on `hosts` entries is optional. All existing configs continue to work without modification. No restart required beyond the normal config reload for any changed proxy.

## Open Questions

_(none)_
