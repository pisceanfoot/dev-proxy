## ADDED Requirements

### Requirement: Watcher monitors both config file and .env file
The system SHALL watch both the YAML config file and the `.env` file (when it exists at startup) for write and create events. A change to either file SHALL trigger the same full config reload pipeline. The `.env` file is only watched if it exists when the proxy starts; a file created after startup is not automatically picked up without a restart.

#### Scenario: Saving .env triggers a config reload
- **WHEN** the proxy is running and the user saves the `.env` file
- **THEN** the watcher detects the change and the config reload pipeline runs, re-expanding all interpolated values

#### Scenario: Saving the YAML config still triggers a reload
- **WHEN** the proxy is running and the user saves `dev-proxy.yaml`
- **THEN** the watcher detects the change and the config reload pipeline runs (unchanged from existing behavior)

#### Scenario: .env not present at startup is not watched
- **WHEN** no `.env` file exists when the proxy starts
- **THEN** the watcher does not attempt to watch it and no error is produced; the proxy runs using only the OS environment
