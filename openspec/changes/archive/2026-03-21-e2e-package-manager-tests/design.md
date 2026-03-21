## Context

pkgproxy has comprehensive unit and integration tests using `httptest.NewServer` mock upstreams, but no tests exercise the full stack against real mirrors with real package managers. Mirror URL structures change across distro releases, and landing page client configuration snippets can drift from reality without anyone noticing until a user reports it.

The production config (`configs/pkgproxy.yaml`) defines 11 repositories across RPM, DEB, and Arch ecosystems. The landing page (`pkg/pkgproxy/landing.go`) generates client configuration snippets, some of which hardcode release codenames (Debian: `bullseye`, Ubuntu: `jammy`) that are already stale.

## Goals / Non-Goals

**Goals:**
- Validate that real package managers (dnf, apt, pacman) can fetch metadata and install packages through pkgproxy using the production mirror configuration.
- Verify that pkgproxy caches package files (`.rpm`, `.deb`, `.tar.zst`) to disk during real downloads.
- Catch stale or broken mirror URLs in `configs/pkgproxy.yaml`.
- Keep landing page snippets, e2e test configs, and README documentation in sync via release-agnostic placeholders.

**Non-Goals:**
- Testing all 11 configured repositories — start with fedora, copr, debian, archlinux.
- Running e2e tests in CI — these require network access and a container runtime (podman or docker); they are developer-local only.
- Building pkgproxy into a container image for testing — the binary runs on the host.
- Testing concurrent cache write safety — already covered by unit tests and naturally exercised by package manager parallelism.

## Decisions

### 1. Host-side pkgproxy process + container runtime distro containers

**Decision:** Start pkgproxy as a subprocess (`go build` + exec) listening on `0.0.0.0:<free-port>`. Run distro containers via the available container runtime, reaching the proxy at `host.containers.internal:<port>` (podman) or `host.docker.internal:<port>` (docker).

The test auto-detects the container runtime: it checks for `podman` first, then `docker`. This can be overridden via the `CONTAINER_RUNTIME` environment variable (e.g. `CONTAINER_RUNTIME=docker make e2e`). The host gateway hostname is set automatically based on the detected runtime.

**Alternatives considered:**
- *Container-to-container with podman network:* More complex networking setup, no additional value over host gateway DNS.
- *Build pkgproxy into a container and test container-to-container:* Couples e2e tests to container build pipeline, slower feedback loop.
- *Podman-only:* Simpler but excludes contributors/environments where only Docker is available.

**Rationale:** Simplest setup. Both podman and docker support host gateway DNS names. Avoids container image build dependency. Auto-detection with override keeps the default frictionless while allowing explicit control.

### 2. Real package managers in real distro containers

**Decision:** Use `fedora:43`, `debian:trixie`, and `archlinux:latest` container images. Run actual `dnf`, `apt`, and `pacman` commands to refresh metadata and install a small package.

**Alternatives considered:**
- *Replay HTTP request patterns with curl/Go client:* Would miss package manager quirks (parallel downloads, GPG verification, redirect handling).
- *Mock upstreams in containers:* Defeats the purpose of testing real mirrors.

**Rationale:** The goal is to validate the real request patterns of real package managers against real mirrors. Only running the actual tools achieves this.

### 3. Per-distro-family shell scripts mounted into containers

**Decision:** Write short shell scripts (`test-dnf.sh`, `test-apt.sh`, `test-pacman.sh`) in `test/e2e/` that accept parameters (proxy address, release, packages). The Go test writes repo config files to a temp dir, mounts both the script and configs into the container. The `test-apt.sh` script configures a complete `sources.list` with both `debian` and `debian-security` entries in a single invocation, since `apt update` naturally hits all configured repos.

**Alternatives considered:**
- *Inline podman run commands:* Harder to read, debug, and maintain.
- *Custom Containerfiles per distro:* Adds build step, slower iteration.

**Rationale:** Scripts are easy to debug manually (`podman run ... bash /test-dnf.sh ...`), and no image build is needed.

### 4. Shared cache directory across all distro tests

**Decision:** All distro tests share a single cache directory (a temp dir created once per test run). Each pkgproxy repository writes to its own subdirectory naturally.

**Alternatives considered:**
- *Separate cache dir per test:* Cleaner isolation but hides real cross-repo interactions.

**Rationale:** Mirrors real-world usage. If shared cache causes issues, it would also cause issues in production. Sequential execution avoids race conditions.

### 5. Sequential test execution

**Decision:** Distro tests run sequentially within a single Go test function (subtests).

**Rationale:** Network bandwidth is the bottleneck, not CPU. Sequential execution simplifies debugging. Package managers already introduce internal parallelism (multi-package downloads).

### 6. GPG enabled except for COPR

**Decision:** Keep GPG signature verification enabled for fedora, debian, and archlinux (keys ship with base images). Disable for COPR (`gpgcheck=0`) since the COPR GPG key requires a separate fetch.

**Rationale:** GPG verification proves pkgproxy doesn't corrupt packages or strip headers. COPR key setup adds complexity with little additional value since the RPM flow is already tested via fedora.

### 7. Release-agnostic landing page snippets

**Decision:** Change Debian/Ubuntu snippets in `landing.go` from hardcoded codenames (`bullseye`, `jammy`) to `<release>` placeholders, matching the existing COPR pattern (`<user>/<repo>`). RPM-based snippets already use `$releasever`. The README keeps concrete examples (updated from `bullseye` to `trixie` for Debian, from `jammy` to current Ubuntu release) with a note to substitute the actual codename.

**Rationale:** Eliminates snippet staleness in the landing page. E2e tests substitute concrete releases; users substitute their own. The README retains concrete examples for readability. The landing page snippets match the README in *structure* but use placeholders where the README uses concrete codenames.

### 8. Build tag gating

**Decision:** Use `//go:build e2e` build tag. Add `make e2e` target that runs `go test -tags e2e ./test/e2e/`.

**Rationale:** Prevents e2e tests from running during `make test` or `make ci-check`. Explicit opt-in via `make e2e` or `-tags e2e`.

## Risks / Trade-offs

- **[Network dependency]** Tests require internet access to reach real mirrors. → Mitigation: Tests are gated behind build tag, never run in CI. Failures due to transient network issues are expected and acceptable.
- **[Mirror availability]** Upstream mirrors may be temporarily down or change URL structure. → Mitigation: Multiple mirrors per repo in config provide failover. Test failures signal real configuration issues that need attention.
- **[Container image pulls]** First run pulls multi-hundred-MB distro images. → Mitigation: Container runtime caches images locally. Document this in test README/comments.
- **[Host gateway DNS support]** `host.containers.internal` (podman) or `host.docker.internal` (docker) may not work in all configurations. → Mitigation: Document prerequisites. Note that `--network=host` is not a viable fallback as it requires elevated privileges.
- **[COPR repo stability]** `ganto/jo` COPR repo could be deleted or renamed. → Mitigation: Repo is maintained by the project owner; test will clearly fail pointing to the cause.
- **[Distro version pinning]** `fedora:43` will eventually become unsupported. → Mitigation: Version numbers are explicit constants in the test code, easy to bump.
