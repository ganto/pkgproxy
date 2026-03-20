## Requirements

### Requirement: Container images are built for multiple platforms
The Makefile SHALL define an `ARCHS` variable that controls which architectures are built:
- Default value: `$(shell go env GOARCH)` — the native host architecture, so local builds produce a single-platform image without cross-compilation overhead
- CI workflows SHALL override `ARCHS=amd64,arm64` via environment to produce a multi-platform OCI image index

The `image-build` target SHALL pass `--platform` to `ko build` by prefixing `linux/` to each entry in `ARCHS` (e.g. `amd64` → `linux/amd64`, `amd64,arm64` → `linux/amd64,linux/arm64`). `.ko.yaml` SHALL NOT define `defaultPlatforms`.

#### Scenario: Local build uses native architecture only
- **WHEN** a developer runs `make image-build` on an `amd64` host without setting `ARCHS`
- **THEN** `ko build` SHALL target `linux/amd64` only and produce a single-platform image

#### Scenario: Local build on arm64 host
- **WHEN** a developer runs `make image-build` on an `arm64` host without setting `ARCHS`
- **THEN** `ko build` SHALL target `linux/arm64` only and produce a single-platform image

#### Scenario: Multi-arch manifest list is published via CI
- **WHEN** the publish workflow runs `make image-build` with `ARCHS=amd64,arm64` set via environment
- **THEN** the resulting artifact in the container registry SHALL be an OCI image index containing at least two manifests: one for `linux/amd64` and one for `linux/arm64`

#### Scenario: Each platform image is functional
- **WHEN** a user pulls the published image on an `arm64` host
- **THEN** the container runtime SHALL automatically resolve the `linux/arm64` variant and the pkgproxy binary SHALL execute successfully

#### Scenario: Existing amd64 users are unaffected
- **WHEN** a user pulls the published image on an `amd64` host
- **THEN** the container runtime SHALL automatically resolve the `linux/amd64` variant, identical in behavior to the previous single-arch image
