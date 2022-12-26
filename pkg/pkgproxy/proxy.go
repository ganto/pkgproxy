// Copyright (c) 2021 LabStack
// SPDX-License-Identifier: MIT
package pkgproxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
	"strings"

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// ProxyWithConfig returns a Proxy middleware with config.
// See: `Proxy()`
func ProxyWithConfig(config middleware.ProxyConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = middleware.DefaultProxyConfig.Skipper
	}
	if config.Balancer == nil {
		panic("echo: proxy middleware requires balancer")
	}

	if config.Rewrite != nil {
		if config.RegexRewrite == nil {
			config.RegexRewrite = make(map[*regexp.Regexp]string)
		}
		for k, v := range rewriteRulesRegex(config.Rewrite) {
			config.RegexRewrite[k] = v
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			req := c.Request()
			res := c.Response()
			tgt := config.Balancer.Next(c)
			c.Set(config.ContextKey, tgt)
			req.Host = tgt.URL.Host
			fmt.Printf("tgt = +%s\n", tgt)

			if err := rewriteURL(config.RegexRewrite, req); err != nil {
				return err
			}

			// Proxy
			proxyHTTP(tgt, c, config).ServeHTTP(res, req)
			if e, ok := c.Get("_error").(error); ok {
				err = e
			}

			return
		}
	}
}

func proxyHTTP(tgt *middleware.ProxyTarget, c echo.Context, config middleware.ProxyConfig) http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(tgt.URL)
	proxy.ErrorHandler = func(resp http.ResponseWriter, req *http.Request, err error) {
		desc := tgt.URL.String()
		if tgt.Name != "" {
			desc = fmt.Sprintf("%s(%s)", tgt.Name, tgt.URL.String())
		}
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
			httpError := echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("remote %s unreachable, could not forward: %v", desc, err))
			httpError.Internal = err
			c.Set("_error", httpError)
		}
	}
	proxy.Transport = config.Transport
	proxy.ModifyResponse = config.ModifyResponse
	return proxy
}
