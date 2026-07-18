# AGENTS.md — dev-proxy

A lightweight, config-driven HTTP/HTTPS reverse proxy in Go. Single module, minimal dependencies.

## Build & Run

- **No Makefile, no task runner** — use raw `go` commands:
  - Build: `go build ./cmd/dev-proxy`
  - Run: `./dev-proxy -config dev-proxy.yaml`
  - Default config path is `dev-proxy.yaml`; override with `-config` flag
  - Env file override: `-env-file .env.production`

## Test

- Run all tests: `go test ./...`
- Per-package coverage: `go test -cover ./internal/<pkg>`
- Detailed coverage report:
  ```bash
  go test -coverprofile=/tmp/c.out ./internal/<pkg> && go tool cover -func=/tmp/c.out
  ```
- **Coverage target: ≥ 80% per package** — retry up to 3 times if missed, document blockers if still below target
- **Standard library only** — no testify, no mock frameworks, no external test deps

## Architecture

- **Module**: `dev-proxy`, Go 1.23.3
- **Only external dependency**: `github.com/fsnotify/fsnotify` (file watcher)
- **Entrypoint**: `cmd/dev-proxy/main.go`
- **Internal packages**:
  - `config` — YAML parsing + `${VAR}` / `${VAR:-default}` env interpolation
  - `proxy` — `httputil.ReverseProxy` wrapper with host/path rewriting
  - `router` — two-phase host-then-path matching (glob + exact/prefix/regex)
  - `tls` — self-signed cert generation + cert caching (`CertManager`)
  - `shutdown` — graceful shutdown on SIGINT/SIGTERM
  - `watcher` — config hot-reload via `fsnotify`
  - `cors` — CORS preflight middleware
  - `static` — static file serving
  - `logger` — levelled stderr logging
  - `review` — interactive request/response review (TUI-like)

## Config & Env

- Config file: `dev-proxy.yaml` (YAML, default in repo root)
- Env interpolation in config values: `${VAR}` (required) or `${VAR:-default}`
- `.env` file + OS env variables combined; OS env wins
- Override via env vars: `DEV_PROXY_CONFIG`, `DEV_PROXY_ENV_FILE`

## OpenSpec Workflow

This repo uses OpenSpec for structured change management via `.opencode/skills/`:

- **Propose**: `/opsx-propose <change-name>` — creates change with `proposal.md`, `design.md`, `tasks.md`
- **Implement**: `/opsx-apply <change-name>` — reads tasks, makes code changes
- **Archive**: `/opsx-archive <change-name>` — moves completed change to `openspec/changes/archive/`
- Delta specs live in `openspec/changes/<name>/specs/`

## Style & Constraints

- No generated code (except self-signed TLS certs at runtime)
- No lint config present — rely on `go vet ./...`
- Keep tests table-driven, minimal, and standard-library-only
- No custom build tags or integration test suites
