// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	echo "github.com/labstack/echo/v5"
)

// isLandingPageRequest returns true when the URI targets a repository root (e.g. /fedora or /fedora/).
func isLandingPageRequest(uri string) bool {
	repo := getRepoFromURI(uri)
	if repo == "" {
		return false
	}
	rest := strings.TrimPrefix(uri, "/"+repo)
	// Strip query string if present
	if i := strings.IndexByte(rest, '?'); i >= 0 {
		rest = rest[:i]
	}
	return rest == "" || rest == "/"
}

var landingPageTmpl = template.Must(template.New("landing").Funcs(template.FuncMap{
	"formatSize": formatSize,
}).Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>pkgproxy: {{.Name}}</title>
  <style>
    body { font-family: sans-serif; max-width: 800px; margin: 2rem auto; padding: 0 1rem; }
    h1 { border-bottom: 2px solid #333; padding-bottom: 0.25rem; }
    h2 { margin-top: 1.5rem; }
    ul { list-style: none; padding: 0; }
    li { padding: 0.25rem 0; }
    a { color: #0066cc; }
    .stat { color: #555; font-size: 0.95em; }
  </style>
</head>
<body>
  <h1>{{.Name}}</h1>
  <h2>Mirrors</h2>
  <ul>
    {{range .Mirrors}}<li><a href="{{.}}">{{.}}</a></li>
    {{end}}
  </ul>
  <h2>Cached file types</h2>
  <ul>
    {{range .Suffixes}}<li>{{.}}</li>
    {{end}}
  </ul>
  <h2>Local cache</h2>
  <p class="stat">{{.CacheFiles}} files &mdash; {{formatSize .CacheSize}}</p>
  <p><a href="{{.BrowseURL}}">Browse cache &rarr;</a></p>
</body>
</html>`))

// LandingPage middleware serves repository metadata for requests to the repository root path.
// The response format (JSON or HTML) is selected based on the Accept request header.
func (pp *pkgProxy) LandingPage(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		uri := c.Request().RequestURI

		if !isLandingPageRequest(uri) || !pp.isRepositoryRequest(uri) {
			return next(c)
		}

		repo := getRepoFromURI(uri)
		mirrors := make([]string, len(pp.upstreams[repo].mirrors))
		for i, m := range pp.upstreams[repo].mirrors {
			mirrors[i] = m.String()
		}
		suffixes := pp.repoConfig.Repositories[repo].CacheSuffixes

		var stats dirStats
		if repoDir, err := pp.upstreams[repo].cache.GetFilePath("/" + repo); err == nil {
			stats, _ = computeDirStats(repoDir)
		}

		if strings.Contains(c.Request().Header.Get("Accept"), "application/json") {
			return c.JSON(http.StatusOK, map[string]any{
				"name":             repo,
				"mirrors":          mirrors,
				"suffixes":         suffixes,
				"cache_files":      stats.fileCount,
				"cache_size_bytes": stats.totalSize,
			})
		}

		type templateData struct {
			Name       string
			Mirrors    []string
			Suffixes   []string
			CacheFiles int
			CacheSize  int64
			BrowseURL  string
		}
		var buf bytes.Buffer
		if err := landingPageTmpl.Execute(&buf, templateData{
			Name:       repo,
			Mirrors:    mirrors,
			Suffixes:   suffixes,
			CacheFiles: stats.fileCount,
			CacheSize:  stats.totalSize,
			BrowseURL:  "/" + repo + "/?browse",
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.HTML(http.StatusOK, buf.String())
	}
}
