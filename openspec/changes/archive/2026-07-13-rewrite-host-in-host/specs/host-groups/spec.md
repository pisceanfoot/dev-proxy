## MODIFIED Requirements

### Requirement: Define route groups by host pattern in config
The system SHALL support a top-level `hosts` list in `dev-proxy.yaml`. Each entry has a `match` field (glob pattern, as per enhanced-route-matching), an optional `upstream` field (inline URL or named upstream reference), an optional `rewrite_host` boolean field, and a `routes` list. Entries are evaluated in declaration order.

#### Scenario: hosts list parsed correctly
- **WHEN** `dev-proxy.yaml` contains a `hosts` list with two entries, the first matching `api.local` and the second `*.local`
- **THEN** the proxy constructs two host groups in the declared order, each with its own route list

#### Scenario: hosts entry with no routes is a fatal error
- **WHEN** a `hosts` entry has an empty or missing `routes` list
- **THEN** the proxy prints a fatal error identifying the `match` pattern and exits with code 1

#### Scenario: hosts entry with a valid upstream field is parsed correctly
- **WHEN** a `hosts` entry specifies `upstream: api-v1` and `api-v1` is defined in the `upstreams` map
- **THEN** the proxy parses the host group successfully and stores the host-level upstream for route resolution

#### Scenario: hosts entry with an inline upstream URL is parsed correctly
- **WHEN** a `hosts` entry specifies `upstream: "http://localhost:3001"`
- **THEN** the proxy parses the host group successfully and uses that URL as the host-level upstream default

#### Scenario: hosts entry with rewrite_host field is parsed correctly
- **WHEN** a `hosts` entry specifies `rewrite_host: true`
- **THEN** the proxy parses the host group successfully and stores the host-level rewrite_host default for route resolution

## ADDED Requirements

### Requirement: Inline-upstream route inherits host-level rewrite_host when not explicitly set
When a route within a `hosts` entry uses an inline upstream URL and does not explicitly set its own `rewrite_host` field, the system SHALL use the host-level `rewrite_host` value as the effective setting for that route. A route that explicitly sets `rewrite_host` (to either `true` or `false`) always uses its own value, overriding the host default.

#### Scenario: Route with no rewrite_host inherits host default true
- **WHEN** a host group specifies `rewrite_host: true` and a route with an inline upstream omits its own `rewrite_host`
- **THEN** the proxy rewrites the Host header on requests matching that route

#### Scenario: Route with no rewrite_host and no host default behaves as false
- **WHEN** neither the route nor the host group specifies `rewrite_host`
- **THEN** the proxy does not rewrite the Host header (same behaviour as today)

#### Scenario: Route explicitly sets rewrite_host true — overrides host false
- **WHEN** a host group specifies `rewrite_host: false` (or omits it) and a route sets `rewrite_host: true`
- **THEN** the proxy rewrites the Host header for that route

#### Scenario: Route explicitly sets rewrite_host false — overrides host true
- **WHEN** a host group specifies `rewrite_host: true` and a route explicitly sets `rewrite_host: false`
- **THEN** the proxy does not rewrite the Host header for that route

#### Scenario: Named-upstream route is not affected by host-level rewrite_host
- **WHEN** a host group specifies `rewrite_host: true` and a route uses a named upstream (e.g., `upstream: api-v1`)
- **THEN** the effective `rewrite_host` for that route comes from the named upstream's definition, not the host default
