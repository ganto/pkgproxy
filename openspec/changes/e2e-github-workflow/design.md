## Context

pkgproxy has a working e2e test suite (`test/e2e/e2e_test.go`) that validates proxying for Fedora, COPR, Debian, and Arch Linux using real containers and upstream mirrors. Currently these tests can only be run locally via `make e2e`. The test is structured as a single `TestE2E` function with subtests, sharing one pkgproxy instance.

The goal is to run these tests in GitHub Actions on demand, expand coverage to more distributions, and expose results via a matrix strategy for clear per-distro visibility.

## Goals / Non-Goals

**Goals:**
- Run e2e tests in GitHub Actions via `workflow_dispatch` with a matrix strategy (one job per distro/release tuple)
- Add e2e coverage for CentOS Stream 10, AlmaLinux 10, Rocky Linux 10, and Ubuntu Noble
- Add EPEL metadata testing for non-Fedora DNF distros
- Add COPR testing for all DNF-based distros
- Parameterize the `make e2e` target with optional `DISTRO` and `RELEASE` variables
- Add `ubuntu-security` repository to pkgproxy config

**Non-Goals:**
- Running e2e tests automatically on push or PR (manual trigger only)
- Testing multiple releases of the same distro in the initial matrix (infrastructure supports it, but not needed yet)
- Caching container images between workflow runs
- EPEL package installation testing (metadata-only, like debian-security)

## Decisions

### 1. One top-level test function per distro instead of subtests under TestE2E

Each distro gets its own top-level test function (`TestFedora`, `TestDebian`, etc.) rather than subtests under a shared `TestE2E`. This enables clean `-run` filtering from the matrix without regex escaping issues, and makes each distro test self-contained with its own setup (repo files, COPR, EPEL).

**Alternative considered:** Keeping `TestE2E` with subtests and filtering via `-run TestE2E/Fedora`. This works but couples all distros to a shared function and makes adding distro-specific setup (EPEL, different repo file patterns) messier.

### 2. E2E_RELEASE env var for release parameterization

Each test function reads `E2E_RELEASE` to determine the release version (e.g., `43` for Fedora, `trixie` for Debian). If unset, a sensible default is used. The matrix passes the release, and local `make e2e` works without any env vars.

**Alternative considered:** Separate test functions per release (e.g., `TestFedora42`, `TestFedora43`). This doesn't scale and duplicates code.

### 3. Inline matrix with explicit include entries

The workflow matrix uses `include` with explicit objects rather than a cross-product strategy. Each entry specifies the display name, test function, and release. This is verbose but immediately readable and avoids unexpected combinations.

### 4. Sources.list generation moves from shell script to Go test

For apt-based distros, the Go test generates the `sources.list` file and mounts it into the container, consistent with how DNF repo files are already handled. The `test-apt.sh` script is simplified to just `apt-get update && apt-get install`. This allows the Go test to control repo prefixes (`debian` vs `ubuntu`) and security repo inclusion per distro.

### 5. EPEL handled via additional repo file mount, metadata-only validation

For CentOS Stream, AlmaLinux, and Rocky Linux, the Go test mounts an additional EPEL `.repo` file. The existing `dnf makecache` in `test-dnf.sh` validates that EPEL metadata is accessible. No separate package install or cache assertion is needed for EPEL (mirrors metadata, not packages).

### 6. COPR for non-Fedora DNF distros uses epel-$releasever-$basearch pattern

The COPR repo URL pattern for non-Fedora distros is `/copr/ganto/jo/epel-$releasever-$basearch/` (vs `/copr/ganto/jo/fedora-$releasever-$basearch/` for Fedora).

### 7. Fully qualified container image names

All container images use fully qualified names (`docker.io/library/fedora:43`, `quay.io/centos/centos:stream10`) to avoid dependency on container runtime defaults. This ensures consistent behavior across podman and docker.

### 8. TestMain builds pkgproxy binary once for all test functions

A `TestMain` function in the e2e package builds the pkgproxy binary once into a shared temporary directory and stores the path in a package-level variable. Each top-level test function reuses this pre-built binary when starting its own pkgproxy process (with its own port, cache directory, and lifecycle). This avoids rebuilding the binary 7 times when running `make e2e` locally, and costs nothing in the GitHub Actions matrix (where each job runs only one test function).

**Alternative considered:** Building the binary in each test function (the current approach for `TestE2E`). This is simpler but wasteful when running all tests locally.

### 9. $basearch and $releasever instead of hardcoded values

Repo files use DNF/YUM variables (`$basearch`, `$releasever`) rather than hardcoded `x86_64` or version numbers. This allows the same tests to work on ARM64 (e.g., developer MacBooks) without modification.

## Risks / Trade-offs

- **Upstream mirror availability** → Tests depend on real upstream mirrors. Transient mirror failures will cause test failures. Mitigation: manual trigger only, so failures don't block CI. pkgproxy's retry configuration helps for mirrors that support it.
- **Container image pull rate limits** → Docker Hub rate limits may cause occasional failures. Mitigation: with only 7 matrix jobs on manual trigger, this is unlikely. Can add `docker login` step later if needed.
- **CentOS Stream URL structure uncertainty** → The exact mirror path for CentOS Stream 10 (`$releasever-stream/BaseOS/...`) needs validation. Mitigation: this is exactly what the e2e test will catch.
- **5-minute timeout per job** → Container image pulls + package installs on slow networks could exceed this. Mitigation: GitHub Actions runners have fast network; timeout can be bumped if needed.
