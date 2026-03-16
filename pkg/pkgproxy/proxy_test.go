// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	echo "github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProxy creates a pkgProxy with a single "testrepo" repository using the given mirrors.
// Returns the proxy and the temporary cache directory path.
func newTestProxy(t *testing.T, mirrors []string) (PkgProxy, string) {
	t.Helper()
	cacheDir := t.TempDir()
	repoConfig := &RepoConfig{
		Repositories: map[string]Repository{
			"testrepo": {
				CacheSuffixes: []string{".rpm"},
				Mirrors:       mirrors,
			},
		},
	}
	pp := New(&PkgProxyConfig{
		CacheBasePath:    cacheDir,
		RepositoryConfig: repoConfig,
	})
	return pp, cacheDir
}

// newTestProxyWithRetries creates a pkgProxy with a single "testrepo" repository using the given mirrors and retries.
// The retry delay is set to zero for fast tests.
func newTestProxyWithRetries(t *testing.T, mirrors []string, retries int) (PkgProxy, string) {
	t.Helper()
	cacheDir := t.TempDir()
	repoConfig := &RepoConfig{
		Repositories: map[string]Repository{
			"testrepo": {
				CacheSuffixes: []string{".rpm"},
				Mirrors:       mirrors,
				Retries:       retries,
			},
		},
	}
	pp := New(&PkgProxyConfig{
		CacheBasePath:    cacheDir,
		RepositoryConfig: repoConfig,
	})
	pp.(*pkgProxy).retryDelay = 0
	return pp, cacheDir
}

// newTestApp creates an Echo app with the standard middleware chain used in tests.
func newTestApp(pp PkgProxy) *echo.Echo {
	app := echo.New()
	app.Use(middleware.RequestID())
	app.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			err := next(c)
			if err != nil {
				app.HTTPErrorHandler(c, err)
			}
			return nil
		}
	})
	app.Use(middleware.Recover())
	app.Use(pp.Cache)
	app.Use(pp.ForwardProxy)
	return app
}

// --- Helper function tests ---

func TestGetRepoFromURI(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"/testrepo/some/path/file.rpm", "testrepo"},
		{"/testrepo/", "testrepo"},
		{"/testrepo", "testrepo"},
		{"/", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			assert.Equal(t, tt.want, getRepoFromURI(tt.uri))
		})
	}
}

func TestIsRepositoryRequest(t *testing.T) {
	pp, _ := newTestProxy(t, []string{"http://example.com/"})
	proxy := pp.(*pkgProxy)

	tests := []struct {
		uri  string
		want bool
	}{
		{"/testrepo/some/file.rpm", true},
		{"/testrepo/", true},
		{"/unknownrepo/file.rpm", false},
		{"/", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			assert.Equal(t, tt.want, proxy.isRepositoryRequest(tt.uri))
		})
	}
}

func TestFilterHeaders(t *testing.T) {
	src := http.Header{
		"Accept":       {"text/html"},
		"User-Agent":   {"test-agent"},
		"X-Custom-Foo": {"should-be-stripped"},
		"Authorization": {"Bearer token"},
	}
	allowed := []string{"Accept", "User-Agent", "Authorization"}

	result := filterHeaders(src, allowed)

	assert.Equal(t, "text/html", result.Get("Accept"))
	assert.Equal(t, "test-agent", result.Get("User-Agent"))
	assert.Equal(t, "Bearer token", result.Get("Authorization"))
	assert.Empty(t, result.Get("X-Custom-Foo"))
	assert.Len(t, result, 3)
}

func TestFilterHeadersEmpty(t *testing.T) {
	result := filterHeaders(http.Header{}, []string{"Accept"})
	assert.Empty(t, result)

	result = filterHeaders(http.Header{"Accept": {"text/html"}}, []string{})
	assert.Empty(t, result)
}

// --- New() constructor tests ---

func TestNewDefaultTransport(t *testing.T) {
	pp, _ := newTestProxy(t, []string{"http://example.com/"})
	proxy := pp.(*pkgProxy)
	assert.Equal(t, http.DefaultTransport, proxy.transport)
}

func TestNewCustomTransport(t *testing.T) {
	customTransport := &http.Transport{}
	cacheDir := t.TempDir()
	pp := New(&PkgProxyConfig{
		CacheBasePath: cacheDir,
		RepositoryConfig: &RepoConfig{
			Repositories: map[string]Repository{
				"testrepo": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"http://example.com/"}},
			},
		},
		Transport: customTransport,
	})
	proxy := pp.(*pkgProxy)
	assert.Same(t, customTransport, proxy.transport)
}

func TestNewParsesUpstreams(t *testing.T) {
	cacheDir := t.TempDir()
	pp := New(&PkgProxyConfig{
		CacheBasePath: cacheDir,
		RepositoryConfig: &RepoConfig{
			Repositories: map[string]Repository{
				"repo1": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"http://a.com/", "http://b.com/"}},
				"repo2": {CacheSuffixes: []string{".deb"}, Mirrors: []string{"http://c.com/"}},
			},
		},
	})
	proxy := pp.(*pkgProxy)

	assert.Len(t, proxy.upstreams, 2)
	assert.Len(t, proxy.upstreams["repo1"].mirrors, 2)
	assert.Len(t, proxy.upstreams["repo2"].mirrors, 1)
	assert.Equal(t, "a.com", proxy.upstreams["repo1"].mirrors[0].Host)
}

// --- Cache middleware tests ---

func TestCacheMethodNotAllowed(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			pp, _ := newTestProxy(t, []string{"http://example.com/"})
			app := newTestApp(pp)

			req := httptest.NewRequest(method, "/testrepo/some/package.rpm", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

func TestCacheServeFromCache(t *testing.T) {
	pp, cacheDir := newTestProxy(t, []string{"http://example.com/"})

	// Pre-populate cache file
	cachedPath := filepath.Join(cacheDir, "testrepo", "some", "path", "package.rpm")
	require.NoError(t, os.MkdirAll(filepath.Dir(cachedPath), 0o750))
	require.NoError(t, os.WriteFile(cachedPath, []byte("cached-content"), 0o644))

	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/some/path/package.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "cached-content", rec.Body.String())
}

func TestCacheMissAndSave(t *testing.T) {
	upstreamBody := "fresh-upstream-content"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, upstreamBody)
	}))
	defer upstream.Close()

	pp, cacheDir := newTestProxy(t, []string{upstream.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/some/path/package.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, upstreamBody, rec.Body.String())

	// Verify file was cached to disk
	cachedPath := filepath.Join(cacheDir, "testrepo", "some", "path", "package.rpm")
	data, err := os.ReadFile(cachedPath)
	require.NoError(t, err)
	assert.Equal(t, upstreamBody, string(data))
}

func TestCacheDeleteExisting(t *testing.T) {
	pp, cacheDir := newTestProxy(t, []string{"http://example.com/"})

	// Pre-populate cache file
	cachedPath := filepath.Join(cacheDir, "testrepo", "some", "package.rpm")
	require.NoError(t, os.MkdirAll(filepath.Dir(cachedPath), 0o750))
	require.NoError(t, os.WriteFile(cachedPath, []byte("cached"), 0o644))

	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodDelete, "/testrepo/some/package.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "Success", body["message"])

	// Verify file was removed
	_, err := os.Stat(cachedPath)
	assert.True(t, os.IsNotExist(err))
}

func TestCacheDeleteNonExistent(t *testing.T) {
	pp, _ := newTestProxy(t, []string{"http://example.com/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodDelete, "/testrepo/some/package.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCacheHeadFromCache(t *testing.T) {
	pp, cacheDir := newTestProxy(t, []string{"http://example.com/"})

	cachedPath := filepath.Join(cacheDir, "testrepo", "some", "package.rpm")
	require.NoError(t, os.MkdirAll(filepath.Dir(cachedPath), 0o750))
	require.NoError(t, os.WriteFile(cachedPath, []byte("cached-content"), 0o644))

	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodHead, "/testrepo/some/package.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// HEAD responses should have no body
	assert.Empty(t, rec.Body.String())
}

func TestCacheNonCacheCandidate(t *testing.T) {
	// .xml is not in the configured suffixes (.rpm only)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "xml-content")
	}))
	defer upstream.Close()

	pp, cacheDir := newTestProxy(t, []string{upstream.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/repodata/repomd.xml", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "xml-content", rec.Body.String())

	// Verify file was NOT cached
	_, err := os.Stat(filepath.Join(cacheDir, "testrepo", "repodata", "repomd.xml"))
	assert.True(t, os.IsNotExist(err))
}

func TestCacheNonRepoRequest(t *testing.T) {
	pp, _ := newTestProxy(t, []string{"http://example.com/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/notarepo/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Non-repo requests pass through to 404 (no route registered)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// --- ForwardProxy middleware tests ---

func TestForwardProxySuccess(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Last-Modified", "Mon, 01 Jan 2024 00:00:00 GMT")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "upstream-body")
	}))
	defer upstream.Close()

	pp, _ := newTestProxy(t, []string{upstream.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "upstream-body", rec.Body.String())
	assert.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
}

func TestForwardProxyMirrorFailover(t *testing.T) {
	// First mirror returns 500
	mirror1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mirror1.Close()

	// Second mirror returns 200
	mirror2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "mirror2-body")
	}))
	defer mirror2.Close()

	pp, _ := newTestProxy(t, []string{mirror1.URL + "/", mirror2.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "mirror2-body", rec.Body.String())
}

func TestForwardProxyAllMirrorsFail(t *testing.T) {
	// Single mirror returns 500 — proxy forwards the last status to client
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "error")
	}))
	defer mirror.Close()

	pp, _ := newTestProxy(t, []string{mirror.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// When upstream returns a non-200 status but no transport error,
	// the last response status is forwarded to the client
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestForwardProxyConnectionFail(t *testing.T) {
	// Bind to a free port then immediately close it so nothing is listening.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	l.Close()

	pp, _ := newTestProxy(t, []string{"http://" + addr + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestForwardProxyNoMirrors(t *testing.T) {
	// A repo with no mirrors should return 502 Bad Gateway instead of panicking
	pp, _ := newTestProxy(t, []string{})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestForwardProxyRedirect(t *testing.T) {
	// Final destination returns 200
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "redirected-body")
	}))
	defer destination.Close()

	// First mirror returns 302 redirect to destination
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", destination.URL+r.URL.Path)
		w.WriteHeader(http.StatusFound)
	}))
	defer mirror.Close()

	pp, _ := newTestProxy(t, []string{mirror.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "redirected-body", rec.Body.String())
}

func TestForwardProxyRequestHeaderFiltering(t *testing.T) {
	var receivedHeaders http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	pp, _ := newTestProxy(t, []string{upstream.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Accept", "application/octet-stream")
	req.Header.Set("X-Custom-Header", "should-be-stripped")
	req.Header.Set("X-Forwarded-For", "should-be-stripped")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "test-agent", receivedHeaders.Get("User-Agent"))
	assert.Equal(t, "application/octet-stream", receivedHeaders.Get("Accept"))
	assert.Empty(t, receivedHeaders.Get("X-Custom-Header"))
	assert.Empty(t, receivedHeaders.Get("X-Forwarded-For"))
}

func TestForwardProxyResponseHeaderFiltering(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Etag", `"abc123"`)
		w.Header().Set("X-Custom-Response", "should-be-stripped")
		w.Header().Set("X-Powered-By", "should-be-stripped")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "body")
	}))
	defer upstream.Close()

	pp, _ := newTestProxy(t, []string{upstream.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/octet-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, `"abc123"`, rec.Header().Get("Etag"))
	assert.Empty(t, rec.Header().Get("X-Custom-Response"))
	assert.Empty(t, rec.Header().Get("X-Powered-By"))
}

func TestForwardProxyMethodNotAllowed(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		t.Run(method, func(t *testing.T) {
			pp, _ := newTestProxy(t, []string{"http://example.com/"})
			// Use only ForwardProxy middleware (skip Cache which handles DELETE differently)
			app := echo.New()
			app.Use(middleware.RequestID())
			app.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
				return func(c *echo.Context) error {
					err := next(c)
					if err != nil {
						app.HTTPErrorHandler(c, err)
					}
					return nil
				}
			})
			app.Use(pp.ForwardProxy)

			req := httptest.NewRequest(method, "/testrepo/some/file.rpm", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
		})
	}
}

func TestForwardProxyNonRepoPassthrough(t *testing.T) {
	pp, _ := newTestProxy(t, []string{"http://example.com/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/notarepo/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Non-repo request should pass through to 404 (no route registered)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestForwardProxyUpstreamPath(t *testing.T) {
	// Verify that the upstream request path is correctly constructed:
	// repo prefix is stripped and mirror base path is prepended
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	// Mirror with a base path
	cacheDir := t.TempDir()
	pp := New(&PkgProxyConfig{
		CacheBasePath: cacheDir,
		RepositoryConfig: &RepoConfig{
			Repositories: map[string]Repository{
				"testrepo": {
					CacheSuffixes: []string{".rpm"},
					Mirrors:       []string{upstream.URL + "/basepath/"},
				},
			},
		},
	})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/sub/dir/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/basepath/sub/dir/file.rpm", receivedPath)
}

// --- httpbin.org tests (gated by environment variable) ---

func TestForwardProxyWithHttpbin(t *testing.T) {
	if os.Getenv("PKGPROXY_HTTPBIN_TESTS") == "" {
		t.Skip("Set PKGPROXY_HTTPBIN_TESTS=1 to run httpbin.org tests")
	}

	// Use httpbin.org to verify the proxy can reach real HTTP servers.
	// httpbin.org/anything/<path> returns 200 with request details in JSON.
	cacheDir := t.TempDir()
	pp := New(&PkgProxyConfig{
		CacheBasePath: cacheDir,
		RepositoryConfig: &RepoConfig{
			Repositories: map[string]Repository{
				"testrepo": {
					CacheSuffixes: []string{".rpm"},
					Mirrors:       []string{"https://httpbin.org/anything/"},
				},
			},
		},
	})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/some/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	t.Logf("Status: %d, Body: %s", rec.Code, rec.Body.String())
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "/anything/some/path/file.rpm")
}

// --- Second GET from cache test ---

func TestCacheSecondRequestServedFromCache(t *testing.T) {
	requestCount := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "upstream-body")
	}))
	defer upstream.Close()

	pp, _ := newTestProxy(t, []string{upstream.URL + "/"})
	app := newTestApp(pp)

	// First request: cache miss, fetches from upstream
	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "upstream-body", rec.Body.String())
	assert.Equal(t, 1, requestCount)

	// Second request: should be served from cache, no upstream hit
	req2 := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec2 := httptest.NewRecorder()
	app.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, "upstream-body", rec2.Body.String())
	assert.Equal(t, 1, requestCount, "expected no additional upstream request")
}

// --- HEAD for non-cached file ---

func TestForwardProxyHead(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	pp, _ := newTestProxy(t, []string{upstream.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodHead, "/testrepo/repodata/repomd.xml", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestForwardProxyMultipleRedirectStatusCodes(t *testing.T) {
	for _, code := range []int{301, 302, 303, 307, 308} {
		t.Run(fmt.Sprintf("redirect_%d", code), func(t *testing.T) {
			destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, "final")
			}))
			defer destination.Close()

			mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", destination.URL+r.URL.Path)
				w.WriteHeader(code)
			}))
			defer mirror.Close()

			pp, _ := newTestProxy(t, []string{mirror.URL + "/"})
			app := newTestApp(pp)

			req := httptest.NewRequest(http.MethodGet, "/testrepo/file.rpm", nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "final", rec.Body.String())
		})
	}
}

// --- Retry tests ---

func TestForwardProxyRetryAfterRedirect5xx(t *testing.T) {
	// Simulates a redirector (like download.fedoraproject.org) that redirects to
	// a broken mirror on the first attempt, then a working mirror on retry.
	attempt := 0
	goodMirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "good-mirror-body")
	}))
	defer goodMirror.Close()

	badMirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer badMirror.Close()

	// Redirector alternates between bad and good mirror
	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt == 1 {
			w.Header().Set("Location", badMirror.URL+r.URL.Path)
		} else {
			w.Header().Set("Location", goodMirror.URL+r.URL.Path)
		}
		w.WriteHeader(http.StatusFound)
	}))
	defer redirector.Close()

	pp, _ := newTestProxyWithRetries(t, []string{redirector.URL + "/"}, 3)
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "good-mirror-body", rec.Body.String())
	assert.Equal(t, 2, attempt, "expected 2 attempts to the redirector")
}

func TestForwardProxyRetryExhausted(t *testing.T) {
	// All retry attempts return 503 — should return the 503 to the client
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "unavailable")
	}))
	defer mirror.Close()

	pp, _ := newTestProxyWithRetries(t, []string{mirror.URL + "/"}, 2)
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestForwardProxyNoRetryOn404(t *testing.T) {
	// 404 should not trigger a retry
	requestCount := 0
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mirror.Close()

	pp, _ := newTestProxyWithRetries(t, []string{mirror.URL + "/"}, 3)
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, 1, requestCount, "expected no retries for 404")
}

func TestForwardProxyDefaultRetriesIsOne(t *testing.T) {
	// With default retries (0 in config = 1 attempt), 503 should not be retried
	requestCount := 0
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer mirror.Close()

	pp, _ := newTestProxy(t, []string{mirror.URL + "/"})
	app := newTestApp(pp)

	req := httptest.NewRequest(http.MethodGet, "/testrepo/path/file.rpm", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, 1, requestCount, "expected only 1 attempt with default retries")
}
