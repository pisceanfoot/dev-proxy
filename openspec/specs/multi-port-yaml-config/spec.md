# Spec: multi-port-yaml-config

## Purpose

Describes how dev-proxy loads its configuration from a YAML file and supports multiple simultaneous routes, each with independent port, TLS, and redirect settings.

## Requirements

### Requirement: Load configuration from dev-proxy.yaml
The system SHALL load its entire configuration from a YAML file named `dev-proxy.yaml` (or a path specified via the `-config` CLI flag). The previous `.env` key=value config format is removed.

#### Scenario: Default config file loaded
- **WHEN** no `-config` flag is provided and `dev-proxy.yaml` exists in the current working directory
- **THEN** the proxy loads configuration from that file

#### Scenario: Custom config path via CLI flag
- **WHEN** the user passes `-config /path/to/custom.yaml` on the command line
- **THEN** the proxy loads configuration from the specified path instead of `dev-proxy.yaml`

#### Scenario: Missing config file produces fatal error
- **WHEN** no config file is found at the expected path (default or flag-specified)
- **THEN** the proxy prints a fatal error identifying the missing path and exits with code 1

### Requirement: Define multiple routes in YAML
The system SHALL support an array of route definitions under the `routes` key in the YAML config. Each route specifies its own port, upstream, TLS settings, and per-route options. The proxy binds one HTTP/HTTPS server per route simultaneously.

#### Scenario: Two routes on different ports
- **WHEN** the YAML defines two routes with ports 80 and 443
- **THEN** the proxy starts listening on both `:80` (HTTP) and `:443` (HTTPS or HTTP depending on TLS setting) in the same process

#### Scenario: Route with all options
- **WHEN** a route specifies port, path_prefix, host_match, upstream, tls, cert_file, key_file, redirect_http, cors_allow_origin, static_dir, rewrite_host, and insecure — **THEN** each field is applied to that route's handler chain exactly as configured

#### Scenario: Empty routes array produces single default route
- **WHEN** the `routes` key is absent or empty in the YAML config
- **THEN** the proxy falls back to a single default route using top-level `port`, `upstream`, and other scalar fields (backward-compatible with simple single-route usage)

### Requirement: Per-route TLS configuration
Each route MAY enable TLS termination by setting `tls: true`. When enabled, the route serves HTTPS. When disabled or absent, the route serves plain HTTP.

#### Scenario: TLS route without custom cert auto-generates self-signed
- **WHEN** a route has `tls: true` and no `cert_file` / `key_file` specified
- **THEN** the proxy generates an in-memory self-signed X.509 certificate (ECDSA P-256, 1-year validity) and serves HTTPS with it

#### Scenario: TLS route with custom cert loads from disk
- **WHEN** a route has `tls: true` and valid `cert_file` / `key_file` paths pointing to PEM-encoded files
- **THEN** the proxy loads those files via `crypto/tls.LoadX509KeyPair` and serves HTTPS with them

### Requirement: HTTP-to-HTTPS redirect per route
A TLS-enabled route MAY set `redirect_http: true` to automatically redirect incoming HTTP requests on port 80 (or any non-TLS route) to the corresponding HTTPS URL. Default is `false`.

#### Scenario: Redirect enabled sends 301
- **WHEN** a route has `tls: true`, `redirect_http: true`, and an HTTP request arrives on its non-TLS counterpart port
- **THEN** the proxy responds with status 301 and a Location header pointing to `https://<host>:<tls-port><path>`

#### Scenario: Redirect disabled passes through
- **WHEN** a route has `redirect_http: false` (or unset) and an HTTP request arrives
- **THEN** the request is processed normally through the handler chain with no redirect

### Requirement: Hot reload of YAML config file
The system SHALL watch the active config file for changes using fsnotify and automatically rebuild the route table when the file is modified.

#### Scenario: Config file change triggers reload
- **WHEN** the `dev-proxy.yaml` file is edited and saved while the proxy is running
- **THEN** the proxy re-reads the YAML, validates it, replaces the active router with the new route table, and prints `[dev-proxy] Routes updated`

#### Scenario: Invalid YAML after reload produces fatal error
- **WHEN** a config change introduces invalid YAML syntax or fails validation
- **THEN** the proxy prints the parse/validation error as a fatal message and exits without applying the broken config
