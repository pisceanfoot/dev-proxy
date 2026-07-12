## Context

The current `Config` has a flat `routes []RouteConfig` where each route embeds its upstream URL inline along with `rewrite_host` and `insecure`. The `Router.Match` is a single-pass scan comparing host and path together. `proxy.NewReverseProxy` takes a URL string directly with no path rewriting capability.

The goal is to layer two independent structural improvements onto this foundation without breaking the existing flat-routes path.

## Goals / Non-Goals

**Goals:**
- Top-level `upstreams` map in YAML; routes reference by name or inline URL
- `upstream_path` on routes for path rewriting when forwarding to named upstreams
- Top-level `hosts` ordered list in YAML; two-phase host-then-path matching
- Flat `routes` still works; `hosts` takes precedence when both present (with warning)
- All validation fatal at startup with clear messages

**Non-Goals:**
- Load balancing across multiple upstream instances per name
- Dynamic upstream registration at runtime
- Per-host TLS settings (TLS remains at the server block level)
- Nested host groups or host inheritance

## Decisions

### D1: Upstream resolution: URL vs. name distinguished by `://` presence

**Choice:** When building routes, check if `RouteConfig.Upstream` contains `"://"`. If yes, treat as an inline URL (current behaviour). If no, treat as a name key and look up in `Config.Upstreams`. Fatal error if the name is not found.

```go
func resolveUpstream(rc RouteConfig, upstreams map[string]UpstreamConfig) (url, rewriteHost, insecure, error) {
    if strings.Contains(rc.Upstream, "://") {
        return rc.Upstream, rc.RewriteHost, rc.Insecure, nil
    }
    up, ok := upstreams[rc.Upstream]
    if !ok {
        return "", false, false, fmt.Errorf("unknown upstream %q", rc.Upstream)
    }
    return up.URL, up.RewriteHost, up.Insecure, nil
}
```

**Rationale:** No new field needed on `RouteConfig`. The `://` heuristic is unambiguous — valid upstream names will never contain `://`, and valid URLs always will.

**Alternatives considered:**
- Separate `upstream_ref` field for named references — rejected: two fields for the same logical concept creates confusion
- Always require names; no inline URLs — rejected: breaks existing configs

### D2: `upstream_path` rewriting in `proxy.NewReverseProxy` Director

**Choice:** Extend `NewReverseProxy` to accept `routePrefix string` and `upstreamPath string`. When `upstreamPath` is non-empty, the Director rewrites `req.URL.Path`:

```go
if upstreamPath != "" {
    suffix := strings.TrimPrefix(req.URL.Path, routePrefix)
    if !strings.HasPrefix(suffix, "/") {
        suffix = "/" + suffix
    }
    req.URL.Path = upstreamPath + suffix
    req.URL.RawPath = ""  // clear encoded path to force re-encoding
}
```

`routePrefix` is the route's `PathPrefix` (used to strip the matched prefix). When `upstreamPath` is empty, path is passed through unchanged (existing behaviour).

**Rationale:** Keeps path rewriting inside the proxy package where URL manipulation already lives. No changes required to calling code in `main.go` other than passing the two new parameters.

**Alternatives considered:**
- Path rewriting in the router before handing off to proxy — rejected: mixes routing and forwarding concerns; the proxy should own its own URL construction

### D3: `hosts` as an ordered slice, not a map

**Choice:** `HostGroup` is a struct with `Match string` and `Routes []RouteConfig`. `Config.Hosts` is `[]HostGroup`. YAML:

```yaml
hosts:
  - match: "api.local"
    routes: [...]
  - match: "*.local"
    routes: [...]
```

**Rationale:** Go maps are unordered; a YAML mapping would require `yaml.MapSlice` or custom unmarshalling to preserve declaration order. A slice with an explicit `match` field preserves order naturally and is immediately clear to the reader. Declaration order IS the evaluation order.

**Alternatives considered:**
- `map[string]HostGroup` with a separate `order []string` field — rejected: redundant and error-prone
- `gopkg.in/yaml.v3` `yaml.Node` custom unmarshalling for ordered map — rejected: adds significant complexity for no readability gain

### D4: No fall-through between host groups

**Choice:** Once a host group is selected (first `match` that passes), only that group's routes are evaluated. If none match, 504 is returned immediately without checking subsequent groups.

**Rationale:** Fall-through semantics (try next group if no path matches) would make config behaviour hard to predict — a wildcard group lower in the list could silently catch requests intended for a specific host group that was misconfigured. No-fall-through makes failures visible and intentional.

### D5: Both `hosts` and `routes` defined — hosts wins with warning

**Choice:** If both are present, `hosts` is used for matching. A startup warning is printed: `[dev-proxy] WARNING: both 'hosts' and 'routes' defined; 'routes' is ignored — use 'hosts' only`.

**Rationale:** Silently ignoring `routes` would confuse authors. A fatal error would block migration. A warning allows gradual migration from flat to grouped config.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| `://` heuristic breaks if someone names an upstream with `://` in it | Document that upstream names must be simple identifiers (letters, digits, hyphens); validate at startup |
| `upstream_path` strip logic is off-by-one on trailing slashes | Document behavior: suffix always starts with `/`; strip is `TrimPrefix` not `TrimLeft` |
| `hosts` and `routes` both present silently ignored `routes` surprises users | Startup WARNING printed to stderr; documented as deprecated path |
| No fall-through between host groups surprises users from nginx backgrounds | nginx's `server` blocks also don't fall through; behaviour is consistent |
