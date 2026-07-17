## ADDED Requirements

### Requirement: Configure URL rewrite on a route
The system SHALL allow a route to declare a `url_rewrite` block with a `match` field (Go regexp pattern) and a `replace` field (replacement string). Both fields are required when `url_rewrite` is present.

#### Scenario: Route with url_rewrite block is accepted
- **WHEN** a route in `dev-proxy.yaml` includes `url_rewrite: {match: "^/api/(.*)", replace: "/v2/$1"}`
- **THEN** the config loads without error

#### Scenario: url_rewrite with missing match field is rejected
- **WHEN** a route declares `url_rewrite` but omits `match`
- **THEN** the proxy prints a validation error and exits with code 1

#### Scenario: url_rewrite with missing replace field is rejected
- **WHEN** a route declares `url_rewrite` but omits `replace`
- **THEN** the proxy prints a validation error and exits with code 1

---

### Requirement: Compile url_rewrite match pattern at startup
The system SHALL compile the `url_rewrite.match` regexp pattern at startup using Go's `regexp.Compile`. An invalid pattern is a fatal startup error.

#### Scenario: Invalid regex causes fatal startup error
- **WHEN** a route specifies `url_rewrite: {match: "[invalid", replace: "$1"}`
- **THEN** the proxy prints a fatal error identifying the route index and the bad pattern, and exits with code 1

#### Scenario: Valid regex compiles silently
- **WHEN** a route specifies a syntactically valid `url_rewrite.match` pattern
- **THEN** the proxy starts without error

---

### Requirement: Rewrite request path before forwarding to upstream
The system SHALL apply `regexp.ReplaceAllString(incomingPath, replace)` to the request path before forwarding to the upstream when a route has `url_rewrite` configured. The original incoming path is not modified for logging purposes.

#### Scenario: Capture group substitution rewrites path
- **WHEN** a route has `url_rewrite: {match: "^/api/(.*)", replace: "/v2/$1"}` and an incoming request path is `/api/users`
- **THEN** the upstream receives the request at path `/v2/users`

#### Scenario: Named capture groups are supported
- **WHEN** a route has `url_rewrite: {match: "^/(?P<ver>v[0-9]+)/(?P<res>.*)", replace: "/${res}?version=${ver}"}` and an incoming request path is `/v3/items`
- **THEN** the upstream receives the request at path `/items?version=v3`

#### Scenario: Pattern with no match leaves path unchanged
- **WHEN** a route has `url_rewrite: {match: "^/legacy/", replace: "/new/"}` and an incoming request path is `/other/path`
- **THEN** the upstream receives the request at path `/other/path` (unchanged)

#### Scenario: Full path replacement with anchored pattern
- **WHEN** a route has `url_rewrite: {match: "^.*$", replace: "/fixed"}` and an incoming request path is `/anything/here`
- **THEN** the upstream receives the request at path `/fixed`

---

### Requirement: url_rewrite and upstream_path are mutually exclusive
The system SHALL reject any route configuration that specifies both `url_rewrite` and `upstream_path`. Only one path-rewriting mechanism may be used per route.

#### Scenario: Both fields on same route causes validation error
- **WHEN** a route declares both `upstream_path: "/new"` and `url_rewrite: {match: "^/(.*)", replace: "/$1"}`
- **THEN** the proxy prints a validation error identifying the route and exits with code 1
