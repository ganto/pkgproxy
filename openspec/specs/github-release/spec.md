## Requirements

### Requirement: CHANGELOG.md is maintained following Keep a Changelog 1.1.0
A `CHANGELOG.md` file at the repository root SHALL be maintained following the [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) format. Unreleased changes SHALL be accumulated under an `[Unreleased]` section. Before pushing a version tag the author SHALL promote the `[Unreleased]` section to a versioned entry with the release date.

#### Scenario: Unreleased section updated for user-facing changes
- **WHEN** a user-facing change is made to the codebase via Claude
- **THEN** Claude SHALL add a corresponding entry under the `[Unreleased]` section in `CHANGELOG.md` as instructed by `CLAUDE.md`

#### Scenario: Unreleased section promoted before tagging
- **WHEN** a version tag is about to be pushed
- **THEN** the `[Unreleased]` section SHALL be renamed to the versioned entry (e.g. `[v0.1.0] - 2026-03-17`) and a new empty `[Unreleased]` section SHALL be added above it

### Requirement: GitHub Release is created on version tag push
A dedicated release workflow SHALL trigger on `push: tags: ['v*']` and create a GitHub Release using `gh release create --notes-file`, with notes extracted from the matching versioned section of `CHANGELOG.md`. The tag name (e.g. `v0.1.0`) matches the `CHANGELOG.md` section header directly (e.g. `## [v0.1.0]`) — no prefix stripping required.

#### Scenario: CHANGELOG.md is prepared before the tag is pushed
- **WHEN** a release is being prepared
- **THEN** the author SHALL manually promote the `[Unreleased]` section to `[v<version>] - <date>` in `CHANGELOG.md` and commit that change before pushing the tag

#### Scenario: Release created for a version tag
- **WHEN** a tag matching `v*` (e.g. `v0.1.0`) is pushed to the repository
- **THEN** the workflow SHALL extract the `## [v0.1.0]` section from `CHANGELOG.md` using the tag name as-is
- **THEN** a GitHub Release SHALL be created for that tag with those extracted notes as the release body

#### Scenario: No release created for branch pushes
- **WHEN** a commit is pushed to a branch (not a tag)
- **THEN** the release workflow SHALL NOT run

### Requirement: Release workflow publishes a versioned container image
The release workflow SHALL build and publish a container image tagged with the version tag (e.g. `v0.1.0`), including the same OCI labels and manifest annotations as the main publish workflow. Future releases may additionally include a static binary or source archive, but these are out of scope for the initial implementation.

#### Scenario: Versioned image published on release
- **WHEN** a GitHub Release is created by the release workflow
- **THEN** a container image tagged with the version (e.g. `v0.1.0`) SHALL be published to GHCR
- **THEN** the image config SHALL carry all eight `org.opencontainers.image.*` labels, for example:
  ```
  org.opencontainers.image.version   = v0.1.0
  org.opencontainers.image.revision  = abc1234def5678901234567890abcdef12345678
  org.opencontainers.image.created   = 2026-03-17T10:00:00Z
  org.opencontainers.image.source    = https://github.com/ganto/pkgproxy
  org.opencontainers.image.title     = pkgproxy
  org.opencontainers.image.vendor    = ganto
  org.opencontainers.image.licenses  = Apache-2.0
  org.opencontainers.image.description = Caching forward proxy for Linux package repositories
  ```
- **THEN** the OCI manifest SHALL carry `source` and `revision` annotations with the same values
- **THEN** the image SHALL be signed with cosign
