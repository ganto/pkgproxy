package pkgproxy

import (
	"errors"
	"os"
	"testing"

	"github.com/ganto/pkgproxy/pkg/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- resilientWriter tests ---

func TestResilientWriterLazyCreation(t *testing.T) {
	fc := cache.New(&cache.CacheConfig{
		BasePath:     t.TempDir(),
		FileSuffixes: []string{".rpm"},
	})
	rw := newResilientWriter(fc, "/repo/package.rpm")

	// Before any write, no temp file should exist
	assert.Empty(t, rw.TmpPath())
	assert.False(t, rw.failed)
	assert.Equal(t, int64(0), rw.bytesWritten)

	// First write triggers lazy creation
	n, err := rw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.NotEmpty(t, rw.TmpPath())
	assert.False(t, rw.failed)
	assert.Equal(t, int64(5), rw.bytesWritten)

	require.NoError(t, rw.Close())
	defer os.Remove(rw.TmpPath())
}

func TestResilientWriterNoTempFileOnZeroWrites(t *testing.T) {
	fc := cache.New(&cache.CacheConfig{
		BasePath:     t.TempDir(),
		FileSuffixes: []string{".rpm"},
	})
	rw := newResilientWriter(fc, "/repo/package.rpm")

	assert.Empty(t, rw.TmpPath())
	assert.NoError(t, rw.Close())
}

func TestResilientWriterSuccessfulWrite(t *testing.T) {
	fc := cache.New(&cache.CacheConfig{
		BasePath:     t.TempDir(),
		FileSuffixes: []string{".rpm"},
	})
	rw := newResilientWriter(fc, "/repo/package.rpm")

	_, _ = rw.Write([]byte("part1"))
	_, _ = rw.Write([]byte("part2"))

	assert.False(t, rw.failed)
	assert.Equal(t, int64(10), rw.bytesWritten)

	require.NoError(t, rw.Close())
	defer os.Remove(rw.TmpPath())

	data, err := os.ReadFile(rw.TmpPath())
	require.NoError(t, err)
	assert.Equal(t, "part1part2", string(data))
}

func TestResilientWriterCreationError(t *testing.T) {
	// Use an invalid base path to trigger creation failure
	fc := cache.New(&cache.CacheConfig{
		BasePath:     "/nonexistent/path/that/cannot/exist",
		FileSuffixes: []string{".rpm"},
	})
	rw := newResilientWriter(fc, "/repo/package.rpm")

	n, err := rw.Write([]byte("data"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.True(t, rw.failed)
	assert.Equal(t, int64(0), rw.bytesWritten)
}

func TestResilientWriterSubsequentWritesDiscarded(t *testing.T) {
	// Use an invalid base path to trigger failure on first write
	fc := cache.New(&cache.CacheConfig{
		BasePath:     "/nonexistent/path",
		FileSuffixes: []string{".rpm"},
	})
	rw := newResilientWriter(fc, "/repo/package.rpm")

	// First write fails (creation error)
	n1, err1 := rw.Write([]byte("first"))
	assert.NoError(t, err1)
	assert.Equal(t, 5, n1)
	assert.True(t, rw.failed)

	// Subsequent writes are silently discarded
	n2, err2 := rw.Write([]byte("second"))
	assert.NoError(t, err2)
	assert.Equal(t, 6, n2)
	assert.Equal(t, int64(0), rw.bytesWritten)
}

func TestResilientWriterBytesWrittenTracking(t *testing.T) {
	fc := cache.New(&cache.CacheConfig{
		BasePath:     t.TempDir(),
		FileSuffixes: []string{".rpm"},
	})
	rw := newResilientWriter(fc, "/repo/package.rpm")

	rw.Write([]byte("abc"))
	assert.Equal(t, int64(3), rw.bytesWritten)

	rw.Write([]byte("defgh"))
	assert.Equal(t, int64(8), rw.bytesWritten)

	require.NoError(t, rw.Close())
	os.Remove(rw.TmpPath())
}

// --- safeWriter tests ---

type errWriter struct {
	failAfter int
	writes    int
}

func (w *errWriter) Write(b []byte) (int, error) {
	w.writes++
	if w.writes > w.failAfter {
		return 0, errors.New("write error")
	}
	return len(b), nil
}

func TestSafeWriterPassthrough(t *testing.T) {
	inner := &errWriter{failAfter: 100}
	sw := newSafeWriter(inner)

	n, err := sw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.False(t, sw.failed)
	assert.Equal(t, 1, inner.writes)
}

func TestSafeWriterErrorAbsorbed(t *testing.T) {
	inner := &errWriter{failAfter: 0} // fail on first write
	sw := newSafeWriter(inner)

	n, err := sw.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.True(t, sw.failed)
}

func TestSafeWriterSubsequentWritesDiscarded(t *testing.T) {
	inner := &errWriter{failAfter: 1} // succeed first, fail second
	sw := newSafeWriter(inner)

	// First write succeeds
	n1, err1 := sw.Write([]byte("first"))
	assert.NoError(t, err1)
	assert.Equal(t, 5, n1)
	assert.False(t, sw.failed)

	// Second write fails and is absorbed
	n2, err2 := sw.Write([]byte("second"))
	assert.NoError(t, err2)
	assert.Equal(t, 6, n2)
	assert.True(t, sw.failed)

	// Third write is discarded without calling inner
	n3, err3 := sw.Write([]byte("third"))
	assert.NoError(t, err3)
	assert.Equal(t, 5, n3)
	assert.Equal(t, 2, inner.writes) // only 2 actual writes to inner
}
