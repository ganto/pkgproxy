# Testing

## Patterns

Tests use `httptest.NewServer` for real local HTTP servers — no mocks for the transport layer.

## Helpers (`pkg/pkgproxy/proxy_test.go`)

- `newTestProxy(t, mirrors)` — creates a `pkgProxy` with a single `testrepo` repository.
- `newTestProxyWithRetries(t, mirrors, retries)` — same, but also sets `retryBaseDelay = 0` so retry tests run instantly.
- `newTestApp(pp)` — builds an Echo app with the same middleware stack as production (`RequestID` → error handler → `Recover` → `Cache` → `ForwardProxy`).

## External Tests

Tests in `proxy_test.go` that hit httpbin.org are skipped by default. Enable with:

```bash
PKGPROXY_HTTPBIN_TESTS=1 go test -v -race ./pkg/pkgproxy/ -run TestForwardProxyWithHttpbin
```
