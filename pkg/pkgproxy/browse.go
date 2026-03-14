// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	echo "github.com/labstack/echo/v5"
)

type browseEntry struct {
	Name        string
	IsDir       bool
	ModTime     time.Time
	Size        int64
	FileCount   int
	BrowseURL   string
	DownloadURL string
}

type breadcrumb struct {
	Label string
	URL   string
}

type browsePageData struct {
	Title       string
	Breadcrumbs []breadcrumb
	Parent      string
	Entries     []browseEntry
}

type dirStats struct {
	fileCount int
	totalSize int64
}

// computeDirStats walks dir and returns the total recursive file count and size.
// Returns zero stats if the directory does not exist.
func computeDirStats(dir string) (dirStats, error) {
	var s dirStats
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			s.fileCount++
			s.totalSize += info.Size()
		}
		return nil
	})
	if os.IsNotExist(err) {
		return dirStats{}, nil
	}
	return s, err
}

// formatSize formats a byte count as a human-readable string (e.g. "1.4 MiB").
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

var browsePageTmpl = template.Must(template.New("browse").Funcs(template.FuncMap{
	"formatSize": formatSize,
}).Parse(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>pkgproxy cache: {{.Title}}</title>
  <style>
    body { font-family: sans-serif; max-width: 1100px; margin: 2rem auto; padding: 0 1rem; }
    h1 { border-bottom: 2px solid #333; padding-bottom: 0.25rem; }
    nav { margin-bottom: 1rem; font-family: monospace; font-size: 1.05em; }
    nav a { color: #0066cc; text-decoration: none; }
    nav a:hover { text-decoration: underline; }
    nav .sep { color: #999; margin: 0 0.3em; }
    table { width: 100%; border-collapse: collapse; font-family: monospace; font-size: 0.95em; }
    th { text-align: left; border-bottom: 2px solid #333; padding: 0.4rem 0.75rem; font-family: sans-serif; }
    td { padding: 0.3rem 0.75rem; border-bottom: 1px solid #eee; }
    td.right { text-align: right; }
    a { color: #0066cc; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .muted { color: #999; }
    .up td { background: #f9f9f9; }
  </style>
</head>
<body>
  <h1>Cache browser</h1>
  <nav>
    {{range $i, $b := .Breadcrumbs}}{{if $i}}<span class="sep">/</span>{{end}}<a href="{{$b.URL}}">{{$b.Label}}</a>{{end}}
  </nav>
  <table>
    <thead>
      <tr>
        <th>Name</th>
        <th>Modified</th>
        <th>Size</th>
        <th>Files</th>
      </tr>
    </thead>
    <tbody>
      {{if .Parent}}<tr class="up">
        <td><a href="{{.Parent}}">..</a></td>
        <td></td><td></td><td></td>
      </tr>{{end}}
      {{range .Entries}}<tr>
        <td>
          {{if .IsDir}}<a href="{{.BrowseURL}}">{{.Name}}/</a>
          {{else}}<a href="{{.DownloadURL}}">{{.Name}}</a>{{end}}
        </td>
        <td class="muted">{{.ModTime.Format "2006-01-02 15:04:05"}}</td>
        <td>{{formatSize .Size}}</td>
        <td>{{if .IsDir}}{{.FileCount}}{{else}}<span class="muted">&mdash;</span>{{end}}</td>
      </tr>{{end}}
      {{if not .Entries}}<tr><td colspan="4" class="muted">No cached files.</td></tr>{{end}}
    </tbody>
  </table>
</body>
</html>`))

// Browse middleware serves a directory listing of the local file cache for requests
// that include the ?browse query parameter on a known repository path.
func (pp *pkgProxy) Browse(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		req := c.Request()

		if _, hasBrowse := req.URL.Query()["browse"]; !hasBrowse {
			return next(c)
		}
		if !pp.isRepositoryRequest(req.URL.Path) {
			return next(c)
		}

		repo := getRepoFromURI(req.URL.Path)

		cachePath, err := pp.upstreams[repo].cache.GetFilePath(req.URL.Path)
		if err != nil {
			return c.JSON(http.StatusForbidden, map[string]string{"message": "Forbidden"})
		}

		info, statErr := os.Stat(cachePath)
		if statErr != nil && !os.IsNotExist(statErr) {
			return echo.NewHTTPError(http.StatusInternalServerError, statErr.Error())
		}

		// If the path resolves to a cached file, redirect to download it.
		if statErr == nil && !info.IsDir() {
			return c.Redirect(http.StatusFound, req.URL.Path)
		}

		// Read directory entries (may not exist yet if nothing cached).
		var rawEntries []os.DirEntry
		if statErr == nil {
			rawEntries, err = os.ReadDir(cachePath)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
			}
		}

		entries := make([]browseEntry, 0, len(rawEntries))
		for _, de := range rawEntries {
			fi, err := de.Info()
			if err != nil {
				continue
			}
			entryURLPath := path.Join(req.URL.Path, de.Name())
			entry := browseEntry{
				Name:    de.Name(),
				IsDir:   de.IsDir(),
				ModTime: fi.ModTime(),
			}
			if de.IsDir() {
				entry.BrowseURL = entryURLPath + "/?browse"
				s, _ := computeDirStats(filepath.Join(cachePath, de.Name()))
				entry.Size = s.totalSize
				entry.FileCount = s.fileCount
			} else {
				entry.DownloadURL = entryURLPath
				entry.Size = fi.Size()
			}
			entries = append(entries, entry)
		}

		cleanPath := strings.TrimRight(req.URL.Path, "/")
		var parent string
		if cleanPath != "/"+repo {
			parent = path.Dir(cleanPath) + "/?browse"
		}

		var buf bytes.Buffer
		if err := browsePageTmpl.Execute(&buf, browsePageData{
			Title:       req.URL.Path,
			Breadcrumbs: buildBreadcrumbs(req.URL.Path),
			Parent:      parent,
			Entries:     entries,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		return c.HTML(http.StatusOK, buf.String())
	}
}

// buildBreadcrumbs generates breadcrumb navigation entries for the given URL path.
func buildBreadcrumbs(urlPath string) []breadcrumb {
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	crumbs := make([]breadcrumb, 0, len(parts))
	for i, part := range parts {
		if part == "" {
			continue
		}
		crumbs = append(crumbs, breadcrumb{
			Label: part,
			URL:   "/" + strings.Join(parts[:i+1], "/") + "/?browse",
		})
	}
	return crumbs
}
