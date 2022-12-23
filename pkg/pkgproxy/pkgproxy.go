package pkgproxy

import (
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/ganto/pkgproxy/pkg/cache"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

func StartServer(enableDebug bool, basePath string, host string, port uint16) error {
	app := echo.New()
	app.Use(middleware.Logger())
	app.Use(middleware.Recover())
	if enableDebug {
		app.Logger.SetLevel(log.DEBUG)
	}

	repos := map[string]string{"fedora": "http://download.fedoraproject.org/pub/fedora/linux"}
	for handle, targetUrl := range repos {
		url, err := url.Parse(targetUrl)
		if err != nil {
			app.Logger.Fatal(err)
		}
		fmt.Printf("Setting up handle '/%s' → %s\n", handle, url)
		group := app.Group("/" + handle)

		// try serving file from local cache directory
		group.Use(middleware.Static(path.Join(basePath, handle)))

		pkgCacheCfg := cache.PkgCacheConfig{
			FileSuffixes: []string{".rpm", ".drpm"},
			BasePath:     basePath,
		}
		pkgCache := cache.NewPkgCache(handle, &pkgCacheCfg)

		proxyTargets := []*middleware.ProxyTarget{
			{
				URL: url,
			},
		}
		proxyCfg := middleware.ProxyConfig{
			Balancer: middleware.NewRoundRobinBalancer(proxyTargets),
			Rewrite:  map[string]string{"/" + handle + "/*": "/$1"},
			Transport: pkgProxyTransport{
				host:  url.Hostname(),
				rt:    http.DefaultTransport,
				cache: pkgCache,
			},
		}
		// forward request to upstream servers
		group.Use(middleware.ProxyWithConfig(proxyCfg))
	}
	app.Logger.Fatal(app.Start(fmt.Sprintf("%s:%d", host, port)))

	return nil
}
