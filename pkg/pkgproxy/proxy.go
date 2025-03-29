// Copyright 2022 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/ganto/pkgproxy/pkg/cache"
	"github.com/ganto/pkgproxy/pkg/utils"
	echo "github.com/labstack/echo/v4"
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
	return func(c echo.Context) error {
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
						fmt.Printf("--> DELETE %s\n", uri)
						if err := repoCache.DeleteFile(uri); err != nil {
							return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
						}
						return c.JSON(http.StatusOK, map[string]string{"message": "Success"})
					}
					return c.File(repoCache.GetFilePath(uri))
				} else {
					if c.Request().Method == "DELETE" {
						return c.JSON(http.StatusNotFound, map[string]string{"message": "Not Found"})
					}
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

		if err := next(c); err != nil {
			return err
		}

		if pp.isRepositoryRequest(uri) {
			if repoCache.IsCacheCandidate(uri) && !repoCache.IsCached(uri) && (c.Response().Status == 200) && len(rspBody.Bytes()) > 0 {
				timestamp := time.Now().Local()
				if c.Response().Header().Get("Last-Modified") != "" {
					timestamp, _ = http.ParseTime(c.Response().Header().Get("Last-Modified"))
				}
				// save buffer to disk
				if err := repoCache.SaveToDisk(uri, rspBody, timestamp); err != nil {
					// don't fail request if we cannot write to cache
					fmt.Printf("Error: %s", err.Error())
				}
			}
		}

		return nil
	}
}

// Proxy request to upstream
func (pp *pkgProxy) ForwardProxy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		clientReq := c.Request()
		clientResp := c.Response()

		if !pp.isRepositoryRequest(clientReq.RequestURI) {
			return next(c)
		}

		if !utils.Contains(allowedProxyMethods, c.Request().Method) {
			return c.JSON(http.StatusMethodNotAllowed, map[string]string{"message": fmt.Sprintf("Forward proxy does not allow method %s\n", c.Request().Method)})
		}

		var rsp *http.Response
		var err error

		repo := getRepoFromURI(clientReq.RequestURI)
		success := false
		index := 0

		for !success && index < len(pp.upstreams[repo].mirrors) {
			// construct new path from upstream mirror and request URI stripped by the repo prefix
			mirror := pp.upstreams[repo].mirrors[index]
			mirrorPath := mirror.Path
			upstreamPath := path.Join(mirrorPath, strings.TrimPrefix(clientReq.URL.Path, "/"+repo))

			rsp, err = pp.forwardClientRequestToOrigin(clientReq, &url.URL{
				Scheme: mirror.Scheme,
				Host:   mirror.Host,
				Path:   upstreamPath,
			})

			if err == nil {
				defer rsp.Body.Close()
				fmt.Printf("<-- %v %+v\n", rsp.Status, rsp.Header)

				// follow HTTP redirects
				if utils.Contains(redirectStatusCodes, rsp.StatusCode) {
					var location *url.URL
					location, err = rsp.Location()
					if err == nil {
						rsp, err = pp.forwardClientRequestToOrigin(clientReq, location)
						if err == nil {
							defer rsp.Body.Close()
							fmt.Printf("<-- %v %+v\n", rsp.Status, rsp.Header)
						}
					}
				}
				success = rsp.StatusCode == 200
			}
			if err != nil {
				fmt.Printf("<-- Error: %s\n", err.Error())
			}

			index += 1
		}

		if err != nil {
			httpError := echo.NewHTTPError(http.StatusBadGateway, fmt.Sprintf("request to upstream server failed: %v", err))
			httpError.Internal = err
			return httpError
		}

		// copy response to client
		headers := clientResp.Header()
		for _, name := range allowedResponseHeaders {
			if value, ok := rsp.Header[name]; ok {
				headers[name] = value
			}
		}
		clientResp.WriteHeader(rsp.StatusCode)
		bodyBytes, _ := io.ReadAll(rsp.Body)
		if len(bodyBytes) > 0 {
			// ignore errors, since there's nothing we can do
			size, _ := io.CopyN(clientResp.Writer, bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
			clientResp.Size = size
		}

		return nil
	}
}

func (pp *pkgProxy) forwardClientRequestToOrigin(req *http.Request, origin *url.URL) (*http.Response, error) {
	// Construct filtered header to send to origin server
	headers := http.Header{}
	for _, name := range allowedRequestHeaders {
		if value, ok := req.Header[name]; ok {
			headers[name] = value
		}
	}

	fmt.Printf("--> %v %v\n", req.Method, origin)
	// Construct request to send to origin server
	return pp.transport.RoundTrip(&http.Request{
		Body:          req.Body,
		Close:         req.Close,
		ContentLength: req.ContentLength,
		Header:        headers,
		Method:        req.Method,
		URL:           origin,
	})
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
