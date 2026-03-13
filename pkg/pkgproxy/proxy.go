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
		transport http.RoundTripper
		upstreams map[string]upstream
	}
	upstream struct {
		cache   cache.FileCache
		mirrors []*url.URL
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
		upstreams[repo] = upstream{
			cache: cache.New(&cache.CacheConfig{
				BasePath:     config.CacheBasePath,
				FileSuffixes: config.RepositoryConfig.Repositories[repo].CacheSuffixes,
			}),
			mirrors: mirrors,
		}
	}
	return &pkgProxy{
		transport: transport,
		upstreams: upstreams,
	}
}

// This middleware function checks if a request can be served from the
// local cache and does so if possible. Otherwise it will make sure the
// response is cached if necessary.
func (pp *pkgProxy) Cache(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		var repoCache cache.FileCache
		var rspBody *bytes.Buffer

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
					return c.FileFS(strings.TrimPrefix(absPath, "/"), os.DirFS("/"))
				} else {
					if c.Request().Method == "DELETE" {
						return c.JSON(http.StatusNotFound, map[string]string{"message": "Not Found"})
					}
					// if not in cache write response body to buffer
					rspBody = new(bytes.Buffer)
					if resp, _ := echo.UnwrapResponse(c.Response()); resp != nil {
						bodyWriter := io.MultiWriter(resp.ResponseWriter, rspBody)
						writer := &bufferWriter{
							Writer:         bodyWriter,
							ResponseWriter: resp.ResponseWriter}
						resp.ResponseWriter = writer
					}
				}
			}
		}

		if err := next(c); err != nil {
			return err
		}

		if pp.isRepositoryRequest(uri) {
			resp, _ := echo.UnwrapResponse(c.Response())
			if repoCache.IsCacheCandidate(uri) && !repoCache.IsCached(uri) && resp != nil && (resp.Status == 200) && len(rspBody.Bytes()) > 0 {
				timestamp := time.Now().Local()
				if c.Response().Header().Get("Last-Modified") != "" {
					timestamp, _ = http.ParseTime(c.Response().Header().Get("Last-Modified"))
				}
				// save buffer to disk
				if err := repoCache.SaveToDisk(uri, rspBody, timestamp); err != nil {
					// don't fail request if we cannot write to cache
					slog.Error("cache write failed", "request_id", requestID(c), "uri", uri, "error", err)
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
// and returns the first 200 response. If no mirror returns 200, the last non-nil response
// (possibly non-200) is returned with a nil error. A non-nil error is only returned when
// the last mirror attempt failed at the connection level (e.g. DNS failure, refused
// connection) — not when the server replied with a non-200 HTTP status.
func (pp *pkgProxy) tryMirrors(ctx context.Context, rid string, req *http.Request, repo string, reqBody []byte) (*http.Response, error) {
	var rsp *http.Response
	var err error

	for i, mirror := range pp.upstreams[repo].mirrors {
		// Close response from previous failed iteration before trying next mirror.
		if rsp != nil {
			_ = rsp.Body.Close()
			rsp = nil
		}

		upstreamPath := path.Join(mirror.Path, strings.TrimPrefix(req.URL.Path, "/"+repo))
		rsp, err = pp.forwardClientRequestToOrigin(ctx, rid, req, &url.URL{
			Scheme: mirror.Scheme,
			Host:   mirror.Host,
			Path:   upstreamPath,
		}, reqBody)
		if err != nil {
			slog.Warn("upstream request failed", "request_id", rid, "mirror_index", i, "error", err)
			continue
		}
		slog.Info("upstream response", "request_id", rid, "status", rsp.Status, "headers", rsp.Header)

		// Follow HTTP redirects.
		if utils.Contains(redirectStatusCodes, rsp.StatusCode) {
			location, locErr := rsp.Location()
			_ = rsp.Body.Close()
			rsp = nil
			if locErr != nil {
				err = locErr
				slog.Warn("upstream request failed", "request_id", rid, "mirror_index", i, "error", err)
				continue
			}
			rsp, err = pp.forwardClientRequestToOrigin(ctx, rid, req, location, reqBody)
			if err != nil {
				slog.Warn("upstream request failed", "request_id", rid, "mirror_index", i, "error", err)
				continue
			}
			slog.Info("upstream response", "request_id", rid, "status", rsp.Status, "headers", rsp.Header)
		}

		if rsp.StatusCode == http.StatusOK {
			return rsp, nil
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
