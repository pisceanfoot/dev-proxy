## ADDED Requirements

### Requirement: Terminate TLS at the proxy for local HTTPS
The system SHALL present a valid TLS certificate to clients when a route is configured to listen on an HTTPS port.

#### Scenario: Client connects over HTTPS
- **WHEN** a route listens on `localhost:8443` with TLS enabled and a browser navigates to `https://localhost:8443/`
- **THEN** the client receives a valid TLS handshake and can complete an HTTPS request

#### Scenario: Self-signed certificate served automatically
- **WHEN** no user-provided certificate is configured for a route
- **THEN** the proxy generates an in-memory self-signed X.509 certificate and uses it for TLS termination

### Requirement: Passthrough TLS to upstream
The system SHALL forward requests to upstream HTTPS backends, terminating TLS at the upstream server.

#### Scenario: Upstream HTTPS connection established
- **WHEN** a route has `upstream = "https://api.example.com"` and a request arrives
- **THEN** the proxy establishes a TLS connection to the upstream using Go's default certificate pool

#### Scenario: Insecure upstream mode bypasses verification
- **WHEN** a route has `insecureSkipVerify = true` and the upstream presents an invalid or self-signed certificate
- **THEN** the proxy proceeds with the TLS handshake without verifying the upstream certificate

### Requirement: Certificate caching across reloads
The system SHALL cache generated certificates in memory so that reloads do not invalidate active client connections prematurely.

#### Scenario: Certificate persists through config reload
- **WHEN** the `.env` file changes and triggers a hot reload while clients have active TLS connections
- **THEN** the proxy continues using the existing certificate for in-flight connections; new connections also use the same cert until next restart
