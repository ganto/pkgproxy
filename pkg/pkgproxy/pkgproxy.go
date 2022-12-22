package pkgproxy

import (
	"fmt"
	"net/http"
	"net/url"

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func StartServer(host string, port uint16) error {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	repos := map[string]string{"fedora": "http://download.fedoraproject.org/pub/fedora/linux"}
	for handle, targetUrl := range repos {
		url, err := url.Parse(targetUrl)
		if err != nil {
			e.Logger.Fatal(err)
		}
		targets := []*middleware.ProxyTarget{
			{
				URL: url,
			},
		}
		g := e.Group("/" + handle)
		c := middleware.ProxyConfig{
			Balancer: middleware.NewRoundRobinBalancer(targets),
			Rewrite:  map[string]string{"/" + handle + "/*": "/$1"},
			Transport: pkgProxyTransport{
				host:             url.Hostname(),
				rt:               http.DefaultTransport,
				cachedFileSuffix: []string{".rpm"},
			},
		}
		g.Use(middleware.ProxyWithConfig(c))
		fmt.Printf("Setting up handle '/%s' → %s\n", handle, url)
	}

	fmt.Printf("Starting reverse proxy on %s:%d\n", host, port)
	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", host, port)))

	return nil
}
