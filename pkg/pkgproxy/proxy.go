// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ganto/pkgproxy/pkg/cache"
	"github.com/ganto/pkgproxy/pkg/utils"
	echo "github.com/labstack/echo/v5"
)

type (
	PkgProxy interface {
		Cache(echo.HandlerFunc) echo.HandlerFunc
		ForwardProxy(echo.HandlerFunc) echo.HandlerFunc
	}

	PkgProxyConfig struct {
		CacheBasePath    string
		RepositoryConfig *RepoConfig

		// To customize the transport to remote.
		// Examples: If custom TLS certificates are required.
		Transport http.RoundTripper
	}

	pkgProxy struct {
		transport      http.RoundTripper
		upstreams      map[string]upstream
		retryBaseDelay time.Duration
	}
	upstream struct {
		cache   cache.FileCache
		mirrors []*url.URL
		retries int
	}
)

var (
	// HTTP methods that are allowed for the cache
	allowedCacheMethods = []string{
		"GET",
		"HEAD",
		"DELETE",
	}

	// HTTP methods that are allowed for the proxy
	allowedProxyMethods = []string{
		"GET",
		"HEAD",
	}

	// HTTP request headers that will be forwarded to origin server
	allowedRequestHeaders = []string{
		"Accept",
		"Accept-Encoding",
		"Accept-Language",
		"Authorization",
		"Cache-Control",
		"Cookie",
		"Range",
		"Referer",
		"User-Agent",
	}

	// HTTP response headers that will be forwarded to client
	allowedResponseHeaders = []string{
		"Accept-Ranges",
		"Age",
		"Allow",
		"Content-Encoding",
		"Content-Language",
		"Content-Length",
		"Content-Type",
		"Cache-Control",
		"Date",
		"Etag",
		"Expires",
		"Last-Modified",
		"Location",
		"Server",
		"Vary",
	}

	// Default number of attempts per mirror (1 = no retry)
	defaultRetries = 1

	// Base delay for exponential backoff between retry attempts (1s, 2s, 4s, ...)
	retryBaseDelay = 1 * time.Second

	// Status codes which will trigger a new request to the "Location" header
	redirectStatusCodes = []int{
		301,
		302,
		303,
		307,
		308,
	}
)

func New(config *PkgProxyConfig) PkgProxy {
	transport := config.Transport
	if config.Transport == nil {
		transport = http.DefaultTransport
	}

	upstreams := map[string]upstream{}
	for _, repo := range utils.KeysFromMap(config.RepositoryConfig.Repositories) {
		var mirrors []*url.URL
		for _, mirror := range config.RepositoryConfig.Repositories[repo].Mirrors {
			url, err := url.Parse(mirror)
			if err == nil {
				mirrors = append(mirrors, url)
			}
		}
		retries := config.RepositoryConfig.Repositories[repo].Retries
		if retries < 1 {
			retries = defaultRetries
		}
		upstreams[repo] = upstream{
			cache: cache.New(&cache.CacheConfig{
				BasePath:     config.CacheBasePath,
				FileSuffixes: config.RepositoryConfig.Repositories[repo].CacheSuffixes,
				Exclude:      config.RepositoryConfig.Repositories[repo].Exclude,
			}),
			mirrors: mirrors,
			retries: retries,
		}
	}
	return &pkgProxy{
		transport:      transport,
		upstreams:      upstreams,
		retryBaseDelay: retryBaseDelay,
	}
}

// This middleware function checks if a request can be served from the
// local cache and does so if possible. Otherwise it will make sure the
// response is cached if necessary.
func (pp *pkgProxy) Cache(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var repoCache cache.FileCache
		var rw *resilientWriter

		// the request URI might be changed later, keep the original value
		uri := strings.Clone(c.Request().RequestURI)

		if pp.isRepositoryRequest(uri) {
			if !utils.Contains(allowedCacheMethods, c.Request().Method) {
				return c.JSON(http.StatusMethodNotAllowed, map[string]string{"message": fmt.Sprintf("Cache does not allow method %s\n", c.Request().Method)})
			}
			repoCache = pp.upstreams[getRepoFromURI(uri)].cache

			if repoCache.IsCacheCandidate(uri) {
				if repoCache.IsCached(uri) {
					// serve or delete from cache
					if c.Request().Method == "DELETE" {
						slog.Info("cache delete", "request_id", requestID(c), "uri", uri)
						if err := repoCache.DeleteFile(uri); err != nil {
							return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
						}
						return c.JSON(http.StatusOK, map[string]string{"message": "Success"})
					}
					filePath, err := repoCache.GetFilePath(uri)
					if err != nil {
						return c.JSON(http.StatusForbidden, map[string]string{"message": "Forbidden"})
					}
					absPath, err := filepath.Abs(filePath)
					if err != nil {
						return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
					}
					return c.FileFS(filepath.Base(absPath), os.DirFS(filepath.Dir(absPath)))
				} else {
					if c.Request().Method == "DELETE" {
						return c.JSON(http.StatusNotFound, map[string]string{"message": "Not Found"})
					}
					// Stream response to both client and cache temp file
					rw = newResilientWriter(repoCache, uri)
					if resp, _ := echo.UnwrapResponse(c.Response()); resp != nil {
						sw := newSafeWriter(resp.ResponseWriter)
						bodyWriter := io.MultiWriter(sw, rw)
						writer := &bufferWriter{
							Writer:         bodyWriter,
							ResponseWriter: resp.ResponseWriter,
							safe:           sw,
						}
						resp.ResponseWriter = writer
					}
				}
			}
		}

		// Ensure temp file cleanup runs regardless of next(c) outcome.
		// The error handler middleware above Cache may write an error response
		// through the wrapped ResponseWriter after Cache returns, so we need
		// cleanup even on error paths.
		if rw != nil {
			defer func() {
				// Disable prevents writes from error handlers that run
				// after Cache returns (e.g. the 502 JSON body).
				rw.Disable()
				// Close before Remove to avoid leaking file descriptors and
				// to ensure removal succeeds on all platforms.
				_ = rw.Close()
				if tmpPath := rw.TmpPath(); tmpPath != "" {
					_ = os.Remove(tmpPath)
				}
			}()
		}

		if err := next(c); err != nil {
			return err
		}

		if pp.isRepositoryRequest(uri) && rw != nil {
			// Close temp file before commit or cleanup. A close error means the
			// file may not have been fully flushed, so skip the commit.
			if err := rw.Close(); err != nil {
				slog.Error("cache temp file close failed", "request_id", requestID(c), "uri", uri, "error", err)
				rw.failed = true
			}

			resp, _ := echo.UnwrapResponse(c.Response())
			if repoCache.IsCacheCandidate(uri) && !repoCache.IsCached(uri) && resp != nil && resp.Status == 200 && rw.bytesWritten > 0 && !rw.failed {
				// Content-Length validation
				commitOK := true
				if clHeader := c.Response().Header().Get("Content-Length"); clHeader != "" {
					if expectedLen, err := strconv.ParseInt(clHeader, 10, 64); err == nil {
						if rw.bytesWritten != expectedLen {
							slog.Warn("cache write skipped: Content-Length mismatch",
								"request_id", requestID(c), "uri", uri,
								"expected", expectedLen, "actual", rw.bytesWritten)
							commitOK = false
						}
					}
				}

				if commitOK {
					timestamp := time.Now().Local()
					if c.Response().Header().Get("Last-Modified") != "" {
						timestamp, _ = http.ParseTime(c.Response().Header().Get("Last-Modified"))
					}
					// CommitTempFile renames the file; the deferred Remove becomes a harmless ENOENT
					if err := repoCache.CommitTempFile(rw.TmpPath(), uri, timestamp); err != nil {
						// don't fail request if we cannot write to cache
						slog.Error("cache commit failed", "request_id", requestID(c), "uri", uri, "error", err)
					}
				}
			}
		}

		return nil
	}
}

// Proxy request to upstream
func (pp *pkgProxy) ForwardProxy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		clientReq := c.Request()
		clientRespW := c.Response()

		if !pp.isRepositoryRequest(clientReq.RequestURI) {
			return next(c)
		}

		if !utils.Contains(allowedProxyMethods, c.Request().Method) {
			return c.JSON(http.StatusMethodNotAllowed, map[string]string{"message": fmt.Sprintf("Forward proxy does not allow method %s\n", c.Request().Method)})
		}

		// Buffer the request body once so it can be replayed across mirror retries and redirects.
		var reqBody []byte
		if clientReq.Body != nil {
			var err error
			reqBody, err = io.ReadAll(clientReq.Body)
			_ = clientReq.Body.Close()
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("failed to read request body: %v", err)).Wrap(err)
			}
		}

		repo := getRepoFromURI(clientReq.RequestURI)

		// Derive an upstream context that is independent of client disconnects
		// but preserves any existing request deadline, so upstream calls remain bounded.
		upstreamCtx := context.Background()
		if deadline, ok := clientReq.Context().Deadline(); ok {
			var cancel context.CancelFunc
			upstreamCtx, cancel = context.WithDeadline(context.Background(), deadline)
			defer cancel()
		}

		rsp, err := pp.tryMirrors(upstreamCtx, requestID(c), clientReq, repo, reqBody)
		if rsp != nil {
			defer rsp.Body.Close()
		}
		if err != nil {
			return echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("request to upstream server failed: %v", err)).Wrap(err)
		}
		if rsp == nil {
			return echo.NewHTTPError(http.StatusBadGateway, "no mirror returned a response")
		}

		// copy response to client
		for name, value := range filterHeaders(rsp.Header, allowedResponseHeaders) {
			clientRespW.Header()[name] = value
		}
		clientRespW.WriteHeader(rsp.StatusCode)
		_, _ = io.Copy(clientRespW, rsp.Body)

		return nil
	}
}

// tryMirrors iterates the mirrors for repo in order, following one redirect per mirror,
// and returns the first 200 response. Each mirror is attempted up to the configured
// number of retries (useful when a redirector like download.fedoraproject.org sends
// traffic to a broken mirror — retrying may yield a different, working mirror).
// If no mirror returns 200, the last non-nil response (possibly non-200) is returned
// with a nil error. A non-nil error is only returned when the last mirror attempt
// failed at the connection level (e.g. DNS failure, refused connection) — not when
// the server replied with a non-200 HTTP status.
func (pp *pkgProxy) tryMirrors(ctx context.Context, rid string, req *http.Request, repo string, reqBody []byte) (*http.Response, error) {
	var rsp *http.Response
	var err error

	retries := pp.upstreams[repo].retries

	for i, mirror := range pp.upstreams[repo].mirrors {
		for attempt := 1; attempt <= retries; attempt++ {
			// Close response from previous failed attempt before retrying.
			if rsp != nil {
				_ = rsp.Body.Close()
				rsp = nil
			}

			if attempt > 1 {
				delay := pp.retryBaseDelay * (1 << (attempt - 2))
				slog.Info("retrying mirror", "request_id", rid, "mirror_index", i, "attempt", attempt, "delay", delay)
				timer := time.NewTimer(delay)
				select {
				case <-timer.C:
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				}
			}

			upstreamPath := path.Join(mirror.Path, strings.TrimPrefix(req.URL.Path, "/"+repo))
			rsp, err = pp.forwardClientRequestToOrigin(ctx, rid, req, &url.URL{
				Scheme:   mirror.Scheme,
				Host:     mirror.Host,
				Path:     upstreamPath,
				RawQuery: req.URL.RawQuery,
			}, reqBody)
			if err != nil {
				slog.Warn("upstream request failed", "request_id", rid, "mirror_index", i, "attempt", attempt, "error", err)
				break // connection-level error, skip to next mirror
			}
			slog.Info("upstream response", "request_id", rid, "status", rsp.Status, "headers", rsp.Header)

			// Follow HTTP redirects.
			if utils.Contains(redirectStatusCodes, rsp.StatusCode) {
				location, locErr := rsp.Location()
				_ = rsp.Body.Close()
				rsp = nil
				if locErr != nil {
					err = locErr
					slog.Warn("upstream request failed", "request_id", rid, "mirror_index", i, "attempt", attempt, "error", err)
					break // bad redirect, skip to next mirror
				}
				rsp, err = pp.forwardClientRequestToOrigin(ctx, rid, req, location, reqBody)
				if err != nil {
					slog.Warn("upstream request failed", "request_id", rid, "mirror_index", i, "attempt", attempt, "error", err)
					break // connection-level error, skip to next mirror
				}
				slog.Info("upstream response", "request_id", rid, "status", rsp.Status, "headers", rsp.Header)
			}

			if rsp.StatusCode == http.StatusOK {
				return rsp, nil
			}

			// Retry this mirror if we got a server error (5xx) and have attempts left.
			if rsp.StatusCode >= 500 && attempt < retries {
				slog.Warn("upstream server error, will retry", "request_id", rid, "mirror_index", i, "attempt", attempt, "status", rsp.StatusCode)
				continue
			}

			// Non-5xx non-200 (e.g. 404): no point retrying this mirror.
			break
		}
	}

	return rsp, err
}

func (pp *pkgProxy) forwardClientRequestToOrigin(ctx context.Context, rid string, req *http.Request, origin *url.URL, bodyBytes []byte) (*http.Response, error) {
	headers := filterHeaders(req.Header, allowedRequestHeaders)

	slog.InfoContext(ctx, "upstream request", "request_id", rid, "method", req.Method, "origin", origin)
	// Construct request to send to origin server
	upstreamReq := (&http.Request{
		Body:          io.NopCloser(bytes.NewReader(bodyBytes)),
		Close:         req.Close,
		ContentLength: req.ContentLength,
		Header:        headers,
		Method:        req.Method,
		URL:           origin,
	}).WithContext(ctx)
	return pp.transport.RoundTrip(upstreamReq)
}

// filterHeaders returns a new http.Header containing only the allowed header keys from src.
func filterHeaders(src http.Header, allowed []string) http.Header {
	dst := http.Header{}
	for _, name := range allowed {
		if value, ok := src[name]; ok {
			dst[name] = value
		}
	}
	return dst
}

// Return the X-Request-ID header value from the echo context
func requestID(c *echo.Context) string {
	return c.Response().Header().Get(echo.HeaderXRequestID)
}

// Check if the request should be handled by PkgProxy
func (pp *pkgProxy) isRepositoryRequest(uri string) bool {
	repo := getRepoFromURI(uri)
	return utils.Contains(utils.KeysFromMap(pp.upstreams), repo)
}

// Return the repository name of the URL without leading "/"
func getRepoFromURI(uri string) string {
	return strings.TrimPrefix(utils.RouteFromURI(uri), "/")
}
