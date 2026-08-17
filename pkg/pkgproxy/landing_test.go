// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	echo "github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

// defaultTestHost is the Host httptest.NewRequest assigns to a relative
// target ("/") when the test doesn't set req.Host explicitly.
const defaultTestHost = "example.com"

func newLandingApp(config *RepoConfig) *echo.Echo {
	return newLandingAppWithVersion(config, "1.2.3")
}

func newLandingAppWithVersion(config *RepoConfig, version string) *echo.Echo {
	app := echo.New()
	app.GET("/", LandingHandler(config, version))
	return app
}

func getLandingBody(t *testing.T, app *echo.Echo) string {
	t.Helper()
	return getLandingBodyWithHost(t, app, "")
}

// getLandingBodyWithHost issues GET / with the given Host header (the header
// curl and other HTTP clients send based on the URL they were given). An
// empty host leaves httptest's default ("example.com") in place.
func getLandingBodyWithHost(t *testing.T, app *echo.Echo, host string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if host != "" {
		req.Host = host
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	return rec.Body.String()
}

func TestLandingHandlerHTTP(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/fedora/"}},
		},
	}
	app := newLandingApp(config)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/html")
}

func TestLandingHandlerRepoNames(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora":  {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/fedora/"}},
			"debian":  {CacheSuffixes: []string{".deb"}, Mirrors: []string{"https://mirror.example.com/debian/"}},
			"unknown": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/unknown/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config))

	assert.Contains(t, body, "fedora")
	assert.Contains(t, body, "debian")
	assert.Contains(t, body, "unknown")
}

func TestLandingHandlerMirrorLinks(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/fedora/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config))

	assert.Contains(t, body, `<a href="https://mirror.example.com/fedora/">https://mirror.example.com/fedora/</a>`)
}

func TestLandingHandlerDefaultBranding(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config))

	assert.Contains(t, body, "<title>pkgproxy</title>")
	assert.Contains(t, body, "<h1>pkgproxy</h1>")
	assert.Contains(t, body, "<p>Caching forward proxy for Linux package repositories.</p>")
}

func TestLandingHandlerCustomBranding(t *testing.T) {
	config := &RepoConfig{
		Branding: &BrandingConfig{
			Title:       "Acme Package Mirror",
			Description: "Internal package cache for Acme Corp.",
		},
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config))

	assert.Contains(t, body, "<title>Acme Package Mirror</title>")
	assert.Contains(t, body, "<h1>Acme Package Mirror</h1>")
	assert.Contains(t, body, "<p>Internal package cache for Acme Corp.</p>")
	assert.NotContains(t, body, "pkgproxy</h1>")
}

func TestLandingHandlerBrandingPartialOverride(t *testing.T) {
	config := &RepoConfig{
		Branding: &BrandingConfig{Title: "Acme Package Mirror"},
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config))

	assert.Contains(t, body, "<h1>Acme Package Mirror</h1>")
	// Description falls back to the default when only the title is customized.
	assert.Contains(t, body, "<p>Caching forward proxy for Linux package repositories.</p>")
}

func TestLandingHandlerVersion(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingAppWithVersion(config, "v0.3.1"))

	assert.Contains(t, body, "<p>pkgproxy v0.3.1</p>")
}

func TestLandingHandlerCDNLink(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"rhel": {
				CacheSuffixes: []string{".rpm"},
				CDN:           "https://cdn.redhat.com/",
				MTLS:          &MTLSConfig{Cert: "entitlement.pem", Key: "entitlement-key.pem"},
			},
		},
	}
	body := getLandingBody(t, newLandingApp(config))

	assert.Contains(t, body, "<strong>CDN:</strong>")
	assert.Contains(t, body, `<a href="https://cdn.redhat.com/">https://cdn.redhat.com/</a>`)
	assert.NotContains(t, body, "<strong>Mirrors:</strong>")
	assert.Contains(t, body, "baseurl=http://"+defaultTestHost+"/rhel/content/dist/rhel$releasever/$releasever/$basearch/baseos/os")
	// The certificate paths are local secrets and must never be rendered.
	assert.NotContains(t, body, "entitlement")
}

func TestLandingHandlerKnownSnippets(t *testing.T) {
	tests := []struct {
		repo   string
		suffix string
		wantIn string
	}{
		{"almalinux", ".rpm", "baseurl=http://" + defaultTestHost + "/almalinux/$releasever/BaseOS/$basearch/os/"},
		{"archlinux", ".tar.zst", "Server = http://" + defaultTestHost + "/archlinux/$repo/os/$arch"},
		{"centos", ".rpm", "baseurl=http://" + defaultTestHost + "/centos/$releasever/os/$basearch/"},
		{"centos-stream", ".rpm", "baseurl=http://" + defaultTestHost + "/centos-stream/$stream/BaseOS/$basearch/os/"},
		{"debian", ".deb", "deb http://" + defaultTestHost + "/debian           &lt;release&gt;            main contrib non-free non-free-firmware"},
		{"debian-security", ".deb", "deb http://" + defaultTestHost + "/debian-security  &lt;release&gt;-security   main contrib non-free non-free-firmware"},
		{"epel", ".rpm", "baseurl=http://" + defaultTestHost + "/epel/$releasever/Everything/$basearch/"},
		{"fedora", ".rpm", "baseurl=http://" + defaultTestHost + "/fedora/releases/$releasever/Everything/$basearch/os/"},
		{"rockylinux", ".rpm", "baseurl=http://" + defaultTestHost + "/rockylinux/$releasever/BaseOS/$basearch/os/"},
		{"ubuntu", ".deb", "deb http://" + defaultTestHost + "/ubuntu           &lt;release&gt;           main restricted universe multiverse"},
		{"ubuntu-security", ".deb", "deb http://" + defaultTestHost + "/ubuntu-security  &lt;release&gt;-security  main restricted universe multiverse"},
	}
	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			config := &RepoConfig{
				Repositories: map[string]Repository{
					tt.repo: {CacheSuffixes: []string{tt.suffix}, Mirrors: []string{"https://mirror.example.com/"}},
				},
			}
			body := getLandingBody(t, newLandingApp(config))
			assert.Contains(t, body, tt.wantIn)
		})
	}
}

func TestLandingHandlerUnknownRepoNoSnippet(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"myprivaterepo": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config))

	assert.NotContains(t, body, "Configuration snippet")
	assert.NotContains(t, body, "baseurl=")
}

// TestLandingHandlerUsesRequestHost is the core of curl support: snippets are
// rendered server-side using whatever Host header the client's request
// carried, so a plain HTTP client that never runs JavaScript still gets a
// working, copy-pasteable address — not a placeholder.
func TestLandingHandlerUsesRequestHost(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	app := newLandingApp(config)

	body := getLandingBodyWithHost(t, app, "myproxy.lan:9090")
	assert.Contains(t, body, "baseurl=http://myproxy.lan:9090/fedora/releases/$releasever/Everything/$basearch/os/")

	body = getLandingBodyWithHost(t, app, "other.example:8080")
	assert.Contains(t, body, "baseurl=http://other.example:8080/fedora/releases/$releasever/Everything/$basearch/os/")
}

// TestLandingHandlerHostSubstitutionScript verifies the page ships the inline
// script that further corrects the server-rendered address to the browser's
// own URL on load (needed when a reverse proxy changes the scheme, e.g. TLS
// termination, which the Host header alone can't reveal). The Go test suite
// has no JS engine, so this only checks the script is wired up correctly
// rather than executing it.
func TestLandingHandlerHostSubstitutionScript(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBodyWithHost(t, newLandingApp(config), "myproxy.lan")

	assert.Contains(t, body, "baseurl=http://myproxy.lan/fedora/")
	assert.Contains(t, body, `var defaultOrigin = "http://myproxy.lan";`)
	assert.Contains(t, body, "var origin = window.location.origin;")
	assert.Contains(t, body, `document.querySelectorAll("pre")`)
}

func TestLandingHandlerDefaultListenAddr(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config))

	assert.Contains(t, body, "http://"+defaultTestHost+"/fedora/")
}

func TestLandingHandlerSelfContained(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config))

	assert.NotContains(t, body, "https://fonts.")
	// The inline hostname-substitution script is allowed; only *external*
	// script/stylesheet/font references are not.
	assert.NotContains(t, body, "<script src")
	assert.NotContains(t, body, `<link rel="stylesheet" href="http`)
}
