// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	yaml "gopkg.in/yaml.v3"
)

// repoHandleRegexp defines which repository names are accepted
var repoHandleRegexp = regexp.MustCompile("^[a-zA-Z0-9_~.-]*$")

// RepoConfig defines the upstream package repositories
type RepoConfig struct {
	Branding     *BrandingConfig       `yaml:"branding,omitempty"`
	Repositories map[string]Repository `yaml:"repositories"`
}

// BrandingConfig customizes the title and description shown on the landing page.
type BrandingConfig struct {
	Title       string `yaml:"title,omitempty"`
	Description string `yaml:"description,omitempty"`
}

type Repository struct {
	CacheSuffixes []string    `yaml:"suffixes"`
	Exclude       []string    `yaml:"exclude,omitempty"`
	Mirrors       []string    `yaml:"mirrors,omitempty"`
	CDN           string      `yaml:"cdn,omitempty"`
	MTLS          *MTLSConfig `yaml:"mtls,omitempty"`
	Retries       int         `yaml:"retries,omitempty"`
}

// MTLSConfig holds the client certificate and private key that are presented to
// a CDN requiring mutual TLS (e.g. a Red Hat entitlement certificate), plus an
// optional CA bundle used to verify the CDN's own server certificate.
type MTLSConfig struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
	CA   string `yaml:"ca,omitempty"`
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
		if err := validateUpstream(handle, repoConfig); err != nil {
			return err
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

// validateUpstream checks that a repository declares exactly one kind of
// upstream — either a list of mirrors or a single CDN — and that an optional
// mTLS block is complete and only used together with a CDN.
func validateUpstream(handle string, repoConfig Repository) error {
	hasMirrors := len(repoConfig.Mirrors) > 0
	hasCDN := repoConfig.CDN != ""

	switch {
	case !hasMirrors && !hasCDN:
		return fmt.Errorf("missing required key for repository '%s': mirrors or cdn", handle)
	case hasMirrors && hasCDN:
		return fmt.Errorf("repository '%s': 'mirrors' and 'cdn' are mutually exclusive", handle)
	}

	if hasCDN {
		if err := validateUpstreamURL(handle, "cdn", repoConfig.CDN); err != nil {
			return err
		}
	}
	for _, mirror := range repoConfig.Mirrors {
		if err := validateUpstreamURL(handle, "mirrors", mirror); err != nil {
			return err
		}
	}

	if repoConfig.MTLS == nil {
		return nil
	}
	if !hasCDN {
		return fmt.Errorf("repository '%s': 'mtls' requires 'cdn'", handle)
	}
	if repoConfig.MTLS.Cert == "" || repoConfig.MTLS.Key == "" {
		return fmt.Errorf("repository '%s': 'mtls' requires both 'cert' and 'key'", handle)
	}
	return nil
}

// validateUpstreamURL ensures an upstream URL is absolute and uses HTTP(S), so
// that misconfigurations surface at startup rather than as proxy errors.
func validateUpstreamURL(handle string, key string, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("repository '%s': invalid '%s' URL '%s': %w", handle, key, rawURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("repository '%s': '%s' URL '%s' must use the http or https scheme", handle, key, rawURL)
	}
	if parsed.Host == "" {
		return fmt.Errorf("repository '%s': '%s' URL '%s' is missing a host", handle, key, rawURL)
	}
	return nil
}
