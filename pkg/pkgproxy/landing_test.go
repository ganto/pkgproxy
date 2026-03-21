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

func newLandingApp(config *RepoConfig, publicAddr string) *echo.Echo {
	app := echo.New()
	app.GET("/", LandingHandler(config, publicAddr))
	return app
}

func getLandingBody(t *testing.T, app *echo.Echo) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
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
	app := newLandingApp(config, "localhost:8080")
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
	body := getLandingBody(t, newLandingApp(config, "localhost:8080"))

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
	body := getLandingBody(t, newLandingApp(config, "localhost:8080"))

	assert.Contains(t, body, `<a href="https://mirror.example.com/fedora/">https://mirror.example.com/fedora/</a>`)
}

func TestLandingHandlerKnownSnippets(t *testing.T) {
	tests := []struct {
		repo   string
		suffix string
		wantIn string
	}{
		{"almalinux", ".rpm", "baseurl=http://localhost:8080/almalinux/$releasever/BaseOS/$basearch/os/"},
		{"archlinux", ".tar.zst", "Server = http://localhost:8080/archlinux/$repo/os/$arch"},
		{"centos", ".rpm", "baseurl=http://localhost:8080/centos/$releasever/os/$basearch/"},
		{"centos-stream", ".rpm", "baseurl=http://localhost:8080/centos-stream/$stream/BaseOS/$basearch/os/"},
		{"debian", ".deb", "deb http://localhost:8080/debian           &lt;release&gt;            main contrib non-free non-free-firmware"},
		{"debian-security", ".deb", "deb http://localhost:8080/debian-security  &lt;release&gt;-security   main contrib non-free non-free-firmware"},
		{"epel", ".rpm", "baseurl=http://localhost:8080/epel/$releasever/Everything/$basearch/"},
		{"fedora", ".rpm", "baseurl=http://localhost:8080/fedora/releases/$releasever/Everything/$basearch/os/"},
		{"rockylinux", ".rpm", "baseurl=http://localhost:8080/rockylinux/$releasever/BaseOS/$basearch/os/"},
		{"ubuntu", ".deb", "deb http://localhost:8080/ubuntu           &lt;release&gt;           main restricted universe multiverse"},
		{"ubuntu-security", ".deb", "deb http://localhost:8080/ubuntu-security  &lt;release&gt;-security  main restricted universe multiverse"},
	}
	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			config := &RepoConfig{
				Repositories: map[string]Repository{
					tt.repo: {CacheSuffixes: []string{tt.suffix}, Mirrors: []string{"https://mirror.example.com/"}},
				},
			}
			body := getLandingBody(t, newLandingApp(config, "localhost:8080"))
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
	body := getLandingBody(t, newLandingApp(config, "localhost:8080"))

	assert.NotContains(t, body, "Configuration snippet")
	assert.NotContains(t, body, "baseurl=")
}

func TestLandingHandlerPublicHostNoPort(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config, "myproxy.lan"))

	assert.Contains(t, body, "http://myproxy.lan/fedora/")
	assert.NotContains(t, body, "myproxy.lan:")
}

func TestLandingHandlerPublicHostWithPort(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config, "myproxy.lan:9090"))

	assert.Contains(t, body, "http://myproxy.lan:9090/fedora/")
}

func TestLandingHandlerDefaultListenAddr(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config, "localhost:8080"))

	assert.Contains(t, body, "http://localhost:8080/fedora/")
}

func TestLandingHandlerSelfContained(t *testing.T) {
	config := &RepoConfig{
		Repositories: map[string]Repository{
			"fedora": {CacheSuffixes: []string{".rpm"}, Mirrors: []string{"https://mirror.example.com/"}},
		},
	}
	body := getLandingBody(t, newLandingApp(config, "localhost:8080"))

	assert.NotContains(t, body, "https://fonts.")
	assert.NotContains(t, body, "<script src")
	assert.NotContains(t, body, `<link rel="stylesheet" href="http`)
}
