## 1. Add YAML dependency and new config module

- [x] 1.1 Add `gopkg.in/yaml.v3` to go.mod via `go get gopkg.in/yaml.v3`
- [x] 1.2 Create `internal/config/yaml.go` with struct definitions: `Config`, `ServerConfig`, `RouteConfig`, and the YAML top-level shape (`server` block + `routes` array)
- [x] 1.3 Implement YAML file loader function that reads and parses the config file path, returning `*Config` or an error
- [x] 1.4 Implement config validation: port range (1-65535), valid upstream URLs, required fields when TLS is enabled

## 2. Update main.go to use YAML config

- [x] 2.1 Replace `config.Load()` call in `main()` with new YAML-based loader
- [x] 2.2 Add `-config` CLI flag for custom config file path (default: `dev-proxy.yaml`)
- [x] 2.3 Update watcher initialization to watch the active YAML config file instead of `.env`
- [x] 2.4 Remove references to old `.env`-based env vars (`DEV_PROXY_PORT`, `DEV_PROXY_UPSTREAM`, etc.)

## 3. CertManager loads once per server

- [x] 3.1 Add disk-load branch in `GetOrGenerate`: when cert_file/key_file are provided, load via `tls.LoadX509KeyPair`
- [x] 3.2 Validate loaded certificate: check PEM parses, verify cert and key match (compare public key bytes), parse leaf certificate
- [x] 3.3 Cache loaded certificates in the existing `certCache` map using `"server"` as key (single entry for all ports)
- [x] 3.4 Return descriptive errors for missing files, mismatched keys, malformed PEM

## 4. Restructure startServers to iterate listen_ports

- [x] 4.1 Replace per-route server loop with per-listen-port server loop using `server.listen_ports`
- [x] 4.2 Each port gets its own `http.Server`; all share the single cached certificate from CertManager
- [x] 4.3 Add port-in-use error detection: wrap `ListenAndServe`/`ListenAndServeTLS` errors, detect "address already in use" via `strings.Contains`
- [x] 4.4 Print formatted fatal message with port number and diagnostic commands (`lsof -i :<N>` / `netstat -an | grep <N>`)

## 5. Add HTTP-to-HTTPS redirect at server level

- [x] 5.1 Create `redirectMiddleware(next http.Handler, targetPort int) http.Handler` that returns 301 to `https://<r.Host>:<targetPort><r.URL.Path>`
- [x] 5.2 Wire redirect middleware onto the HTTP listener's handler when `server.redirect_http == true`; TLS listeners skip it
- [x] 5.3 The router (shared across all ports) matches routes by path_prefix/host_match; no per-route port/TLS fields

## 6. Generate test fixture and verify build

- [x] 6.1 Create `dev-proxy.yaml` with a multi-port example (port 8080 HTTP + port 8443 HTTPS with self-signed fallback, redirect_http: true)
- [x] 6.2 Run `go build ./cmd/dev-proxy/` to confirm the project compiles cleanly
- [x] 6.3 Run `go vet ./...` to check for static analysis issues
