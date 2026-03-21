## Requirements

### Requirement: E2e test framework with container runtime
The project SHALL provide end-to-end tests in `test/e2e/` that start a real pkgproxy process, run Linux distribution containers via a container runtime (podman or docker), and exercise real package managers against real upstream mirrors through the proxy. Tests SHALL be gated behind a `//go:build e2e` build tag and a `make e2e` Makefile target.

#### Scenario: E2e tests do not run during standard test suite
- **WHEN** a developer runs `make test` or `make ci-check`
- **THEN** no e2e tests are executed

#### Scenario: E2e tests run via make target
- **WHEN** a developer runs `make e2e`
- **THEN** the e2e test suite is executed with the `e2e` build tag

### Requirement: Container runtime auto-detection with override
The e2e test SHALL auto-detect the available container runtime by checking for `podman` first, then `docker`. The runtime SHALL be overridable via the `CONTAINER_RUNTIME` environment variable. The host gateway hostname SHALL be set automatically based on the detected runtime: `host.containers.internal` for podman, `host.docker.internal` for docker.

#### Scenario: Podman is auto-detected when available
- **WHEN** `podman` is found on `PATH` and `CONTAINER_RUNTIME` is not set
- **THEN** the tests use `podman` as the container runtime and `host.containers.internal` as the host gateway

#### Scenario: Docker is auto-detected as fallback
- **WHEN** `podman` is not found on `PATH` and `docker` is found and `CONTAINER_RUNTIME` is not set
- **THEN** the tests use `docker` as the container runtime and `host.docker.internal` as the host gateway

#### Scenario: Environment variable overrides auto-detection
- **WHEN** `CONTAINER_RUNTIME=docker` is set
- **THEN** the tests use `docker` regardless of whether `podman` is available

#### Scenario: No container runtime available
- **WHEN** neither `podman` nor `docker` is found and `CONTAINER_RUNTIME` is not set
- **THEN** the test is skipped with a descriptive message

### Requirement: Pkgproxy starts as a host-side subprocess
The e2e test SHALL build and start pkgproxy as a subprocess listening on `0.0.0.0` with a dynamically allocated free port, using the production `configs/pkgproxy.yaml`. The process SHALL be terminated after all subtests complete.

#### Scenario: Pkgproxy binds to a free port
- **WHEN** the e2e test suite starts
- **THEN** pkgproxy is built, started on `0.0.0.0:<free-port>`, and reachable from podman containers via `host.containers.internal:<port>` (podman) or `host.docker.internal:<port>` (docker)

#### Scenario: Pkgproxy is stopped after tests
- **WHEN** all e2e subtests have completed
- **THEN** the pkgproxy subprocess is terminated and the cache directory is cleaned up

### Requirement: Shared cache directory across all distro tests
All distro subtests SHALL share a single temporary cache directory. Each repository writes to its own subdirectory within the cache (this is pkgproxy's default behavior).

#### Scenario: Cache directory is shared
- **WHEN** multiple distro subtests run sequentially
- **THEN** they all use the same cache directory passed to pkgproxy via `--cachedir`

### Requirement: Fedora e2e test
The test suite SHALL include a Fedora subtest using a `fedora:43` container that configures dnf to use pkgproxy for the `fedora` repository, refreshes metadata, and installs a small package (e.g. `tree`). GPG verification SHALL remain enabled.

#### Scenario: Fedora metadata refresh succeeds
- **WHEN** the Fedora container runs `dnf makecache` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Fedora package install succeeds
- **WHEN** the Fedora container runs `dnf install -y tree` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Fedora packages are cached
- **WHEN** the Fedora package install completes
- **THEN** the cache directory contains at least one `.rpm` file under the `fedora/` subdirectory tree (recursive search)

### Requirement: COPR e2e test
The test suite SHALL include a COPR subtest using the same `fedora:43` container that configures an additional dnf repository pointing at `ganto/jo` via the pkgproxy `copr` repository, and installs the `jo` package. GPG verification SHALL be disabled for the COPR repository (`gpgcheck=0`).

#### Scenario: COPR package install succeeds
- **WHEN** the Fedora container runs `dnf install -y jo` from the COPR repository through pkgproxy
- **THEN** the command exits successfully

#### Scenario: COPR packages are cached
- **WHEN** the COPR package install completes
- **THEN** the cache directory contains at least one `.rpm` file under the `copr/` subdirectory tree (recursive search)

### Requirement: Debian e2e test
The test suite SHALL include a Debian subtest using a `debian:trixie` container that configures apt to use pkgproxy for the `debian` and `debian-security` repositories, refreshes metadata, and installs a small package (e.g. `tree`).

#### Scenario: Debian metadata refresh succeeds
- **WHEN** the Debian container runs `apt update` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Debian package install succeeds
- **WHEN** the Debian container runs `apt install -y tree` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Debian packages are cached
- **WHEN** the Debian package install completes
- **THEN** the cache directory contains at least one `.deb` file under the `debian/` subdirectory tree (recursive search)

### Requirement: Arch Linux e2e test
The test suite SHALL include an Arch Linux subtest using an `archlinux:latest` container that configures pacman to use pkgproxy for the `archlinux` repository, refreshes the package database, and installs a small package (e.g. `tree`). GPG verification SHALL remain enabled.

#### Scenario: Arch metadata refresh succeeds
- **WHEN** the Arch container runs `pacman -Sy` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Arch package install succeeds
- **WHEN** the Arch container runs `pacman -S --noconfirm tree` through pkgproxy
- **THEN** the command exits successfully

#### Scenario: Arch packages are cached
- **WHEN** the Arch package install completes
- **THEN** the cache directory contains at least one `.tar.zst` file under the `archlinux/` subdirectory tree (recursive search)

### Requirement: Per-distro-family shell scripts
The e2e tests SHALL use per-distro-family shell scripts (`test-dnf.sh`, `test-apt.sh`, `test-pacman.sh`) located in `test/e2e/` that are mounted into containers. Scripts SHALL accept parameters for proxy address and packages to install.

#### Scenario: Shell script is mounted and executed
- **WHEN** a distro container is started
- **THEN** the corresponding shell script and repo config files are mounted into the container and the script is executed

#### Scenario: DNF script handles both standard and COPR repos
- **WHEN** the Go test generates a `.repo` file with `gpgcheck=0` for COPR
- **THEN** `test-dnf.sh` uses the mounted repo config as-is, without modifying GPG settings

### Requirement: Sequential distro test execution
Distro subtests SHALL run sequentially, not in parallel.

#### Scenario: Tests run one at a time
- **WHEN** the e2e test suite executes
- **THEN** each distro subtest completes before the next one starts
