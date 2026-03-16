// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ganto/pkgproxy/pkg/pkgproxy"
	echo "github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/spf13/cobra"
)

var (
	listenAddress string
	listenPort    uint16
	publicHost    string
)

const (
	defaultAddress   = "localhost"
	defaultPort      = 8080
	publicHostEnvVar = "PKGPROXY_PUBLIC_HOST"
)

func newServeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:              "serve",
		Args:             cobra.ArbitraryArgs,
		Short:            "Start forward proxy",
		RunE:             startServer,
		TraverseChildren: true,
	}
	c.PersistentFlags().StringVar(&listenAddress, "host", defaultAddress, "listen address of the pkgproxy.")
	c.PersistentFlags().Uint16Var(&listenPort, "port", defaultPort, "listen port of the pkgproxy.")
	c.PersistentFlags().StringVar(&publicHost, "public-host", "", "public hostname (or host:port) shown in landing page config snippets; overrides PKGPROXY_PUBLIC_HOST.")

	return c
}

// resolvePublicAddr determines the address rendered in landing page config snippets.
// The CLI flag takes precedence over the environment variable. If neither is set,
// the listen host:port is used.
func resolvePublicAddr(flagValue string, listenAddr string, port uint16) string {
	if flagValue != "" {
		return flagValue
	}
	if v := os.Getenv(publicHostEnvVar); v != "" {
		return v
	}
	return fmt.Sprintf("%s:%d", listenAddr, port)
}

func startServer(_ *cobra.Command, _ []string) error {
	logLevel := slog.LevelInfo
	if enableDebug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	app := echo.New()

	app.Use(middleware.RequestID())
	app.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()
			err := next(c)
			// Let Echo handle the error to ensure the status is set properly.
			// However this will prevent any potential upstream middleware from
			// receiving the error.
			if err != nil {
				app.HTTPErrorHandler(c, err)
			}
			resp, _ := echo.UnwrapResponse(c.Response())
			status := 0
			if resp != nil {
				status = resp.Status
			}
			logFn := slog.Info
			if status >= 500 {
				logFn = slog.Error
			} else if status >= 400 {
				logFn = slog.Warn
			}
			logFn("client response",
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
				"method", c.Request().Method,
				"uri", c.Request().RequestURI,
				"status", status,
				"latency", time.Since(start).String(),
				"remote_ip", c.RealIP(),
			)
			return nil
		}
	})
	app.Use(middleware.Recover())

	pkgProxy := pkgproxy.New(&pkgproxy.PkgProxyConfig{
		CacheBasePath:    cacheDir,
		RepositoryConfig: &repoConfig,
	})
	publicAddr := resolvePublicAddr(publicHost, listenAddress, listenPort)
	app.GET("/", pkgproxy.LandingHandler(&repoConfig, publicAddr))
	app.Use(pkgProxy.Cache)
	app.Use(pkgProxy.ForwardProxy)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{
		Address:    fmt.Sprintf("%s:%d", listenAddress, listenPort),
		HideBanner: true,
	}
	return sc.Start(ctx, app)
}
