## ADDED Requirements

### Requirement: Load configuration from .env file
The system SHALL read proxy settings from a `.env` file in the current working directory when CLI flags are not provided.

#### Scenario: Environment variables populate defaults
- **WHEN** no `--port` flag is given and `.env` contains `DEV_PROXY_PORT=8080`
- **THEN** the proxy listens on port 8080

#### Scenario: CLI flags override .env values
- **WHEN** `--port 9090` is provided as a CLI flag and `.env` contains `DEV_PROXY_PORT=8080`
- **THEN** the proxy listens on port 9090 (flag takes precedence)

#### Scenario: Missing .env file uses zero values
- **WHEN** no `.env` file exists in the current working directory
- **THEN** the proxy starts with default configuration and prints a warning suggesting creating a `.env` file

### Requirement: Watch .env file for changes
The system SHALL watch the `.env` file for modifications and trigger a configuration reload when changes are detected.

#### Scenario: Config reload on .env modification
- **WHEN** the `.env` file is modified (e.g., upstream URL changed) while the proxy is running
- **THEN** the proxy detects the change within 1 second, rebuilds the route table with new values, and begins serving updated configuration

#### Scenario: Reload does not drop in-flight requests
- **WHEN** a config reload occurs while client requests are being processed
- **THEN** in-flight requests complete using the previous configuration; only new requests use the updated configuration

### Requirement: Graceful shutdown on process termination signal
The system SHALL stop accepting new connections and drain existing ones when receiving `SIGINT` or `SIGTERM`.

#### Scenario: Clean shutdown on Ctrl+C
- **WHEN** the user presses Ctrl+C (sends SIGINT) while the proxy is running
- **THEN** the proxy stops listening for new connections, waits up to 5 seconds for in-flight requests to complete, then exits with code 0

#### Scenario: Watcher stopped before exit
- **WHEN** the process receives a termination signal
- **THEN** the `.env` file watcher is closed and resources are released before process exit
