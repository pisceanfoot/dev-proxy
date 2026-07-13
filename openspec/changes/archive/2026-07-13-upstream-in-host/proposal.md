## Why

Hosts in `dev-proxy.yaml` often proxy all their routes to the same upstream, but today every route must repeat the `upstream` field individually. When a host has many routes pointing to the same backend, this is verbose and error-prone — changing the backend requires editing every route.

## What Changes

- Add an optional `upstream` field to each entry in the `hosts` list.
- Routes within a host group that omit their own `upstream` field inherit the host-level default.
- Routes that do specify `upstream` continue to use their own value, overriding the host default.
- Validation ensures that every route has a resolved upstream (either its own or the host default); a route with no upstream and no host default remains a fatal error.

## Capabilities

### New Capabilities

_(none — this extends an existing capability)_

### Modified Capabilities

- `host-groups`: Add optional `upstream` field on host entries; define inheritance and override semantics for route resolution.

## Impact

- `config` package: `HostGroup` struct gains an `Upstream` field; route-resolution logic reads host upstream as fallback.
- `router` package: Route compilation must inherit host upstream when route upstream is absent.
- `watcher` / hot-reload: No changes needed; reload already rebuilds the full route table.
- Config validation: Must check that each route ends up with a resolved upstream after inheritance; emit fatal error if not.
- Existing configs are fully backward-compatible — the new field is optional.
