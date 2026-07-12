## Context

This is a greenfield project — no existing codebase beyond the openspec scaffolding. The tool must be a single Go binary with minimal dependencies, targeting macOS/Linux/Windows developers who need flexible local port forwarding with developer-experience features (hot reload, .env watching, static overrides).

## Goals / Non-Goals

**Goals:**
- Single-port-to-arbitrary-upstream routing with path and host header rewriting
- TLS termination at the proxy for HTTPS upstreams; optional client-side TLS serving
- Per-route CORS header injection
- Request/response review mode for traffic inspection
- Static file override on matching URL paths
- `.env` file watching with live config reload without restart
- Sub-second startup time; low memory footprint

**Non-Goals:**
- Load balancing across multiple upstreams
- WebSocket proxying (out of scope v1)
- Authentication/middleware plugins
- GUI or web dashboard
- DNS server — only resolves via Go's net/http default dialer

## Decisions

### D1: Single binary, standard library first

**Choice:** Use `net/http`, `crypto/tls`, and `os/signal` as the core. Add `fsnotify` for file watching only.

**Rationale:** The proxy surface area is small (HTTP request routing + TLS). Standard library handles 90% of the work. Adding a framework (chi, gin) introduces unnecessary dependencies and startup overhead for a dev tool.

**Alternatives considered:**
- `gin`/`chi` — rejected: adds dependency weight; the project needs only path-based routing
- `httputil.ReverseProxy` — accepted as the core forwarding mechanism; it handles connection pooling, header management, and streaming responses out of the box

### D2: Rule-based router with priority matching

**Choice:** A slice of `Route` structs evaluated in order. First match wins. Each route has: local port, path prefix, host match (optional), upstream target, and per-route options (CORS, static override, review mode).

```go
type Route struct {
    LocalPort   int
    PathPrefix  string      // "/" matches all
    HostMatch   string      // optional exact host match
    Upstream    string      // "http://localhost:3000" or "https://api.example.com"
    RewriteHost bool        // rewrite outgoing Host header to upstream hostname
    CORS        *CORSConfig // nil = no CORS
    StaticDir   string      // if set, serve files from disk instead of proxying
    ReviewMode  bool        // pause at review point before forwarding
}
```

**Rationale:** Slice-based first-match is simple, predictable, and fast. No trie or regex needed for v1.

### D3: TLS termination with auto-generated self-signed certs

**Choice:** When a route serves HTTPS on the local side, generate an in-memory self-signed cert (using `crypto/x509` + `x509/elliptic`) on first start. Cache it per-route for reuse across reloads. For upstream TLS, use Go's default `http.Transport` with configurable `InsecureSkipVerify`.

**Rationale:** Developers don't want to manage certs manually. Self-signed avoids external tooling (mkcert). InsecureSkipVerify is acceptable for dev environments and can be overridden per-route.

### D4: Hot reload via signal + file watcher

**Choice:** Two triggers for reload:
1. `SIGHUP` / Ctrl+C+restart — manual restart
2. `fsnotify.Watch()` on `.env` (and optionally a YAML/JSON config file) — automatic reload

On reload, rebuild the route table, close old listeners, accept new connections on updated ports. In-flight requests complete on old handlers; new requests use new rules immediately.

**Rationale:** fsnotify is the de facto Go library for this. The dual-trigger approach covers both manual and automatic workflows.

### D5: Review mode via buffered channel + blocking receive

**Choice:** When `ReviewMode` is enabled, the proxy pauses at a review point (before forwarding or after receiving response) and sends the request/response to an in-memory channel. A built-in CLI reviewer (or external tool via stdin/stdout JSON) reads from the channel, inspects, and signals continue/discard/replay.

**Rationale:** Channel-based design fits Go's concurrency model naturally. Keeps review logic decoupled from proxy core.

### D6: Configuration via flags + .env overlay

**Choice:** CLI flags provide explicit overrides (port, upstream). If a flag is unset, fall back to `.env` variables (`DEV_PROXY_PORT`, `DEV_PROXY_UPSTREAM`). Default values for everything else.

```
dev-proxy --port 8080 --upstream http://localhost:3000
# or via .env:
# DEV_PROXY_PORT=8080
# DEV_PROXY_UPSTREAM=http://localhost:3000
```

**Rationale:** Flags take precedence over env, matching Unix conventions. `.env` allows committing defaults without hardcoding.

## Architecture

```
┌─────────────────────────────────────────────┐
│                  dev-proxy                   │
│                                              │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐ │
│  │ Listener │  │ Router   │  │ TLS Mgr   │ │
│  │ (http)   │→ │ (routes) │→ │ (certs)   │ │
│  └──────────┘  └────┬─────┘  └───────────┘ │
│                     │                        │
│         ┌───────────┼───────────┐           │
│         ▼           ▼           ▼           │
│   ┌──────────┐ ┌──────────┐ ┌──────────┐  │
│   │Reverse   │ │Static    │ │ CORS     │  │
│   │Proxy     │ │File Serve│ │ Injector │  │
│   └────┬─────┘ └──────────┘ └──────────┘  │
│        │                                    │
│  ┌─────┴──────────┐                         │
│  │ Review Mode    │ (optional)              │
│  │ (channel+CLI)  │                         │
│  └────────────────┘                         │
│                                              │
│  ┌──────────────────┐                        │
│  │ .env Watcher     │ ← fsnotify            │
│  │ (hot reload)     │                        │
│  └──────────────────┘                        │
└─────────────────────────────────────────────┘
```

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| In-memory self-signed certs not trusted by browsers | Document `--insecure` flag; suggest mkcert for production-like testing |
| fsnotify inotify limit on Linux (default 8192 watches) | Not a concern — watching 1-2 files; mention if config file watch scales |
| Review mode becomes a bottleneck | Make review mode opt-in per-route; default routes skip it entirely |
| .env reload mid-request corrupts state | Rebuild route table atomically (RWMutex); in-flight requests use old rules |
| CORS preflight handling incomplete | Use standard `cors` library only if stdlib approach proves insufficient |

## Open Questions

1. Should the tool support multiple routes in a single config, or one proxy per process? → Leaning toward multi-route via a config file + env fallback for flexibility.
2. Should review mode include response body interception, or request-only? → Request-only for v1; body interception adds complexity (body buffering).
