## Context

`dev-proxy.yaml` supports a `hosts` list where host groups can now share an `upstream` default across routes (added in the `upstream-in-host` change). The `rewrite_host` field on routes suffers the same repetition problem: when all inline-upstream routes in a group should rewrite the Host header, every route must repeat `rewrite_host: true`.

The key implementation challenge is that `rewrite_host` is a boolean. Unlike `upstream` (where an empty string unambiguously means "not set"), a `bool` zero value of `false` cannot be distinguished from an explicitly written `rewrite_host: false`. Correct inheritance semantics require this distinction.

## Goals / Non-Goals

**Goals:**
- Add optional `RewriteHost *bool` to `config.HostGroup`.
- Change `RouteConfig.RewriteHost` from `bool` to `*bool` so "omitted" is distinguishable from "explicitly false".
- Routes with inline upstreams that omit `rewrite_host` inherit the host-level value (nil → host default → false).
- Route-level `rewrite_host` (either `true` or `false` when explicitly written) always wins over the host default.
- Named-upstream routes continue to take `rewrite_host` from the upstream definition — unchanged.
- Full backward compatibility: existing configs that omit `rewrite_host` get nil, which resolves to `false` as before.

**Non-Goals:**
- Changing `UpstreamConfig.RewriteHost` — named upstream definitions own their own `rewrite_host`.
- Adding host-level `insecure` default (parallel feature, out of scope).
- Flat `routes` (non-host-grouped) — no host context to inherit from.

## Decisions

### Decision 1: `*bool` for `RouteConfig.RewriteHost`, not a separate sentinel field

**Chosen:** Change `RouteConfig.RewriteHost bool` to `RouteConfig.RewriteHost *bool`. The YAML library (`gopkg.in/yaml.v3`) unmarshals an absent field to `nil` and an explicit `true`/`false` to a pointer, giving us the three-state distinction we need.

**Alternative considered:** Keep `bool` and add a parallel `RewriteHostSet bool` flag. Rejected — two fields for one semantic is ugly and error-prone; the `*bool` pattern is idiomatic Go.

**Backward compatibility:** Configs that omit `rewrite_host` currently get `false` (zero value). After the change they get `nil`, which resolves to `false` when there is no host default — identical runtime behaviour.

### Decision 2: Inheritance applies only to inline-upstream routes

**Chosen:** When `resolveUpstream` is called for a named upstream, it returns the upstream definition's `RewriteHost bool` value and ignores the host default. The host-level default is resolved before calling `resolveUpstream` only for inline-URL routes.

**Rationale:** Named upstream definitions are the authoritative source for connection settings (`rewrite_host`, `insecure`, `url`). Overriding them via host config would create a confusing two-source-of-truth situation.

### Decision 3: Resolve effective value in `buildMatchedRoute`, pass concrete `bool` to `resolveUpstream`

**Chosen:** `buildMatchedRoute` receives `hostRewriteHost *bool` alongside the existing `hostUpstream string`. It computes the effective `rewriteHost bool` for inline-URL routes (route ptr → host ptr → false) and stores it directly on `effectiveRC` before calling `resolveUpstream`. `resolveUpstream` signature stays unchanged.

**Alternative considered:** Pass `*bool` through into `resolveUpstream`. Rejected — keeps the resolution logic in one place and avoids spreading pointer dereferencing across the call chain.

## Risks / Trade-offs

- **Silent behaviour change for routes omitting `rewrite_host` under a host with default `true`**: Routes that previously defaulted to false now inherit true. This is the desired new behaviour, but operators adding `rewrite_host: true` to a host group should be aware all inline-URL routes in that group are affected. Mitigation: debug logs already print `rewrite_host` via the upstream URL, making the effective value observable.
- **`*bool` in YAML**: Some YAML authors may be surprised that `rewrite_host: false` is now semantically different from omitting it. Mitigation: a YAML comment in `dev-proxy.yaml` can document the inheritance rule.

## Migration Plan

No migration required. The `*bool` change is fully backward compatible — existing configs produce the same runtime behaviour. No restart or config changes needed for deployments not using the new host-level field.

## Open Questions

_(none)_
