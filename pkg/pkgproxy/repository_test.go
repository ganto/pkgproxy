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
