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

- `pkgProxy` (`pkg/pkgproxy/proxy.go`) — holds `upstreams` map (repo name → mirrors + cache instance), `transport`, and `retryBaseDelay`. The `PkgProxy` interface exposes only `Cache` and `ForwardProxy` middleware funcs.
- `upstream` — per-repository struct bundling a `FileCache`, a list of parsed mirror `*url.URL`s, and the retry count.
- `FileCache` (`pkg/cache/cache.go`) — interface backed by a filesystem cache. Uses atomic write (temp file + `os.Rename`) to prevent partial reads. Path traversal is prevented in `resolvedFilePath`.
- `RepoConfig` / `Repository` (`pkg/pkgproxy/repository.go`) — YAML-loaded config: each repository has `mirrors`, `suffixes` (cache candidates), and optional `retries`.

## Mirror Failover & Retry (`tryMirrors`)

Mirrors are tried in order. Per mirror, up to `retries` attempts are made (default 1). Exponential backoff (`retryBaseDelay * 2^(attempt-2)`, starting at 1 s) is triggered only on 5xx responses. A single redirect (301/302/303/307/308) is followed per attempt. Connection-level errors skip immediately to the next mirror. The first 200 response wins; otherwise the last non-nil response is returned.

## Cache Write Path

When a file is a cache candidate and not yet cached, the `http.ResponseWriter` is replaced with a `bufferWriter` that tee-writes to both the original writer and an in-memory `bytes.Buffer`. After `next(c)` returns with status 200, the buffer is flushed to disk via `FileCache.SaveToDisk`. The file mtime is set to the upstream `Last-Modified` header value if present.

## Header Filtering

Both request and response headers are whitelisted via `allowedRequestHeaders` / `allowedResponseHeaders` slices in `proxy.go`. Non-listed headers are stripped before forwarding.
