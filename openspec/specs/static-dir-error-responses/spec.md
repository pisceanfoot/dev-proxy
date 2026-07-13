## Purpose

HTTP error responses and path safety for static file serving via `static_dir` configuration.

## ADDED Requirements

### Requirement: File not found returns 404
When a request targets a `static_dir` route and the requested path does not exist on the filesystem, the server SHALL return HTTP 404 Not Found with a plain-text body.

#### Scenario: Missing file on static-only route
- **WHEN** a GET request is made for `/missing.txt` on a route with `static_dir` set and no upstream
- **THEN** the server responds with HTTP 404
- **THEN** the response body contains `404 Not Found`
- **THEN** the response `Content-Type` is `text/plain`

#### Scenario: Missing nested path on static-only route
- **WHEN** a GET request is made for `/a/b/c.json` and the path does not exist under `static_dir`
- **THEN** the server responds with HTTP 404
- **THEN** the response body contains `404 Not Found`

### Requirement: Stat failure returns 500 with reason
When `os.Stat` fails for a reason other than the file not existing (e.g. permission denied, I/O error), the server SHALL return HTTP 500 Internal Server Error with a plain-text body that includes the OS error reason.

#### Scenario: Permission denied on stat
- **WHEN** a GET request is made for a path whose parent directory has no execute permission
- **THEN** the server responds with HTTP 500
- **THEN** the response body contains the OS-level error message
- **THEN** the response `Content-Type` is `text/plain`

### Requirement: Path traversal outside static_dir returns 403
When a request URL resolves (after joining with `static_dir`) to a path outside `static_dir`, the server SHALL return HTTP 403 Forbidden without serving any file or directory listing.

#### Scenario: Double-dot traversal attempt
- **WHEN** a GET request is made with a URL like `/../../../etc/passwd`
- **THEN** the resolved path falls outside `static_dir`
- **THEN** the server responds with HTTP 403
- **THEN** no file content is returned

### Requirement: Existing file is served correctly
When the requested path exists and is a regular file within `static_dir`, the server SHALL serve the file with the correct `Content-Type` and HTTP 200.

#### Scenario: Serving a known file
- **WHEN** a GET request is made for `/index.html` and the file exists under `static_dir`
- **THEN** the server responds with HTTP 200
- **THEN** the response `Content-Type` is `text/html`
- **THEN** the response body contains the file content
