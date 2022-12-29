// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	yaml "gopkg.in/yaml.v3"
)

// RepoConfig defines the upstream package repositories
type RepoConfig struct {
	Repositories map[string]Repository `yaml:"repositories"`
}

type Repository struct {
	CacheSuffixes []string `yaml:"suffixes"`
	Mirrors       []string `yaml:"mirrors"`
}

func LoadConfig(config *RepoConfig, path string) error {
	fullPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	file, err := os.ReadFile(fullPath)
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
		if alphanum := regexp.MustCompile("^[a-zA-Z0-9_~.-]*$").MatchString(handle); !alphanum {
			return fmt.Errorf("invalid repository name '%s'. Must be alphanumeric or in '-', '_', '.', '~'", handle)
		}
		if repoConfig.CacheSuffixes == nil {
			return fmt.Errorf("missing required key for repository '%s': suffixes", handle)
		}
		if repoConfig.Mirrors == nil {
			return fmt.Errorf("missing required key for repository '%s': mirrors", handle)
		}
	}
	return nil
}
