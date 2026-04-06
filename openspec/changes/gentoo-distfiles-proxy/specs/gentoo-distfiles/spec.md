## ADDED Requirements

### Requirement: Gentoo distfiles repository config entry
The `configs/pkgproxy.yaml` SHALL include a `gentoo` repository configured with `suffixes: ["*"]`, an `exclude` list covering mirror-specific metadata files (`layout.conf`, `timestamp.mirmon`, `timestamp.dev-local`), and at least two Swiss HTTPS mirrors plus `distfiles.gentoo.org` as authoritative fallback.

#### Scenario: Gentoo distfiles repository is configured
- **WHEN** pkgproxy loads its configuration
- **THEN** the `gentoo` repository is available with at least one upstream mirror

#### Scenario: layout.conf is not cached
- **WHEN** a client fetches `<proxy>/gentoo/distfiles/layout.conf`
- **THEN** pkgproxy proxies the file upstream but does not write it to the local cache

#### Scenario: Distfile fetched via emerge --fetchonly is proxied and cached
- **WHEN** portage runs `emerge --fetchonly app-text/tree` with `GENTOO_MIRRORS` pointing at pkgproxy
- **THEN** pkgproxy proxies the distfile from the upstream mirror and saves it to the local cache under `gentoo/distfiles/<xx>/<filename>`

#### Scenario: Cached distfile is served from disk on subsequent request
- **WHEN** portage fetches the same distfile a second time
- **THEN** pkgproxy serves the file from the local cache without contacting the upstream mirror

### Requirement: make.conf snippet in README and landing page
The README.md and HTTP landing page SHALL include a Gentoo `make.conf` snippet showing how to configure `GENTOO_MIRRORS` to point at the proxy.

#### Scenario: Gentoo configuration snippet is present
- **WHEN** a user views the README or the pkgproxy landing page
- **THEN** a `make.conf` snippet with `GENTOO_MIRRORS="http://<proxy>/gentoo"` is visible
