## Context

`config.Load` currently reads YAML bytes and unmarshals them directly. All string values are literal — there is no mechanism to inject environment-specific values without editing the file. The watcher (`internal/watcher/watcher.go`) watches exactly one file via `fsnotify`, using a single `filePath string`. Both constraints are straightforward to loosen.

The design must not require a new external dependency (no `godotenv` library) — the `.env` format used here is simple enough to parse with a few lines of Go.

## Goals / Non-Goals

**Goals:**
- `${VAR}` and `${VAR:-default}` expansion in all YAML string values before unmarshal
- OS env takes precedence over `.env` file values (Unix convention)
- `.env` file is optional; absence is not an error
- `--env-file` flag to override the default `.env` path
- Watcher fires a reload when either the YAML config or the `.env` file changes
- Missing var with no default is a clear load error naming the variable

**Non-Goals:**
- Variable references within the `.env` file itself (`FOO=${BAR}`)
- Multi-line values in `.env`
- Expanding non-string YAML fields (booleans, integers) — `listen_ports: ${PORT}` is not supported; only string fields
- Watching additional arbitrary files beyond config + `.env`
- Shell-style quoting edge cases beyond `"..."` and `'...'`

## Decisions

### Decision 1: Expand raw bytes before `yaml.Unmarshal`, not after

We apply interpolation on the raw YAML byte slice before passing to the YAML parser, using a single `regexp.ReplaceAllStringFunc` pass over the text.

**Why:** Post-unmarshal expansion would require walking the entire config struct with reflection to find string fields. Pre-unmarshal expansion is a simple string operation, requires no struct knowledge, and naturally handles any new fields added in the future.

**Alternative considered:** A custom `yaml.Node` walk post-unmarshal. Rejected — significantly more complex and fragile with struct changes.

**Concern:** Expanding inside YAML string values that happen to contain `${` for unrelated reasons (e.g., a regex pattern). Mitigation: the interpolation regex is `\$\{[^}]+\}` — it only matches `${...}` tokens. Real-world YAML values rarely contain this. If they do, users can escape with `$${VAR}` → produces literal `${VAR}` (same convention as docker-compose).

### Decision 2: `Load` signature gains `envPath string`; no global state

```go
func Load(configPath, envPath string) (*Config, error)
```

`envPath` is the resolved path to the `.env` file (empty string = skip). `parseDotEnv` and `expandVars` are pure functions with no side effects on `os.Setenv` — env vars from `.env` are NOT injected into the process environment, only used locally for expansion.

**Why:** Polluting `os.Setenv` would affect any library or sub-process that reads env vars. Keeping it local is safer and more predictable.

### Decision 3: Precedence via merge — OS env wins

```go
func mergeEnv(dotEnv map[string]string) map[string]string {
    merged := make(map[string]string, len(dotEnv))
    for k, v := range dotEnv { merged[k] = v }
    for _, k := range keys(dotEnv) {
        if v, ok := os.LookupEnv(k); ok { merged[k] = v }
    }
    return merged
}
```

Only vars present in the `.env` file are checked against the OS env. We don't enumerate all OS vars — we only override dotenv values where the OS has the same key.

### Decision 4: Watcher accepts `[]string` file paths

```go
func New(filePaths []string, reloadFunc func() error, ...) (*Watcher, error)
```

`Start()` calls `fw.Add(path)` for each path, skipping any that don't exist (`.env` may not be present). All paths share the same `reloadFunc` — the distinction of *which* file changed doesn't matter since both changes require a full `config.Load`.

**Alternative considered:** A second independent watcher for `.env`. Rejected — doubles goroutines and makes shutdown more complex; `fsnotify` handles multiple paths natively.

### Decision 5: `${VAR}` with no default and no value → load error

```
error: env var "PAYMENTS_URL" is not set and has no default
```

Silent empty string expansion would cause a proxy that starts up but forwards to an empty upstream URL — a confusing runtime failure. A clear startup error is safer.

### Decision 6: `.env` parse rules (intentionally minimal)

```
# comment lines           → skip
empty lines               → skip
KEY=value                 → KEY → "value"
KEY="value with spaces"   → KEY → "value with spaces"  (strip double quotes)
KEY='value'               → KEY → "value"              (strip single quotes)
export KEY=value          → KEY → "value"              (strip export prefix)
KEY=                      → KEY → ""
```

No variable interpolation within `.env`, no multiline values. Unrecognised lines are silently skipped.

## Risks / Trade-offs

- **`${` in existing YAML values** (e.g., regex patterns like `\${1}`) → Mitigation: only `${UPPERCASE_OR_UNDERSCORE}` patterns are expanded by default; add `$$` escape if needed.
- **`.env` not watched when it doesn't exist at startup** → If a user creates `.env` after starting dev-proxy, the watcher won't pick it up (file must exist at startup for `fsnotify.Add` to succeed). Mitigation: document this; user restarts once to activate `.env` watching.
- **Secrets in `.env`** → `.env` should be gitignored. Not a code concern but worth noting in docs.

## Migration Plan

- Purely additive: existing configs with no `${...}` are unaffected
- `config.Load` signature change is internal; only `main.go` calls it
- No config file format version bump needed
- Rollback: revert binary; `.env` files are inert without the expansion logic
