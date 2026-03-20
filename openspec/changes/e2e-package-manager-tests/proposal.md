## Why

pkgproxy's existing tests use mock HTTP servers (`httptest.NewServer`) to simulate upstreams. While these validate the proxy logic in isolation, they cannot catch real-world breakage: stale mirror URLs in `configs/pkgproxy.yaml`, incorrect URL path structures in the landing page client configuration snippets, or subtle incompatibilities with actual package manager request patterns. When a mirror changes its layout or a new distro release alters its repo structure, there is no automated way to detect the problem before users hit it.

## What Changes

- Add end-to-end tests that start a real pkgproxy process, run real Linux distribution containers (via podman or docker, auto-detected with override), and exercise real package managers (`dnf`, `apt`, `pacman`) against real upstream mirrors through the proxy.
- Update landing page configuration snippets for Debian/Ubuntu to use `<release>` placeholders instead of hardcoded codenames, keeping them release-agnostic and in sync with what e2e tests validate.
- Update README client configuration examples for Debian/Ubuntu to use current stable releases (trixie, noble) with a note to substitute the actual codename. Fix stale hostname in CentOS Stream example.
- Add a `make e2e` target and gate the tests behind a `//go:build e2e` build tag.

## Capabilities

### New Capabilities
- `e2e-testing`: End-to-end test framework using podman containers with real package managers (dnf, apt, pacman) validating proxy behavior, mirror configs, and cache population against real upstream mirrors.

### Modified Capabilities
- `http-landing-page`: Debian/Ubuntu configuration snippets switch from hardcoded release codenames to `<release>` placeholders.

## Impact

- **New files:** `test/e2e/e2e_test.go`, per-distro shell scripts and repo config templates in `test/e2e/`.
- **Modified files:** `pkg/pkgproxy/landing.go` (snippet placeholders), `Makefile` (new `e2e` target), `CHANGELOG.md`.
- **Dependencies:** Requires `podman` or `docker` on the test host. No new Go dependencies.
- **CI:** The e2e tests are gated and will not run in the standard CI pipeline (`make ci-check`).
