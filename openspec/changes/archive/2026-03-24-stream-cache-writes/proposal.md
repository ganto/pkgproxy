## Why

Cache-miss responses are fully buffered in a `bytes.Buffer` before being written to disk. Large packages (kernel packages, firmware blobs, etc. can exceed 100 MB) cause memory spikes proportional to their size, and concurrent downloads multiply the problem. Since the data is already being streamed to the client, there is no reason to also hold it in RAM — it can be teed directly to a temp file on disk.

## What Changes

- Extend the `FileCache` interface with two new methods (`CreateTempWriter`, `CommitTempFile`) that let callers stream data to a temp file and atomically commit it into the cache. `CommitTempFile` trusts that the URI was already validated by `CreateTempWriter`
- Modify the `Cache` middleware to tee upstream responses to a temp file instead of a `bytes.Buffer`
- Wrap the temp file writer in a resilient writer that lazily creates the temp file on the first write (avoiding filesystem work for connection-level failures) and absorbs all disk write errors from the first failure onward, including temp file creation failure, without affecting the client response
- Wrap the client-side `ResponseWriter` in a safe writer that absorbs write errors starting from the first failure (e.g. client disconnect), ensuring the upstream body is always fully consumed and the cache receives the complete response
- Add `Content-Length` to `allowedResponseHeaders` so it is forwarded from upstream and accessible to the Cache middleware
- Validate bytes written against `Content-Length` before committing to cache, preventing truncated upstream responses from being cached (**bugfix**)
- Refactor `SaveToDisk` internally to use the same temp-file primitives (no behavioral change for existing callers)

## Capabilities

### New Capabilities

- `streaming-cache-write`: Streaming write path for caching upstream responses to disk without buffering the full response in memory

### Modified Capabilities

_(none — no existing spec-level requirements change)_

## Impact

- **Code**: `pkg/cache/cache.go` (new interface methods, refactored internals), `pkg/pkgproxy/proxy.go` (cache middleware rewrite for streaming path, `Content-Length` added to `allowedResponseHeaders`), `pkg/pkgproxy/` (new `resilientWriter` and `safeWriter` types)
- **Tests**: `pkg/cache/cache_test.go` (new method tests), `pkg/pkgproxy/proxy_test.go` (updated cache-miss tests)
- **API**: `FileCache` interface gains two methods — any external implementers would need to add them (**BREAKING** for out-of-tree implementations, though none are known)
- **Behavior**: Functionally identical from the client's perspective for completed downloads; client disconnects no longer prevent caching — if a client aborts and retries, the file may already be cached from the first (completed upstream) request
