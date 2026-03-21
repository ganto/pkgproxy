## 1. Configuration

- [ ] 1.1 Add `ubuntu-security` repository to `configs/pkgproxy.yaml` with `.deb` suffix and `https://security.ubuntu.com/ubuntu/` mirror

## 2. Client config updates

- [ ] 2.1 Update Ubuntu client config section in `README.md` to point `noble-security` at `/ubuntu-security` instead of `/ubuntu`
- [ ] 2.2 Update Ubuntu landing page snippet to point `noble-security` at `/ubuntu-security` instead of `/ubuntu`

## 3. Test restructuring

- [ ] 3.1 Refactor `test/e2e/e2e_test.go`: add `TestMain` that builds pkgproxy binary once into a shared temp directory and stores path in a package-level variable. Extract shared helpers (container runtime detection, port allocation, pkgproxy start from pre-built binary, cache assertion, container run) and add `E2E_RELEASE` env var reading with default fallback helper
- [ ] 3.2 Convert existing `TestE2E/Fedora` subtest into top-level `TestFedora` function with its own pkgproxy instance, cache dir, and default release `43`. Include COPR subtest using `fedora-$releasever-$basearch` pattern
- [ ] 3.3 Convert existing `TestE2E/Debian` subtest into top-level `TestDebian` function. Generate `sources.list` in Go test (pointing at `/debian` and `/debian-security`), mount it into the container. Default release `trixie`
- [ ] 3.4 Convert existing `TestE2E/Arch` subtest into top-level `TestArch` function. No `E2E_RELEASE` support (rolling release). Use fully qualified image `docker.io/library/archlinux:latest`
- [ ] 3.5 Remove the old `TestE2E` function and `TestE2E/COPR` subtest (COPR is now part of each DNF distro test)

## 4. New distro tests

- [ ] 4.1 Add `TestCentOSStream` function using `quay.io/centos/centos:stream<release>` image, default release `10`. Include base repo (`centos-stream`), EPEL repo, and COPR subtest (`epel-$releasever-$basearch` pattern)
- [ ] 4.2 Add `TestAlmaLinux` function using `docker.io/library/almalinux:<release>` image, default release `10`. Include base repo (`almalinux`), EPEL repo, and COPR subtest
- [ ] 4.3 Add `TestRockyLinux` function using `docker.io/library/rockylinux:<release>` image, default release `10`. Include base repo (`rockylinux`), EPEL repo, and COPR subtest
- [ ] 4.4 Add `TestUbuntu` function using `docker.io/library/ubuntu:<release>` image, default release `noble`. Generate `sources.list` with `/ubuntu` and `/ubuntu-security` repos

## 5. Shell script changes

- [ ] 5.1 Remove `sources.list` generation from `test/e2e/test-apt.sh` (now handled by Go test). Retain `sources.list.d/` cleanup, `DEBIAN_FRONTEND=noninteractive`, and `apt-get update`/`apt-get install`. Accept packages as arguments

## 6. Makefile

- [ ] 6.1 Update `make e2e` target to support optional `DISTRO` and `RELEASE` parameters. Convert `DISTRO` value to test function name via `-run`. Pass `RELEASE` as `E2E_RELEASE` env var. Set 15-minute timeout

## 7. GitHub Actions workflow

- [ ] 7.1 Create `.github/workflows/e2e.yaml` with `workflow_dispatch` trigger, matrix strategy with 7 entries (fedora/43, centos-stream/10, almalinux/10, rockylinux/10, debian/trixie, ubuntu/noble, archlinux/latest), 5-minute timeout per job, checkout + setup-go + go test steps

## 8. Fully qualified image names

- [ ] 8.1 Update all existing container image references in `TestFedora` and `TestDebian` to use fully qualified names (`docker.io/library/...`)

## 9. Documentation

- [ ] 9.1 Update `README.md`: add a "Testing" or "Development" section documenting `make e2e` usage (with `DISTRO`/`RELEASE` parameters) and note that e2e tests should be extended when adding support for new Linux distributions
- [ ] 9.2 Update `CLAUDE.md`: add `make e2e` to the Commands section, add a rule that e2e tests must pass before a feature is considered complete, add a rule that adding support for a new Linux distribution requires adding corresponding e2e tests, and add a rule that changes to client config snippets (sources.list, .repo files) must be replicated in the landing page snippets and in README.md

## 10. Validation

- [ ] 10.1 Update `CHANGELOG.md` with the changes in the `[Unreleased]` section
- [ ] 10.2 Run `make ci-check` to verify no regressions in unit tests or linting
