## ADDED Requirements

### Requirement: Apply url_rewrite transform after route is matched
The system SHALL, after selecting a matching route that has `url_rewrite` configured, apply the regex path rewrite to the upstream request before forwarding. Route matching itself is unaffected by the `url_rewrite` field; it participates in forwarding only.

#### Scenario: url_rewrite does not affect route matching
- **WHEN** a route has `path_prefix: "/api"` and `url_rewrite: {match: "^/api/(.*)", replace: "/v2/$1"}` and an incoming request path is `/api/users`
- **THEN** the route is selected by the `path_prefix` criterion, and the upstream receives path `/v2/users`

#### Scenario: Route without url_rewrite forwards path unchanged
- **WHEN** a route has `path_prefix: "/api"` and no `url_rewrite` field and an incoming request path is `/api/users`
- **THEN** the upstream receives path `/api/users` (or the prefix-stripped path per existing behavior)
