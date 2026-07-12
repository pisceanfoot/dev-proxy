## ADDED Requirements

### Requirement: Load custom TLS certificate from disk
When a route has `tls: true` and specifies `cert_file` and `key_file`, the system SHALL load the PEM-encoded certificate and private key from those file paths using Go's `crypto/tls.LoadX509KeyPair`.

#### Scenario: Valid cert and key files loaded successfully
- **WHEN** a route configures `cert_file: "./certs/cert.pem"` and `key_file: "./certs/key.pem"`, both pointing to valid PEM files with matching keys
- **THEN** the proxy loads the certificate, parses it into a `tls.Certificate`, caches it in CertManager, and serves HTTPS using it

#### Scenario: Certificate file is missing produces fatal error
- **WHEN** a route specifies a `cert_file` path that does not exist on disk
- **THEN** the proxy prints a fatal error showing the full file path and exits with code 1

#### Scenario: Key file is missing produces fatal error
- **WHEN** a route specifies a `key_file` path that does not exist on disk
- **THEN** the proxy prints a fatal error showing the full file path and exits with code 1

#### Scenario: Cert and key do not match produces fatal error
- **WHEN** the certificate's public key does not correspond to the private key in `key_file` (e.g., mismatched pair)
- **THEN** the proxy prints a fatal error stating "certificate and key do not match" with both file paths and exits with code 1

#### Scenario: Invalid PEM encoding produces fatal error
- **WHEN** a cert or key file contains malformed PEM data that `LoadX509KeyPair` cannot parse
- **THEN** the proxy prints a fatal error including the underlying parse error message, the file path, and exits with code 1

### Requirement: Cache loaded certificates in CertManager
Loaded custom certificates SHALL be cached in the existing `CertManager` using the certificate file path as the cache key. This prevents reloading the same cert when multiple routes share it or when hot reload re-processes the config.

#### Scenario: Two routes sharing one cert use cached copy
- **WHEN** two routes both reference the same `cert_file` / `key_file` paths
- **THEN** the certificate is loaded from disk once and reused for both routes without a second disk read

#### Scenario: Certificate survives hot reload
- **WHEN** the YAML config changes and triggers a route table rebuild while custom certs are in use
- **THEN** the cached certificates remain active; no re-read of cert files occurs during reload

### Requirement: Port-in-use error with actionable message
When `ListenAndServe` or `ListenAndServeTLS` fails because the port is already bound by another process, the proxy SHALL print a fatal error that identifies the port and suggests how to find the conflicting process.

#### Scenario: Port 443 already in use shows helpful error
- **WHEN** the proxy attempts to bind port 443 and receives "address already in use"
- **THEN** it prints `[dev-proxy] FATAL: port 443 is already in use — another process is bound to it. Run 'lsof -i :443' or 'netstat -an | grep 443' to find the conflicting process.` and exits with code 1

#### Scenario: Port 80 already in use shows helpful error
- **WHEN** the proxy attempts to bind port 80 (a privileged port) and receives "address already in use" or a permission-related bind failure
- **THEN** it prints the same actionable fatal message identifying port 80 and exits with code 1
