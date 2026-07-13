## MODIFIED Requirements

### Requirement: Define route groups by host pattern in config
The system SHALL support a top-level `hosts` list in `dev-proxy.yaml`. Each entry has a `match` field (glob pattern, as per enhanced-route-matching), an optional `upstream` field (inline URL or named upstream reference), and a `routes` list. Entries are evaluated in declaration order.

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

## ADDED Requirements

### Requirement: Route inherits host-level upstream when no route upstream is specified
When a route within a `hosts` entry does not specify its own `upstream` field, and the enclosing `hosts` entry has an `upstream` field, the system SHALL use the host-level `upstream` as the effective upstream for that route.

#### Scenario: Route with no upstream inherits host upstream
- **WHEN** a host group specifies `upstream: api-v1` and a route within it omits its own `upstream` field
- **THEN** requests matching that route are forwarded to `api-v1` as if the route had declared `upstream: api-v1`

#### Scenario: Route-level upstream overrides host-level upstream
- **WHEN** a host group specifies `upstream: api-v1` and one of its routes specifies `upstream: api-v2`
- **THEN** requests matching that route are forwarded to `api-v2`, not `api-v1`

#### Scenario: Route with no upstream and no host upstream is a fatal error
- **WHEN** a route within a host group has no `upstream` field and the host group has no `upstream` field
- **THEN** the proxy prints a fatal error identifying the host group and route index, and exits with code 1

#### Scenario: Host upstream works with named upstream references
- **WHEN** the host-level `upstream` references a named upstream key and the named upstream is valid
- **THEN** the route inherits the named upstream's `url`, `rewrite_host`, and `insecure` settings

#### Scenario: Host upstream works with inline URLs
- **WHEN** the host-level `upstream` is an inline URL (contains `://`)
- **THEN** routes inheriting it use that URL directly with the route's own `rewrite_host` and `insecure` fields

### Requirement: Host-level upstream reference is validated at startup
When a `hosts` entry specifies an `upstream` field, the system SHALL validate it using the same rules as route-level upstreams: inline URLs must parse correctly; named references must exist in the `upstreams` map.

#### Scenario: Invalid host upstream name is a fatal error
- **WHEN** a `hosts` entry specifies `upstream: nonexistent` and `nonexistent` is not in the `upstreams` map
- **THEN** the proxy prints a fatal error identifying the host group and exits with code 1

#### Scenario: Host upstream with invalid inline URL is a fatal error
- **WHEN** a `hosts` entry specifies `upstream: "://bad-url"`
- **THEN** the proxy prints a fatal error identifying the host group and the URL, and exits with code 1
