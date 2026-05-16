// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/ganto/pkgproxy/pkg/pkgproxy"
	echo "github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/spf13/cobra"
)

var (
	listenAddress      string
	listenPort         uint16
	publicHost         string
	trustProxy         string
	ipExtractor        echo.IPExtractor
	resolvedTrustProxy string
)

const (
	defaultAddress   = "localhost"
	defaultPort      = 8080
	hostEnvVar       = "PKGPROXY_HOST"
	publicHostEnvVar = "PKGPROXY_PUBLIC_HOST"
	trustProxyEnvVar = "PKGPROXY_TRUST_PROXY"
)

func newServeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "serve",
		Args:  cobra.ArbitraryArgs,
		Short: "Start forward proxy",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			listenAddress = resolveListenHost(cmd.Flag("host").Changed, listenAddress, os.Getenv(hostEnvVar))
			resolvedTrustProxy = resolveTrustProxy(cmd.Flag("trust-proxy").Changed, trustProxy, os.Getenv(trustProxyEnvVar))
			var err error
			ipExtractor, err = parseTrustProxy(resolvedTrustProxy)
			if err != nil {
				return err
			}
			return initConfig()
		},
		RunE:             startServer,
		TraverseChildren: true,
	}
	c.PersistentFlags().StringVar(&listenAddress, "host", defaultAddress, "listen address of the pkgproxy.")
	c.PersistentFlags().Uint16Var(&listenPort, "port", defaultPort, "listen port of the pkgproxy.")
	c.PersistentFlags().StringVar(&publicHost, "public-host", "", "public hostname (or host:port) shown in landing page config snippets; overrides PKGPROXY_PUBLIC_HOST.")
	c.PersistentFlags().StringVar(&trustProxy, "trust-proxy", "", "comma-separated list of trusted proxy addresses for X-Forwarded-For: none, loopback, private, CIDR, or IP; overrides PKGPROXY_TRUST_PROXY.")

	return c
}

// resolveListenHost determines the listen host using flag → env var → default precedence.
func resolveListenHost(flagChanged bool, flagValue, envValue string) string {
	if flagChanged {
		return flagValue
	}
	if envValue != "" {
		return envValue
	}
	return defaultAddress
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

// resolveTrustProxy determines the trust-proxy value using flag → env var → default precedence.
func resolveTrustProxy(flagChanged bool, flagValue, envValue string) string {
	if flagChanged {
		return flagValue
	}
	if envValue != "" {
		return envValue
	}
	return ""
}

// parseTrustProxy converts the resolved trust-proxy string into an echo.IPExtractor.
// Empty or "none" installs ExtractIPDirect (XFF ignored). Other values install
// ExtractIPFromXFFHeader with only the operator-specified trust options; echo's
// implicit defaults (loopback/link-local/private) are never applied automatically.
func parseTrustProxy(value string) (echo.IPExtractor, error) {
	var entries []string
	for p := range strings.SplitSeq(value, ",") {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			entries = append(entries, p)
		}
	}
	if len(entries) == 0 {
		return echo.ExtractIPDirect(), nil
	}
	if len(entries) == 1 && entries[0] == "none" {
		return echo.ExtractIPDirect(), nil
	}
	if slices.Contains(entries, "none") {
		return nil, errors.New("trust-proxy: 'none' cannot be combined with other entries")
	}

	var (
		trustLoopback bool
		trustPrivate  bool
		extraRanges   []*net.IPNet
	)
	for _, e := range entries {
		switch e {
		case "loopback":
			trustLoopback = true
		case "private":
			trustPrivate = true
		default:
			_, ipNet, err := net.ParseCIDR(e)
			if err != nil {
				ip := net.ParseIP(e)
				if ip == nil {
					return nil, fmt.Errorf("trust-proxy: unrecognized entry %q", e)
				}
				bits := 128
				if ip.To4() != nil {
					bits = 32
				}
				_, ipNet, _ = net.ParseCIDR(fmt.Sprintf("%s/%d", ip.String(), bits))
			}
			extraRanges = append(extraRanges, ipNet)
		}
	}

	opts := []echo.TrustOption{
		echo.TrustLoopback(trustLoopback),
		echo.TrustLinkLocal(false),
		echo.TrustPrivateNet(trustPrivate),
	}
	for _, ipNet := range extraRanges {
		opts = append(opts, echo.TrustIPRange(ipNet))
	}
	return echo.ExtractIPFromXFFHeader(opts...), nil
}

func startServer(_ *cobra.Command, _ []string) error {
	logLevel := slog.LevelInfo
	if enableDebug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))
	slog.Info("starting pkgproxy",
		"version", Version,
		"gitCommit", GitCommit,
		"goVersion", runtime.Version(),
		"buildDate", buildDate(),
	)
	trustProxyLog := resolvedTrustProxy
	if trustProxyLog == "" {
		trustProxyLog = "none"
	}
	slog.Info("trust-proxy", "value", trustProxyLog)

	app := echo.New()
	// Extract client IP from X-Forwarded-For only when a trusted proxy is explicitly configured
	// via --trust-proxy. By default, XFF is ignored and the direct connecting IP is used.
	app.IPExtractor = ipExtractor

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
