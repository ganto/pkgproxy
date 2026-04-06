// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	yaml "gopkg.in/yaml.v3"
)

// repoHandleRegexp defines which repository names are accepted
var repoHandleRegexp = regexp.MustCompile("^[a-zA-Z0-9_~.-]*$")

// RepoConfig defines the upstream package repositories
type RepoConfig struct {
	Repositories map[string]Repository `yaml:"repositories"`
}

type Repository struct {
	CacheSuffixes []string `yaml:"suffixes"`
	Exclude       []string `yaml:"exclude,omitempty"`
	Mirrors       []string `yaml:"mirrors"`
	Retries       int      `yaml:"retries,omitempty"`
}

func LoadConfig(config *RepoConfig, path string) error {
	fullPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	file, err := os.ReadFile(fullPath) //nolint:gosec
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return err
	}

	if err = validateConfig(config); err != nil {
		return err
	}

	return nil
}

func validateConfig(config *RepoConfig) error {
	if config.Repositories == nil {
		return errors.New("missing required key 'repositories'")
	}
	for handle, repoConfig := range config.Repositories {
		if alphanum := repoHandleRegexp.MatchString(handle); !alphanum {
			return fmt.Errorf("invalid repository name '%s'. Must be alphanumeric or in '-', '_', '.', '~'", handle)
		}
		if repoConfig.CacheSuffixes == nil {
			return fmt.Errorf("missing required key for repository '%s': suffixes", handle)
		}
		if repoConfig.Mirrors == nil {
			return fmt.Errorf("missing required key for repository '%s': mirrors", handle)
		}
		// Warn if suffixes contains "*" alongside other entries (redundant).
		hasWildcard := false
		var redundant []string
		for _, s := range repoConfig.CacheSuffixes {
			if s == "*" {
				hasWildcard = true
			} else {
				redundant = append(redundant, s)
			}
		}
		if hasWildcard && len(redundant) > 0 {
			slog.Warn("repository has wildcard suffix '*' with redundant explicit suffixes",
				"repository", handle, "redundant_suffixes", redundant)
		}
	}
	return nil
}
