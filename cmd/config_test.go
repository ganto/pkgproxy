// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveConfigPath(t *testing.T) {
	tests := []struct {
		name         string
		localExists  bool
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
			want:         func(koDir string) string { return koDir + "/pkgproxy.yaml" },
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
			want:         func(koDir string) string { return koDir + "/pkgproxy.yaml" },
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
				err := os.WriteFile(filepath.Join(localDir, "pkgproxy.yaml"), []byte("repos: []\n"), 0600)
				if err != nil {
					t.Fatal(err)
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

			got := resolveConfigPath()
			assert.Equal(t, tt.want(koDir), got)
		})
	}
}
