## Why

As the number of proxied services grows, the flat `routes` list becomes repetitive: every route that targets the same upstream service must repeat its URL, `rewrite_host`, and `insecure` flags. Similarly, routes for the same virtual host are scattered across the list with no structural grouping, making the config hard to read and the matching order hard to reason about. Defining upstreams once by name and grouping routes by host mirrors how every production proxy (nginx, Caddy, HAProxy) is structured, and is the natural next step after multi-port and enhanced path matching.

## What Changes

- Add a top-level `upstreams` map to `dev-proxy.yaml` — each entry names an upstream with its URL, `rewrite_host`, and `insecure` settings
- A route's `upstream` field accepts either an inline URL (current behaviour) or a named upstream key; routes that reference a name inherit that upstream's connection settings
- A route that references a named upstream MAY specify `upstream_path` to rewrite the forwarded path: the route's matched prefix is stripped from the request path, and `upstream_path` is prepended before forwarding
- Add a top-level `hosts` list to `dev-proxy.yaml` — an ordered list of host groups, each with a `match` pattern (glob, as per enhanced-route-matching) and a nested `routes` list
- Matching order: host groups are evaluated in declaration order; the first host group whose `match` passes is entered; within it, routes are evaluated in order; first path match wins; if no path matches within a matched host group, 504 is returned (no fall-through to subsequent host groups)
- The existing flat `routes` list remains valid for configs without host grouping (treated as a single implicit host group that matches any host)
- **BREAKING**: `RouteConfig.host_match` is unused and ignored when routes are defined under a `hosts` entry; `host_match` only applies to flat `routes`

## Capabilities

### New Capabilities

- `named-upstreams`: Define reusable upstream targets by name; routes reference them by key and optionally override the forwarded path via `upstream_path`
- `host-groups`: Group routes under ordered host match patterns in `dev-proxy.yaml`; host is matched first, then path within the group

### Modified Capabilities

- `enhanced-route-matching`: Route resolution now operates within a host group context; the 504 no-match path is reached when no path matches within the selected host group

## Impact

- **Modified files:** `internal/config/yaml.go` (new `UpstreamConfig`, `HostGroup` structs; extend `RouteConfig` with `upstream_path`; extend `Config` with `Upstreams` and `Hosts`; update `validate` to resolve named upstream refs), `internal/router/router.go` (`MatchedRoute` gains `UpstreamPath`; `Router` gains `MatchHost` + `MatchPath` two-phase logic; `Router` stores host groups), `internal/proxy/proxy.go` (`NewReverseProxy` accepts optional `upstreamPath` and `routePrefix` for path rewriting), `cmd/dev-proxy/main.go` (`buildRoutes` resolves named upstream refs, populates `UpstreamPath`)
- **No new dependencies** — all stdlib
