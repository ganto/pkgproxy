// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfigWildcardWithRedundantSuffixes(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	config := &RepoConfig{
		Repositories: map[string]Repository{
			"testrepo": {
				CacheSuffixes: []string{"*", ".rpm", ".drpm"},
				Mirrors:       []string{"https://example.com/"},
			},
		},
	}

	err := validateConfig(config)
	require.NoError(t, err)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "testrepo")
	assert.Contains(t, logOutput, "redundant")
}

func TestValidateConfigWildcardAloneNoWarning(t *testing.T) {
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	config := &RepoConfig{
		Repositories: map[string]Repository{
			"testrepo": {
				CacheSuffixes: []string{"*"},
				Mirrors:       []string{"https://example.com/"},
			},
		},
	}

	err := validateConfig(config)
	require.NoError(t, err)

	assert.Empty(t, buf.String())
}

func TestValidateConfigUpstreams(t *testing.T) {
	tests := []struct {
		name    string
		repo    Repository
		wantErr string
	}{
		{
			name: "mirrors only",
			repo: Repository{CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
		{
			name: "cdn only",
			repo: Repository{CacheSuffixes: []string{".rpm"}, CDN: "https://cdn.example.com/"},
		},
		{
			name: "cdn with complete mtls",
			repo: Repository{
				CacheSuffixes: []string{".rpm"},
				CDN:           "https://cdn.example.com/",
				MTLS:          &MTLSConfig{Cert: "cert.pem", Key: "key.pem"},
			},
		},
		{
			name:    "neither mirrors nor cdn",
			repo:    Repository{CacheSuffixes: []string{".rpm"}},
			wantErr: "mirrors or cdn",
		},
		{
			name: "mirrors and cdn together",
			repo: Repository{
				CacheSuffixes: []string{".rpm"},
				Mirrors:       []string{"https://mirror.example.com/"},
				CDN:           "https://cdn.example.com/",
			},
			wantErr: "mutually exclusive",
		},
		{
			name: "mtls without cert",
			repo: Repository{
				CacheSuffixes: []string{".rpm"},
				CDN:           "https://cdn.example.com/",
				MTLS:          &MTLSConfig{Key: "key.pem"},
			},
			wantErr: "requires both 'cert' and 'key'",
		},
		{
			name: "mtls without key",
			repo: Repository{
				CacheSuffixes: []string{".rpm"},
				CDN:           "https://cdn.example.com/",
				MTLS:          &MTLSConfig{Cert: "cert.pem"},
			},
			wantErr: "requires both 'cert' and 'key'",
		},
		{
			name: "mtls without cdn",
			repo: Repository{
				CacheSuffixes: []string{".rpm"},
				Mirrors:       []string{"https://mirror.example.com/"},
				MTLS:          &MTLSConfig{Cert: "cert.pem", Key: "key.pem"},
			},
			wantErr: "'mtls' requires 'cdn'",
		},
		{
			name:    "cdn without scheme",
			repo:    Repository{CacheSuffixes: []string{".rpm"}, CDN: "cdn.example.com/"},
			wantErr: "must use the http or https scheme",
		},
		{
			name:    "cdn without host",
			repo:    Repository{CacheSuffixes: []string{".rpm"}, CDN: "https:///content/"},
			wantErr: "missing a host",
		},
		{
			name:    "mirror with unsupported scheme",
			repo:    Repository{CacheSuffixes: []string{".rpm"}, Mirrors: []string{"ftp://mirror.example.com/"}},
			wantErr: "must use the http or https scheme",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RepoConfig{Repositories: map[string]Repository{"testrepo": tt.repo}}
			err := validateConfig(config)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Contains(t, err.Error(), "testrepo")
		})
	}
}

func TestLoadConfigCDNRepository(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pkgproxy.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`---
repositories:
  rhel:
    suffixes:
      - .rpm
    cdn: https://cdn.redhat.com/
    mtls:
      cert: entitlement.pem
      key: /etc/pki/entitlement/key.pem
`), 0o600))

	var config RepoConfig
	require.NoError(t, LoadConfig(&config, configPath))

	rhel := config.Repositories["rhel"]
	assert.Equal(t, "https://cdn.redhat.com/", rhel.CDN)
	assert.Empty(t, rhel.Mirrors)
	require.NotNil(t, rhel.MTLS)
	// Paths are passed through unchanged and resolved by the process at load time.
	assert.Equal(t, "entitlement.pem", rhel.MTLS.Cert)
	assert.Equal(t, "/etc/pki/entitlement/key.pem", rhel.MTLS.Key)
}

func TestLoadConfigBranding(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pkgproxy.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`---
branding:
  title: Acme Package Mirror
  description: Internal package cache for Acme Corp.
repositories:
  fedora:
    suffixes:
      - .rpm
    mirrors:
      - https://mirror.example.com/
`), 0o600))

	var config RepoConfig
	require.NoError(t, LoadConfig(&config, configPath))

	require.NotNil(t, config.Branding)
	assert.Equal(t, "Acme Package Mirror", config.Branding.Title)
	assert.Equal(t, "Internal package cache for Acme Corp.", config.Branding.Description)
}

func TestLoadConfigNoBranding(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pkgproxy.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`---
repositories:
  fedora:
    suffixes:
      - .rpm
    mirrors:
      - https://mirror.example.com/
`), 0o600))

	var config RepoConfig
	require.NoError(t, LoadConfig(&config, configPath))

	assert.Nil(t, config.Branding)
}
