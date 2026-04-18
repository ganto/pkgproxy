// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalConfig = "repositories: {}\n"

func writeConfig(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(minimalConfig), 0600))
	return p
}

func TestResolveConfigPath(t *testing.T) {
	tests := []struct {
		name         string
		localExists  bool
		localIsDir   bool
		koDataSet    bool
		koFileExists bool
		want         func(koDir string) string
	}{
		{
			name:         "local file wins over ko fallback",
			localExists:  true,
			koDataSet:    true,
			koFileExists: true,
			want:         func(_ string) string { return defaultConfigPath },
		},
		{
			name:         "ko fallback used when local missing",
			localExists:  false,
			koDataSet:    true,
			koFileExists: true,
			want:         func(koDir string) string { return filepath.Join(koDir, "pkgproxy.yaml") },
		},
		{
			name:        "both missing returns default path",
			localExists: false,
			koDataSet:   false,
			want:        func(_ string) string { return defaultConfigPath },
		},
		{
			name:         "KO_DATA_PATH set but no file returns ko path",
			localExists:  false,
			koDataSet:    true,
			koFileExists: false,
			want:         func(koDir string) string { return filepath.Join(koDir, "pkgproxy.yaml") },
		},
		{
			name:         "directory named pkgproxy.yaml falls through to ko",
			localExists:  true,
			localIsDir:   true,
			koDataSet:    true,
			koFileExists: true,
			want:         func(koDir string) string { return filepath.Join(koDir, "pkgproxy.yaml") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localDir := t.TempDir()
			koDir := t.TempDir()

			origDir, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Chdir(localDir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(origDir) })

			if tt.localExists {
				if tt.localIsDir {
					if err := os.Mkdir(filepath.Join(localDir, "pkgproxy.yaml"), 0700); err != nil {
						t.Fatal(err)
					}
				} else {
					if err := os.WriteFile(filepath.Join(localDir, "pkgproxy.yaml"), []byte("repos: []\n"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}

			if tt.koDataSet {
				t.Setenv(koDataPathEnvVar, koDir)
				if tt.koFileExists {
					err := os.WriteFile(filepath.Join(koDir, "pkgproxy.yaml"), []byte("repos: []\n"), 0600)
					if err != nil {
						t.Fatal(err)
					}
				}
			} else {
				// empty string is treated as unset by resolveConfigPath
				t.Setenv(koDataPathEnvVar, "")
			}

			got, err := resolveConfigPath()
			assert.NoError(t, err)
			assert.Equal(t, tt.want(koDir), got)
		})
	}
}

func TestInitConfig(t *testing.T) {
	t.Run("explicit --config bypasses lookup and env var", func(t *testing.T) {
		dir := t.TempDir()
		explicit := writeConfig(t, dir, "explicit.yaml")
		_ = writeConfig(t, dir, "env.yaml")

		t.Setenv(configPathEnvVar, filepath.Join(dir, "env.yaml"))
		t.Setenv(koDataPathEnvVar, "")

		configPath = explicit
		t.Cleanup(func() { configPath = defaultConfigPath })

		require.NoError(t, initConfig())
		assert.Equal(t, explicit, configPath)
	})

	t.Run("PKGPROXY_CONFIG bypasses ordered lookup", func(t *testing.T) {
		dir := t.TempDir()
		envFile := writeConfig(t, dir, "env.yaml")

		t.Setenv(configPathEnvVar, envFile)
		t.Setenv(koDataPathEnvVar, "")

		configPath = defaultConfigPath
		t.Cleanup(func() { configPath = defaultConfigPath })

		require.NoError(t, initConfig())
		assert.Equal(t, envFile, configPath)
	})

	t.Run("ordered lookup used when flag and env var unset", func(t *testing.T) {
		localDir := t.TempDir()

		origDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(localDir))
		t.Cleanup(func() { _ = os.Chdir(origDir) })

		writeConfig(t, localDir, "pkgproxy.yaml")
		// unset both env vars so the ordered lookup runs
		for _, key := range []string{configPathEnvVar, koDataPathEnvVar} {
			if prev, ok := os.LookupEnv(key); ok {
				require.NoError(t, os.Unsetenv(key))
				t.Cleanup(func() { _ = os.Setenv(key, prev) })
			}
		}

		configPath = defaultConfigPath
		t.Cleanup(func() { configPath = defaultConfigPath })

		require.NoError(t, initConfig())
		assert.Equal(t, defaultConfigPath, configPath)
	})
}
