// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/ganto/pkgproxy/pkg/pkgproxy"
	"github.com/spf13/cobra"
)

var (
	cacheDir    string
	configPath  string
	enableDebug bool
	repoConfig  pkgproxy.RepoConfig
)

const (
	configPathEnvVar  = "PKGPROXY_CONFIG"
	defaultConfigPath = "./pkgproxy.yaml"
	defaultDir        = "cache"
)

// NewRootCommand creates a new root cli command instance
func NewRootCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "pkgproxy",
		Args:  cobra.MinimumNArgs(1),
		Short: "pkgproxy is a caching forward proxy for Linux package repositories ",
		Long: `pkgproxy is a Web server that serves Linux packages for various
repository types (RPM, DEB, ...) from a local cache. It can be used
as a central package server in a local network. Packages not available
in the local cache will be fetched transparently from configurable
upstream mirrors.

Complete documentation is available at https://github.com/ganto/pkgproxy`,
	}
	c.PersistentFlags().StringVar(&cacheDir, "cachedir", defaultDir, "path to the local cache directory")
	c.PersistentFlags().StringVarP(&configPath, "config", "c", defaultConfigPath, "path to the repository config file")
	c.PersistentFlags().BoolVar(&enableDebug, "debug", false, "enable debugging")
	c.AddCommand(newServeCommand())
	c.AddCommand(newVersionCommand())

	return c
}

const koDataPathEnvVar = "KO_DATA_PATH"

// injectServeDefault prepends "serve" to os.Args when the binary is called
// with no arguments, making the container image work without an explicit subcommand.
func injectServeDefault() {
	if len(os.Args) == 1 {
		os.Args = append([]string{os.Args[0], "serve"}, os.Args[1:]...)
	}
}

// resolveConfigPath returns the config file path to use when neither --config
// nor $PKGPROXY_CONFIG has been set explicitly.
func resolveConfigPath() string {
	if _, err := os.Stat(defaultConfigPath); err == nil {
		return defaultConfigPath
	} else if !errors.Is(err, os.ErrNotExist) {
		return defaultConfigPath
	}
	if koDataPath, ok := os.LookupEnv(koDataPathEnvVar); ok && koDataPath != "" {
		return koDataPath + "/pkgproxy.yaml"
	}
	return defaultConfigPath
}

func initConfig() error {
	if configPath == defaultConfigPath {
		if value, found := os.LookupEnv(configPathEnvVar); found {
			configPath = value
		} else {
			configPath = resolveConfigPath()
		}
	}

	if err := pkgproxy.LoadConfig(&repoConfig, configPath); err != nil {
		return fmt.Errorf("unable to load configuration from %s: %w", configPath, err)
	}
	return nil
}

// Execute starts the command
func Execute() {
	injectServeDefault()
	if err := NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
