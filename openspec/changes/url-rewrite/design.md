## Context

The proxy currently rewrites the forwarded path in two ways:
1. **Prefix stripping** — the `path_prefix` is stripped from the incoming path before forwarding (handled in `proxy.go`'s director via `routePrefix`).
2. **Static upstream path** — `upstream_path` prepends a fixed prefix after prefix stripping (also in the director).

Neither mechanism allows dynamic path transformation based on captured groups. Regex-based rewriting is a standard capability in reverse proxies (nginx `rewrite`, Caddy `rewrite`, Traefik `ReplacePathRegex`) and is needed for path versioning, legacy remapping, and path normalization.

The existing route-matching pipeline already compiles `path_regex` at startup and stores a `*regexp.Regexp` on `MatchedRoute`. The director in `proxy.go` is the natural extension point for path transformation.

## Goals / Non-Goals

**Goals:**
- Allow a route to declare a `url_rewrite` block with a `match` regex and a `replace` template (supporting `$1`, `${name}` capture group references)
- Apply the rewrite to the upstream request path after route matching, inside the proxy director
- Fail fast at startup for invalid regex patterns
- Reject configs that specify both `url_rewrite` and `upstream_path` on the same route (mutually exclusive)

**Non-Goals:**
- Rewriting query strings, headers, or the upstream host/scheme
- Conditional rewrites (e.g. rewrite only when a header is present)
- Chaining multiple rewrites on one route
- Modifying the path logged in access/structured logs (logged path remains the original)

## Decisions

### Decision 1: Store compiled regex on `MatchedRoute`, apply rewrite in proxy director

The `MatchedRoute` struct already holds a `*regexp.Regexp` for path matching. We add a second field `URLRewriteRegex *regexp.Regexp` (the compiled `match` pattern) alongside `URLRewriteReplace string`. The director in `proxy.NewReverseProxy` receives these and calls `regexp.ReplaceAllString(path, replace)` when both are non-nil/non-empty.

**Alternative considered:** Apply the rewrite in a middleware before the proxy director. Rejected — it would require threading the route's regex down through additional layers. The director already mutates `req.URL.Path`; keeping all path mutation in one place is simpler.

### Decision 2: Mutual exclusion of `url_rewrite` and `upstream_path`

Both fields rewrite the forwarded path. Combining them produces confusing semantics (which applies first?). Validation in `config.validateRoute` returns an error if both are set on the same route.

**Alternative considered:** Apply them sequentially (`upstream_path` after regex rewrite). Rejected — the interaction is non-obvious and the use cases don't overlap; users should pick one.

### Decision 3: Compile `url_rewrite.match` at startup, fail fast on invalid pattern

Consistent with the existing `path_regex` approach. Invalid patterns produce a fatal startup error with the route index and the bad pattern, matching the existing error format.

### Decision 4: Rewrite applies to the full incoming path (not post-strip path)

`regexp.ReplaceAllString` is called on `req.URL.Path` as set by `originalDirector`. This is the full path after the upstream base URL is applied by `httputil.NewSingleHostReverseProxy`. This means `match` patterns should be written against the full incoming path, consistent with how `path_regex` matching works.

## Risks / Trade-offs

- **Regex complexity** → Mitigation: document that patterns follow Go `regexp/syntax`; invalid patterns are caught at startup.
- **Unexpected full-path replacement** — `ReplaceAllString` replaces all non-overlapping matches, not just the first. A pattern like `(.*)` would replace the entire path (intended), but a pattern matching a sub-segment would replace every occurrence. → Mitigation: document the semantics clearly; users who want a single replacement should anchor with `^`.
- **No query-string rewriting** — out of scope; users needing it can raise a follow-up.

## Migration Plan

- Feature is purely additive; existing configs without `url_rewrite` are unaffected.
- No schema version bump required.
- Rollback: revert the binary; no persistent state is involved.
