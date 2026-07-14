## 1. .env Parser

- [x] 1.1 Create `internal/config/expand.go` with `parseDotEnv(path string) (map[string]string, error)` — reads KEY=VALUE lines, strips quotes and `export` prefix, skips comments and blanks, returns empty map (no error) if file not found
- [x] 1.2 Implement `mergeEnv(dotEnv map[string]string) map[string]string` — merges dotenv map with OS env, OS env wins per-key
- [x] 1.3 Implement `expandVars(data []byte, env map[string]string) ([]byte, error)` — regexp scan for `${VAR}` and `${VAR:-default}`, resolves each token against merged env, returns error naming any unresolved variable with no default

## 2. Config Load Integration

- [x] 2.1 Update `config.Load` signature to `Load(configPath, envPath string) (*Config, error)`
- [x] 2.2 In `Load`: call `parseDotEnv(envPath)`, `mergeEnv`, then `expandVars(rawBytes, env)` before `yaml.Unmarshal`
- [x] 2.3 Propagate expansion errors as load errors (clear message: `env var "X" is not set and has no default`)

## 3. Watcher Multi-File Support

- [x] 3.1 Change `watcher.New` to accept `filePaths []string` instead of `filePath string`
- [x] 3.2 In `watcher.Start`, call `fw.Add(path)` for each path; skip (no error) if a path does not exist
- [x] 3.3 Update `watcher.watchLoop` — no change needed, fsnotify events already carry the path; reload callback is the same regardless of which file changed

## 4. CLI Flag + Wiring

- [x] 4.1 Add `--env-file` flag to `main.go` with default `".env"`; also check `DEV_PROXY_ENV_FILE` env var as an alternative (consistent with `--config` / `DEV_PROXY_CONFIG` pattern)
- [x] 4.2 Update all `config.Load(configPath)` call sites in `main.go` to `config.Load(configPath, envPath)`
- [x] 4.3 Update `watcher.New(configPath, ...)` to `watcher.New([]string{configPath, envPath}, ...)` — pass resolved envPath so watcher watches both files

## 5. Tests

- [x] 5.1 Unit test `parseDotEnv`: plain `KEY=value`, double-quoted, single-quoted, `export` prefix, comment lines, empty lines, missing file returns empty map
- [x] 5.2 Unit test `expandVars`: `${VAR}` resolves from env, `${VAR:-default}` uses default when unset, OS env beats dotenv, missing var with no default returns error
- [x] 5.3 Unit test `config.Load` with a temp YAML using `${VAR:-fallback}` and a temp `.env` file — verify correct resolution
- [x] 5.4 Unit test `config.Load` with a missing `.env` path — verify load succeeds using OS env only
- [x] 5.5 Unit test watcher `Start` with a non-existent path in the list — verify it skips without error
- [x] 5.6 Integration test in `main_test.go`: build a config with `${TEST_UPSTREAM:-http://localhost:9999}`, load without setting the var, verify fallback URL used in the built route

## 6. Documentation

- [x] 6.1 Update `dev-proxy.yaml` example to show `${VAR:-default}` usage in an upstream URL with a comment explaining the pattern
