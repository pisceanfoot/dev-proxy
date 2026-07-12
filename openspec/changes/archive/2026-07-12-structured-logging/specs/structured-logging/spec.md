## ADDED Requirements

### Requirement: Configure log level in dev-proxy.yaml
The system SHALL read a top-level `log_level` field from `dev-proxy.yaml`. Valid values are `error`, `info`, and `debug` (case-insensitive). The default when the field is absent is `error`. An unrecognised value SHALL produce a fatal error at startup identifying the invalid value and listing the valid options.

#### Scenario: Default log level is error when field is absent
- **WHEN** `dev-proxy.yaml` contains no `log_level` field
- **THEN** the proxy starts with log level `error` and produces no runtime output beyond fatal errors

#### Scenario: Valid log level accepted
- **WHEN** `dev-proxy.yaml` contains `log_level: debug`
- **THEN** the proxy starts with log level `debug` and emits info and debug output

#### Scenario: Invalid log level is a fatal error
- **WHEN** `dev-proxy.yaml` contains `log_level: verbose`
- **THEN** the proxy prints a fatal error: `invalid log_level "verbose" — must be one of: error, info, debug` and exits with code 1

### Requirement: Error level suppresses all runtime output
At log level `error`, the system SHALL print nothing during normal operation. Only fatal errors (via `log.Fatalf`) and the pre-existing startup warning for conflicting config keys are emitted.

#### Scenario: No output at error level during startup and request handling
- **WHEN** `log_level: error` and the proxy starts and handles requests
- **THEN** no output is written to stdout or stderr (other than fatal errors)

### Requirement: Info level prints startup summary
At log level `info` (or `debug`), the system SHALL print a startup summary after all listeners are bound. The summary includes: each listen port with its scheme (http/https), number of host groups configured, and total number of routes across all groups.

#### Scenario: Info summary shows ports and route counts
- **WHEN** `log_level: info` and the server has listen ports `[8080, 8443]` with TLS on 8443, two host groups, and five routes total
- **THEN** the proxy prints lines reporting each port's scheme, the host group count (2), and total route count (5)

#### Scenario: Info level also prints config reload notification
- **WHEN** `log_level: info` and the YAML config file is saved while the proxy is running
- **THEN** the proxy prints `[info] config reloaded` (or equivalent) after rebuilding the route table

### Requirement: Debug level prints per-request trace
At log level `debug`, the system SHALL print one log line per request containing: HTTP method, request URL (scheme + host + path), matched host group pattern, matched route criteria (the active match field and its value), resolved upstream base URL, full upstream request URL (after any path rewriting), response status code, and request duration.

#### Scenario: Debug line includes all routing fields
- **WHEN** `log_level: debug` and a `GET /api/v1/users` request arrives on host `api.local`, matching host group `api.local`, route with `path_prefix: /api/v1`, upstream `http://localhost:3001`, upstream_path `/v1`
- **THEN** the proxy emits a single debug line containing: method `GET`, path `/api/v1/users`, host group `api.local`, route criterion `prefix:/api/v1`, upstream `http://localhost:3001`, upstream URL `http://localhost:3001/v1/users`, response status, and duration

#### Scenario: Debug line for no-match 504
- **WHEN** `log_level: debug` and a request matches no route (504 response)
- **THEN** the proxy emits a debug line with method, path, host, status `504`, duration, and `no match` for route and upstream fields

#### Scenario: Debug level includes all info-level output
- **WHEN** `log_level: debug`
- **THEN** the proxy also emits all info-level output (startup summary, reload notification)
