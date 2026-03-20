## Context

Ko builds container images using Chainguard's `cgr.dev/chainguard/static` as the default base image. This base image ships with its own OCI labels and annotations (e.g., `org.opencontainers.image.vendor=Chainguard`, `org.opencontainers.image.title=static`, Chainguard source URLs). Ko applies labels via direct map assignment, so `--image-label` does override matching base image label keys. The current `ko build` invocation already sets eight labels correctly, but `org.opencontainers.image.authors` and `org.opencontainers.image.url` are not set, allowing the Chainguard base image values for those keys to pass through. For annotations, the same map-based override applies, but the current invocation only sets `source` and `revision`, leaving `created`, `title`, `vendor`, `url`, and others pointing at Chainguard values.

Additionally, the current build produces only `linux/amd64` images. ARM64 usage is growing (Raspberry Pi servers, ARM cloud instances), and Ko natively supports multi-platform builds.

## Goals / Non-Goals

**Goals:**
- Ensure all OCI labels on the published image reflect pkgproxy metadata, overriding any Chainguard base image defaults
- Ensure all OCI manifest annotations reflect pkgproxy metadata, overriding any Chainguard base image defaults
- Produce multi-architecture images (`linux/amd64` + `linux/arm64`) as a manifest list
- Centralize the `ko build` invocation in the Makefile so it can be tested locally and reused by CI workflows

**Non-Goals:**
- Replacing the Chainguard base image with a different base image
- Supporting architectures beyond amd64 and arm64
- Modifying the cosign signing workflow — `cosign sign --yes "${IMAGE}"` where `IMAGE` is the manifest list digest returned by `ko build` signs the OCI image index as a whole, which is the standard pattern and requires no changes to the existing invocation

## Decisions

### 1. Use `ko build --image-label` to override base image labels
**Decision**: Continue using `--image-label` flags, which override base image labels with the same key.

**Rationale**: Ko applies labels via direct map assignment (`cfg.Config.Labels[k] = v`), so any key supplied via `--image-label` unconditionally replaces the value for that key inherited from the base image. All ten pkgproxy labels will overwrite any matching Chainguard values. If the base image introduces new label keys we do not explicitly set, those will pass through — this is acceptable since all ten standard OCI keys are covered.

**Alternatives considered**:
- Custom Dockerfile with explicit `LABEL` directives — adds build complexity, loses Ko's advantages

### 2. Mirror all labels as annotations for full consistency
**Decision**: Every `--image-label` flag in the `ko build` invocation SHALL have a corresponding `--image-annotation` flag with the same key and value.

**Rationale**: Ko applies annotations via the same map-based assignment as labels, so `--image-annotation` likewise overwrites matching annotation keys inherited from the base image. Chainguard's base image sets its own annotations (e.g., `org.opencontainers.image.created`, `org.opencontainers.image.title`, `org.opencontainers.image.vendor`, `org.opencontainers.image.url`). Without explicit annotation overrides for these same keys, the base image annotations persist on the final manifest and conflict with the pkgproxy label values. Full label/annotation parity eliminates this inconsistency and ensures tools that inspect either labels or annotations see coherent pkgproxy-specific metadata.

**Alternatives considered**:
- Override only `source` and `revision` annotations (current state) — leaves `created`, `title`, `vendor`, and `url` annotations pointing at Chainguard, which contradicts the pkgproxy labels

### 3. Control target platforms via an `ARCHS` Makefile variable
**Decision**: Define `ARCHS ?= $(shell go env GOARCH)` in the Makefile. The `image-build` target passes `--platform` to `ko build` by prefixing `linux/` to each entry in `ARCHS`. CI workflows set `ARCHS=amd64,arm64` via environment to produce a multi-platform image index; local builds default to the native host architecture. `.ko.yaml` does not define `defaultPlatforms`.

**Rationale**: Local container daemons (including podman) do not support loading multi-arch OCI image indexes via the Docker socket API — they only accept single-platform images. Building both platforms locally would either fail at load time or require a local registry. Using the native architecture as the local default keeps `make image-build` fast and functional for local development without special setup. CI, which pushes to a real registry, is the right place to build and publish the multi-arch manifest list. The `ARCHS` variable follows the same `?=` override pattern used for `VERSION`, `REVISION`, and `BUILD_DATE`, making the CI override consistent with the rest of the Makefile.

**Alternatives considered**:
- `defaultPlatforms` in `.ko.yaml` — would build both platforms locally on every `make image-build`, causing load failures with local container daemons and unnecessary cross-compilation overhead for developers

### 4. Centralize `ko build` in a Makefile target
**Decision**: Add an `image-build` Make target that encapsulates the full `ko build` command with all `--image-label` and `--image-annotation` flags. GitHub Actions workflows call `make image-build` instead of inlining the command. The target uses Makefile variables `VERSION`, `REVISION` (full git SHA), and `BUILD_DATE` for dynamic metadata, and accepts an optional `IMAGE_TAGS` variable for release tagging. The `source` and `url` labels/annotations are derived from the `SOURCE_URL` environment variable and are only included when `SOURCE_URL` is set. The target writes the image reference to `image-ref.out` in the repository root and also prints it to stdout. This allows the target to use `$(info ...)` banners like all other Makefile targets, since workflows read the image reference from the file rather than capturing stdout. `image-ref.out` is added to `.gitignore`.

**Rationale**: The current approach duplicates the `ko build` invocation across `publish.yaml` and `release.yaml`, making it hard to test locally and error-prone to keep in sync. A Makefile target provides a single source of truth that developers can run locally (`make image-build`) with sensible defaults (e.g., `KO_DOCKER_REPO=ko.local`) and CI can invoke with the appropriate registry and tags. As part of this change, the existing Makefile variables `COMMIT` and `DATE` are renamed to `REVISION` (using `git rev-parse HEAD` for the full SHA) and `BUILD_DATE` respectively, aligning the Makefile with `.ko.yaml` ldflags and GitHub Actions variable names. The `LDFLAGS` definition and all targets referencing these variables (`ci-build`) are updated accordingly. `VERSION`, `REVISION`, and `BUILD_DATE` are defined with `?=` so that values injected by CI via `$GITHUB_ENV` take precedence over the Makefile defaults. `SOURCE_URL` is provided by CI via environment variable; when unset (e.g., local builds), the `source` and `url` labels/annotations are omitted since they are only meaningful for images pushed to a registry.

**Alternatives considered**:
- Shell script wrapping `ko build` — adds another file; the Makefile already serves as the project's build entry point
- Keep `ko build` inline in workflows — current state; duplicated, untestable locally without copy-pasting the command

## Risks / Trade-offs

- **[Build time increase]** Multi-platform builds compile Go for two architectures, roughly doubling build time. → Acceptable trade-off for broader platform support; CI runners have sufficient capacity.
- **[Base image label/annotation drift]** If Chainguard adds new labels or annotations in future base image updates, they won't be overridden automatically. → Mitigated by explicit overrides for all standard OCI keys plus `authors` and `url`. Periodic review is sufficient.
- **[arm64 testing gap]** CI runs on amd64 only; arm64 images won't be integration-tested in CI. → Go cross-compilation is reliable; the binary is statically linked with CGO disabled, minimizing platform-specific issues.
- **[Missing `source`/`url` in local builds]** When `SOURCE_URL` is not set, the `source` and `url` labels/annotations are omitted, so local builds will not fully match CI builds. → Acceptable; these labels point to the GitHub repository and are only meaningful for published images.
