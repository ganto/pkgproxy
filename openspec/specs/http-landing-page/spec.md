## Requirements

### Requirement: Landing page served at root path
pkgproxy SHALL serve an HTML landing page at `GET /` that lists all configured repositories and their upstream mirrors.

#### Scenario: Root path returns HTML
- **WHEN** a client sends `GET /` to pkgproxy
- **THEN** the server responds with HTTP 200 and `Content-Type: text/html`

#### Scenario: Page lists configured repositories
- **WHEN** the landing page is rendered
- **THEN** it displays the name of every repository defined in the configuration

#### Scenario: Page renders upstream mirrors as clickable links
- **WHEN** the landing page is rendered
- **THEN** each upstream mirror URL is rendered as an HTML anchor (`<a href="...">`) that opens the mirror in the browser

### Requirement: Package manager configuration snippets match README
The landing page SHALL include copy-paste configuration snippets for repositories whose names appear in the project README client configuration section. Snippets MUST match the URL structure from the README including the full URI path suffix after the repository name (e.g. `/$releasever/BaseOS/$basearch/os/`), with `<pkgproxy>` replaced by an address resolved automatically (see "Automatic hostname substitution"). Repositories not documented in the README SHALL have their snippet omitted entirely. DEB-based snippets (Debian, Ubuntu) SHALL use a `<release>` placeholder instead of hardcoded release codenames, matching the placeholder convention used by the COPR snippet (`<user>`, `<repo>`). The README retains concrete codename examples for readability; the landing page uses placeholders.

#### Scenario: Known RPM repository shows dnf/yum baseurl snippet with full path
- **WHEN** a repository name matches one documented in the README with `.rpm` suffixes
- **THEN** the landing page shows the exact `baseurl=http://<address>/<repo>/<path-suffix>` snippet from the README for that repository

#### Scenario: Known DEB repository shows apt sources snippet with release placeholder
- **WHEN** a repository name matches one documented in the README with `.deb` suffixes
- **THEN** the landing page shows one or more `deb http://<address>/<repo> <release> <components>` lines using `<release>` as a placeholder instead of a hardcoded codename

#### Scenario: Known Arch repository shows pacman mirrorlist snippet with full path
- **WHEN** a repository name matches one documented in the README with `.tar.zst` or `.pkg.tar.*` suffixes
- **THEN** the landing page shows the exact `Server = http://<address>/<repo>/$repo/os/$arch` snippet from the README

#### Scenario: Unknown repository has no snippet
- **WHEN** a repository name has no matching entry in the README client configuration section
- **THEN** no configuration snippet is shown for that repository

### Requirement: Automatic hostname substitution
Configuration snippets SHALL NOT depend on a server-side public-address setting. Instead, the address is resolved automatically in two layers:

1. **Server-side default.** Every response renders snippets using the `Host` header of the incoming request, so any client — including `curl` and other non-browser HTTP clients — receives a working, copy-pasteable address without needing JavaScript.
2. **Client-side correction.** The page additionally includes a small inline script that, once loaded in a browser, compares the server-rendered default against the page's own `window.location.origin` and, if they differ, rewrites the snippets to match it. This corrects cases the `Host` header alone cannot reveal, such as a reverse proxy that terminates TLS (the browser's scheme is `https`, but pkgproxy itself only ever sees plain HTTP).

This keeps snippets correct with no pkgproxy-side configuration in the common case (reverse proxy forwarding the original `Host` header), and self-corrects in the browser when it doesn't.

#### Scenario: Server renders snippets using the request's Host header
- **WHEN** a client sends `GET /` with a given `Host` header
- **THEN** every configuration snippet on the returned page uses that `Host` value as the address, with no placeholder text

#### Scenario: Inline script corrects the address to the page's origin when it differs
- **WHEN** the landing page is loaded in a browser whose `window.location.origin` differs from the server-rendered default (e.g. behind a TLS-terminating reverse proxy)
- **THEN** an inline script rewrites the address in the rendered snippets to `window.location.origin`

#### Scenario: No client-side change when origins already match
- **WHEN** the landing page is loaded in a browser whose `window.location.origin` matches the server-rendered default
- **THEN** the inline script makes no changes to the rendered snippets

#### Scenario: Server-rendered address is used verbatim without JavaScript
- **WHEN** the landing page is loaded with JavaScript disabled or unavailable, or fetched with a non-browser client such as `curl`
- **THEN** the configuration snippets show the server-rendered address derived from the request's `Host` header

### Requirement: README documents CLI flags
The project README SHALL contain a CLI flags reference table covering all `serve` subcommand flags.

#### Scenario: README CLI flags table includes all serve flags
- **WHEN** a user reads the README
- **THEN** they find a table listing all `serve` subcommand flags with their defaults and descriptions

### Requirement: No external script, stylesheet, or font dependencies
The landing page SHALL be rendered using only the Go standard library (`html/template`) plus a small inline script used solely for client-side hostname correction (see "Automatic hostname substitution"). It SHALL NOT load external JavaScript, stylesheets, or fonts.

#### Scenario: Page is self-contained
- **WHEN** the landing page HTML is served
- **THEN** it contains no references to external scripts, stylesheets, or fonts
