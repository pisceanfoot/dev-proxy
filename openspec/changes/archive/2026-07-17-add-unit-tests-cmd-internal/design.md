## Context

The dev-proxy codebase contains six packages under `/internal` that currently ship without any unit-test coverage:

| Package | File(s) | Current Tests |
|---------|---------|---------------|
| `logger` | `logger.go` | None |
| `proxy` | `proxy.go` | None |
| `review` | `review.go` | None |
| `router` | `router.go` | None |
| `shutdown` | `shutdown.go` | None |
| `tls` | `cert.go` | None |

Some packages (`config`, `cors`, `static`, `watcher`) already have test files and will be left untouched unless gaps are discovered during the coverage audit.

The project uses the standard Go toolchain (`go test`) and there are no custom test frameworks or mocking libraries in use today.

## Goals / Non-Goals

**Goals:**
- Achieve ≥ 80 % statement coverage for each of the six target packages.
- Write idiomatic Go table-driven tests using the standard library only.
- Where external dependencies make direct testing hard (e.g., `os.Stderr`, OS signals), use testable interfaces or dependency injection without changing public API signatures.
- If 80 % cannot be reached after three attempts, document the blocker and move on.

**Non-Goals:**
- Refactoring production code for testability beyond minimal, safe adaptations.
- Adding integration or end-to-end tests.
- Changing any user-facing behaviour or public API signatures.
- Covering `cmd/dev-proxy/main.go` beyond its existing `main_test.go`.

## Decisions

1. **Standard-library-only testing**
   - *Rationale*: Keeps the build simple and consistent with existing tests. No new dependencies.

2. **Table-driven tests for all packages**
   - *Rationale*: Reduces boilerplate and makes it easy to add new cases as edge conditions are discovered.

3. **Minimal dependency injection for side-effecting code**
   - `logger`: The `Logger` struct already holds state; tests will instantiate `Logger` directly and, where needed, redirect output to a custom `io.Writer` by extending the struct or capturing `os.Stderr` via pipe.
   - `shutdown`: `Manager` exposes `Register` and `DoShutdown`; signals will be injected via a channel or by directly calling `DoShutdown`.
   - `review`: The `Reviewer` struct already has `SetInput`; output will be captured through an `io.Writer` field.
   - `tls`: `CertManager` methods are deterministic; temporary directories will be used for disk-loading tests.

4. **Coverage measured with `go test -cover`**
   - *Rationale*: The built-in coverage tool is the single source of truth. A small script or Makefile target can iterate packages and report per-package percentages.

## Risks / Trade-offs

- **[Risk]** Reaching 80 % coverage in `proxy` may require spinning up an HTTP server, making tests slower.  
  → *Mitigation*: Test the `Director` function and error paths directly; use `httptest` for lightweight server stubs.

- **[Risk]** `shutdown.Wait()` blocks on OS signals, which is hard to unit-test without race conditions.  
  → *Mitigation*: Test `DoShutdown`, `Register`, and context cancellation directly; leave signal-path coverage as a known gap if necessary.

- **[Risk]** Self-signed certificate generation in `tls` is non-deterministic due to randomness.  
  → *Mitigation*: Assert on structural properties (PEM type, parsed leaf validity) rather than byte-for-byte equality.

- **[Risk]** `review` uses `fmt.Printf` directly in `SendRequest`, bypassing the configurable `outputWriter`.  
  → *Mitigation*: Test the method as-is for now; note the inconsistency in coverage rationale if it blocks the target.
