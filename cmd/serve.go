// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/ganto/pkgproxy/pkg/cache"
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
	app.Use(middleware.Logger())
	app.Use(middleware.Recover())
	if enableDebug {
		app.Logger.SetLevel(log.DEBUG)
	}

	for handle, repoConfig := range pkgProxyConfig.Repositories {
		group := app.Group("/" + handle)

		// try serving file from local cache directory
		group.Use(middleware.Static(path.Join(cacheDir, handle)))

		cacheCfg := cache.PkgCacheConfig{
			FileSuffixes: repoConfig.Suffixes,
			BasePath:     cacheDir,
		}
		cache := cache.NewPkgCache(handle, &cacheCfg)

		fmt.Printf("Setting up handle '/%s' → %s\n", handle, strings.Join(repoConfig.Upstreams, ", "))
		var targetUrls []*url.URL
		for _, upstreamUrl := range repoConfig.Upstreams {
			url, err := url.Parse(upstreamUrl)
			if err != nil {
				app.Logger.Fatal(err)
			}
			targetUrls = append(targetUrls, url)
		}
		repoConfig := pkgproxy.RepoConfig{
			Cache:   cache,
			Mirrors: targetUrls,
			UrlPath: handle,
		}
		group.Use(pkgproxy.RepositoryWithConfig(repoConfig))
	}
	app.Logger.Fatal(app.Start(fmt.Sprintf("%s:%d", listenAddress, listenPort)))

	return nil
}
