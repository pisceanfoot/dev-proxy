## Context

The current dev-proxy uses a single `DEV_PROXY_PORT` env var plus CLI flags to configure one route. The `-routes` JSON flag is an unimplemented TODO stub. There is no mechanism for multiple simultaneous port bindings, per-route TLS control, or custom certificate loading. The `internal/tls/cert.go` module only generates self-signed certs.

The project targets developers who need to mirror production-like setups locally — e.g., serving HTTPS on :443 while proxying HTTP on :80, or running multiple services behind one proxy with different upstreams per route.

## Goals / Non-Goals

**Goals:**
- Replace `.env` key=value config with `dev-proxy.yaml` separating server-level concerns (ports, TLS, redirect) from route-level concerns (proxy targets, CORS, static overrides)
- Load custom TLS certificates from PEM files at the server level; fall back to self-signed otherwise
- Fatal errors with actionable messages for missing certs, invalid PEM, port conflicts
- HTTP-to-HTTPS redirect as a server-level setting (`redirect_http`, default `false`)
- Hot reload the YAML config file via fsnotify (same mechanism as current .env watching)

**Non-Goals:**
- Support for PKCS#12 / PFX certificate formats
- Automatic cert renewal or Let's Encrypt integration
- SNI-based multi-cert routing on a single port
- Backward compatibility with `.env` key=value format
- CLI flag overrides per-route (flags remain top-level only)
- Multiple independent server blocks (single server; extend later if needed)

## Decisions

### D0: Server block owns ports and TLS; routes are pure proxy rules

**Choice:** The YAML config has two top-level sections: `server` (listen_ports, tls, redirect_http) and `routes` (path_prefix, upstream, host_match, cors, static_dir). A single server binds one listener per listen port. All routes share the same router behind those listeners. TLS termination happens at the server layer — there is no per-route TLS config.

```yaml
server:                              # ← ports + TLS + redirect
  listen_ports: [80, 443]
  tls:
    cert_file: ./certs/cert.pem      # omit → self-signed
    key_file: ./certs/key.pem
  redirect_http: true                # :80 → https://host:443

routes:                              # ← pure proxy rules, no port/TLS
  - path_prefix: "/"
    upstream: http://localhost:3000
    rewrite_host: true
```

Architecture:
```
┌─────────────────────────────────────────────────────┐
│                   Server (one process)                │
│                                                       │
│  :80 (HTTP)  ──▶ redirect or pass-through             │
│  :443 (TLS)  ──▶ tls.Config from server block         │
│                    │                                  │
│                    ▼                                  │
│              Shared Router (first-match)               │
│                /api → upstream A                       │
│                /app → upstream B                       │
│                /static → static dir C                  │
└─────────────────────────────────────────────────────┘
```

**Rationale:** In production proxies (nginx, Caddy, Traefik), TLS and ports are server/vhost concerns; routes are path-based forwarding rules. Separating them matches developer mental models and eliminates the per-route `tls`/`cert_file`/`redirect_http` fields that don't make sense when one cert serves all paths on a port.

**Alternatives considered:**
- Per-route TLS (each route picks its own cert) — rejected: most dev setups use one cert for everything; adds complexity without value
- Multiple independent server blocks — rejected: single-server covers 95% of dev cases; extend later if multi-vhost is needed

## Decisions

### D1: YAML config file replaces .env entirely

**Choice:** `dev-proxy.yaml` is the sole config source. The `-config` flag overrides the filename; default is `dev-proxy.yaml` in CWD. The `.env` parser (`internal/config/env.go`) and all `DEV_PROXY_*` env vars are removed.

**Rationale:** Key=value format cannot express arrays, nested objects, or per-route settings. YAML is native to Go via `gopkg.in/yaml.v3`, readable for humans, and widely used in dev tooling (docker-compose, k8s, etc.). Removing `.env` avoids dual-config confusion.

**Alternatives considered:**
- Keep `.env` with extended syntax (`ROUTE_1_PORT=80`) — rejected: fragile, non-standard, hard to validate
- JSON config — rejected: no comments, verbose for humans; YAML is a superset of readability for this use case
- CLI flags only — rejected: impractical for multi-route configs

### D2: CertManager loads once per server, shared by all routes

**Choice:** `CertManager.GetOrGenerate()` loads the certificate **once at startup** (or on config reload) based on the server block's `cert_file`/`key_file`. The loaded cert is cached under a single key (`"server"`). All listeners and routes share this one `tls.Certificate`. When no cert files are specified, generate one self-signed cert for the whole server.

```
startup:
  if server.tls.cert_file != "":
      cp = LoadX509KeyPair(cert_file, key_file)   // validate PEM, keys match
  else:
      cp = generateSelfSigned()                     // ECDSA P-256, 1 year
  CertManager.cache["server"] = cp

startServers(listen_ports):
  for each port in listen_ports:
      cert = CertManager.cache["server"]            // shared across all ports
      configure tls.Config with cert
```

The `CertPair` struct is unchanged. The downstream code in `main.go:startServers()` iterates over `listen_ports` instead of over routes, building one `http.Server` per port — each sharing the same certificate from the cache.

**Rationale:** One server = one identity. All ports on that server present the same cert. This matches how nginx/caddy work and eliminates per-route cert logic entirely. The cache key is a single string (`"server"`), not a map of port→cert or path→cert.

### D3: Port-in-use detection via error string matching

**Choice:** Wrap `srv.ListenAndServe()` / `srv.ListenAndServeTLS("")` errors. When the error message contains "address already in use", print a formatted fatal message with the port number and platform-appropriate diagnostic commands (`lsof -i :<port>` on macOS, `netstat -an | grep <port>` as fallback).

```go
if err := srv.ListenAndServeTLS("", ""); err != nil {
    if strings.Contains(err.Error(), "address already in use") {
        log.Fatalf("[dev-proxy] FATAL: port %d is already in use — "+
            "another process is bound to it. Run 'lsof -i :%d' or "+
            "'netstat -an | grep %d' to find the conflicting process.",
            route.LocalPort, route.LocalPort, route.LocalPort)
    }
    log.Fatalf("HTTPS server error: %v", err)
}
```

**Rationale:** Go's `net.Listen` returns a wrapped syscall error. The string "address already in use" is the consistent user-facing message across platforms. Using `strings.Contains` rather than `errors.Is` with a sentinel avoids adding platform-specific error types.

### D4: HTTP-to-HTTPS redirect as middleware wrapper

**Choice:** Add a `redirectMiddleware` that wraps the handler chain for non-TLS routes when the corresponding TLS route has `redirect_http: true`. The middleware checks if the request is plain HTTP (no TLS), constructs an HTTPS URL with the target port, and returns 301.

```
buildHandler(route):
  base = upstream proxy OR static handler
  if route.StaticDir != "": base = static.Serve(...)
  if route.CORS.Enabled:    base = cors.Middleware(base, ...)
  if route.redirect_http && !route.TLSEnabled:
      base = redirectMiddleware(base, targetPort)   // only on non-TLS routes
  return loggingMiddleware(base, &route)
```

The redirect constructs the Location header from `r.Host` (preserving the original Host header) and the configured TLS port. If no TLS route exists for this upstream, the redirect is skipped silently.

**Rationale:** Middleware wrapping keeps the change localized to `buildHandler`. Using `r.Host` preserves whatever hostname the client used rather than hardcoding localhost. 301 (permanent) is appropriate for dev — developers want browsers to remember the HTTPS URL.

### D5: Cert files not watched for hot reload

**Choice:** Only the YAML config file is fsnotify-watched. Certificate files on disk are loaded once at startup/reload time and cached in memory via CertManager. Changing a cert requires restarting or editing the YAML (which triggers reload, which re-loads the cert from disk).

**Rationale:** Cert changes are rare events (unlike .env/config edits). Watching cert files adds complexity for negligible benefit in a dev tool. The in-memory cache already survives config reloads — if a user changes a cert and saves the YAML, the next reload picks up the new file automatically without needing a separate watcher.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| `gopkg.in/yaml.v3` adds a dependency | It's the standard Go YAML library; small transitive dep footprint (~1MB compiled). Already used by virtually every Go project that needs YAML. |
| Users expect `.env` to still work after migration | Fatal error on missing YAML file with clear message telling user to create `dev-proxy.yaml`. No silent fallback. |
| Custom cert loaded from disk could be modified while proxy is running | CertManager caches the parsed certificate; in-flight TLS connections keep using the old cert. New connections after reload get the new one. Not a security issue — just expected behavior. |
| Port 80 bind fails with permission error on macOS (not "address already in use") | The fatal message covers both cases generically ("port X is already in use or cannot be bound"). On privileged ports, suggest running with `sudo` or changing the port. |
| YAML validation catches only syntax errors, not semantic ones (e.g., cert file missing) | Validation runs in two phases: parse YAML → validate structure (ports, URLs) → load certs → validate PEM/files. Each phase produces a fatal error with full context before proceeding. |

## Migration Plan

1. Add `gopkg.in/yaml.v3` to `go.mod`
2. Create `internal/config/yaml.go` with new struct definitions and YAML parser
3. Update `cmd/dev-proxy/main.go`: replace `config.Load()` call site, update watcher target from `.env` to YAML file path
4. Extend `internal/tls/cert.go::GetOrGenerate` with disk-load branch
5. Add port-in-use error detection in `startServers()` goroutines
6. Add `redirectMiddleware` and wire into `buildHandler()`
7. Generate `dev-proxy.yaml` test fixture for development
8. Remove `internal/config/env.go` (the .env parser)

## Open Questions

1. Should the config file extension be flexible (`.yaml`, `.yml`) or locked to one? → Locked to `.yaml` for consistency; the `-config` flag accepts any path.
2. Should top-level scalar fields (`port`, `upstream`) still exist as shorthand when `routes` is empty? → Yes, for backward-compatible single-route usage patterns.
