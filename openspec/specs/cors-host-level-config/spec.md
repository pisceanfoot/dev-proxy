## Purpose

Host-level and route-level CORS configuration with inheritance semantics: route-level `cors_allow_origin` overrides host-level, and CORS headers are sent on all HTTP methods when an `Origin` header is present.

## ADDED Requirements

### Requirement: Host-level cors_allow_origin configuration
A `HostGroup` SHALL accept a `cors_allow_origin` field in YAML configuration, which sets the default CORS allowed origin for all routes within that host group.

#### Scenario: Host-level CORS applies to all routes when no route override
- **WHEN** a host group has `cors_allow_origin: "https://example.com"` and a route within it has no `cors_allow_origin`
- **THEN** cross-origin requests to that route receive `Access-Control-Allow-Origin: https://example.com`

#### Scenario: Route-level cors_allow_origin overrides host-level
- **WHEN** a host group has `cors_allow_origin: "https://example.com"` and a route within it sets `cors_allow_origin: "https://other.com"`
- **THEN** cross-origin requests to that route receive `Access-Control-Allow-Origin: https://other.com`

#### Scenario: Route with empty cors_allow_origin inherits host value
- **WHEN** a host group has `cors_allow_origin: "*"` and a route within it has `cors_allow_origin` absent or empty
- **THEN** requests to that route receive `Access-Control-Allow-Origin: *`

#### Scenario: No CORS when neither host nor route configures cors_allow_origin
- **WHEN** a host group has no `cors_allow_origin` and a route within it has no `cors_allow_origin`
- **THEN** CORS headers are not added to responses for that route

### Requirement: CORS headers sent on all HTTP methods with Origin header
When `cors_allow_origin` is configured (at host or route level), the server SHALL include CORS response headers on every request that carries an `Origin` request header, regardless of the HTTP method.

#### Scenario: GET request with Origin receives CORS headers
- **WHEN** a GET request is made with an `Origin` header and `cors_allow_origin` is configured
- **THEN** the response includes `Access-Control-Allow-Origin`

#### Scenario: POST request with Origin receives CORS headers
- **WHEN** a POST request is made with an `Origin` header and `cors_allow_origin` is configured
- **THEN** the response includes `Access-Control-Allow-Origin`

#### Scenario: PUT request with Origin receives CORS headers
- **WHEN** a PUT request is made with an `Origin` header and `cors_allow_origin` is configured
- **THEN** the response includes `Access-Control-Allow-Origin`

#### Scenario: DELETE request with Origin receives CORS headers
- **WHEN** a DELETE request is made with an `Origin` header and `cors_allow_origin` is configured
- **THEN** the response includes `Access-Control-Allow-Origin`

#### Scenario: OPTIONS preflight receives CORS headers and 204
- **WHEN** an OPTIONS request is made with `Origin` and `Access-Control-Request-Method` headers and `cors_allow_origin` is configured
- **THEN** the response status is 204
- **THEN** the response includes `Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, and `Access-Control-Allow-Headers`

#### Scenario: Request without Origin header receives no CORS headers
- **WHEN** a request is made without an `Origin` header
- **THEN** the response does not include `Access-Control-Allow-Origin`
