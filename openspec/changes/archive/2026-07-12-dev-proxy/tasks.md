## 1. Project Setup

- [ ] 1.1 Initialize Go module with `go mod init dev-proxy`
- [ ] 1.2 Create directory structure: `cmd/dev-proxy/main.go`, `internal/config/`, `internal/router/`, `internal/proxy/`, `internal/tls/`, `internal/cors/`, `internal/static/`, `internal/review/`, `internal/watcher/`
- [ ] 1.3 Add `fsnotify` dependency for file watching (`go get github.com/fsnotify/fsnotify`)

## 2. Configuration and .env Loading

- [ ] 2.1 Define CLI flag set using Go's `flag` package (port, upstream, static-dir, insecure)
- [ ] 2.2 Implement `.env` file parser that reads key=value pairs from `.env` in cwd
- [ ] 2.3 Wire flag values to override .env variables with precedence: flags > env file > defaults
- [ ] 2.4 Validate configuration at startup (port range, valid upstream URL) and exit with error on invalid config

## 3. Route Types and Router

- [ ] 3.1 Define `Route` struct with fields: LocalPort, PathPrefix, HostMatch, Upstream, RewriteHost, CORS, StaticDir, ReviewMode
- [ ] 3.2 Implement route table as an ordered slice with first-match evaluation logic
- [ ] 3.3 Implement path-prefix matching that strips the matched prefix before forwarding
- [ ] 3.4 Return HTTP 404 when no route matches an incoming request

## 4. Reverse Proxy Core

- [ ] 4.1 Use `net/http/httputil.ReverseProxy` as the forwarding engine per route
- [ ] 3.5 Implement Host header rewriting when `RewriteHost` is true on the matched route
- [ ] 3.6 Ensure connection pooling and streaming responses via ReverseProxy defaults

## 5. TLS Termination

- [ ] 5.1 Implement in-memory self-signed X.509 certificate generation using `crypto/x509` and `elliptic.P256()`
- [ ] 5.2 Cache generated certificates per-route in memory across reloads
- [ ] 5.3 Add HTTPS listener support: when a route's local port is configured with TLS, serve over `https://` using the cached cert
- [ ] 5.4 Support upstream TLS passthrough via Go's default `http.Transport`
- [ ] 5.5 Add `InsecureSkipVerify` option on the transport for routes that need it

## 6. CORS Handling

- [ ] 6.1 Implement CORS header injection middleware that runs before the reverse proxy
- [ ] 6.2 Handle OPTIONS preflight requests by returning 204 with CORS headers without forwarding upstream
- [ ] 6.3 Support configurable `AllowOrigin` (wildcard or specific origin) with rejection of non-matching origins
- [ ] 6.4 Add option to forward upstream CORS headers instead of injecting own

## 7. Static File Serving

- [ ] 7.1 Implement static file handler that checks for a matching file in `StaticDir` before proxying
- [ ] 7.2 Detect content-type from file extension using Go's `net/http.ParseMediaType` or a built-in map
- [ ] 7.3 Return HTTP 403 when requesting a directory without an index file (autoIndex off by default)
- [ ] 7.4 Fall through to upstream proxy when no static file matches the request path

## 8. Review Mode

- [ ] 8.1 Implement review channel: buffered channel of `ReviewRequest` structs (method, URL, headers, body bytes)
- [ ] 8.2 When `ReviewMode` is true on a route, pause forwarding and send request to the review channel
- [ ] 8.3 Add CLI subcommand or interactive mode for reviewer to approve/discard requests via stdin
- [ ] 8.4 On approval: resume proxying with original request; on discard: return 403 to client

## 9. .env Watcher and Hot Reload

- [ ] 9.1 Start `fsnotify.Watcher` goroutine that watches the `.env` file for Write events
- [ ] 9.2 On change detection: re-read `.env`, rebuild route table under a write lock, swap in new routes atomically using `sync.RWMutex`
- [ ] 9.3 Ensure in-flight requests continue on old handlers during reload (RWMutex read-lock for route access)
- [ ] 9.4 Close watcher gracefully on shutdown

## 10. Signal Handling and Graceful Shutdown

- [ ] 10.1 Register `os/signal.Notify` for SIGINT and SIGTERM
- [ ] 10.2 On signal: stop accepting new connections, close all listeners, wait up to 5 seconds for in-flight requests, then exit with code 0

## 11. Main Entry Point and Wiring

- [ ] 11.1 Wire all components in `main.go`: config → router → proxy → TLS → CORS → static → review → watcher
- [ ] 11.2 Print startup banner showing active routes, ports, and upstream targets
- [ ] 11.3 Log each incoming request (method, path, upstream target, status code) to stdout

## 12. Testing

- [ ] 12.1 Write unit tests for route matching logic (first-match priority, prefix stripping, no-match 404)
- [ ] 12.2 Write unit tests for .env parser (valid lines, comments, missing file)
- [ ] 12.3 Write integration test: start proxy, send HTTP request to local port, verify upstream response received correctly
- [ ] 12.4 Write integration test: static file override returns correct content-type and body
- [ ] 12.5 Write integration test: CORS preflight returns 204 with correct headers
