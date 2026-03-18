## Why

Container images and release artefacts built by pkgproxy have no embedded version or provenance metadata, making it impossible to trace which source commit or tag a running instance corresponds to and preventing users from knowing when a new release is available with its associated changes.

## What Changes

- Embed `org.opencontainers.image.*` labels (source, version, revision, created, title, vendor, licenses, description) into container images at build time via the ko publish step
- Attach `org.opencontainers.image.source` and `org.opencontainers.image.revision` as OCI manifest annotations; these are required so that cosign attestations can be anchored to verifiable provenance metadata on the manifest
- Maintain a `CHANGELOG.md` following the [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format; a `CLAUDE.md` instruction ensures Claude updates the `[Unreleased]` section for every user-facing code change; the section is promoted to a versioned entry before each tag is pushed
- Introduce a Git tag–based release workflow that creates a GitHub Release whose notes are sourced from the relevant `CHANGELOG.md` section

## Capabilities

### New Capabilities

- `image-labels`: Attach OCI-standard labels to the container image config during the ko build step, sourced from GitHub Actions context (repo URL, SHA, build timestamp), `git describe --always` (version), and static project metadata (title, vendor, licenses, description)
- `image-annotations`: Attach `source` and `revision` as OCI manifest annotations during the ko build step to enable cosign attestation anchoring
- `github-release`: Automatically create a GitHub Release with changelog notes when a version tag (e.g. `v*`) is pushed, triggered by a new CI/CD workflow
- `version-command`: A `pkgproxy version` CLI subcommand that prints `Version`, `GitCommit`, `GoVersion`, and `BuildDate`; `Version`, `GitCommit`, and `BuildDate` are injected at build time via `-ldflags`, `GoVersion` is read from `runtime.Version()`; the same fields are emitted as structured log attributes when `pkgproxy serve` starts

### Modified Capabilities

- (none)

## Impact

- `.github/workflows/publish.yaml`: add `--image-label` and `--image-annotation` flags to the `ko build` invocation
- New `.github/workflows/release.yaml`: separate workflow triggered on `push: tags: ['v*']`; builds and publishes a versioned container image, signs it with cosign, and creates a GitHub Release with notes extracted from `CHANGELOG.md`
- `CHANGELOG.md`: maintained at the repository root following Keep a Changelog 1.1.0; `[Unreleased]` promoted to a versioned entry manually before each tag push
- `cmd/version.go`: new Cobra subcommand; `cmd/root.go` updated to register it; `Makefile` updated to inject version, commit, and date via `-ldflags` at build time
