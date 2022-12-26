package pkgproxy

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	Repositories map[string]Repository `yaml:"repositories"`
}

type Repository struct {
	Suffixes  []string `yaml:"suffixes"`
	Upstreams []string `yaml:"upstreams"`
}

func LoadConfig(config *Config, path string) error {
	fullPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	file, err := ioutil.ReadFile(fullPath)
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

func validateConfig(config *Config) error {
	if config.Repositories == nil {
		return errors.New("missing required key 'repositories'")
	}
	for handle, repoConfig := range config.Repositories {
		if alphanum := regexp.MustCompile("^[a-zA-Z0-9]*$").MatchString(handle); !alphanum {
			return errors.New(fmt.Sprintf("invalid repository name '%s'. Must be alphanumeric.", handle))
		}
		if repoConfig.Suffixes == nil {
			return errors.New(fmt.Sprintf("missing required key for repository '%s': suffixes", handle))
		}
		if repoConfig.Upstreams == nil {
			return errors.New(fmt.Sprintf("missing required key for repository '%s': upstreams", handle))
		}
	}
	return nil
}
