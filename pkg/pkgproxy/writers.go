package pkgproxy

import (
	"io"
	"log/slog"
	"os"

	"github.com/ganto/pkgproxy/pkg/cache"
)

// resilientWriter lazily creates a temp file on the first Write() call and
// absorbs all disk write errors without propagating them. On any error
// (including temp file creation failure), it returns len(b), nil for that
// and all subsequent writes, satisfying io.MultiWriter's short-write check.
type resilientWriter struct {
	fc           cache.FileCache
	uri          string
	file         *os.File
	failed       bool
	bytesWritten int64
}

func newResilientWriter(fc cache.FileCache, uri string) *resilientWriter {
	return &resilientWriter{fc: fc, uri: uri}
}

func (w *resilientWriter) Write(b []byte) (int, error) {
	if w.failed {
		return len(b), nil
	}

	// Lazy creation of temp file on first write
	if w.file == nil {
		f, err := w.fc.CreateTempWriter(w.uri)
		if err != nil {
			slog.Error("cache temp file creation failed", "uri", w.uri, "error", err)
			w.failed = true
			return len(b), nil
		}
		w.file = f
	}

	n, err := w.file.Write(b)
	w.bytesWritten += int64(n)
	if err != nil {
		slog.Error("cache write failed", "uri", w.uri, "error", err)
		w.failed = true
		return len(b), nil
	}
	return len(b), nil
}

// Close closes the underlying temp file if it was created.
func (w *resilientWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Disable marks the writer as failed so any subsequent writes (e.g. from
// an error handler running after the Cache middleware returns) are silently
// discarded without creating temp files.
func (w *resilientWriter) Disable() {
	w.failed = true
}

// TmpPath returns the path of the temp file, or empty string if not created.
func (w *resilientWriter) TmpPath() string {
	if w.file != nil {
		return w.file.Name()
	}
	return ""
}

// safeWriter wraps an io.Writer and absorbs write errors starting from the
// first failure. After the first error, all subsequent writes return
// len(b), nil without attempting the underlying write.
type safeWriter struct {
	inner  io.Writer
	failed bool
}

func newSafeWriter(w io.Writer) *safeWriter {
	return &safeWriter{inner: w}
}

func (w *safeWriter) Write(b []byte) (int, error) {
	if w.failed {
		return len(b), nil
	}

	n, writeErr := w.inner.Write(b)
	if writeErr != nil {
		w.failed = true
		return len(b), nil //nolint:nilerr // intentionally absorbing write errors
	}
	return n, nil
}
