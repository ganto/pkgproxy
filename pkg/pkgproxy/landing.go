// Copyright 2026 Reto Gantenbein
// SPDX-License-Identifier: Apache-2.0
package pkgproxy

import (
	"bytes"
	"html/template"
	"net/http"
	"sort"

	echo "github.com/labstack/echo/v5"
)

const landingTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{.Title}}</title>
<style>
body { font-family: monospace; max-width: 900px; margin: 2em auto; padding: 0 1em; color: #222; }
h1 { border-bottom: 2px solid #444; padding-bottom: 0.3em; }
h2 { margin-top: 2em; border-bottom: 1px solid #ccc; }
pre { background: #f4f4f4; padding: 0.8em 1em; overflow-x: auto; }
ul { padding-left: 1.4em; }
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<p>{{.Description}}</p>
<p>pkgproxy {{.Version}}</p>
{{range .Repos}}
<h2>{{.Name}}</h2>
<p><strong>Mirrors:</strong></p>s
<ul>{{range .Mirrors}}<li><a href="{{.}}">{{.}}</a></li>{{end}}</ul>
{{with repoSnippet .Name}}
<p><strong>Configuration snippet:</strong></p>
<pre>{{.}}</pre>
{{end}}
{{end}}
</body>
</html>
`

// defaultTitle and defaultDescription are used when the config's 'branding'
// block is absent or leaves a field empty.
const (
	defaultTitle       = "pkgproxy"
	defaultDescription = "Caching forward proxy for Linux package repositories."
)

// snippetFuncs maps known repository names to functions that generate
// package manager configuration snippets for the landing page.
// Each function takes the public address (host or host:port) and returns
// the snippet string matching the format documented in the project README.
var snippetFuncs = map[string]func(string) string{
	"almalinux": func(addr string) string {
		return "[baseos]\n" +
			"# mirrorlist=https://mirrors.almalinux.org/mirrorlist/$releasever/baseos\n" +
			"baseurl=http://" + addr + "/almalinux/$releasever/BaseOS/$basearch/os/"
	},
	"archlinux": func(addr string) string {
		return "Server = http://" + addr + "/archlinux/$repo/os/$arch"
	},
	"centos": func(addr string) string {
		return "[base]\n" +
			"# mirrorlist=http://mirrorlist.centos.org/?release=$releasever&arch=$basearch&repo=os&infra=$infra\n" +
			"baseurl=http://" + addr + "/centos/$releasever/os/$basearch/"
	},
	"centos-stream": func(addr string) string {
		return "[baseos]\n" +
			"# metalink=https://mirrors.centos.org/metalink?repo=centos-baseos-$stream&arch=$basearch&protocol=https,http\n" +
			"baseurl=http://" + addr + "/centos-stream/$stream/BaseOS/$basearch/os/"
	},
	"copr": func(addr string) string {
		return "[copr:copr.fedorainfracloud.org:<user>:<repo>]\n" +
			"# baseurl=https://download.copr.fedorainfracloud.org/results/<user>/<repo>/fedora-$releasever-$basearch/\n" +
			"baseurl=http://" + addr + "/copr/<user>/<repo>/fedora-$releasever-$basearch/"
	},
	"debian": func(addr string) string {
		return "deb http://" + addr + "/debian           <release>            main contrib non-free non-free-firmware\n" +
			"deb http://" + addr + "/debian           <release>-updates    main contrib non-free non-free-firmware\n" +
			"deb http://" + addr + "/debian           <release>-backports  main contrib non-free non-free-firmware"
	},
	"debian-security": func(addr string) string {
		return "deb http://" + addr + "/debian-security  <release>-security   main contrib non-free non-free-firmware"
	},
	"epel": func(addr string) string {
		return "[epel]\n" +
			"# metalink=https://mirrors.fedoraproject.org/metalink?repo=epel-$releasever&arch=$basearch\n" +
			"baseurl=http://" + addr + "/epel/$releasever/Everything/$basearch/"
	},
	"gentoo": func(addr string) string {
		return "GENTOO_MIRRORS=\"http://" + addr + "/gentoo\""
	},
	"fedora": func(addr string) string {
		return "[fedora]\n" +
			"# metalink=https://mirrors.fedoraproject.org/metalink?repo=fedora-$releasever&arch=$basearch\n" +
			"baseurl=http://" + addr + "/fedora/releases/$releasever/Everything/$basearch/os/"
	},
	"rockylinux": func(addr string) string {
		return "[baseos]\n" +
			"# mirrorlist=https://mirrors.rockylinux.org/mirrorlist?arch=$basearch&repo=BaseOS-$releasever$rltype\n" +
			"baseurl=http://" + addr + "/rockylinux/$releasever/BaseOS/$basearch/os/"
	},
	"ubuntu": func(addr string) string {
		return "deb http://" + addr + "/ubuntu           <release>           main restricted universe multiverse\n" +
			"deb http://" + addr + "/ubuntu           <release>-updates   main restricted universe multiverse"
	},
	"ubuntu-security": func(addr string) string {
		return "deb http://" + addr + "/ubuntu-security  <release>-security  main restricted universe multiverse"
	},
}

// repoEntry holds a repository name and its configuration for template rendering.
type repoEntry struct {
	Name    string
	Mirrors []string
}

type landingData struct {
	Title       string
	Description string
	Version     string
	Repos []repoEntry
}

// brandingOrDefault returns the configured title and description, falling
// back to the built-in pkgproxy defaults for whichever field is unset.
func brandingOrDefault(branding *BrandingConfig) (title string, description string) {
	title, description = defaultTitle, defaultDescription
	if branding == nil {
		return title, description
	}
	if branding.Title != "" {
		title = branding.Title
	}
	if branding.Description != "" {
		description = branding.Description
	}
	return title, description
}

// sortedRepos returns repository entries sorted alphabetically by name.
func sortedRepos(config *RepoConfig) []repoEntry {
	names := make([]string, 0, len(config.Repositories))
	for name := range config.Repositories {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]repoEntry, 0, len(names))
	for _, name := range names {
		entries = append(entries, repoEntry{Name: name, Mirrors: config.Repositories[name].Mirrors})
	}
	return entries
}

// LandingHandler returns an Echo handler that renders an HTML overview page
// listing all configured repositories, their mirrors, and package manager snippets.
// publicAddr is the address (host or host:port) rendered in config snippets.
func LandingHandler(config *RepoConfig, publicAddr string, version string) echo.HandlerFunc {
	funcMap := template.FuncMap{
		"repoSnippet": func(name string) string {
			fn, ok := snippetFuncs[name]
			if !ok {
				return ""
			}
			return fn(publicAddr)
		},
	}
	tmpl := template.Must(template.New("landing").Funcs(funcMap).Parse(landingTemplate))

	title, description := brandingOrDefault(config.Branding)
	repos := sortedRepos(config)

	return func(c *echo.Context) error {
		data := landingData{
			Title:       title,
			Description: description,
			Version:     version,
			Repos:       repos,
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return err
		}
		c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=UTF-8")
		c.Response().WriteHeader(http.StatusOK)
		_, err := buf.WriteTo(c.Response())
		return err
	}
}
