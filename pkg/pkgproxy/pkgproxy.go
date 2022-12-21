package pkgproxy

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/ganto/pkgproxy/pkg/utils"
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
			Transport: headerUpdate{
				host: url.Hostname(),
				r:    http.DefaultTransport,
			},
			ModifyResponse: followRedirect,
		}
		g.Use(middleware.ProxyWithConfig(c))
		fmt.Printf("Setting up handle '/%s' → %s\n", handle, url)
	}

	fmt.Printf("Starting reverse proxy on %s:%d\n", host, port)
	e.Logger.Fatal(e.Start(fmt.Sprintf("%s:%d", host, port)))

	return nil
}

func followRedirect(rsp *http.Response) error {
	if rsp.StatusCode == 301 || rsp.StatusCode == 302 {
		location, err := rsp.Location()
		if err != nil {
			return err
		}
		r, err := http.Get(location.String())
		if err != nil {
			return err
		}
		updateResponse(r, rsp)
	}
	return nil
}

func updateResponse(src *http.Response, dst *http.Response) error {
	dst.ProtoMajor = src.ProtoMajor
	dst.ProtoMinor = src.ProtoMinor
	dst.Request.Host = src.Request.Host
	dst.Request.RequestURI = src.Request.RequestURI
	dst.Status = src.Status
	dst.StatusCode = src.StatusCode
	dst.ContentLength = src.ContentLength
	dst.Body = src.Body

	srcHeaders := utils.KeysFromMap(src.Header)
	dstHeaders := utils.KeysFromMap(dst.Header)

	for _, header := range utils.ListIntersection(srcHeaders, dstHeaders) {
		dst.Header.Set(header, src.Header.Get(header))
	}
	for _, header := range utils.ListDifference(dstHeaders, srcHeaders) {
		dst.Header.Del(header)
	}
	for _, header := range utils.ListDifference(srcHeaders, dstHeaders) {
		dst.Header.Add(header, src.Header.Get(header))
	}

	return nil
}
