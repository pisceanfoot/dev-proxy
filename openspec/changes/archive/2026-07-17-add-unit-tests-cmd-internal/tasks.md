## 1. Logger Package Tests

- [x] 1.1 Create `internal/logger/logger_test.go` with tests for `ParseLevel` (valid inputs, invalid input, case-insensitivity, empty string)
- [x] 1.2 Add tests for `Logger.Info` and `Logger.Debug` at all level thresholds (level filtering)
- [x] 1.3 Add tests for `Logger.Error` (always prints regardless of level)
- [x] 1.4 Add tests for package-level convenience functions (`Info`, `Debug`, `Error`)
- [x] 1.5 Run coverage check; if < 80 %, iterate up to two more times and document blockers if still below target (Result: 100%)

## 2. Proxy Package Tests

- [x] 2.1 Create `internal/proxy/proxy_test.go` with tests for `NewReverseProxy` (valid upstream, invalid URL)
- [x] 2.2 Add tests for `NewReverseProxy` with `rewriteHost = true` and `rewriteHost = false`
- [x] 2.3 Add tests for `NewReverseProxy` with `insecure = true` on HTTPS upstream
- [x] 2.4 Add tests for path rewriting with `routePrefix` and `upstreamPath`
- [x] 2.5 Add tests for `ServeHTTP` handler wrapper
- [x] 2.6 Run coverage check; if < 80 %, iterate up to two more times and document blockers if still below target (Result: 100%)

## 3. Review Package Tests

- [x] 3.1 Create `internal/review/review_test.go` with tests for `New` and `SetInput`
- [x] 3.2 Add tests for `ReviewRequest` (approve, discard, modify decisions)
- [x] 3.3 Add tests for `ReviewResponse` (approve, discard decisions)
- [x] 3.4 Add tests for `SendRequest` (approved vs discarded)
- [x] 3.5 Add tests for `readAnswer` (empty default, whitespace trimming)
- [x] 3.6 Add tests for `truncate` (short string, long string, exact length)
- [x] 3.7 Add tests for `CloneRequest` (body cloning, nil body handling)
- [x] 3.8 Run coverage check; if < 80 %, iterate up to two more times and document blockers if still below target (Result: 100%)

## 4. Router Package Tests

- [x] 4.1 Create `internal/router/router_test.go` with tests for `New` and `NewFromRoutes`
- [x] 4.2 Add tests for `Match` with exact path matching
- [x] 4.3 Add tests for `Match` with prefix path matching
- [x] 4.4 Add tests for `Match` with regex path matching
- [x] 4.5 Add tests for `Match` with host group matching (glob patterns)
- [x] 4.6 Add tests for `Match` with no-match scenarios (no host group, no path match)
- [x] 4.7 Add tests for `StripPrefix` (with and without matching prefix, empty suffix)
- [x] 4.8 Add tests for `HandleNoMatch` (status code, response body)
- [x] 4.9 Run coverage check; if < 80 %, iterate up to two more times and document blockers if still below target (Result: 100%)

## 5. Shutdown Package Tests

- [x] 5.1 Create `internal/shutdown/shutdown_test.go` with tests for `New`
- [x] 5.2 Add tests for `Register` (single and multiple cleanup functions)
- [x] 5.3 Add tests for `DoShutdown` (successful cleanup, error propagation)
- [x] 5.4 Add tests for context cancellation after `Wait` (signal-path testing via direct invocation)
- [x] 5.5 Run coverage check; if < 80 %, iterate up to two more times and document blockers if still below target (Result: 100%)

## 6. TLS Package Tests

- [x] 6.1 Create `internal/tls/cert_test.go` with tests for `NewCertManager`
- [x] 6.2 Add tests for `GetOrGenerate` (cache hit, cache miss, concurrent access)
- [x] 6.3 Add tests for `LoadFromDisk` (successful load, missing cert file, missing key file, mismatched cert/key)
- [x] 6.4 Add tests for `generateSelfSigned` (PEM structure, parsed leaf validity, key type)
- [x] 6.5 Run coverage check; if < 80 %, iterate up to two more times and document blockers if still below target (Result: 84.6%)

## 7. Final Verification

- [x] 7.1 Run full test suite: `go test ./...` (Result: all 11 packages pass)
- [x] 7.2 Run coverage report for all target packages: `go test -cover ./internal/logger ./internal/proxy ./internal/review ./internal/router ./internal/shutdown ./internal/tls`
- [x] 7.3 Document any packages that failed to reach 80 % coverage after three attempts, including reason and next steps (Result: none — all packages ≥ 80%)
- [x] 7.4 Ensure `go vet ./...` passes with no new warnings (Result: clean)
