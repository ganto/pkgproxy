## Why

When users or administrators browse to the pkgproxy root URL (`/`), they currently receive a blank or error response. A landing page would make the service self-documenting, showing what repositories are available and how to configure package managers to use them.

## What Changes

- Add an HTTP handler for the root path (`/`) that renders an HTML landing page
- The landing page lists all configured repositories with their cache paths and upstream mirrors
- Provides copy-paste configuration snippets for common package managers (dnf/yum, apt, pacman)
- Shows basic service status (uptime, cache directory)

## Capabilities

### New Capabilities

- `http-landing-page`: HTTP handler serving an HTML overview page at `/` listing configured repositories, mirror URLs, and package manager configuration snippets

### Modified Capabilities

<!-- No existing spec-level behavior changes -->

## Impact

- `cmd/serve.go`: Register new route handler
- `pkg/pkgproxy/`: New handler file (e.g. `landing.go`) with template rendering
- No changes to caching or proxy logic
