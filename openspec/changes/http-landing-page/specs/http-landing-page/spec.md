## ADDED Requirements

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
The landing page SHALL include copy-paste configuration snippets for repositories whose names appear in the project README client configuration section. Snippets MUST use the exact format from the README including the full URI path suffix after the repository name (e.g. `/$releasever/BaseOS/$basearch/os/`), with `<pkgproxy>` replaced by the configured public address. Repositories not documented in the README SHALL have their snippet omitted entirely.

#### Scenario: Known RPM repository shows dnf/yum baseurl snippet with full path
- **WHEN** a repository name matches one documented in the README with `.rpm` suffixes
- **THEN** the landing page shows the exact `baseurl=http://<address>/<repo>/<path-suffix>` snippet from the README for that repository

#### Scenario: Known DEB repository shows apt sources snippet with suite and components
- **WHEN** a repository name matches one documented in the README with `.deb` suffixes
- **THEN** the landing page shows the exact one or more `deb http://<address>/<repo> <suite> <components>` lines from the README for that repository

#### Scenario: Known Arch repository shows pacman mirrorlist snippet with full path
- **WHEN** a repository name matches one documented in the README with `.tar.zst` or `.pkg.tar.*` suffixes
- **THEN** the landing page shows the exact `Server = http://<address>/<repo>/$repo/os/$arch` snippet from the README

#### Scenario: Unknown repository has no snippet
- **WHEN** a repository name has no matching entry in the README client configuration section
- **THEN** no configuration snippet is shown for that repository

#### Scenario: Snippet uses listen host:port when no public host is set
- **WHEN** no public host is configured and pkgproxy is started with `--host h --port p`
- **THEN** all config snippets on the landing page use `h:p` as the address

#### Scenario: Snippet uses public address verbatim without appending listen port
- **WHEN** a public address is configured via `--public-host` or `PKGPROXY_PUBLIC_HOST`
- **THEN** all config snippets use that value verbatim and the listen port is not appended

### Requirement: Configurable public address
pkgproxy SHALL expose a `--public-host` CLI flag (on the `serve` subcommand) and a `PKGPROXY_PUBLIC_HOST` environment variable that set the address rendered in landing page config snippets. The value MAY include a port (e.g. `myproxy.lan:9090`), in which case that port is used as-is. When a public host is set, the listen port is NOT appended. When no public host is set, the listen `host:port` is used. The CLI flag takes precedence over the environment variable when both are set.

#### Scenario: Flag sets the public address without appending listen port
- **WHEN** pkgproxy is started with `--public-host myproxy.lan`
- **THEN** the landing page config snippets use `myproxy.lan` with no port suffix

#### Scenario: Flag value with embedded port is used verbatim
- **WHEN** pkgproxy is started with `--public-host myproxy.lan:9090`
- **THEN** the landing page config snippets use `myproxy.lan:9090` verbatim

#### Scenario: Environment variable sets the public address
- **WHEN** `PKGPROXY_PUBLIC_HOST=myproxy.lan` is set and no `--public-host` flag is given
- **THEN** the landing page config snippets use `myproxy.lan` with no port suffix

#### Scenario: CLI flag takes precedence over environment variable
- **WHEN** both `--public-host myproxy.lan` and `PKGPROXY_PUBLIC_HOST=other.host` are set
- **THEN** the landing page config snippets use `myproxy.lan`

#### Scenario: Default renders listen address with port
- **WHEN** neither `--public-host` nor `PKGPROXY_PUBLIC_HOST` is set
- **THEN** the landing page config snippets use `<host>:<port>` from the listen configuration

### Requirement: README documents CLI flags and the public host option
The project README SHALL contain a CLI flags reference table covering all `serve` subcommand flags, including `--public-host` and the `PKGPROXY_PUBLIC_HOST` environment variable with a description of their effect.

#### Scenario: README CLI flags table includes all serve flags
- **WHEN** a user reads the README
- **THEN** they find a table listing all `serve` subcommand flags with their defaults and descriptions

#### Scenario: README CLI flags table includes public-host and its env var
- **WHEN** a user reads the README
- **THEN** they can find `--public-host` and `PKGPROXY_PUBLIC_HOST` with a description of their effect

### Requirement: No external dependencies for page rendering
The landing page SHALL be rendered using only Go standard library (`html/template`), with no JavaScript or external stylesheet resources.

#### Scenario: Page is self-contained
- **WHEN** the landing page HTML is served
- **THEN** it contains no references to external scripts, stylesheets, or fonts
