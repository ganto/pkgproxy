## Why

Ko uses a Chainguard `cgr.dev/chainguard/static` base image by default, which embeds its own OCI labels and annotations (e.g., vendor, description, source URL) that refer to the Chainguard project rather than pkgproxy. These base image metadata values leak through into the final published image, causing confusion when inspecting the container. Additionally, the current build only produces an `amd64` image, but users running on ARM-based infrastructure (e.g., Raspberry Pi, ARM cloud instances) need an `arm64` variant.

## What Changes

- Add the two missing OCI labels `authors` and `url` to the `ko build` invocation; the eight existing labels (`source`, `revision`, `version`, `created`, `title`, `vendor`, `description`, `licenses`) already override Chainguard base image values correctly
- Mirror all OCI labels as OCI manifest annotations so that annotations are fully consistent with labels; this prevents Chainguard base image annotations for `created`, `title`, `vendor`, `authors`, and `url` from conflicting with pkgproxy label values
- Enable multi-platform builds (`linux/amd64` + `linux/arm64`) controlled via the `ARCHS` Makefile variable; local builds default to the native host architecture, CI sets `ARCHS=amd64,arm64`
- Refactor the `ko build` invocation into a Makefile target (`image-build`) so the same build logic is shared between GitHub Actions workflows and local development; workflows call `make image-build` instead of inlining the `ko build` command
- Rename Makefile variables `COMMIT` → `REVISION` (full git SHA via `git rev-parse HEAD`) and `DATE` → `BUILD_DATE` to align with `.ko.yaml` ldflags and GitHub Actions variable names; update `LDFLAGS` and all targets that reference these variables

## Capabilities

### New Capabilities
- `multi-arch-build`: Build and publish multi-platform container images (amd64 + arm64)
- `makefile-image-build`: Makefile `image-build` target wrapping `ko build` with all labels, annotations, and metadata, usable both locally and in CI

### Modified Capabilities
- `image-labels`: Override Chainguard base image labels so all ten OCI labels reflect pkgproxy metadata, not the base image defaults
- `image-annotations`: Override Chainguard base image annotations so all manifest annotations reflect pkgproxy metadata, not the base image defaults

## Impact

- `Makefile` — Rename `COMMIT` → `REVISION` (full SHA) and `DATE` → `BUILD_DATE`; update `LDFLAGS` and the `ci-build` target referencing these variables; `VERSION`, `REVISION`, and `BUILD_DATE` defined with `?=` so CI-injected values take precedence; new `image-build` target encapsulating the full `ko build` invocation with all labels, annotations, and metadata variables; accepts optional `IMAGE_TAGS` for release tagging; writes image reference to `image-ref.out`
- `.gitignore` — Add `image-ref.out`
- `.github/workflows/publish.yaml` — Inline `ko build` replaced with `make image-build`; cosign reads image reference from `image-ref.out`; build metadata computation retained in workflow
- `.github/workflows/release.yaml` — Inline `ko build` replaced with `make image-build IMAGE_TAGS=${GITHUB_REF_NAME}`; cosign reads image reference from `image-ref.out`; build metadata computation retained in workflow
- `Makefile` (additional) — `ARCHS ?= $(shell go env GOARCH)` defaults to the native host architecture; CI sets `ARCHS=amd64,arm64` to produce a multi-platform image index; `.ko.yaml` is not changed
- Published container images will change from single-arch (`amd64`) to multi-arch (`amd64` + `arm64`) manifest lists
- Existing image pull workflows are unaffected (the container runtime resolves the correct platform automatically)
