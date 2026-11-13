// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"bytes"
	"log/slog"
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
