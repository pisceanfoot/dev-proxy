## Why

Developers frequently need to route local development traffic to different upstream hosts (local services, remote APIs, staging environments) with flexible routing rules. Existing tools are either too rigid (single-port forward only), lack developer-friendly features (hot reload, .env watching, static file overrides), or require complex configuration. A lightweight Go-based dev proxy that handles these concerns out of the box eliminates context switching and speeds up local development workflows.

## What Changes

- A CLI tool (`dev-proxy`) written in Go that acts as a reverse proxy for local development
- Port forwarding from a local port to any upstream host (IP, DNS name, or another local service)
- TLS/SSL termination and passthrough support for HTTPS backends
- CORS header injection and configuration per route
- Request/response review mode — intercept traffic for inspection before forwarding
- Local static file serving that overrides upstream responses on match
- Host header rewriting per route
- `.env` file watching with automatic reload of environment-driven config changes
- Hot reload of proxy rules without restarting the process

## Capabilities

### New Capabilities

- `proxy-routing`: Route local ports to arbitrary upstream hosts (IP, DNS) with configurable paths and host headers
- `tls-termination`: Terminate TLS at the proxy or passthrough to upstream HTTPS backends
- `cors-handling`: Inject and configure CORS headers per route
- `request-review`: Intercept requests/responses for manual review before forwarding
- `static-overrides`: Serve static files from disk, overriding upstream responses on path match
- `env-driven-config`: Watch `.env` files and hot-reload proxy configuration on change

### Modified Capabilities

_(none — this is a greenfield project)_

## Impact

- New Go module with minimal dependencies (net/http, crypto/tls, os/signal, fsnotify)
- Single binary output via `go build`
- No external runtime dependencies — runs anywhere Go compiles to
- Configuration driven by CLI flags and/or `.env` file
- Affects: developer workstation tooling only; no server-side or API changes
