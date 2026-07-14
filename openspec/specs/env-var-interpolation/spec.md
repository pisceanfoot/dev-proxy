## Purpose

Support environment variable interpolation in dev-proxy YAML configuration files, enabling secrets and hostnames to be externalized from config while remaining safe for hot-reload.

## Requirements

### Requirement: Expand ${VAR} and ${VAR:-default} in YAML string values
The system SHALL expand `${VAR}` and `${VAR:-default}` tokens in all string-typed YAML values before parsing. Expansion resolves the variable name against the merged environment (OS env taking precedence over `.env` file values). Non-string YAML fields (integers, booleans, lists of integers) are not subject to expansion.

#### Scenario: ${VAR} expands to OS environment value
- **WHEN** the OS environment has `PAYMENTS_URL=https://prod.payments.internal` and the config contains `url: ${PAYMENTS_URL}`
- **THEN** the upstream URL is resolved as `https://prod.payments.internal`

#### Scenario: ${VAR:-default} uses default when variable is unset
- **WHEN** the variable `PAYMENTS_URL` is not set in OS env or `.env` file and the config contains `url: ${PAYMENTS_URL:-http://localhost:3010}`
- **THEN** the upstream URL is resolved as `http://localhost:3010`

#### Scenario: ${VAR:-default} uses variable value when set
- **WHEN** `PAYMENTS_URL=https://preprod.internal` is set and the config contains `url: ${PAYMENTS_URL:-http://localhost:3010}`
- **THEN** the upstream URL is resolved as `https://preprod.internal`, ignoring the default

#### Scenario: ${VAR} with no default and variable unset is a load error
- **WHEN** `PAYMENTS_URL` is not set anywhere and the config contains `url: ${PAYMENTS_URL}` (no default)
- **THEN** the proxy prints an error identifying the missing variable name and refuses to start (or rejects the reload)

#### Scenario: Config without interpolation tokens is unaffected
- **WHEN** a `dev-proxy.yaml` file contains no `${...}` tokens
- **THEN** the config loads identically to before this feature was introduced

### Requirement: OS environment takes precedence over .env file
The system SHALL resolve variables by checking the OS environment first, then the `.env` file. A variable set in both places SHALL use the OS environment value.

#### Scenario: OS env overrides .env file value
- **WHEN** the `.env` file contains `PAYMENTS_URL=http://localhost:3010` and the OS environment has `PAYMENTS_URL=https://prod.payments.internal`
- **THEN** the upstream resolves to `https://prod.payments.internal`

#### Scenario: .env value used when OS env does not have the variable
- **WHEN** the `.env` file contains `PAYMENTS_URL=http://localhost:3010` and the OS environment does not have `PAYMENTS_URL`
- **THEN** the upstream resolves to `http://localhost:3010`

### Requirement: Load .env file from configurable path
The system SHALL accept an `--env-file <path>` CLI flag specifying the dot-env file to load. The default SHALL be `.env` in the current working directory. If the specified file does not exist, the proxy SHALL start without error using only the OS environment.

#### Scenario: Default .env in CWD is loaded automatically
- **WHEN** a `.env` file exists in the current working directory and `--env-file` is not specified
- **THEN** the proxy loads it and uses its values for interpolation

#### Scenario: --env-file flag overrides default path
- **WHEN** the user starts the proxy with `--env-file .env.preprod`
- **THEN** the proxy loads `.env.preprod` instead of `.env`

#### Scenario: Missing .env file is not an error
- **WHEN** no `.env` file exists at the default or specified path
- **THEN** the proxy starts normally using only the OS environment for interpolation

### Requirement: Parse .env file in standard KEY=VALUE format
The system SHALL parse the `.env` file line by line, supporting the following formats. Unrecognized lines SHALL be silently skipped.

#### Scenario: Plain KEY=value line is parsed
- **WHEN** the `.env` file contains `AUTH_URL=http://localhost:4000`
- **THEN** `AUTH_URL` resolves to `http://localhost:4000`

#### Scenario: Double-quoted value strips quotes
- **WHEN** the `.env` file contains `AUTH_URL="http://localhost:4000"`
- **THEN** `AUTH_URL` resolves to `http://localhost:4000` (quotes removed)

#### Scenario: Single-quoted value strips quotes
- **WHEN** the `.env` file contains `AUTH_URL='http://localhost:4000'`
- **THEN** `AUTH_URL` resolves to `http://localhost:4000` (quotes removed)

#### Scenario: export prefix is stripped
- **WHEN** the `.env` file contains `export AUTH_URL=http://localhost:4000`
- **THEN** `AUTH_URL` resolves to `http://localhost:4000`

#### Scenario: Comment lines are skipped
- **WHEN** a line in `.env` starts with `#`
- **THEN** the line is ignored

#### Scenario: Empty lines are skipped
- **WHEN** the `.env` file contains blank lines
- **THEN** they are silently ignored

### Requirement: Interpolation is re-evaluated on every config reload
The system SHALL re-read and re-apply the `.env` file on every config reload (triggered by changes to either the YAML config or the `.env` file). Updated variable values SHALL be reflected in the reloaded config.

#### Scenario: Editing .env and saving updates upstream URL live
- **WHEN** the proxy is running, the user changes `PAYMENTS_URL` in `.env` and saves the file
- **THEN** the watcher triggers a reload, the new value is expanded, and subsequent requests use the updated upstream URL

#### Scenario: Reload error on missing required variable keeps old config
- **WHEN** the user removes a required variable from `.env` (one that has no default in the YAML) and saves
- **THEN** the reload is rejected with an error naming the missing variable and the previous config remains active
