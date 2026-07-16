# dev-proxy

A lightweight, config-driven HTTP/HTTPS reverse proxy with hot-reload, TLS termination, URL rewriting, CORS preflight handling, and environment variable interpolation.

## Quick Start

```bash
# Install from source
go install github.com/opendev-dev/dev-proxy/cmd/dev-proxy@latest

# Or build locally
make build   # produces bin/dev-proxy

# Run with default config (dev-proxy.yaml)
./bin/dev-proxy

# Run with a custom config file
./bin/dev-proxy -config my-config.yaml
```

## CLI Usage

```
Usage of dev-proxy:
  -config string
        Path to YAML config file (default "dev-proxy.yaml")
  -env-file string
        Path to .env file for config variable interpolation (default ".env")
```

### Flags

| Flag        | Type   | Default          | Description                                                                                      |
| ----------- | ------ | ---------------- | ------------------------------------------------------------------------------------------------ |
| `-config`   | string | `dev-proxy.yaml` | Path to the YAML configuration file.                                                             |
| `-env-file` | string | `.env`           | Path to a `.env` file whose variables are available for `${VAR}` interpolation in config values. |

### Examples

```bash
# Run with default settings
dev-proxy

# Use a custom config and env file
dev-proxy -config prod.yaml -env-file .env.production

```

## Configuration

The proxy is configured entirely through a YAML file. The default path is `dev-proxy.yaml`; specify another with `-config`.

### Top-Level Structure

```yaml
log:
  level: error # "error" | "info" | "debug" (default: "info")

server:
  listen_ports: [8080, 8443]
  tls:
    enabled: true # omit cert_file/key_file to auto-generate a self-signed cert
    cert_file: ./certs/cert.pem
    key_file: ./certs/key.pem

upstreams: # named upstream definitions (optional)
  api-v1:
    url: ${API_V1_URL:-http://localhost:3001}
    rewrite_host: true
    insecure: false
    cors_allow_origin: "*"

hosts: # host-grouped routes — evaluated in order, first match wins
  - match: "api.local"
    upstream: api-v1
    rewrite_host: true
    cors_allow_origin: "*"
    routes: [...]
```

### Log Level

Controls verbosity of startup and request logging.

| Value             | Description                              |
| ----------------- | ---------------------------------------- |
| `error` (default) | Errors only; silent in normal operation. |
| `info`            | Startup summary plus errors.             |
| `debug`           | Per-request trace output.                |

```yaml
log:
  level: debug
```

### Server

Defines which ports the proxy listens on and optional TLS / CORS settings.

#### listen_ports

Required. At least one port in range 1–65535.

```yaml
server:
  listen_ports: [8080, 443]
```

#### tls

Optional. When present, the proxy terminates TLS on its listener(s).

| Field       | Required           | Description                                                                          |
| ----------- | ------------------ | ------------------------------------------------------------------------------------ |
| `enabled`   | —                  | Set to `true`.                                                                       |
| `cert_file` | if `key_file` set  | Path to a PEM certificate file. Omit to auto-generate a self-signed cert at startup. |
| `key_file`  | if `cert_file` set | Path to the matching PEM private key file.                                           |

```yaml
server:
  tls:
    enabled: true
    # cert_file and key_file omitted → self-signed cert generated automatically
```

### Upstreams

Named upstream definitions referenced by routes below. Each entry has a key (the name) and configuration.

| Field          | Default       | Description                                                                                                                      |
| -------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------- |
| `url`          | _(required)_  | Base URL of the upstream service. Supports `${VAR}` and `${VAR:-default}` interpolation from `.env` or OS environment variables. |
| `rewrite_host` | default(true) | If `true`, sets the `Host` header on proxied requests to match the inbound host.                                                 |
| `insecure`     | `false`       | If `true`, skips TLS certificate verification for HTTPS upstreams.                                                               |

```yaml
upstreams:
  api-v1:
    url: ${API_V1_URL:-http://localhost:3001}
    rewrite_host: true
    insecure: false
    cors_allow_origin: "*"

  web:
    url: https://web.example.com
    rewrite_host: true
```

#### Environment Variable Interpolation

Values in the config support two interpolation syntaxes, sourced from `.env` files and OS environment variables (OS env takes precedence):

| Syntax            | Behavior                                             |
| ----------------- | ---------------------------------------------------- |
| `${VAR}`          | Replaced with the value of `VAR`. Errors if not set. |
| `${VAR:-default}` | Uses `default` if `VAR` is unset or empty.           |

Example `.env`:

```
API_V1_URL=http://localhost:3001
API_V2_URL=https://preprod.api.internal
WEB_URL=http://localhost:3003
```

Quick environment switch without editing the config file:

```bash
dev-proxy --env-file .env.preprod
```

### Hosts

Host-grouped routes evaluated in order. The first `match` that matches the request host wins; within a group, the first matching route wins. No fall-through between groups.

#### Host Group Fields

| Field          | Required | Description                                                                                                  |
| -------------- | -------- | ------------------------------------------------------------------------------------------------------------ |
| `match`        | yes      | Hostname or glob pattern (e.g. `"api.local"`, `"*.local"`, `"*"`).                                           |
| `upstream`     | no       | Default upstream for routes in this group that don't specify one. Can be a named upstream key or a full URL. |
| `rewrite_host` | no       | Host-level default; inline routes without their own `rewrite_host` inherit this value.                       |
| `routes`       | yes      | List of route entries within this host group.                                                                |

```yaml
hosts:
  - match: "api.local"
    upstream: api-v1
    rewrite_host: true
    routes: [...]
```

#### Glob Matching

- `"*.local"` matches any subdomain ending in `.local` (e.g. `app.local`, `staging.local`).
- `"*"` is a catch-all that matches every host.

### Routes

A route defines how to forward or transform an incoming request within its host group.

#### Route Matching Fields

Exactly one of these should be set per route:

| Field         | Description                                                                 | Example       |
| ------------- | --------------------------------------------------------------------------- | ------------- |
| `path_exact`  | Exact path match (case-sensitive). Only this exact path triggers the route. | `"/health"`   |
| `path_prefix` | Prefix match — any path starting with this prefix.                          | `"/api/v2"`   |
| `path_regex`  | Regex match against the request path.                                       | `"^/api/v1/"` |

#### Route Action Fields

| Field           | Description                                                                                                                           |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `upstream`      | Override the host-level default upstream for this route. Can be a named upstream key or a full URL (supports `${VAR}` interpolation). |
| `upstream_path` | Path prefix to replace the matched portion when forwarding. Mutually exclusive with `url_rewrite`.                                    |
| `methods`       | Restrict this route to specific HTTP methods (e.g. `["GET", "POST"]`).                                                                |

#### URL Rewrite

Regex-based path rewriting using capture groups. Mutually exclusive with `upstream_path` on the same route.

```yaml
- path_prefix: "/api/v2"
  upstream: api-v2
  url_rewrite:
    match: "^/api/v2/(.*)"
    replace: "/v2/$1"
# Incoming /api/v2/users → forwarded as /v2/users
```

| Field     | Required | Description                                                                |
| --------- | -------- | -------------------------------------------------------------------------- |
| `match`   | yes      | Regex pattern to match against the request path.                           |
| `replace` | yes      | Replacement string; `$1`, `$2`, etc. refer to capture groups from `match`. |

#### Static Files

Instead of proxying, serve files from a local directory:

```yaml
- path_prefix: "/"
  static_dir: "./public"
```

#### CORS Per-Route

Override the server-level CORS settings for a specific route:

```yaml
- path_prefix: "/"
  cors_allow_origin: "*"
```

### Complete Example

```yaml
log:
  level: debug

server:
  listen_ports: [8080, 8443]
  tls:
    enabled: true

upstreams:
  api-v1:
    url: ${API_V1_URL:-http://localhost:3001}
    rewrite_host: true
  api-v2:
    url: ${API_V2_URL:-http://localhost:3002}
    rewrite_host: true

hosts:
  - match: "api.local"
    upstream: api-v1
    routes:
      - path_exact: "/health"

      - path_regex: "^/api/v1/"
        upstream_path: "/v1"

      - path_prefix: "/api/v2"
        upstream: api-v2
        upstream_path: "/v2"

      - path_prefix: "/"

  - match: "*.local"
    routes:
      - path_prefix: "/"
        upstream: ${HTTPBUN_URL}

  - match: "*"
    upstream: "https://httpbun.com"
    rewrite_host: true
    routes:
      - path_prefix: "/"
```

## Hot Reload

By default, dev-proxy watching the config file and reloads when it changes.
No restart is needed — updates take effect immediately for in-flight requests.
