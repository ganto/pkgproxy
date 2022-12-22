package pkgproxy

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/ganto/pkgproxy/pkg/cache"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

func StartServer(enableDebug bool, path string, host string, port uint16) error {
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
		targets := []*middleware.ProxyTarget{
			{
				URL: url,
			},
		}
		g := app.Group("/" + handle)
		cacheCfg := cache.PkgCacheConfig{
			FileSuffixes: []string{".rpm", ".drpm"},
			Path:         path,
		}
		c := middleware.ProxyConfig{
			Balancer: middleware.NewRoundRobinBalancer(targets),
			Rewrite:  map[string]string{"/" + handle + "/*": "/$1"},
			Transport: pkgProxyTransport{
				host:  url.Hostname(),
				rt:    http.DefaultTransport,
				cache: cache.NewPkgCache(handle, &cacheCfg),
			},
		}
		g.Use(middleware.ProxyWithConfig(c))
		fmt.Printf("Setting up handle '/%s' → %s\n", handle, url)
	}

	fmt.Printf("Starting reverse proxy on %s:%d\n", host, port)
	app.Logger.Fatal(app.Start(fmt.Sprintf("%s:%d", host, port)))

	return nil
}
