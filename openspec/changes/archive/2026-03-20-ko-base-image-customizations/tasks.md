## 1. Configure Multi-Platform Builds via Makefile

- [x] 1.1 Add `ARCHS ?= $(shell go env GOARCH)` to the Makefile variable definitions
- [x] 1.2 Pass `--platform` to `ko build` in the `image-build` target by prefixing `linux/` to each entry in `$(ARCHS)`

## 2. Rename Makefile Variables

- [x] 2.1 Rename `COMMIT` to `REVISION` and change from `git rev-parse --short HEAD` to `git rev-parse HEAD` (full SHA)
- [x] 2.2 Rename `DATE` to `BUILD_DATE`
- [x] 2.3 Update `LDFLAGS` to reference `$(REVISION)` and `$(BUILD_DATE)`
- [x] 2.4 Verify all existing targets (`ci-build`) work with the renamed variables

## 3. Add `image-build` Makefile Target

- [x] 3.1 Add an `image-build` target to the Makefile that runs `ko build --bare` with all `--image-label` and `--image-annotation` flags (alphabetically ordered), using Makefile variables `VERSION`, `REVISION`, and `BUILD_DATE` defined with `?=` so environment values set by CI take precedence
- [x] 3.2 Support an optional `IMAGE_TAGS` variable; when set, pass `--tags $(IMAGE_TAGS)` to `ko build`
- [x] 3.3 Set `KO_DATA_DATE_EPOCH` from `git log -1 --format='%ct'` inside the target
- [x] 3.4 Export `VERSION`, `REVISION`, and `BUILD_DATE` as environment variables for `.ko.yaml` ldflags
- [x] 3.5 Conditionally include `source` and `url` labels/annotations only when `SOURCE_URL` is set
- [x] 3.6 Write the image reference to `image-ref.out` in the repository root and also print it to stdout
- [x] 3.7 Add `image-ref.out` to `.gitignore`
- [x] 3.8 Add removal of `image-ref.out` to the `clean` Makefile target

## 4. Update publish.yaml to Use Makefile

- [x] 4.1 Replace the inline `ko build` invocation in `publish.yaml` with `make image-build`, passing `VERSION`, `REVISION`, `BUILD_DATE`, `SOURCE_URL`, and `ARCHS=amd64,arm64` via environment; read the image reference from `image-ref.out` for cosign signing

## 5. Update release.yaml to Use Makefile

- [x] 5.1 Replace the inline `ko build` invocation in `release.yaml` with `make image-build IMAGE_TAGS="${GITHUB_REF_NAME}"`, passing `VERSION`, `REVISION`, `BUILD_DATE`, `SOURCE_URL`, and `ARCHS=amd64,arm64` via environment; read the image reference from `image-ref.out` for cosign signing

## 6. Verify

- [x] 6.1 (manual) Run `KO_DOCKER_REPO=ko.local make image-build` locally and inspect the output image to confirm labels and annotations are present with pkgproxy-specific values
- [x] 6.2 (manual) Inspect the OCI manifest to confirm annotations are present and consistent with the labels
- [x] 6.3 (manual) Confirm the published image is an OCI image index with both `linux/amd64` and `linux/arm64` manifests

## 7. Update CHANGELOG.md

- [x] 7.1 Add entry in the `[Unreleased]` section for multi-arch image support (amd64 + arm64)
