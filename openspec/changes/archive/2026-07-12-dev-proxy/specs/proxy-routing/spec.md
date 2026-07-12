## ADDED Requirements

### Requirement: Route local port to upstream host
The system SHALL forward HTTP requests received on a configured local port to an arbitrary upstream host specified by IP address or DNS name.

#### Scenario: Forward request to upstream by IP
- **WHEN** a request arrives at `localhost:8080/api/users` and the route is configured with `upstream = "192.168.1.50:3000"` and `pathPrefix = "/"`
- **THEN** the proxy forwards the request to `http://192.168.1.50:3000/api/users` preserving method, headers, and body

#### Scenario: Forward request to upstream by DNS name
- **WHEN** a request arrives at `localhost:8080/` and the route is configured with `upstream = "https://api.example.com"` and `pathPrefix = "/"`
- **THEN** the proxy resolves the DNS name, forwards the request to `https://api.example.com/`, and returns the response to the client

#### Scenario: Path prefix matching strips prefix
- **WHEN** a route is configured with `pathPrefix = "/api"` and `upstream = "http://localhost:3000"` and a request arrives at `/api/users`
- **THEN** the proxy forwards the request as `/users` to the upstream (prefix stripped)

#### Scenario: Default path prefix matches all
- **WHEN** a route has no explicit `pathPrefix` configured (defaults to `/`) and a request arrives at any path
- **THEN** the proxy forwards the full request path unchanged to the upstream

### Requirement: Rewrite Host header on forwarding
The system SHALL optionally rewrite the outgoing `Host` header to match the upstream hostname when the route has `rewriteHost = true`.

#### Scenario: Host header rewritten when enabled
- **WHEN** a route has `rewriteHost = true` and `upstream = "https://api.example.com"` and a request arrives with `Host: localhost:8080`
- **THEN** the proxy sends the upstream request with `Host: api.example.com`

#### Scenario: Host header preserved when disabled
- **WHEN** a route has `rewriteHost = false` (default) and a request arrives with `Host: myapp.local`
- **THEN** the proxy forwards the original `Host: myapp.local` to the upstream unchanged

### Requirement: First-match routing priority
The system SHALL evaluate routes in configuration order and use the first matching route for each incoming request.

#### Scenario: First matching route wins
- **WHEN** two routes match a request — Route A (`pathPrefix = "/api"`) defined before Route B (`pathPrefix = "/"`) — and a request arrives at `/api/data`
- **THEN** Route A handles the request; Route B is not evaluated

#### Scenario: No matching route returns 404
- **WHEN** no configured route matches an incoming request path
- **THEN** the proxy returns HTTP 404 with a descriptive body
