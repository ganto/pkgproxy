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
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RepoConfig defines the configuration of a package repository
type RepoConfig struct {
	Cache   cache.Cache
	Mirrors []*url.URL
	UrlPath string
}

func RepositoryWithConfig(config RepoConfig) echo.MiddlewareFunc {
	transport := PkgProxyTransport{
		Rt:    http.DefaultTransport,
		Cache: config.Cache,
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			req := c.Request()
			res := c.Response()

			tgt := config.Mirrors[0]
			req.Host = tgt.Host

			// trim repository handle
			if len(config.UrlPath) > 0 {
				req.RequestURI = strings.TrimPrefix(req.RequestURI, "/"+config.UrlPath)
				req.URL.Path = strings.TrimPrefix(req.URL.Path, "/"+config.UrlPath)
			}

			// Proxy
			proxyHTTP(tgt, c, &transport).ServeHTTP(res, req)
			if e, ok := c.Get("_error").(error); ok {
				err = e
			}

			return
		}
	}
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
