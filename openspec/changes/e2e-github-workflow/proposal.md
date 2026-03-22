## Why

The e2e tests validate pkgproxy against real package managers and upstream mirrors but can only be run locally today. Adding a GitHub Actions workflow enables on-demand CI validation across all supported distributions, catches regressions from upstream mirror changes, and expands test coverage to distros not yet tested (CentOS Stream, AlmaLinux, Rocky Linux, Ubuntu).

## What Changes

- Add a new GitHub Actions workflow (`.github/workflows/e2e.yaml`) triggered only via `workflow_dispatch`, using a matrix strategy with one job per distro/release tuple and a 5-minute timeout per job.
- Restructure `test/e2e/e2e_test.go` from a single `TestE2E` function with subtests into separate top-level test functions per distro (`TestFedora`, `TestCentOSStream`, `TestAlmaLinux`, `TestRockyLinux`, `TestDebian`, `TestUbuntu`, `TestArch`), each parameterized by `E2E_RELEASE` env var with sensible defaults.
- Add new distro tests: CentOS Stream 10, AlmaLinux 10, Rocky Linux 10, Ubuntu Noble.
- DNF-based distros (except Fedora) include COPR (`epel-$releasever-$basearch` pattern) and EPEL repo metadata tests. Fedora includes COPR only.
- Move `sources.list` generation from `test-apt.sh` into the Go test (consistent with DNF repo file generation). Simplify `test-apt.sh` to just run `apt-get update && apt-get install`.
- Add `ubuntu-security` repository to `configs/pkgproxy.yaml`.
- Update `Makefile` with parameterized `e2e` target supporting optional `DISTRO` and `RELEASE` variables.
- Use fully qualified container image names everywhere (e.g., `docker.io/library/fedora:43`, `quay.io/centos/centos:stream10`).
- Use `$basearch` and `$releasever` variables instead of hardcoded architecture/release values.

## Capabilities

### New Capabilities
- `e2e-github-workflow`: GitHub Actions workflow with matrix strategy for running e2e tests per distro/release tuple on demand.
- `e2e-multi-distro`: Extended e2e test coverage for CentOS Stream, AlmaLinux, Rocky Linux, and Ubuntu, including EPEL and COPR for DNF-based distros.

### Modified Capabilities
- `e2e-testing`: Restructure from single TestE2E into per-distro top-level test functions, parameterize by E2E_RELEASE env var, move apt sources.list generation into Go test, support DISTRO/RELEASE make parameters.

## Impact

- **New file**: `.github/workflows/e2e.yaml`
- **Modified files**: `test/e2e/e2e_test.go`, `test/e2e/test-apt.sh`, `configs/pkgproxy.yaml`, `Makefile`, `README.md`, `CLAUDE.md`
- **New shell scripts**: None (existing scripts reused; `test-dnf.sh` and `test-pacman.sh` unchanged)
- **External dependencies**: Container images from Docker Hub and Quay.io; upstream package mirrors (Fedora, Debian, Ubuntu, Arch, CentOS, AlmaLinux, Rocky)
