## ADDED Requirements

### Requirement: Serve static files from disk on path match
The system SHALL serve files from a configured local directory when a route has `staticDir` set and the request path matches a file in that directory.

#### Scenario: Static file served for matching path
- **WHEN** a route has `staticDir = "./public"` and `pathPrefix = "/"` and a request arrives at `/index.html` with a corresponding file at `./public/index.html`
- **THEN** the proxy reads the file from disk, sets appropriate content-type headers, and returns it as the response without contacting upstream

#### Scenario: Static file not found falls through to upstream
- **WHEN** a route has `staticDir = "./public"` and a request arrives at `/missing.html` with no corresponding file in `./public/`
- **THEN** the proxy forwards the request to the configured upstream as if static override were not set

#### Scenario: Directory listing disabled by default
- **WHEN** a request arrives for a directory path (e.g., `/assets/`) and no index file exists in that directory
- **THEN** the proxy returns HTTP 403 Forbidden unless `staticDir.autoIndex = true` is explicitly configured

### Requirement: Content-type detection for static files
The system SHALL detect and set the correct `Content-Type` response header based on the served file's extension.

#### Scenario: HTML content type detected
- **WHEN** a request serves `index.html` from the static directory
- **THEN** the response includes `Content-Type: text/html; charset=utf-8`

#### Scenario: Unknown extension defaults to application/octet-stream
- **WHEN** a request serves a file with an unrecognized extension (e.g., `.xyz`) from the static directory
- **THEN** the response includes `Content-Type: application/octet-stream`
