// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/ganto/pkgproxy/pkg/cache"
	"github.com/ganto/pkgproxy/pkg/utils"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	PkgProxy interface {
		Cache(echo.HandlerFunc) echo.HandlerFunc
		Upstream(echo.HandlerFunc) echo.HandlerFunc
	}

	PkgProxyConfig struct {
		CacheBasePath    string
		RepositoryConfig *RepoConfig
	}

	pkgProxy struct {
		Upstreams map[string]Upstream
	}
	Upstream struct {
		Cache   cache.Cache
		Mirrors []*url.URL
	}
)

func New(config *PkgProxyConfig) PkgProxy {
	upstreams := map[string]Upstream{}
	for _, repo := range utils.KeysFromMap(config.RepositoryConfig.Repositories) {
		var mirrors []*url.URL
		for _, mirror := range config.RepositoryConfig.Repositories[repo].Mirrors {
			url, err := url.Parse(mirror)
			if err == nil {
				mirrors = append(mirrors, url)
			}
		}
		cacheCfg := cache.PkgCacheConfig{
			BasePath:     config.CacheBasePath,
			FileSuffixes: config.RepositoryConfig.Repositories[repo].CacheSuffixes,
		}
		upstreams[repo] = Upstream{
			Cache:   cache.NewPkgCache(repo, &cacheCfg),
			Mirrors: mirrors,
		}
	}
	return &pkgProxy{
		Upstreams: upstreams,
	}
}

func (pp *pkgProxy) Cache(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		uri := c.Request().RequestURI

		if pp.isRepositoryRequest(uri) {
			cache := pp.Upstreams[getRepofromUri(uri)].Cache
			if cache.IsCacheCandidate(uri) {
				if cache.IsCached(uri) {
					return c.File(cache.GetFilePath(uri))
				}
			}
		}

		fmt.Println("Cache(): exec next() middleware")
		return next(c)
	}
}

func (pp *pkgProxy) Upstream(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		uri := c.Request().RequestURI
		if !pp.isRepositoryRequest(uri) {
			fmt.Println("Upstream(): exec next() middleware")
			return next(c)
		}

		req := c.Request()
		res := c.Response()

		repo := getRepofromUri(uri)
		tgt := pp.Upstreams[repo].Mirrors[0]
		req.Host = tgt.Host

		// trim repository handle
		req.RequestURI = strings.TrimPrefix(req.RequestURI, "/"+repo)
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/"+repo)

		transport := PkgProxyTransport{
			Rt:    http.DefaultTransport,
			Cache: pp.Upstreams[repo].Cache,
		}

		// Proxy
		var err error
		proxyHTTP(tgt, c, &transport).ServeHTTP(res, req)
		if e, ok := c.Get("_error").(error); ok {
			err = e
		}

		return err
	}
}

// Check if the request should be handled by PkgProxy
func (pp *pkgProxy) isRepositoryRequest(uri string) bool {
	repo := getRepofromUri(uri)
	return utils.Contains(utils.KeysFromMap(pp.Upstreams), repo)
}

// Return the repository name of the URL without leading "/"
func getRepofromUri(uri string) string {
	return strings.TrimPrefix(utils.RouteFromUri(uri), "/")
}

func proxyHTTP(tgt *url.URL, c echo.Context, transport *PkgProxyTransport) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(tgt)
	proxy.ErrorHandler = func(resp http.ResponseWriter, req *http.Request, err error) {
		// If the client canceled the request (usually by closing the connection), we can report a
		// client error (4xx) instead of a server error (5xx) to correctly identify the situation.
		// The Go standard library (at of late 2020) wraps the exported, standard
		// context.Canceled error with unexported garbage value requiring a substring check, see
		// https://github.com/golang/go/blob/6965b01ea248cabb70c3749fd218b36089a21efb/src/net/net.go#L416-L430
		if err == context.Canceled || strings.Contains(err.Error(), "operation was canceled") {
			httpError := echo.NewHTTPError(middleware.StatusCodeContextCanceled, fmt.Sprintf("client closed connection: %v", err))
			httpError.Internal = err
			c.Set("_error", httpError)
		} else {
			httpError := echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("upstream %s unreachable, could not forward: %v", tgt.Host, err))
			httpError.Internal = err
			c.Set("_error", httpError)
		}
	}
	proxy.Transport = transport
	return proxy
}
