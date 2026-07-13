## 1. Fix data race — atomic router swap

- [x] 1.1 In `cmd/dev-proxy/main.go`, replace `rt *router.Router` variable with `var rt atomic.Pointer[router.Router]`; add `"sync/atomic"` to imports
- [x] 1.2 Replace `rt = router.New(groups)` at startup with `rt.Store(router.New(groups))`
- [x] 1.3 In `buildHandler`, replace the `rt` capture with a per-request `r := rt.Load()` call before routing

## 2. Update watcher to accept logging callbacks

- [x] 2.1 Extend `watcher.New` signature to accept `onReloadErr func(error)` and `onFSErr func(error)` parameters after `reloadFunc`
- [x] 2.2 In `watchLoop`, replace the three `fmt.Printf`/`fmt.Println` calls with calls to `onReloadErr` / `onFSErr` as appropriate; remove the `fmt` import if it becomes unused
- [x] 2.3 Update the `watcher.New` call in `main.go` to pass `func(e error) { logger.Error("config reload failed: %v", e) }` and `func(e error) { logger.Error("watcher: %v", e) }` as the new callbacks

## 3. Snapshot server config and detect changes on reload

- [x] 3.1 Add helper `serverConfigChanged(a, b config.ServerConfig) []string` in `main.go` that returns a slice of changed field names; compare `ListenPorts` (sort copies), `RedirectHTTP`, and `TLS` (nil check + field compare)
- [x] 3.2 In `main()`, capture `origServer := cfg.Server` after initial config load
- [x] 3.3 In the reload callback, after `config.Load()` succeeds, call `serverConfigChanged(origServer, newCfg.Server)`; if non-empty, call `logger.Info("server config changed (%s); restart required to apply", strings.Join(changed, ", "))` and continue with the route rebuild (do not skip it)

## 4. Re-apply log_level on reload

- [x] 4.1 In the reload callback, after `config.Load()` succeeds and before rebuilding routes, call `logger.ParseLevel(newCfg.LogLevel)` then `logger.SetLevel(...)` to apply the new level immediately
- [x] 4.2 If `logger.ParseLevel` returns an error, call `logger.Error("config reload failed: invalid log_level: %v", err)` and return the error (abort reload — keep old config)

## 5. Explicit error logging on reload failure

- [x] 5.1 In the reload callback, if `config.Load()` returns an error, log it via `logger.Error("config reload failed: %v", err)` before returning the error (the watcher's `onReloadErr` callback also fires, but the reload callback itself should log clearly)

## 6. Verify

- [x] 6.1 Run `go build ./...` to confirm clean compile
- [x] 6.2 Run `go vet ./...` to check for static analysis issues
