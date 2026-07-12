# Spec: Host Groups

## Purpose

Defines a top-level `hosts` list in `dev-proxy.yaml` that groups routes by host pattern, enabling two-phase matching (host then path). Flat `routes` without `hosts` continues to work unchanged.

---

## Requirements

### Requirement: Define route groups by host pattern in config
The system SHALL support a top-level `hosts` list in `dev-proxy.yaml`. Each entry has a `match` field (glob pattern, as per enhanced-route-matching) and a `routes` list. Entries are evaluated in declaration order.

#### Scenario: hosts list parsed correctly
- **WHEN** `dev-proxy.yaml` contains a `hosts` list with two entries, the first matching `api.local` and the second `*.local`
- **THEN** the proxy constructs two host groups in the declared order, each with its own route list

#### Scenario: hosts entry with no routes is a fatal error
- **WHEN** a `hosts` entry has an empty or missing `routes` list
- **THEN** the proxy prints a fatal error identifying the `match` pattern and exits with code 1

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
