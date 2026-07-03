# Testing

## Patterns

Tests use `httptest.NewServer` for real local HTTP servers — no mocks for the transport layer.

## Helpers (`pkg/pkgproxy/proxy_test.go`)

- `newTestProxy(t, mirrors)` — creates a `pkgProxy` with a single `testrepo` repository.
- `newTestProxyWithRetries(t, mirrors, retries)` — same, but also sets `retryBaseDelay = 0` so retry tests run instantly.
- `newTestApp(pp)` — builds an Echo app with the same middleware stack as production (`RequestID` → error handler → `Recover` → `Cache` → `ForwardProxy`).

## E2E Tests in CI

E2e tests (`.github/workflows/e2e.yaml`) run a full distro container matrix and are
expensive, so they don't run on every push. They run once per pull request, triggered
manually by a maintainer before merging:

```bash
gh workflow run e2e.yaml --ref <branch>
```

or via the Actions UI ("E2E Tests" → "Run workflow").

Merge gating: each PR event sets a pending `e2e` commit status, and the dispatched run
reports the result back as the `e2e` status used as a merge check.

Security constraint of the implementation: the status-setting job runs under
`pull_request_target` with a privileged token and therefore never checks out or
executes PR code.

## External Tests

Tests in `proxy_test.go` that hit httpbin.org are skipped by default. Enable with:

```bash
PKGPROXY_HTTPBIN_TESTS=1 go test -v -race ./pkg/pkgproxy/ -run TestForwardProxyWithHttpbin
```
