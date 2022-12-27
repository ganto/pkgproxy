// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"os"

	"github.com/ganto/pkgproxy/pkg/pkgproxy"
	"github.com/spf13/cobra"
)

var (
	cacheDir       string
	configPath     string
	enableDebug    bool
	pkgProxyConfig pkgproxy.Config
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
	c.PersistentFlags().StringVarP(&configPath, "config", "c", defaultConfigPath, "path to config file")
	c.PersistentFlags().BoolVar(&enableDebug, "debug", false, "enable debugging")
	c.AddCommand(newServeCommand())

	return c
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if configPath == defaultConfigPath {
		value, found := os.LookupEnv(configPathEnvVar)
		if found {
			configPath = value
		}
	}

	if err := pkgproxy.LoadConfig(&pkgProxyConfig, configPath); err != nil {
		fmt.Printf("unable to load configuration '%s': %s\n", configPath, err.Error())
		os.Exit(1)
	}
}

// Execute starts the command
func Execute() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
