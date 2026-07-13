## Context

The current reload mechanism (in `main.go`) works as follows:
1. `watcher.New(configPath, callback)` watches the YAML file via fsnotify
2. On every Write/Create event, the callback: calls `config.Load()`, calls `buildHostGroups(newCfg)`, assigns `rt = router.New(newGroups)`
3. The watcher's `watchLoop` uses `fmt.Printf` for its own messages

Problems:
- **Silent server-config drift**: if `listen_ports` or `tls` changes, the new config is loaded but the server block is simply not used — no warning
- **`log_level` ignored on reload**: `logger.SetLevel` is called once at startup; hot-editing log level has no effect
- **Data race on `rt`**: `rt` is a `*router.Router` variable captured by the handler closure. The callback assigns `rt = router.New(...)` from a goroutine (watchLoop) while request goroutines read `rt` concurrently. This is undefined behaviour under the Go memory model
- **`fmt.Printf` in watcher**: bypasses the logger; always prints regardless of log level

## Goals / Non-Goals

**Goals:**
- Atomic `rt` swap via `sync/atomic.Pointer[router.Router]`
- Reload callback also calls `logger.SetLevel` when `log_level` changes
- Snapshot `cfg.Server` at startup; compare on every reload; warn on diff
- Watcher callbacks (reload triggered, reload error, fsnotify error) flow through logger
- Keep old router on reload parse/validation error

**Non-Goals:**
- Live-reloading server ports (requires rebinding OS sockets; out of scope)
- Live-reloading TLS certificates independently of server config
- Debouncing rapid saves (fsnotify already coalesces events on most platforms)
- Per-field diff output in the restart-required warning (field names are enough)

## Decisions

### D1: `sync/atomic.Pointer[router.Router]` for safe rt swap

**Choice:** Replace the bare `rt *router.Router` variable with `var rt atomic.Pointer[router.Router]`. The handler calls `rt.Load()` on every request; the reload callback calls `rt.Store(router.New(newGroups))`.

```go
var rt atomic.Pointer[router.Router]
rt.Store(router.New(groups))

// in buildHandler:
r := rt.Load()
result := r.Match(req)

// in reload callback:
rt.Store(router.New(newGroups))
```

**Rationale:** `atomic.Pointer` provides sequentially-consistent load/store with no mutex overhead on the read path. Given that reloads are rare and reads are per-request, this is the minimal-overhead solution. Available in Go 1.19+; project uses 1.23.3.

**Alternatives considered:**
- `sync.RWMutex` — correct but adds lock/unlock overhead on every request (hot path)
- Channel-based swap — correct but adds goroutine indirection with no benefit

### D2: Server config snapshot compared field-by-field

**Choice:** Capture `origServer := cfg.Server` (a value copy) at startup. On each reload, compare `newCfg.Server` to `origServer` field by field:
- `ListenPorts`: sorted slice equality
- `TLS`: pointer nil-check + field equality
- `RedirectHTTP`: bool equality

When any field differs, log the warning naming the changed fields and skip applying the server block. The route table is still updated regardless.

```go
changed := serverConfigChanged(origServer, newCfg.Server)
if len(changed) > 0 {
    logger.Info("server config changed (%s); restart required to apply",
        strings.Join(changed, ", "))
}
```

**Rationale:** A simple field-by-field compare is readable, requires no serialisation dependency, and makes the warning message precise ("listen_ports changed" vs. "tls changed").

**Alternatives considered:**
- Marshal both to YAML/JSON and compare bytes — adds serialisation overhead and loses field-name precision in the warning
- `reflect.DeepEqual` — works but is opaque and doesn't tell us which field changed

### D3: Watcher accepts `onEvent func(error)` — caller owns logging

**Choice:** Extend `watcher.New` to accept two callbacks alongside `reloadFunc`:

```go
func New(filePath string, reloadFunc func() error, onReloadErr func(error), onFSErr func(error)) (*Watcher, error)
```

The `watchLoop` calls these instead of `fmt.Printf`. `main.go` passes closures that call `logger.Error`. Remove `fmt.Printf`/`fmt.Println` from the watcher entirely.

**Rationale:** The watcher package should not import `logger` (that would create a coupling from a utility package to an application-level concern). Passing callbacks keeps the watcher generic. The caller (`main.go`) owns the logging policy.

**Alternatives considered:**
- Import `logger` directly in watcher — creates an unnecessary dependency; also, the watcher package predates the logger package
- Keep `fmt.Printf` and accept the noise — rejected: the whole point of structured-logging was to silence output at `error` level

### D4: Reload re-applies log_level before rebuilding routes

**Choice:** In the reload callback, after `config.Load()` succeeds, call `logger.ParseLevel` + `logger.SetLevel` before rebuilding host groups. If parse fails (invalid log_level), log the error and abort the reload entirely (don't apply partial config).

**Rationale:** Log level is cheap to change and has no server-state side effects. Applying it first means subsequent reload log messages use the new level. Aborting on invalid log_level is consistent with the "keep old config on error" principle.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| `origServer` snapshot never updates — server config warning fires forever after first change | Documented: the warning is accurate; the running server config never changes. If user wants new ports they must restart. |
| `atomic.Pointer` requires buildHandler to call `rt.Load()` on every request | One atomic load per request — negligible overhead (single cacheline read on modern hardware) |
| Watcher callback signature change breaks existing callers | Only one call site in `main.go`; updated in the same PR |
| Sorted slice comparison for `listen_ports` — sorting mutates the slice | Sort copies, not the originals |
