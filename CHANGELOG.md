# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased](https://github.com/ganto/pkgproxy/commits/HEAD/)

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
