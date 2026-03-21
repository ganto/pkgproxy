## MODIFIED Requirements

### Requirement: Package manager configuration snippets match README
The landing page SHALL include copy-paste configuration snippets for repositories whose names appear in the project README client configuration section. Snippets MUST match the URL structure from the README including the full URI path suffix after the repository name (e.g. `/$releasever/BaseOS/$basearch/os/`), with `<pkgproxy>` replaced by the configured public address. Repositories not documented in the README SHALL have their snippet omitted entirely. DEB-based snippets (Debian, Ubuntu) SHALL use a `<release>` placeholder instead of hardcoded release codenames, matching the placeholder convention used by the COPR snippet (`<user>`, `<repo>`). The README retains concrete codename examples for readability; the landing page uses placeholders.

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

#### Scenario: Snippet uses listen host:port when no public host is set
- **WHEN** no public host is configured and pkgproxy is started with `--host h --port p`
- **THEN** all config snippets on the landing page use `h:p` as the address

#### Scenario: Snippet uses public address verbatim without appending listen port
- **WHEN** a public address is configured via `--public-host` or `PKGPROXY_PUBLIC_HOST`
- **THEN** all config snippets use that value verbatim and the listen port is not appended
