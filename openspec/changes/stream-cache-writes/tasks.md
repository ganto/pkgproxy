## 1. FileCache Interface & Implementation

- [ ] 1.1 Add `CreateTempWriter(uri string) (*os.File, error)` to the `FileCache` interface and implement it in `pkg/cache/cache.go` — creates parent dirs, validates path traversal, returns a temp file handle
- [ ] 1.2 Add `CommitTempFile(tmpPath string, uri string, mtime time.Time) error` to the `FileCache` interface and implement it in `pkg/cache/cache.go` — sets mtime via `os.Chtimes`, renames temp file to final path
- [ ] 1.3 Refactor `SaveToDisk` to use `CreateTempWriter` and `CommitTempFile` internally
- [ ] 1.4 Add unit tests for `CreateTempWriter` (valid URI, missing parent dirs, path traversal rejection)
- [ ] 1.5 Add unit tests for `CommitTempFile` (successful commit, `IsCached` returns true after commit, path traversal rejection)
- [ ] 1.6 Verify existing `SaveToDisk` tests still pass

## 2. Prerequisite: Forward Content-Length

- [ ] 2.0 Add `Content-Length` to `allowedResponseHeaders` in `pkg/pkgproxy/proxy.go` so the Cache middleware can read it after `next(c)` returns

## 3. Cache Middleware Streaming

- [ ] 3.1 Implement a `resilientWriter` struct in `pkg/pkgproxy/` that lazily creates a temp file (via `CreateTempWriter`) on the first `Write()` call; on any write error (including creation failure), returns `len(b), nil` for that write, logs the error, and marks itself as failed so all subsequent writes are also discarded (returning `len(b), nil`); exposes a `failed` flag, tracks bytes written, and provides a `Close()` method for the underlying temp file
- [ ] 3.2 Implement a `safeWriter` struct in `pkg/pkgproxy/` that wraps an `io.Writer` and absorbs write errors starting from the first failure — on any write error, returns `len(b), nil` for that write and marks itself as failed so all subsequent writes are also discarded (returning `len(b), nil`) — isolating the cache from client disconnects
- [ ] 3.3 Modify the `Cache` middleware in `pkg/pkgproxy/proxy.go` to use the `resilientWriter` (cache side) and `safeWriter` (client side) instead of allocating a `bytes.Buffer` for cache-miss responses
- [ ] 3.4 Wire the `io.MultiWriter` with both the `safeWriter`-wrapped client `ResponseWriter` and the `resilientWriter`, keeping the existing `bufferWriter` wrapper around the `io.MultiWriter`. Give `bufferWriter` a reference to the `safeWriter` so that `Flush()` and `WriteHeader()` become no-ops when `safeWriter.failed` is true
- [ ] 3.5 After `next(c)` returns, close the `resilientWriter`'s temp file (if created), then defer its removal — after a successful commit the file has been renamed so the deferred remove is a harmless ENOENT; on any failure path the deferred remove ensures cleanup
- [ ] 3.6 Replace the post-response `SaveToDisk` call with `CommitTempFile` (on status 200, `resilientWriter.bytesWritten > 0`, `resilientWriter.failed` is false, and Content-Length matches if present); log and continue on `CommitTempFile` errors (same as current `SaveToDisk` error handling)
- [ ] 3.7 Add Content-Length validation: parse the response `Content-Length` header as an integer; if present and valid, compare against `resilientWriter.bytesWritten` before committing — skip commit and log warning on mismatch; if absent or non-numeric, skip validation
- [ ] 3.8 Add unit tests for `resilientWriter` (lazy creation on first write, no temp file on zero writes, successful write, error caught and suppressed, subsequent writes discarded, bytes-written tracking, close)
- [ ] 3.9 Add unit tests for `safeWriter` (passthrough on success, error absorbed after first failure, subsequent writes discarded)
- [ ] 3.10 Update proxy unit tests to cover the streaming cache-write path (cache miss with 200, non-200 cleanup, connection error creates no temp file, disk write error does not affect client, client disconnect does not prevent caching, truncated upstream rejected via Content-Length mismatch, missing Content-Length skips validation)

## 4. Validation

- [ ] 4.1 Run `make ci-check` — all linting, vulnerability checks, and tests pass
- [ ] 4.2 Run `make e2e` — end-to-end tests pass with the new streaming path
- [ ] 4.3 Update `CHANGELOG.md` unreleased section with the change
