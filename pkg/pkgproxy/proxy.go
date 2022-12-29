// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
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
		Proxy(echo.HandlerFunc) echo.HandlerFunc
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
		upstreams[repo] = Upstream{
			Cache: cache.New(&cache.CacheConfig{
				BasePath:     config.CacheBasePath,
				FileSuffixes: config.RepositoryConfig.Repositories[repo].CacheSuffixes,
			}),
			Mirrors: mirrors,
		}
	}
	return &pkgProxy{
		Upstreams: upstreams,
	}
}

// This middleware function checks if a request can be served from the
// local cache and does so if possible. Otherwise it will make sure the
// response is cached if necessary.
func (pp *pkgProxy) Cache(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var repoCache cache.Cache
		var rspBody *bytes.Buffer

		// the request URI might be changed later, keep the original value
		uri := strings.Clone(c.Request().RequestURI)

		if pp.isRepositoryRequest(uri) {
			repoCache = pp.Upstreams[getRepofromUri(uri)].Cache

			if repoCache.IsCacheCandidate(uri) {
				// serve from cache if possible
				if repoCache.IsCached(uri) {
					return c.File(repoCache.GetFilePath(uri))

				} else {
					// if not in cache write response body to buffer
					rspBody = new(bytes.Buffer)
					bodyWriter := io.MultiWriter(c.Response().Writer, rspBody)
					writer := &bufferWriter{
						Writer:         bodyWriter,
						ResponseWriter: c.Response().Writer}
					c.Response().Writer = writer
				}
			}
		}

		fmt.Println("Cache(): exec next() middleware")
		if err := next(c); err != nil {
			return err
		}
		fmt.Println("Cache(): handle response")

		if pp.isRepositoryRequest(uri) {
			if repoCache.IsCacheCandidate(uri) && !repoCache.IsCached(uri) && len(rspBody.Bytes()) > 0 {
				// save buffer to disk
				if err := repoCache.SaveToDisk(uri, rspBody); err != nil {
					// don't fail request if we cannot write to cache
					fmt.Printf("Error: %s", err.Error())
				}
			}
		}

		return nil
	}
}

// Adjust request to use upstream mirror depending on repository configuration
func (pp *pkgProxy) Upstream(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()

		if !pp.isRepositoryRequest(req.RequestURI) {
			fmt.Println("Upstream(): exec next() middleware")
			return next(c)
		}

		repo := getRepofromUri(req.RequestURI)
		req.Host = pp.Upstreams[repo].Mirrors[0].Host
		req.URL.Scheme = pp.Upstreams[repo].Mirrors[0].Scheme

		// construct new path from upstream mirror and request URI stripped by the repo prefix
		mirrorPath := pp.Upstreams[repo].Mirrors[0].Path
		upstreamPath := path.Join(mirrorPath, strings.TrimPrefix(req.RequestURI, "/"+repo))
		req.RequestURI = upstreamPath
		req.URL.Path = upstreamPath

		return next(c)
	}
}

// Proxy request to upstream
func (pp *pkgProxy) Proxy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()

		if !pp.isRepositoryRequest(req.RequestURI) {
			fmt.Println("Upstream(): exec next() middleware")
			return next(c)
		}

		target, _ := url.Parse(req.URL.Scheme + "://" + req.Host)
		followRedirect := transport{RT: http.DefaultTransport}

		var err error
		proxyHTTP(target, c, &followRedirect).ServeHTTP(c.Response(), c.Request())
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

func proxyHTTP(target *url.URL, c echo.Context, transport *transport) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(target)
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
			httpError := echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("upstream %s unreachable, could not forward: %v", target.Host, err))
			httpError.Internal = err
			c.Set("_error", httpError)
		}
	}
	proxy.Transport = transport
	return proxy
}
