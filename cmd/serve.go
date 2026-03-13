// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ganto/pkgproxy/pkg/pkgproxy"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/cobra"
)

var (
	listenAddress string
	listenPort    uint16
)

const (
	defaultAddress = "localhost"
	defaultPort    = 8080
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

	return c
}

func startServer(_ *cobra.Command, _ []string) error {
	logLevel := slog.LevelInfo
	if enableDebug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	app := echo.New()
	app.HideBanner = true

	app.Use(middleware.RequestID())
	app.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			// Let Echo handle the error to ensure the status is set properly.
			// However this will prevent any potential upstream middleware from
			// receiving the error.
			if err != nil {
				c.Error(err)
			}
			status := c.Response().Status
			logFn := slog.Info
			if status >= 500 {
				logFn = slog.Error
			} else if status >= 400 {
				logFn = slog.Warn
			}
			logFn("downstream response",
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
				"method", c.Request().Method,
				"uri", c.Request().RequestURI,
				"status", status,
				"latency", time.Since(start),
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
	app.Use(pkgProxy.Cache)
	app.Use(pkgProxy.ForwardProxy)

	err := app.Start(fmt.Sprintf("%s:%d", listenAddress, listenPort))
	// ignore normal shutdown returning http.ErrServerClosed
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	return err
}
