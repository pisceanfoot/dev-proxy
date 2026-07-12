## ADDED Requirements

### Requirement: Inject CORS headers per route
The system SHALL add Cross-Origin Resource Sharing (CORS) response headers when a route has a `CORS` configuration.

#### Scenario: Standard CORS headers added to simple request
- **WHEN** a request arrives at a route with `CORS.enabled = true` and `CORS.allowOrigin = "*"` and the method is GET
- **THEN** the proxy adds `Access-Control-Allow-Origin: *`, `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`, and `Access-Control-Allow-Headers: Content-Type, Authorization` to the response

#### Scenario: CORS preflight request handled
- **WHEN** an HTTP OPTIONS request arrives at a route with CORS enabled and the request includes `Origin` and `Access-Control-Request-Method` headers
- **THEN** the proxy returns 204 No Content with appropriate CORS preflight headers (`Access-Control-Allow-Origin`, `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers`, `Access-Control-Max-Age`) without forwarding to upstream

#### Scenario: Custom allow origin enforced
- **WHEN** a route has `CORS.allowOrigin = "http://localhost:3000"` and a request arrives with `Origin: http://evil.com`
- **THEN** the proxy does not include `Access-Control-Allow-Origin` in the response (origin rejected)

#### Scenario: CORS headers forwarded from upstream when configured
- **WHEN** a route has `CORS.forwardUpstream = true` and the upstream response already contains CORS headers
- **THEN** the proxy passes through the upstream CORS headers instead of injecting its own
