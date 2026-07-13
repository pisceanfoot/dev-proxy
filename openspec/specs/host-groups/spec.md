# Spec: Host Groups

## Purpose

Defines a top-level `hosts` list in `dev-proxy.yaml` that groups routes by host pattern, enabling two-phase matching (host then path). Flat `routes` without `hosts` continues to work unchanged.

---

## Requirements

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

---

### Requirement: Host-first two-phase matching
When `hosts` is defined, the system SHALL match an incoming request in two phases: first find the earliest host group whose `match` pattern matches the request's Host header, then find the first route within that group whose path criteria pass. If the host phase finds no match, 504 is returned. If the path phase finds no match within the selected group, 504 is returned.

#### Scenario: Host matched, path matched
- **WHEN** a request with `Host: api.local` and path `/v1/users` arrives, and a host group with `match: api.local` contains a route with `path_prefix: "/v1"`
- **THEN** that route handles the request

#### Scenario: Host matched, no path match returns 504
- **WHEN** a request with `Host: api.local` arrives but no route in the `api.local` host group matches the request path
- **THEN** the proxy returns 504 with body `no route matched: host=api.local path=<path>`

#### Scenario: No host match returns 504
- **WHEN** a request's Host header does not match the `match` pattern of any host group
- **THEN** the proxy returns 504 with body `no route matched: host=<host> path=<path>`

#### Scenario: Earlier host group takes precedence
- **WHEN** two host groups both could match the request Host (e.g., `api.local` exact and `*.local` wildcard), and the exact entry appears first
- **THEN** the first group (exact) is selected; the second group is never evaluated

#### Scenario: No fall-through between host groups
- **WHEN** the first matching host group contains no path match for the request
- **THEN** the proxy returns 504 immediately; it does NOT evaluate subsequent host groups

---

### Requirement: Flat routes list remains valid without hosts
When no `hosts` key is present in `dev-proxy.yaml`, the flat `routes` list continues to work as before: all routes are evaluated in order using their individual `host_match` fields (or no host check if `host_match` is empty).

#### Scenario: Config with only flat routes works unchanged
- **WHEN** `dev-proxy.yaml` has a `routes` list and no `hosts` key
- **THEN** the proxy behaves exactly as before — routes evaluated in order, host_match applied per route

#### Scenario: Config with both hosts and routes uses hosts only
- **WHEN** `dev-proxy.yaml` defines both `hosts` and `routes`
- **THEN** the proxy uses only `hosts` for matching; the flat `routes` list is ignored with a startup warning logged

---

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

---

### Requirement: Host-level upstream reference is validated at startup
When a `hosts` entry specifies an `upstream` field, the system SHALL validate it using the same rules as route-level upstreams: inline URLs must parse correctly; named references must exist in the `upstreams` map.

#### Scenario: Invalid host upstream name is a fatal error
- **WHEN** a `hosts` entry specifies `upstream: nonexistent` and `nonexistent` is not in the `upstreams` map
- **THEN** the proxy prints a fatal error identifying the host group and exits with code 1

#### Scenario: Host upstream with invalid inline URL is a fatal error
- **WHEN** a `hosts` entry specifies `upstream: "://bad-url"`
- **THEN** the proxy prints a fatal error identifying the host group and the URL, and exits with code 1

---

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
