// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	echo "github.com/labstack/echo/v5"
)

// NewEcho returns an Echo instance with pkgproxy's router configuration. It
// registers no routes and no middleware; callers wire those up themselves.
//
// AutoHandleHEAD lets the router answer HEAD requests with the matching GET
// route. Without it the landing page only accepts GET and replies 405 to HEAD,
// which is inconsistent with the proxy itself (HEAD is an allowed method) and
// breaks health check probes that use HEAD on "/".
//
// AllowOverwritingRoute must be set explicitly: NewWithConfig replaces the
// router wholesale and does not inherit the defaults echo.New() applies.
func NewEcho() *echo.Echo {
	return echo.NewWithConfig(echo.Config{
		Router: echo.NewRouter(echo.RouterConfig{
			AutoHandleHEAD:        true,
			AllowOverwritingRoute: true,
		}),
	})
}
