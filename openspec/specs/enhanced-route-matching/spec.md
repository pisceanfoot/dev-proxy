# Spec: Enhanced Route Matching

## Purpose

Defines how the proxy selects a backend route for an incoming request. Routes are evaluated in order; the first route whose every configured criterion passes is selected. Unmatched requests receive a 504 response.

---

## Requirements

### Requirement: Match route by host glob pattern
The system SHALL evaluate the route's `host_match` field as a glob pattern against the incoming request's `Host` header using `path.Match` semantics. `*` matches any sequence of non-separator characters; `?` matches exactly one character. If `host_match` is empty, the criterion is skipped (always passes).

#### Scenario: Exact hostname match
- **WHEN** a route has `host_match: "api.example.com"` and the request Host header is `api.example.com`
- **THEN** the host criterion passes and matching continues to path criteria

#### Scenario: Wildcard subdomain match with `*`
- **WHEN** a route has `host_match: "*.example.com"` and the request Host is `api.example.com`
- **THEN** the host criterion passes

#### Scenario: Wildcard does not match across dots
- **WHEN** a route has `host_match: "*.example.com"` and the request Host is `api.v2.example.com`
- **THEN** the host criterion fails and the route is skipped

#### Scenario: Single-character wildcard with `?`
- **WHEN** a route has `host_match: "app?.local"` and the request Host is `app1.local`
- **THEN** the host criterion passes

#### Scenario: No host_match set always passes
- **WHEN** a route has no `host_match` field (empty string)
- **THEN** the host criterion is skipped and always passes regardless of the incoming Host header

---

### Requirement: Match route by exact path
The system SHALL support a `path_exact` field on a route. When set, the request path must equal the field value exactly (no prefix logic). This criterion is independent of `path_prefix` and `path_regex`.

#### Scenario: Exact path matches correctly
- **WHEN** a route has `path_exact: "/health"` and the request path is `/health`
- **THEN** the path criterion passes

#### Scenario: Exact path does not match on prefix
- **WHEN** a route has `path_exact: "/health"` and the request path is `/health/check`
- **THEN** the path criterion fails and the route is skipped

#### Scenario: No path_exact set skips criterion
- **WHEN** a route has no `path_exact` field
- **THEN** the exact path criterion is skipped

---

### Requirement: Match route by path prefix
The system SHALL retain the existing `path_prefix` matching behaviour. When `path_prefix` is set, the request path must begin with the field value. An empty `path_prefix` skips this criterion.

#### Scenario: Prefix match passes for longer path
- **WHEN** a route has `path_prefix: "/api"` and the request path is `/api/v1/users`
- **THEN** the prefix criterion passes

#### Scenario: Empty path_prefix skips criterion
- **WHEN** a route has no `path_prefix` field
- **THEN** the prefix criterion is skipped

---

### Requirement: Match route by path regex
The system SHALL support a `path_regex` field on a route containing a Go `regexp` pattern. The pattern is compiled at startup. When set, the request path must match the compiled pattern for the criterion to pass.

#### Scenario: Regex matches request path
- **WHEN** a route has `path_regex: "^/api/v[0-9]+/"` and the request path is `/api/v2/users`
- **THEN** the regex criterion passes

#### Scenario: Regex does not match
- **WHEN** a route has `path_regex: "^/api/v[0-9]+/"` and the request path is `/api/beta/users`
- **THEN** the regex criterion fails and the route is skipped

#### Scenario: Invalid regex is a fatal startup error
- **WHEN** a route specifies `path_regex: "[invalid"` that fails `regexp.Compile`
- **THEN** the proxy prints a fatal error identifying the route index and the pattern and exits with code 1

#### Scenario: No path_regex set skips criterion
- **WHEN** a route has no `path_regex` field
- **THEN** the regex criterion is skipped

---

### Requirement: All configured criteria must match (AND logic)
When a route has multiple matching criteria set, ALL of them must pass for the route to be selected. A criterion that is not set (empty/nil) is ignored and does not affect match outcome.

#### Scenario: Host and path prefix both match
- **WHEN** a route has `host_match: "*.local"` and `path_prefix: "/api"`, and the request has Host `app.local` and path `/api/v1`
- **THEN** the route matches

#### Scenario: Host matches but path does not
- **WHEN** a route has `host_match: "*.local"` and `path_prefix: "/api"`, and the request has Host `app.local` and path `/static/logo.png`
- **THEN** the route does not match and evaluation continues to the next route

---

### Requirement: First matching route wins
The system SHALL evaluate routes in the order they appear in `dev-proxy.yaml`. The first route whose every configured criterion passes is selected for the request.

#### Scenario: Earlier route takes precedence
- **WHEN** two routes both match a request (e.g., one with `path_prefix: "/"` and one with `path_exact: "/health"`), and the first in order is the prefix route
- **THEN** the prefix route handles the request

---

### Requirement: Return 504 with error body when no route matches
When no route in the list matches the incoming request, the system SHALL respond with HTTP status 504 (Gateway Timeout) and a plain-text body identifying the unmatched host and path.

#### Scenario: No route matches returns 504
- **WHEN** an incoming request matches no configured route
- **THEN** the proxy responds with status 504 and body `no route matched: host=<host> path=<path>`

#### Scenario: 504 body includes request details
- **WHEN** a request to `GET /missing` on host `app.local` matches no route
- **THEN** the response body is `no route matched: host=app.local path=/missing`

---

### Requirement: Apply url_rewrite transform after route is matched
The system SHALL, after selecting a matching route that has `url_rewrite` configured, apply the regex path rewrite to the upstream request before forwarding. Route matching itself is unaffected by the `url_rewrite` field; it participates in forwarding only.

#### Scenario: url_rewrite does not affect route matching
- **WHEN** a route has `path_prefix: "/api"` and `url_rewrite: {match: "^/api/(.*)", replace: "/v2/$1"}` and an incoming request path is `/api/users`
- **THEN** the route is selected by the `path_prefix` criterion, and the upstream receives path `/v2/users`

#### Scenario: Route without url_rewrite forwards path unchanged
- **WHEN** a route has `path_prefix: "/api"` and no `url_rewrite` field and an incoming request path is `/api/users`
- **THEN** the upstream receives path `/api/users` (or the prefix-stripped path per existing behavior)
