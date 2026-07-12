## ADDED Requirements

### Requirement: Define named upstreams in config
The system SHALL support a top-level `upstreams` map in `dev-proxy.yaml`. Each key is a name; each value is an upstream definition with `url`, `rewrite_host`, and `insecure` fields. Names must be unique and non-empty.

#### Scenario: Named upstream defined and valid
- **WHEN** `dev-proxy.yaml` contains an `upstreams` map with entry `api: { url: "http://localhost:3001", rewrite_host: true }`
- **THEN** the proxy parses it successfully and makes `api` available for route references

#### Scenario: Named upstream with missing URL is a fatal error
- **WHEN** an upstream entry has an empty or missing `url` field
- **THEN** the proxy prints a fatal error identifying the upstream name and exits with code 1

#### Scenario: Named upstream with invalid URL is a fatal error
- **WHEN** an upstream entry has a `url` that fails URL parsing
- **THEN** the proxy prints a fatal error with the upstream name, the invalid URL, and the parse error, and exits with code 1

### Requirement: Route upstream field accepts name reference
A route's `upstream` field SHALL accept either an inline URL (containing `://`) or the name of an entry in the `upstreams` map. When a name is used, the route inherits that upstream's `url`, `rewrite_host`, and `insecure` settings.

#### Scenario: Route references named upstream by key
- **WHEN** a route has `upstream: api` and `api` is defined in the `upstreams` map
- **THEN** the proxy forwards requests to the upstream's URL using its `rewrite_host` and `insecure` settings

#### Scenario: Route with inline URL continues to work unchanged
- **WHEN** a route has `upstream: "http://localhost:3000"`
- **THEN** the proxy forwards using that URL directly, using the route's own `rewrite_host` and `insecure` fields

#### Scenario: Route references undefined upstream name is a fatal error
- **WHEN** a route's `upstream` field is a name (no `://`) that does not match any key in `upstreams`
- **THEN** the proxy prints a fatal error identifying the route and the unknown name, and exits with code 1

### Requirement: Route may override upstream path via upstream_path
When a route references a named upstream and specifies `upstream_path`, the system SHALL rewrite the forwarded path before sending to the upstream. The route's matched path prefix is stripped from the request path; `upstream_path` is prepended to the remainder.

#### Scenario: upstream_path rewrites forwarded path
- **WHEN** a route has `path_prefix: "/v2"`, `upstream: api`, and `upstream_path: "/api/v2"`, and a request arrives at `/v2/users/123`
- **THEN** the proxy forwards to the upstream at path `/api/v2/users/123`

#### Scenario: upstream_path with no remaining path
- **WHEN** a route has `path_prefix: "/health"`, `upstream: api`, `upstream_path: "/healthz"`, and a request arrives at `/health`
- **THEN** the proxy forwards to the upstream at path `/healthz`

#### Scenario: upstream_path not set forwards original path
- **WHEN** a route references a named upstream but has no `upstream_path`
- **THEN** the request path is forwarded unchanged to the upstream
