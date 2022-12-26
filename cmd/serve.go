package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"path"

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
		Short:            "Start reverse proxy",
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

		pkgCacheCfg := cache.PkgCacheConfig{
			FileSuffixes: repoConfig.Suffixes,
			BasePath:     cacheDir,
		}
		pkgCache := cache.NewPkgCache(handle, &pkgCacheCfg)

		var targetUrls []*url.URL
		for _, upstreamUrl := range repoConfig.Upstreams {
			url, err := url.Parse(upstreamUrl)
			if err != nil {
				app.Logger.Fatal(err)
			}
			targetUrls = append(targetUrls, url)
			fmt.Printf("Setting up handle '/%s' → %s\n", handle, url)
		}
		proxyTargets := []*middleware.ProxyTarget{
			{
				URL: targetUrls[0],
			},
		}
		proxyCfg := middleware.ProxyConfig{
			Balancer: middleware.NewRoundRobinBalancer(proxyTargets),
			Rewrite:  map[string]string{"/" + handle + "/*": "/$1"},
			Transport: pkgproxy.PkgProxyTransport{
				Host:  targetUrls[0].Hostname(),
				Rt:    http.DefaultTransport,
				Cache: pkgCache,
			},
		}
		// forward request to upstream servers
		group.Use(middleware.ProxyWithConfig(proxyCfg))
	}
	app.Logger.Fatal(app.Start(fmt.Sprintf("%s:%d", listenAddress, listenPort)))

	return nil
}
