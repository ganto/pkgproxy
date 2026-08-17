# Architecture

pkgproxy is a caching forward proxy for Linux package repositories, written in Go.

## Request Flow

```
Client → Cache middleware → ForwardProxy middleware → upstream mirrors
```

Both middlewares are registered as Echo middleware in `cmd/serve.go`. Order is significant: `Cache` runs first and either serves the file directly (cache hit) or installs a tee-writer to capture the response body for later caching. `ForwardProxy` then does the actual upstream fetch.

## Routing Convention

The **first path segment** of the URL is the repository name (e.g. `/fedora/...` → repo `fedora`). This is how `getRepoFromURI` / `isRepositoryRequest` route requests to the correct upstream config. Repository names must match `^[a-zA-Z0-9_~.-]*$`.

## Key Types

- `pkgProxy` (`pkg/pkgproxy/proxy.go`) — holds `upstreams` map (repo name → targets + cache instance), `transport`, and `retryBaseDelay`. The `PkgProxy` interface exposes only `Cache` and `ForwardProxy` middleware funcs. `New` returns an error, so misconfiguration (notably an unloadable mTLS key pair) aborts startup instead of failing per request.
- `upstream` — per-repository struct bundling a `FileCache`, the parsed upstream `targets` (`*url.URL`), an optional repository-scoped `transport`, and the retry count.
- `FileCache` (`pkg/cache/cache.go`) — interface backed by a filesystem cache. Uses atomic write (temp file + `os.Rename`) to prevent partial reads. Path traversal is prevented in `resolvedFilePath`.
- `RepoConfig` / `Repository` (`pkg/pkgproxy/repository.go`) — YAML-loaded config: each repository has `suffixes` (cache candidates), exactly one of `mirrors` or `cdn`, and optional `mtls` and `retries`.

## Upstream Kinds

A repository is backed either by an ordered list of `mirrors` or by a single `cdn`; the two are mutually exclusive and validated at config load. Both are normalized into the same `targets` slice, so failover, retry, redirect handling and path mapping are one code path.

A `cdn` may carry an `mtls` block. `New` loads the key pair and clones the proxy-wide `*http.Transport`, adding the client certificate to the clone's `TLSClientConfig`. An optional `mtls.ca` is appended to the system trust store (or to the base transport's existing `RootCAs`) so CDNs signed by a private CA — such as `cdn.redhat.com` — verify. The clone is stored on the `upstream`, which scopes both the credential and the extra trust to that one repository; `transportFor` falls back to the shared transport for everything else. Cloning preserves proxy env vars and timeouts.

## Mirror Failover & Retry (`tryUpstreams`)

Upstream targets are tried in order. Per target, up to `retries` attempts are made (default 1). Exponential backoff (`retryBaseDelay * 2^(attempt-2)`, starting at 1 s) is triggered only on 5xx responses. A single redirect (301/302/303/307/308) is followed per attempt. Connection-level errors skip immediately to the next target. The first 200 response wins; otherwise the last non-nil response is returned.

When a repository uses a repository-scoped (mTLS) transport and the redirect points at a different host, the redirect is followed with the shared transport instead, so the client certificate is never presented to a host other than the configured CDN.

## Cache Write Path

When a file is a cache candidate and not yet cached, the `http.ResponseWriter` is replaced with a `bufferWriter` that tee-writes to both the original writer and an in-memory `bytes.Buffer`. After `next(c)` returns with status 200, the buffer is flushed to disk via `FileCache.SaveToDisk`. The file mtime is set to the upstream `Last-Modified` header value if present.

## Header Filtering

Both request and response headers are whitelisted via `allowedRequestHeaders` / `allowedResponseHeaders` slices in `proxy.go`. Non-listed headers are stripped before forwarding.
