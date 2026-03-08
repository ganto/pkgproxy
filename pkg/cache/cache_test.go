// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
