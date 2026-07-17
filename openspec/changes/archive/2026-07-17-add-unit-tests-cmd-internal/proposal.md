## Why

The dev-proxy project currently has six Go source files under `/cmd` and `/internal` that lack any unit-test coverage. Without tests, regressions in core functionality (logging, proxying, routing, request review, shutdown handling, and TLS certificate management) can only be caught at integration time or in production. Adding comprehensive unit tests now will improve confidence in future refactors and reduce the risk of silent breakages.

## What Changes

- Add `*_test.go` files for every `.go` source file under `/cmd/dev-proxy` and `/internal` that does not already have one.
- Target packages: `logger`, `proxy`, `review`, `router`, `shutdown`, `tls`.
- Aim for **≥ 80 % code coverage** per package. If a package cannot reach 80 % after three iterations, document the blocking reason and move on.
- No behavioural changes to production code (purely additive test coverage).

## Capabilities

### New Capabilities
<!-- No new user-facing capabilities are being introduced; this is an engineering-quality change. -->
*None — this change is purely additive test coverage for existing capabilities.*

### Modified Capabilities
<!-- Existing capabilities whose REQUIREMENTS are changing (not just implementation).
     Only list here if spec-level behavior changes. Each needs a delta spec file.
     Use existing spec names from openspec/specs/. Leave empty if no requirement changes. -->
*None — no spec-level behaviour changes.*

## Impact

- Affected packages: `internal/logger`, `internal/proxy`, `internal/review`, `internal/router`, `internal/shutdown`, `internal/tls`.
- `cmd/dev-proxy` already has `main_test.go`; it will be kept as-is unless gaps are found.
- No API, dependency, or system changes.
- CI/test runner will execute the new tests automatically.
