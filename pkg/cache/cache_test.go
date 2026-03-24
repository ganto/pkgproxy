// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cache

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheNew(t *testing.T) {
	c := New(&CacheConfig{BasePath: "cache", FileSuffixes: []string{}})
	assert.IsType(t, &cache{}, c)

	c = New(&CacheConfig{})
	assert.IsType(t, &cache{}, c)
}

func TestCacheConfig(t *testing.T) {
	cc := CacheConfig{BasePath: "cache", FileSuffixes: []string{}}

	c := New(&cc)
	assert.Equal(t, 0, len(c.GetFileSuffixes()))
}

func TestResolvedFilePath(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		uri     string
		want    string
		wantErr bool
	}{
		{
			name: "uri with leading slash",
			base: "/cache",
			uri:  "/myrepo/subdir/package.rpm",
			want: "/cache/myrepo/subdir/package.rpm",
		},
		{
			name: "uri without leading slash",
			base: "/cache",
			uri:  "myrepo/subdir/package.rpm",
			want: "/cache/myrepo/subdir/package.rpm",
		},
		{
			name: "relative base path",
			base: "cache",
			uri:  "/myrepo/package.rpm",
			want: "cache/myrepo/package.rpm",
		},
		{
			name: "base path with trailing slash is normalized",
			base: "/cache/",
			uri:  "/myrepo/package.rpm",
			want: "/cache/myrepo/package.rpm",
		},
		{
			name:    "traversal above base with dotdot segments",
			base:    "/cache",
			uri:     "/myrepo/../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "traversal via leading dotdot",
			base:    "/cache",
			uri:     "../other",
			wantErr: true,
		},
		{
			name:    "traversal to filesystem root",
			base:    "/cache",
			uri:     "/../../../etc/passwd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cache{config: &CacheConfig{BasePath: tt.base}}
			got, err := c.resolvedFilePath(tt.uri)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestCreateTempWriter(t *testing.T) {
	t.Run("valid URI creates temp file", func(t *testing.T) {
		baseDir := t.TempDir()
		c := New(&CacheConfig{BasePath: baseDir})

		f, err := c.CreateTempWriter("/myrepo/subdir/package.rpm")
		require.NoError(t, err)
		defer f.Close()
		defer os.Remove(f.Name())

		// Temp file should exist in the correct directory
		assert.DirExists(t, filepath.Join(baseDir, "myrepo", "subdir"))
		assert.FileExists(t, f.Name())
		assert.Equal(t, filepath.Join(baseDir, "myrepo", "subdir"), filepath.Dir(f.Name()))
	})

	t.Run("missing parent dirs created", func(t *testing.T) {
		baseDir := t.TempDir()
		c := New(&CacheConfig{BasePath: baseDir})

		f, err := c.CreateTempWriter("/deep/nested/path/package.rpm")
		require.NoError(t, err)
		defer f.Close()
		defer os.Remove(f.Name())

		assert.DirExists(t, filepath.Join(baseDir, "deep", "nested", "path"))
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		baseDir := t.TempDir()
		c := New(&CacheConfig{BasePath: baseDir})

		f, err := c.CreateTempWriter("/../../../etc/passwd")
		assert.Error(t, err)
		assert.Nil(t, f)
	})
}

func TestCommitTempFile(t *testing.T) {
	t.Run("successful commit", func(t *testing.T) {
		baseDir := t.TempDir()
		c := New(&CacheConfig{BasePath: baseDir})

		// Create a temp file via CreateTempWriter
		f, err := c.CreateTempWriter("/myrepo/package.rpm")
		require.NoError(t, err)
		_, err = f.WriteString("test content")
		require.NoError(t, err)
		tmpPath := f.Name()
		require.NoError(t, f.Close())

		mtime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
		err = c.CommitTempFile(tmpPath, "/myrepo/package.rpm", mtime)
		require.NoError(t, err)

		// Final file should exist with correct content and mtime
		finalPath := filepath.Join(baseDir, "myrepo", "package.rpm")
		data, err := os.ReadFile(finalPath)
		require.NoError(t, err)
		assert.Equal(t, "test content", string(data))

		info, err := os.Stat(finalPath)
		require.NoError(t, err)
		assert.Equal(t, mtime, info.ModTime().UTC())
	})

	t.Run("IsCached returns true after commit", func(t *testing.T) {
		baseDir := t.TempDir()
		c := New(&CacheConfig{BasePath: baseDir})

		assert.False(t, c.IsCached("/myrepo/package.rpm"))

		f, err := c.CreateTempWriter("/myrepo/package.rpm")
		require.NoError(t, err)
		_, err = f.WriteString("data")
		require.NoError(t, err)
		tmpPath := f.Name()
		require.NoError(t, f.Close())

		require.NoError(t, c.CommitTempFile(tmpPath, "/myrepo/package.rpm", time.Now()))
		assert.True(t, c.IsCached("/myrepo/package.rpm"))
	})

	t.Run("path traversal rejected", func(t *testing.T) {
		baseDir := t.TempDir()
		c := New(&CacheConfig{BasePath: baseDir})

		err := c.CommitTempFile("/tmp/fake.tmp", "/../../../etc/passwd", time.Now())
		assert.Error(t, err)
	})
}

func TestSaveToDiskStillWorks(t *testing.T) {
	baseDir := t.TempDir()
	c := New(&CacheConfig{BasePath: baseDir})

	buf := bytes.NewBufferString("buffered content")
	mtime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	err := c.SaveToDisk("/myrepo/path/package.rpm", buf, mtime)
	require.NoError(t, err)

	finalPath := filepath.Join(baseDir, "myrepo", "path", "package.rpm")
	data, err := os.ReadFile(finalPath)
	require.NoError(t, err)
	assert.Equal(t, "buffered content", string(data))

	info, err := os.Stat(finalPath)
	require.NoError(t, err)
	assert.Equal(t, mtime, info.ModTime().UTC())

	assert.True(t, c.IsCached("/myrepo/path/package.rpm"))
}
