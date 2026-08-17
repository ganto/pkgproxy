# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased](https://github.com/ganto/pkgproxy/commits/HEAD/)

### Added

- Per-repository `cdn` config field to proxy vendor CDNs that publish no public mirrors
- Per-repository `mtls` config field with client `cert`, `key` and optional `ca` for CDNs using mutual TLS
- Support for proxying entitled Red Hat content from `cdn.redhat.com` without client-side certificates
- Client config snippet for Red Hat Enterprise Linux on the landing page and in the README
- Top-level `branding` config field to customize the landing page title and description
- Landing page now shows the running pkgproxy version
- Container image now runs `serve` by default and loads bundled config from `$KO_DATA_PATH`
- `PKGPROXY_TRUST_PROXY` env var (and `--trust-proxy` flag) to opt in to X-Forwarded-For trust
- `PKGPROXY_HOST` env var to set the listen address without passing `--host` on the command line

### Changed

- Repositories must now define exactly one of `mirrors` or `cdn`; setting both is rejected
- Upstream URLs are validated at startup: they must be absolute and use `http` or `https`
- **Breaking:** `remote_ip` in access logs now reflects the direct connecting peer by default; set `PKGPROXY_TRUST_PROXY` to restore XFF-based IP extraction when running behind a reverse proxy
- **Breaking:** Removed the `--public-host` flag and `PKGPROXY_PUBLIC_HOST` env var; the landing page now fills in config snippet hostnames automatically — server-side from the request's `Host` header (works for `curl` too), further corrected client-side to the browser's own URL when that differs (e.g. behind a TLS-terminating reverse proxy)
- Upgraded Echo web framework to v5.1.1
- Config-file errors now list all default paths attempted, not just the last one

## [v0.2.0](https://github.com/ganto/pkgproxy/releases/tag/v0.2.0) - 2026-04-06

### Added

- Per-repository `exclude` config field to prevent specific filenames or suffixes from being cached
- Support caching Gentoo distfiles with `suffixes: ["*"]` wildcard and `exclude` list
- Gentoo e2e test using `emerge --fetchonly` in a `gentoo/stage3` container

## [v0.1.2](https://github.com/ganto/pkgproxy/releases/tag/v0.1.2) - 2026-03-28

### Fixed

- Disable Echo v5's default 30-second `WriteTimeout` which killed streaming responses for large package files

## [v0.1.1](https://github.com/ganto/pkgproxy/releases/tag/v0.1.1) - 2026-03-25

### Added

- End-to-end tests using real package managers (dnf, apt, pacman) in containers via `make e2e`
- `ubuntu-security` repository in default configuration

### Changed

- Cache-miss responses are now streamed directly to a temp file on disk instead of being buffered in memory, eliminating memory spikes for large packages
- `Content-Length` is validated before committing cached files, preventing truncated upstream responses from being cached
- Client disconnects no longer prevent caching — if a client aborts mid-download, the upstream response is still fully received and cached
- Landing page snippets for Debian/Ubuntu use `<release>` placeholder instead of hardcoded codenames
- README client configuration examples updated to current stable releases (Debian trixie, Ubuntu noble)

## [v0.1.0](https://github.com/ganto/pkgproxy/releases/tag/v0.1.0) - 2026-03-20

### Added

- Caching forward proxy for Linux package repositories
- Support for RPM-based distributions: Fedora, CentOS, CentOS Stream, AlmaLinux, Rocky Linux, and EPEL/COPR
- Support for DEB-based distributions: Debian (including security updates) and Ubuntu
- Support for Arch Linux repositories
- YAML-based configuration of repositories, upstream mirrors, and cacheable file suffixes
- Automatic failover across multiple configured upstream mirrors with optional retry on 5xx responses
- HTTP landing page listing all configured repositories with ready-to-use client configuration snippets for supported package managers
- Cache invalidation via HTTP `DELETE` requests to remove individual cached files
- Multi-architecture container image (amd64 + arm64) published to GitHub Container Registry, signed with cosign via GitHub OIDC
