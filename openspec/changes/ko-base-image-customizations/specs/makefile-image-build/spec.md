## MODIFIED Requirements

### Requirement: Makefile variables use consistent naming
The existing Makefile variables `COMMIT` and `DATE` SHALL be renamed to `REVISION` and `BUILD_DATE` respectively:
- `REVISION` SHALL use `git rev-parse HEAD` (full 40-character SHA, not the short form)
- `BUILD_DATE` SHALL use `date -u +%Y-%m-%dT%H:%M:%SZ` (unchanged format)
- The `LDFLAGS` definition SHALL be updated to reference `$(REVISION)` and `$(BUILD_DATE)`
- All existing targets that reference these variables (`ci-build`, and any others) SHALL continue to work with the renamed variables

## ADDED Requirements

### Requirement: ko build is invoked via Makefile target
The Makefile SHALL provide an `image-build` target that encapsulates the full `ko build --bare` invocation with all `--image-label` and `--image-annotation` flags, sorted alphabetically by annotation/label name. The target SHALL:
- Use Makefile variables `VERSION`, `REVISION` (full git SHA), `BUILD_DATE`, and `ARCHS` for dynamic metadata; these variables SHALL be defined with `?=` so that values set in the environment (e.g. by a CI workflow via `$GITHUB_ENV`) take precedence over the Makefile defaults
- Accept an optional `IMAGE_TAGS` variable; when set, pass `--tags $(IMAGE_TAGS)` to `ko build`
- Set `KO_DATA_DATE_EPOCH` from the latest git commit timestamp
- Derive the `source` and `url` labels/annotations from the `SOURCE_URL` environment variable; when `SOURCE_URL` is not set, these labels/annotations SHALL be omitted
- Export `VERSION`, `REVISION`, and `BUILD_DATE` as environment variables for `.ko.yaml` ldflags
- Write the image reference to `image-ref.out` in the repository root and also print it to stdout
- `image-ref.out` SHALL be added to `.gitignore`
- The existing `clean` Makefile target SHALL remove `image-ref.out`

GitHub Actions workflows (`publish.yaml`, `release.yaml`) SHALL call `make image-build` instead of inlining the `ko build` command. Workflows read the image reference from `image-ref.out` for cosign signing. Workflows remain responsible for setting environment variables (`VERSION`, `REVISION`, `BUILD_DATE`, `SOURCE_URL`, `ARCHS`) and for cosign signing.

#### Scenario: Local build with defaults
- **WHEN** a developer runs `KO_DOCKER_REPO=ko.local make image-build`
- **THEN** `ko build` SHALL execute with all labels and annotations except `source` and `url` (since `SOURCE_URL` is not set), using locally computed `VERSION`, `REVISION`, and `BUILD_DATE` values
- **AND** the host-native platform variant SHALL be loaded into the local container runtime

#### Scenario: CI publish build
- **WHEN** the publish workflow runs `make image-build` with `VERSION`, `REVISION`, `BUILD_DATE`, `SOURCE_URL`, and `ARCHS=amd64,arm64` set via environment
- **THEN** the Makefile target SHALL use these values for all ten labels and ten annotations
- **AND** the image SHALL be pushed to the registry configured in `KO_DOCKER_REPO`

#### Scenario: CI release build with tag
- **WHEN** the release workflow runs `make image-build IMAGE_TAGS=v1.0.0`
- **THEN** the Makefile target SHALL pass `--tags v1.0.0` to `ko build`
- **AND** all labels and annotations SHALL be present as in the publish build

#### Scenario: Workflows stay in sync
- **WHEN** a label or annotation is added or modified
- **THEN** the change SHALL be made only in the Makefile, not duplicated across workflow files
