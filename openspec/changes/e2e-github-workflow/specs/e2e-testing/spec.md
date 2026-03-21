## MODIFIED Requirements

### Requirement: E2e test framework with container runtime
The project SHALL provide end-to-end tests in `test/e2e/` that start a real pkgproxy process, run Linux distribution containers via a container runtime (podman or docker), and exercise real package managers against real upstream mirrors through the proxy. Tests SHALL be gated behind a `//go:build e2e` build tag and a `make e2e` Makefile target. Each distribution SHALL have its own top-level test function (e.g., `TestFedora`, `TestDebian`) instead of subtests under a single `TestE2E` function.

#### Scenario: E2e tests do not run during standard test suite
- **WHEN** a developer runs `make test` or `make ci-check`
- **THEN** no e2e tests are executed

#### Scenario: E2e tests run via make target
- **WHEN** a developer runs `make e2e`
- **THEN** all e2e test functions are executed with the `e2e` build tag and default release versions

#### Scenario: E2e tests run for a specific distro
- **WHEN** a developer runs `make e2e DISTRO=fedora`
- **THEN** only `TestFedora` is executed with the default release version

#### Scenario: E2e tests run for a specific distro and release
- **WHEN** a developer runs `make e2e DISTRO=fedora RELEASE=42`
- **THEN** only `TestFedora` is executed with `E2E_RELEASE=42`

### Requirement: Pkgproxy binary built once via TestMain
The e2e test package SHALL use a `TestMain` function to build the pkgproxy binary once into a shared temporary directory before any test functions run. The binary path SHALL be stored in a package-level variable. Each top-level test function SHALL reuse this pre-built binary when starting its own pkgproxy process. The temporary directory SHALL be cleaned up after all tests complete.

#### Scenario: Binary is built once for all tests
- **WHEN** `go test -tags e2e ./test/e2e/` runs multiple test functions
- **THEN** the pkgproxy binary is built exactly once via `TestMain`

#### Scenario: Each test function starts its own pkgproxy process
- **WHEN** `TestFedora` and `TestDebian` both run
- **THEN** each starts a separate pkgproxy process from the shared binary, with its own port and cache directory

### Requirement: E2E_RELEASE environment variable for release parameterization
Each top-level e2e test function SHALL read the `E2E_RELEASE` environment variable to determine the distribution release version. If `E2E_RELEASE` is not set, the test SHALL use a sensible default release (e.g., `43` for Fedora, `trixie` for Debian, `noble` for Ubuntu, `10` for CentOS Stream/AlmaLinux/Rocky Linux).

#### Scenario: Default release is used when E2E_RELEASE is unset
- **WHEN** `TestFedora` runs without `E2E_RELEASE` set
- **THEN** the test uses the default release `43`

#### Scenario: E2E_RELEASE overrides the default
- **WHEN** `TestFedora` runs with `E2E_RELEASE=42`
- **THEN** the test uses release `42` for the container image and repo configuration

### Requirement: Per-distro-family shell scripts
The e2e tests SHALL use per-distro-family shell scripts (`test-dnf.sh`, `test-pacman.sh`) located in `test/e2e/` that are mounted into containers. The `test-apt.sh` script SHALL have its `sources.list` generation removed (handled by the Go test instead), but SHALL retain cleanup of `sources.list.d/` entries, setting `DEBIAN_FRONTEND=noninteractive`, and running `apt-get update` and `apt-get install`. DNF repo file generation SHALL remain in the Go test.

#### Scenario: apt script receives pre-generated sources.list
- **WHEN** a Debian or Ubuntu container is started
- **THEN** the Go test generates and mounts the `sources.list` file, and `test-apt.sh` removes interfering `sources.list.d/` entries, sets `DEBIAN_FRONTEND=noninteractive`, and runs `apt-get update` and `apt-get install`

#### Scenario: DNF script handles both standard and COPR repos
- **WHEN** the Go test generates a `.repo` file with `gpgcheck=0` for COPR
- **THEN** `test-dnf.sh` uses the mounted repo config as-is, without modifying GPG settings

### Requirement: Makefile e2e target with DISTRO and RELEASE parameters
The `make e2e` target SHALL accept optional `DISTRO` and `RELEASE` parameters. When `DISTRO` is set, it SHALL be converted to the corresponding test function name and passed via `-run`. When `RELEASE` is set, it SHALL be passed as the `E2E_RELEASE` environment variable. The target SHALL use a 15-minute timeout.

#### Scenario: make e2e without parameters runs all tests
- **WHEN** a developer runs `make e2e`
- **THEN** all e2e test functions run with default releases and a 15-minute timeout

#### Scenario: make e2e with DISTRO filters to one test
- **WHEN** a developer runs `make e2e DISTRO=fedora`
- **THEN** only `TestFedora` runs

#### Scenario: make e2e with DISTRO and RELEASE
- **WHEN** a developer runs `make e2e DISTRO=fedora RELEASE=42`
- **THEN** only `TestFedora` runs with `E2E_RELEASE=42`

### Requirement: Fedora e2e test
The test suite SHALL include a top-level `TestFedora` test function using a `docker.io/library/fedora:<release>` container (where release defaults to `43` or is set via `E2E_RELEASE`). The test SHALL configure dnf to use pkgproxy for the `fedora` repository and include a COPR subtest for `ganto/jo` using the `fedora-$releasever-$basearch` pattern. GPG verification SHALL remain enabled for the base repo, disabled for COPR.

#### Scenario: Fedora metadata refresh succeeds
- **WHEN** the Fedora container runs `dnf makecache` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Fedora package install succeeds
- **WHEN** the Fedora container runs `dnf install -y tree` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Fedora packages are cached
- **WHEN** the Fedora package install completes
- **THEN** the cache directory contains at least one `.rpm` file under the `fedora/` subdirectory tree

#### Scenario: Fedora COPR package install succeeds
- **WHEN** the Fedora container runs `dnf install -y jo` from the COPR repository through pkgproxy
- **THEN** the command exits successfully and cached `.rpm` files exist under the `copr/` cache subdirectory

### Requirement: Debian e2e test
The test suite SHALL include a top-level `TestDebian` test function using a `docker.io/library/debian:<release>` container (where release defaults to `trixie` or is set via `E2E_RELEASE`). The test SHALL generate a `sources.list` in the Go test pointing at pkgproxy for the `debian` and `debian-security` repositories, mount it into the container, and install a small package.

#### Scenario: Debian metadata refresh succeeds
- **WHEN** the Debian container runs `apt-get update` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Debian package install succeeds
- **WHEN** the Debian container runs `apt-get install -y tree` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Debian packages are cached
- **WHEN** the Debian package install completes
- **THEN** the cache directory contains at least one `.deb` file under the `debian/` subdirectory tree

### Requirement: Arch Linux e2e test
The test suite SHALL include a top-level `TestArch` test function using a `docker.io/library/archlinux:latest` container. The Arch Linux test SHALL NOT use `E2E_RELEASE` as Arch is a rolling release. GPG verification SHALL remain enabled.

#### Scenario: Arch metadata refresh succeeds
- **WHEN** the Arch container runs `pacman -Sy` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Arch package install succeeds
- **WHEN** the Arch container runs `pacman -S --noconfirm tree` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Arch packages are cached
- **WHEN** the Arch package install completes
- **THEN** the cache directory contains at least one `.tar.zst` file under the `archlinux/` subdirectory tree

### Requirement: COPR e2e test
The COPR test SHALL be a subtest within each DNF-based distro's top-level test function rather than a standalone test. For Fedora, the COPR URL pattern SHALL be `/copr/ganto/jo/fedora-$releasever-$basearch/`. For non-Fedora DNF distros, the pattern SHALL be `/copr/ganto/jo/epel-$releasever-$basearch/`. GPG verification SHALL be disabled for COPR (`gpgcheck=0`).

#### Scenario: COPR package install succeeds within Fedora test
- **WHEN** `TestFedora` runs the COPR subtest
- **THEN** `jo` is installed from the COPR repository and cached `.rpm` files exist under the `copr/` subdirectory

#### Scenario: COPR package install succeeds within non-Fedora DNF test
- **WHEN** `TestAlmaLinux` runs the COPR subtest
- **THEN** `jo` is installed from the COPR repository using the `epel-$releasever-$basearch` URL pattern

## ADDED Requirements

### Requirement: E2e test documentation in README.md
The `README.md` SHALL document how to run e2e tests via `make e2e` including the optional `DISTRO` and `RELEASE` parameters. It SHALL note that e2e tests should be extended when adding support for new Linux distributions.

#### Scenario: Developer finds e2e test instructions in README
- **WHEN** a developer reads the README.md
- **THEN** they find instructions for running `make e2e`, `make e2e DISTRO=fedora`, and `make e2e DISTRO=fedora RELEASE=42`

#### Scenario: README reminds to add e2e tests for new distros
- **WHEN** a developer reads the e2e testing section in README.md
- **THEN** they find guidance that adding support for a new Linux distribution requires adding corresponding e2e tests

### Requirement: E2e test rules in CLAUDE.md
The `CLAUDE.md` SHALL list `make e2e` in the Commands section. The Rules section SHALL state that e2e tests must pass before a feature is considered complete. The Rules section SHALL state that adding support for a new Linux distribution requires adding corresponding e2e tests. The Rules section SHALL state that changes to client configuration snippets (e.g. `sources.list`, `.repo` files) must be replicated in the landing page snippets and in `README.md`.

#### Scenario: CLAUDE.md commands include e2e
- **WHEN** Claude Code reads CLAUDE.md
- **THEN** `make e2e` is listed in the Commands section with usage examples

#### Scenario: CLAUDE.md rules require e2e completion
- **WHEN** Claude Code checks the Rules section
- **THEN** it finds that e2e tests must pass before a feature is considered complete

#### Scenario: CLAUDE.md rules require e2e for new distros
- **WHEN** Claude Code checks the Rules section
- **THEN** it finds that adding a new Linux distribution requires adding corresponding e2e tests

#### Scenario: CLAUDE.md rules require client config consistency
- **WHEN** Claude Code checks the Rules section
- **THEN** it finds that changes to client configuration snippets (sources.list, .repo files) must be replicated in the landing page snippets and in README.md

## REMOVED Requirements

### Requirement: Sequential distro test execution
**Reason**: With separate top-level test functions per distro, sequential execution is no longer enforced at the test level. Each test function manages its own pkgproxy instance. In GitHub Actions, tests run in parallel via the matrix strategy. Locally, `go test` runs top-level test functions sequentially by default.
**Migration**: No migration needed. Each top-level test function is self-contained with its own pkgproxy instance and cache directory.

### Requirement: Shared cache directory across all distro tests
**Reason**: With separate top-level test functions, each test manages its own pkgproxy process and cache directory. The pkgproxy binary is shared (built once in `TestMain`), but each test function has its own process, port, and cache.
**Migration**: Each test function creates its own temporary cache directory via `t.TempDir()`.
