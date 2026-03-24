## Context

The cache middleware in `pkg/pkgproxy/proxy.go` intercepts cache-miss responses by replacing the `http.ResponseWriter` with a `bufferWriter` that tees writes to both the client and an in-memory `bytes.Buffer`. After the request completes with status 200, the buffer is flushed to disk via `FileCache.SaveToDisk`, which itself writes to a temp file and renames it into place.

This means every cacheable response byte is held in RAM until the entire response is received. For large packages (kernel, firmware — 100+ MB), this causes significant memory spikes. Concurrent large downloads multiply the problem.

## Goals / Non-Goals

**Goals:**
- Eliminate memory buffering of cache-miss responses by streaming directly to a temp file on disk
- Keep the `FileCache` interface as the owner of path resolution, traversal protection, and atomic write semantics
- Maintain identical client-facing behavior (same response headers, same caching semantics)

**Non-Goals:**
- Cache eviction, size limits, or TTL policies (separate concern)
- Changing the cache-hit serving path (already efficient — serves from disk)
- Supporting concurrent writes to the same cache key (current behavior: last writer wins via rename, which is acceptable)

## Decisions

### 1. Tee to temp file instead of `bytes.Buffer`

**Decision:** Replace the `bytes.Buffer` in the cache middleware with an `*os.File` (temp file) as the second writer in the `io.MultiWriter`.

**Alternatives considered:**
- **Hybrid memory/disk (spill at threshold):** Buffer small responses in memory, spill to disk above a configurable size. Rejected: adds complexity (two code paths, arbitrary threshold) for minimal benefit — disk writes happen anyway.
- **Check `Content-Length` to decide:** Use the upstream response header to choose buffer vs file upfront. Rejected: not all upstreams send `Content-Length` (chunked encoding), so a fallback is still needed.

**Rationale:** The disk write is unavoidable for cached files. Moving it earlier in the pipeline (streaming vs. post-response) keeps memory flat with no behavioral trade-offs.

### 2. Extend `FileCache` with `CreateTempWriter` and `CommitTempFile`

**Decision:** Add two methods to the `FileCache` interface:
- `CreateTempWriter(uri string) (*os.File, error)` — creates a temp file in the correct cache subdirectory (creating parent dirs as needed), with path traversal protection
- `CommitTempFile(tmpPath string, uri string, mtime time.Time) error` — sets mtime and atomically renames the temp file to the final cache path. This method trusts that the URI was already validated by `CreateTempWriter` and does not perform its own path traversal check

**Alternatives considered:**
- **Middleware handles file I/O directly:** The middleware calls `GetFilePath()`, creates the temp file, and renames. Rejected: leaks cache internals (path resolution, directory creation, traversal protection) into the middleware.

**Rationale:** Keeps all filesystem and security logic encapsulated in `FileCache`. The middleware only deals with an `*os.File` writer.

### 3. Refactor `SaveToDisk` to use the same primitives

**Decision:** Reimplement `SaveToDisk` internally using `CreateTempWriter` + write + `CommitTempFile`. This keeps `SaveToDisk` available for callers that have an in-memory buffer, while ensuring a single code path for atomic writes.

### 4. Resilient writer wrapper with lazy temp file creation (cache side)

**Decision:** Wrap the cache write side in a `resilientWriter` that lazily creates the temp file (via `CreateTempWriter`) on the first `Write()` call. This avoids creating temp files for connection-level failures where `ForwardProxy` returns an error before any response bytes are written. On any write error (including temp file creation failure), the wrapper absorbs all disk write errors from the first failure onward: it returns `len(b), nil` for that write (satisfying `io.MultiWriter`'s short-write check), logs the error, and marks itself as failed — all subsequent writes are also silently discarded (returning `len(b), nil`). It exposes a `failed` flag that the post-response code checks — if set, the commit is skipped and the temp file is cleaned up. The wrapper maintains a `bytesWritten` counter that is incremented only by the number of bytes actually written to the underlying temp file (the return value of the successful `os.File.Write` call), not by the `len(b)` returned to callers. This counter is used for the "data was written" check and Content-Length validation. After streaming completes (i.e. `next(c)` returns), the temp file MUST be closed before attempting commit or cleanup.

The existing `bufferWriter` struct remains as the outermost wrapper around the `io.MultiWriter`. It is needed to satisfy the `http.ResponseWriter` interface (`WriteHeader`, `Flush`, `Hijack`) while routing `Write` calls through the `io.MultiWriter`. The `resilientWriter` and `safeWriter` (Decision 5) are the two legs *inside* the `io.MultiWriter`. The `bufferWriter` SHALL hold a reference to the `safeWriter` so that `Flush()` and `WriteHeader()` become no-ops when `safeWriter.failed` is true — these methods bypass the `io.MultiWriter` and call the raw `ResponseWriter` directly, which could panic or error after a client disconnect.

**Rationale:** `io.MultiWriter` propagates errors from any underlying writer. Without this wrapper, a disk write error would surface as a response write error to the client and would stop `io.Copy`, preventing the upstream body from being fully consumed. The existing code already follows the principle that cache failures must not affect clients (see the `SaveToDisk` error handling). The resilient wrapper preserves this contract in the streaming path. Lazy creation avoids unnecessary filesystem operations when upstream requests fail at the connection level.

### 5. Client-side write isolation

**Decision:** Wrap the client-side `ResponseWriter` in the `io.MultiWriter` with a writer that absorbs write errors starting from the first failure (e.g. a lightweight `safeWriter` that, on any write error, returns `len(b), nil` for that write and marks itself as failed so all subsequent writes are also discarded). Combined with the cache-side `resilientWriter` (Decision 4), both legs of the `io.MultiWriter` absorb errors, ensuring `io.Copy` in `ForwardProxy` reads the entire upstream response body regardless of client disconnects or disk failures.

**Alternatives considered:**
- **Custom "best-effort" MultiWriter that ignores individual writer errors:** A replacement for `io.MultiWriter` that continues writing to remaining writers when one fails. Rejected: more complex to implement and test; wrapping individual writers is simpler and reuses the existing pattern.
- **Detect client disconnect and switch to cache-only reads:** Monitor the client connection state and, on disconnect, stop teeing and drain the upstream body directly to the cache writer. Rejected: adds complexity with connection state monitoring; wrapping both writers achieves the same effect with no conditional logic.

**Rationale:** `ForwardProxy` already intentionally detaches the upstream context from the client (`context.Background()`), so the upstream connection stays open even when the client disconnects. However, `io.Copy` still stops when _any_ writer in the `io.MultiWriter` returns an error. By absorbing errors on both the cache and client sides, `io.Copy` drains the upstream body to completion. If a client aborts a download and retries moments later, the cache already has (or is completing) the full file from the first request.

### 6. Content-Length validation before commit

**Decision:** Before committing the temp file, compare the `resilientWriter`'s bytes-written count against the `Content-Length` response header (if present). If they don't match (fewer or more bytes than expected), skip the commit and clean up the temp file. Since both the client and cache writers absorb errors (Decisions 4 and 5), `io.Copy` reads the entire upstream body — a Content-Length mismatch indicates an upstream issue (e.g. truncated connection, or a buggy upstream sending more data than declared), not a client disconnect.

**Prerequisite:** `Content-Length` must be added to `allowedResponseHeaders` so it is forwarded from the upstream response and accessible to the Cache middleware via `c.Response().Header().Get("Content-Length")` after `next(c)` returns.

**Alternatives considered:**
- **Check if upstream body was fully consumed:** Attempt a small read from `rsp.Body` after `io.Copy` returns. Works without `Content-Length` but is fragile and hacky.

**Rationale:** Most package mirrors send `Content-Length` for binary files. The `resilientWriter` already tracks bytes written, so the comparison is nearly free. When `Content-Length` is absent (chunked encoding), the validation is skipped — this is acceptable because chunked transfers are uncommon for large binary packages served by Linux mirrors.

### 7. Cleanup via `defer` close and remove

**Decision:** After streaming completes, the `resilientWriter`'s temp file (if created) MUST be closed. If the temp file was created, register cleanup via `defer os.Remove(tmpPath)`. On the success path (200, commit), the file has been renamed so the remove is a harmless no-op (ENOENT). On any failure path (non-200, streaming error, partial response, or resilient writer failure), the temp file is cleaned up automatically. If the temp file was never created (connection-level error), no cleanup is needed.

## Risks / Trade-offs

- **Increased disk I/O during streaming:** Writes happen incrementally during the response instead of in one burst at the end. Mitigation: OS page cache absorbs this well; the total bytes written are identical. Lazy temp file creation avoids disk I/O entirely for connection-level failures.
- **Orphaned temp files on crash:** If the process is killed mid-stream, the deferred cleanup won't run. Mitigation: Temp files use a recognizable `*.tmp` pattern and live in the cache directory — a startup cleanup sweep could be added later if needed (non-goal for this change).
- **`FileCache` interface change is breaking:** External implementers of `FileCache` must add the new methods. Mitigation: No known external implementers exist. The interface is internal to this project.
- **Disk write failure during streaming:** `io.MultiWriter` propagates errors from any writer, so a disk error would kill the client response and stop upstream consumption. Mitigation: a `resilientWriter` wrapper swallows disk errors (returning `len(b), nil` to satisfy `io.MultiWriter`'s short-write check), sets a `failed` flag, and the commit is skipped — the client response is unaffected.
- **Full upstream consumption on client disconnect:** With both writers absorbing errors, `io.Copy` always reads the entire upstream response even if the client disconnected early. This uses bandwidth and disk I/O for a response no client is waiting for. Mitigation: the cached file serves subsequent requests immediately, avoiding a redundant upstream fetch. The upstream context already has a deadline if one was set on the original request, bounding the maximum consumption time.
