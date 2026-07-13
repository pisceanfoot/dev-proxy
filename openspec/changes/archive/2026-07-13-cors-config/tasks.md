## 1. Config Schema

- [x] 1.1 Add `CORSAllowOrigin string \`yaml:"cors_allow_origin"\`` field to `HostGroup` in `internal/config/yaml.go`

## 2. Inheritance Resolution

- [x] 2.1 In `buildMatchedRoute` (`cmd/dev-proxy/main.go`), read the host group's `CORSAllowOrigin` and use it as the fallback when the route's `CORSAllowOrigin` is empty
- [x] 2.2 Ensure the resolved effective value (route overrides host) is used when constructing `router.CORSConfig`

## 3. CORS Middleware Audit

- [x] 3.1 Read `internal/cors/cors.go` and verify `addHeaders` is called for every request with an `Origin` header on all HTTP methods (GET, POST, PUT, DELETE, plain OPTIONS)
- [x] 3.2 Fix `cors.Middleware` if any method path skips `addHeaders` for requests that carry an `Origin` header

## 4. Tests

- [x] 4.1 Write a test: host-level `cors_allow_origin` is applied to a route that has none (`internal/cors/cors_test.go` or integration test in `cmd/dev-proxy/`)
- [x] 4.2 Write a test: route-level `cors_allow_origin` overrides the host-level value
- [x] 4.3 Write a test: no CORS headers when neither host nor route configures `cors_allow_origin`
- [x] 4.4 Write a test: GET request with `Origin` header receives `Access-Control-Allow-Origin`
- [x] 4.5 Write a test: POST request with `Origin` header receives `Access-Control-Allow-Origin`
- [x] 4.6 Write a test: OPTIONS preflight receives 204 with CORS headers
- [x] 4.7 Write a test: request without `Origin` header receives no CORS headers
