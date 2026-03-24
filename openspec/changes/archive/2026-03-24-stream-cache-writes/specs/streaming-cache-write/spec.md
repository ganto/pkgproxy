## ADDED Requirements

### Requirement: FileCache provides temp file creation for streaming writes

The `FileCache` interface SHALL expose a `CreateTempWriter(uri string) (*os.File, error)` method that creates a temporary file in the correct cache subdirectory for the given URI. The method MUST create parent directories if they do not exist. The method MUST reject URIs that would resolve outside the cache base directory.

#### Scenario: Temp file created in correct directory
- **WHEN** `CreateTempWriter` is called with a valid URI (e.g. `/fedora/releases/42/Everything/x86_64/os/Packages/k/kernel-6.12.rpm`)
- **THEN** a temporary file is created in the same directory where the final cached file would reside, and the file handle is returned

#### Scenario: Parent directories created automatically
- **WHEN** `CreateTempWriter` is called with a URI whose parent directories do not yet exist in the cache
- **THEN** the required directories are created before the temp file is created

#### Scenario: Path traversal rejected
- **WHEN** `CreateTempWriter` is called with a URI containing path traversal sequences (e.g. `/../../../etc/passwd`)
- **THEN** an error is returned and no file is created

### Requirement: FileCache provides atomic commit of temp files into cache

The `FileCache` interface SHALL expose a `CommitTempFile(tmpPath string, uri string, mtime time.Time) error` method that atomically moves a temp file to the final cache path for the given URI and sets the file modification time.

#### Scenario: Successful commit
- **WHEN** `CommitTempFile` is called with a valid temp file path, a valid URI, and a timestamp
- **THEN** the temp file is renamed to the final cache path and the file's modification time is set to the given timestamp

#### Scenario: File becomes visible to IsCached after commit
- **WHEN** `CommitTempFile` completes successfully for a given URI
- **THEN** `IsCached` returns `true` for that URI

### Requirement: Cache middleware streams responses to disk instead of memory

The cache middleware SHALL tee cache-miss responses to a temp file on disk instead of an in-memory `bytes.Buffer`. The response MUST be streamed simultaneously to the client and to the temp file via a resilient writer that lazily creates the temp file on the first write and absorbs disk write errors without affecting the client response.

#### Scenario: Large package cached without memory spike
- **WHEN** a cache-miss request is received for a large cacheable file (e.g. 200 MB)
- **THEN** the response is streamed to both the client and a temp file, without accumulating the full response in memory

#### Scenario: Successful response committed to cache
- **WHEN** the upstream returns status 200 for a cacheable file and streaming completes without cache write errors
- **THEN** the temp file is committed to the cache via `CommitTempFile` with the upstream `Last-Modified` timestamp (or current time if absent)

#### Scenario: Non-200 response cleans up temp file
- **WHEN** the upstream returns a non-200 status for a cacheable file
- **THEN** the temp file is removed and nothing is committed to the cache

#### Scenario: Upstream error does not create temp file
- **WHEN** the upstream request fails at the network/connection level (e.g. DNS failure, connection reset) and the `ForwardProxy` middleware returns an error before writing any response bytes
- **THEN** no temp file is created and nothing is committed to the cache

#### Scenario: Client disconnect does not prevent caching
- **WHEN** a client disconnects mid-download of a cacheable file and the upstream response is still streaming
- **THEN** the upstream body continues to be read to completion, the response is fully written to the temp file, and (if all other conditions are met) the temp file is committed to the cache

#### Scenario: CommitTempFile failure does not affect client response
- **WHEN** the response has been fully streamed to both the client and the temp file, but `CommitTempFile` fails (e.g. rename fails due to permissions)
- **THEN** the error is logged, the temp file is cleaned up, and the client response is unaffected (the client already received the full response)

#### Scenario: Disk write error does not affect client response
- **WHEN** a disk write error occurs while teeing the response to the temp file (e.g. disk full)
- **THEN** the client response continues uninterrupted, the cache write is abandoned, and the temp file is cleaned up

### Requirement: Resilient writer isolates cache write failures from client responses

The cache middleware SHALL use a resilient writer wrapper that lazily creates the temp file (via `CreateTempWriter`) on the first `Write()` call. This avoids creating temp files for connection-level failures where no response bytes are written. On any write error (including temp file creation failure), the wrapper MUST log the error, return `len(b), nil` for that write (not propagate the error), and mark itself as failed; all subsequent writes MUST also return `len(b), nil` without attempting the underlying write. The wrapper MUST expose a mechanism (e.g. a `failed` flag) for the post-response code to detect that the cache write failed, so the commit can be skipped. The wrapper MUST maintain a `bytesWritten` counter that is incremented only by the number of bytes actually written to the underlying temp file (i.e. the return value of the successful `os.File.Write` call); this counter is NOT incremented for discarded writes after failure. This counter serves as the "data was written" check in the post-response code (replacing the previous `len(rspBody.Bytes()) > 0` check). Note: the `len(b), nil` return value to `io.MultiWriter` is independent of the `bytesWritten` counter — the return value satisfies the `io.MultiWriter` contract, while the counter tracks actual bytes on disk. The wrapper MUST provide a way to close the underlying temp file after streaming completes and before commit or cleanup.

#### Scenario: Temp file created lazily on first write
- **WHEN** the first `Write()` call is made to the resilient writer
- **THEN** the temp file is created via `CreateTempWriter` before writing data

#### Scenario: No temp file created on connection failure
- **WHEN** the upstream request fails at the connection level and `ForwardProxy` returns an error before writing any response bytes
- **THEN** no temp file is created and no filesystem cleanup is needed

#### Scenario: Write error caught and suppressed
- **WHEN** the underlying temp file write (or lazy creation) returns an error
- **THEN** the resilient writer logs the error, returns `len(b), nil` to the `io.MultiWriter` (since `io.MultiWriter` treats a short write as `io.ErrShortWrite`), and marks itself as failed

#### Scenario: Subsequent writes discarded after failure
- **WHEN** a write error has previously occurred
- **THEN** all subsequent writes to the resilient writer are silently discarded (returning success)

#### Scenario: Temp file closed after streaming completes
- **WHEN** `next(c)` returns (streaming is done)
- **THEN** the resilient writer's temp file (if created) is closed before the commit-or-cleanup decision

#### Scenario: Commit skipped when writer has failed
- **WHEN** the response completes with status 200 but the resilient writer's failed flag is set
- **THEN** `CommitTempFile` is not called and the temp file is cleaned up

#### Scenario: Commit skipped when no bytes were written
- **WHEN** the response completes with status 200 but the resilient writer's bytes-written count is zero
- **THEN** `CommitTempFile` is not called and the temp file is cleaned up

### Requirement: Client-side writer isolates cache from client disconnects

The cache middleware SHALL wrap the client-side `ResponseWriter` in the `io.MultiWriter` with a writer that absorbs write errors starting from the first failure. On any write error, the wrapper MUST return `len(b), nil` for that write (not propagate the error) and mark itself as failed; all subsequent writes MUST also return `len(b), nil` without attempting the underlying write. Combined with the cache-side `resilientWriter`, this ensures `io.Copy` in `ForwardProxy` reads the entire upstream response body to completion regardless of whether the client disconnects mid-download.

#### Scenario: First write error absorbed silently
- **WHEN** a write to the client `ResponseWriter` fails for the first time
- **THEN** the client-side wrapper returns `len(b), nil` for that write (does not propagate the error) and marks itself as failed

#### Scenario: Subsequent writes discarded after client failure
- **WHEN** the client-side wrapper has previously encountered a write error
- **THEN** all subsequent writes return `len(b), nil` without attempting the underlying write

#### Scenario: Cache receives full response despite client disconnect
- **WHEN** the client disconnects mid-download but the upstream continues streaming
- **THEN** the `resilientWriter` (cache side) receives every byte from the upstream response body because `io.Copy` is not stopped by client-side write errors

#### Scenario: Client-side wrapper does not suppress errors before first failure
- **WHEN** writes to the client `ResponseWriter` succeed
- **THEN** the client-side wrapper passes data through transparently with no overhead beyond the function call

### Requirement: bufferWriter skips Flush and WriteHeader after client disconnect

The `bufferWriter` struct SHALL hold a reference to the `safeWriter`. When `safeWriter.failed` is true, `bufferWriter.Flush()` and `bufferWriter.WriteHeader()` MUST be no-ops — they SHALL NOT call through to the underlying `ResponseWriter`. This prevents panics or errors from calling methods on a disconnected client's `ResponseWriter`.

#### Scenario: Flush is a no-op after client disconnect
- **WHEN** the `safeWriter` has marked itself as failed (client disconnected) and `Flush()` is called on the `bufferWriter`
- **THEN** the call returns without invoking the underlying `ResponseWriter`'s `Flush()`

#### Scenario: WriteHeader is a no-op after client disconnect
- **WHEN** the `safeWriter` has marked itself as failed (client disconnected) and `WriteHeader()` is called on the `bufferWriter`
- **THEN** the call returns without invoking the underlying `ResponseWriter`'s `WriteHeader()`

#### Scenario: Flush works normally before client disconnect
- **WHEN** the `safeWriter` has not failed and `Flush()` is called on the `bufferWriter`
- **THEN** the call delegates to the underlying `ResponseWriter`'s `Flush()` as normal

### Requirement: Cache middleware validates Content-Length before committing

The cache middleware SHALL compare the resilient writer's bytes-written count against the `Content-Length` response header (forwarded from upstream via `allowedResponseHeaders`) before committing the temp file. Since both the client and cache writers absorb errors, `io.Copy` reads the entire upstream body — a Content-Length mismatch indicates the upstream connection was truncated (e.g. upstream server closed the connection early or a network interruption between the proxy and upstream). If `Content-Length` is present and the counts do not match, the commit MUST be skipped and the temp file cleaned up. If `Content-Length` is absent, this validation SHALL be skipped.

#### Scenario: Complete download committed
- **WHEN** the upstream response has `Content-Length: 1000` and the resilient writer has written 1000 bytes
- **THEN** the temp file is committed to the cache

#### Scenario: Truncated upstream response rejected
- **WHEN** the upstream response has `Content-Length: 1000` but the resilient writer has only written 500 bytes (e.g. upstream connection dropped mid-stream)
- **THEN** `CommitTempFile` is not called, the temp file is cleaned up, and a warning is logged

#### Scenario: Oversized upstream response rejected
- **WHEN** the upstream response has `Content-Length: 1000` but the resilient writer has written more than 1000 bytes (e.g. buggy upstream)
- **THEN** `CommitTempFile` is not called, the temp file is cleaned up, and a warning is logged

#### Scenario: Missing Content-Length skips validation
- **WHEN** the upstream response does not include a `Content-Length` header (e.g. chunked encoding)
- **THEN** the Content-Length validation is skipped and the commit proceeds based on the other checks (status 200, bytes written > 0, writer not failed)

#### Scenario: Malformed Content-Length skips validation
- **WHEN** the upstream response includes a `Content-Length` header that cannot be parsed as an integer
- **THEN** the Content-Length validation is skipped and the commit proceeds based on the other checks (status 200, bytes written > 0, writer not failed)

### Requirement: SaveToDisk remains functional

The existing `SaveToDisk(uri string, buf *bytes.Buffer, mtime time.Time) error` method SHALL continue to work for callers that have an in-memory buffer. Its internal implementation MAY be refactored to use `CreateTempWriter` and `CommitTempFile`.

#### Scenario: SaveToDisk still works with buffer input
- **WHEN** `SaveToDisk` is called with a URI, a `bytes.Buffer`, and a timestamp
- **THEN** the file is written to the cache at the correct path with the correct modification time, identical to current behavior
