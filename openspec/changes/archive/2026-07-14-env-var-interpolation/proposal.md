## Why

Developers frequently need to point one or more routes at a different environment (preprod, production) for a debugging session, but editing `dev-proxy.yaml` directly risks accidentally committing the override. A `.env`-file-based interpolation system lets environment-specific values live outside the committed config, following the Unix convention already used by docker-compose, vite, and dotenv.

## What Changes

- Add `${VAR}` and `${VAR:-default}` interpolation to all string values in `dev-proxy.yaml`, resolved at config load time
- Environment variable precedence: OS environment > `.env` file > inline default
- Add a `--env-file` CLI flag (default: `.env` in CWD) to specify the dot-env file path; if the file does not exist the proxy starts silently without it
- The watcher now watches both the YAML config file and the `.env` file; a change to either triggers a full config reload
- `${VAR}` with no default and no value set is a fatal load/reload error identifying the missing variable name

## Capabilities

### New Capabilities
- `env-var-interpolation`: `${VAR}` / `${VAR:-default}` expansion in YAML string fields, sourced from OS env and an optional `.env` file, evaluated at every config load including hot-reloads

### Modified Capabilities
- `hot-reload-config`: The watcher now tracks multiple files (YAML config + `.env`); a write to either file triggers the existing reload pipeline

## Impact

- `internal/config/yaml.go`: `Load` gains an `envPath string` parameter; raw YAML bytes are expanded before unmarshal
- New `internal/config/expand.go`: `parseDotEnv`, `expandVars` functions
- `internal/watcher/watcher.go`: supports watching multiple file paths
- `cmd/dev-proxy/main.go`: `--env-file` flag; passes env path to `config.Load` and `watcher.New`
- No breaking changes to existing configs — interpolation is opt-in per field; configs without `${...}` are unaffected
