## Why

The current single-port `.env` key=value config cannot express multiple routes, TLS settings per route, or custom certificate paths. Developers who need to proxy both HTTP (port 80) and HTTPS (port 443) simultaneously — or run several routes with different upstreams from one process — must launch multiple dev-proxy instances or use the incomplete JSON `-routes` CLI flag. A YAML config file with a clear separation between server-level concerns (ports, TLS, redirect) and route-level concerns (proxy targets, CORS, static overrides) gives each developer a single readable file that mirrors how production proxies are configured.

## What Changes

- Replace `.env` key=value configuration with `dev-proxy.yaml` as the primary config format
- Introduce a top-level `server` block that owns port bindings (`listen_ports`), TLS termination (cert/key files or auto self-signed), and HTTP-to-HTTPS redirect (`redirect_http`, default `false`)
- Routes become pure proxy rules — path prefix, upstream, host match, CORS, static dir, rewrite flags — with no port or TLS fields
- Custom SSL certificate loading from disk at the server level; falls back to auto-generated self-signed cert when no cert files specified
- Fatal error with actionable message when a listen port is already bound by another process
- Fatal error with full certificate validation details when cert/key files are missing or invalid
- Hot reload the YAML config file via fsnotify (same mechanism as current .env watching)

## Capabilities

### New Capabilities

- `multi-port-yaml-config`: Load configuration from `dev-proxy.yaml` with a server block (ports, TLS, redirect) and a routes array (proxy rules); bind one listener per listen port sharing a single router
- `custom-tls-certs`: Load user-provided TLS certificates from disk (`cert_file` / `key_file`) at the server level; validate PEM format and key pairing at startup

### Modified Capabilities

_(none — the existing `env-driven-config` capability is replaced by YAML-based config with hot reload)_

## Impact

- **New files:** `internal/config/yaml.go` (YAML parser + struct definitions), test fixture `dev-proxy.yaml`, new spec files under `openspec/changes/multi-port-yaml-config/specs/`
- **Modified files:** `cmd/dev-proxy/main.go` (config loading, watcher target, server startup, redirect middleware, error handling for port bind and cert validation), `internal/config/config.go` (remove .env parsing, add YAML load + validate), `internal/tls/cert.go` (add disk-load path to CertManager — loads once per server, not per route), `internal/watcher/watcher.go` (watch YAML file instead of .env)
- **New dependency:** `gopkg.in/yaml.v3` for YAML parsing
- **Breaking change:** `.env` file is no longer the config source; users must migrate to `dev-proxy.yaml`
