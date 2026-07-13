## Purpose

Enable configuration hot-reloading for route, upstream, and logging changes in dev-proxy while preserving safety guarantees for server configuration changes.

## Requirements

### Requirement: Route and upstream changes apply immediately on config save
When `dev-proxy.yaml` is saved, the system SHALL rebuild the route table (host groups, routes, upstreams) and make it active for all new requests within the same process lifetime. No restart is required for changes to `routes`, `hosts`, `upstreams`, or `log_level`.

#### Scenario: Route change takes effect without restart
- **WHEN** a route's `upstream` or `path_prefix` is edited and the file is saved
- **THEN** the next request uses the updated route without restarting the proxy

#### Scenario: log_level change takes effect without restart
- **WHEN** `log_level` is changed from `error` to `debug` in `dev-proxy.yaml` and saved
- **THEN** subsequent requests are logged at debug verbosity with no restart

#### Scenario: upstreams map change takes effect without restart
- **WHEN** a named upstream's URL is edited and the file is saved
- **THEN** routes referencing that upstream immediately forward to the new URL

### Requirement: Server config changes produce a restart-required warning
When a reload detects that the `server` block has changed (`listen_ports`, `tls.*`, or `redirect_http`), the system SHALL log a warning identifying which field(s) changed and state that a restart is required. The proxy SHALL continue running on the original ports and TLS settings.

#### Scenario: listen_ports change warns and keeps original ports
- **WHEN** `server.listen_ports` is changed from `[8080]` to `[8080, 8443]` and the file is saved
- **THEN** the proxy logs `[info] server config changed (listen_ports); restart required to apply` and continues listening only on port 8080

#### Scenario: tls change warns and keeps original TLS state
- **WHEN** `server.tls.enabled` is toggled and the file is saved
- **THEN** the proxy logs a warning naming the changed server field and continues with the original TLS configuration

#### Scenario: Non-server changes do not produce the warning
- **WHEN** only routes or upstreams are changed
- **THEN** no restart-required warning is emitted

### Requirement: Config reload error keeps previous config running
When a file-save triggers a reload and the new config fails to parse or validate, the system SHALL log the error, keep the previous route table active, and remain fully operational.

#### Scenario: Invalid YAML on save keeps old routes
- **WHEN** the user saves `dev-proxy.yaml` with a YAML syntax error
- **THEN** the proxy logs `[error] config reload failed: <parse error>` and continues serving requests with the previous configuration

#### Scenario: Validation error on save keeps old routes
- **WHEN** the user saves a config that references an undefined upstream name
- **THEN** the proxy logs the validation error and continues with the previous route table

### Requirement: Router replacement is safe under concurrent requests
The system SHALL replace the active router atomically so that no in-flight request observes a partially-constructed route table. Requests that arrive during a reload see either the previous router or the new one in full — never a mix.

#### Scenario: In-flight requests complete with consistent router
- **WHEN** a config reload occurs while requests are being processed
- **THEN** each in-flight request uses the router that was active when the request was dispatched; subsequent requests use the new router

### Requirement: Watcher error messages use the logger
**Previously:** the watcher's internal `watchLoop` wrote directly to stdout/stderr via `fmt.Printf`, bypassing the level-controlled logger.
**Now:** all watcher-originated messages (reload triggered, reload error, fsnotify error) are routed through `logger.Info` / `logger.Error` so they respect the configured log level.

#### Scenario: Watcher reload message respects log level
- **WHEN** `log_level: error` and a config reload is triggered
- **THEN** no reload message is printed (it is info-level, suppressed at error threshold)

#### Scenario: Watcher reload error always appears
- **WHEN** a config reload fails regardless of log level
- **THEN** the error is printed via `logger.Error` (error level is always active)
