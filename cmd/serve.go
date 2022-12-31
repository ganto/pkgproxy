// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"

	"github.com/ganto/pkgproxy/pkg/pkgproxy"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/spf13/cobra"
)

var (
	listenAddress string
	listenPort    uint16
)

const (
	defaultAddress = "localhost"
	defaultPort    = 8080
)

func newServeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:              "serve",
		Args:             cobra.ArbitraryArgs,
		Short:            "Start forward proxy",
		RunE:             startServer,
		TraverseChildren: true,
	}
	c.PersistentFlags().StringVar(&listenAddress, "host", defaultAddress, "listen address of the pkgproxy.")
	c.PersistentFlags().Uint16Var(&listenPort, "port", defaultPort, "listen port of the pkgproxy.")

	return c
}

func startServer(_ *cobra.Command, _ []string) error {
	app := echo.New()
	app.HideBanner = true

	app.Use(middleware.Logger())
	app.Use(middleware.Recover())
	if enableDebug {
		app.Logger.SetLevel(log.DEBUG)
	}

	pkgProxy := pkgproxy.New(&pkgproxy.PkgProxyConfig{
		CacheBasePath:    cacheDir,
		RepositoryConfig: &repoConfig,
	})
	app.Use(pkgProxy.Cache)
	app.Use(pkgProxy.Upstream)
	app.Use(pkgProxy.Proxy)

	app.Logger.Fatal(app.Start(fmt.Sprintf("%s:%d", listenAddress, listenPort)))

	return nil
}
