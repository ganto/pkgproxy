## Context

pkgproxy is a Go binary distributed as a container image via GitHub Container Registry. Currently:
- The binary has no embedded version information; `pkgproxy --help` shows no version.
- Container images carry no OCI labels, so there is no way to correlate a running container to its source commit or release.
- There is no release workflow; no tags are created and no changelog exists.

The Makefile already computes `VERSION := $(shell git describe --always)` but never passes it to the linker.

## Goals / Non-Goals

**Goals:**
- Binary reports version, git commit, Go compiler version, and build date via a `pkgproxy version` subcommand; the same fields are logged at `INFO` level when `pkgproxy serve` starts.
- Container images carry eight `org.opencontainers.image.*` labels.
- A `CHANGELOG.md` following Keep a Changelog 1.1.0 is maintained and updated before each release.
- Pushing a `v*` tag triggers a GitHub Release whose notes are extracted from `CHANGELOG.md`.
- The release workflow also builds and publishes a tagged container image.

**Non-Goals:**
- Conventional-commit enforcement or commit linting.
- Automated changelog generation from commit messages.
- Semantic version validation or automated tag bumping.
- Releasing a static binary or source archive (deferred to a future change).
- Structured (`--json`) output for the `version` command.
- Publishing to registries other than GHCR.

## Decisions

### 1. Version injection via `-ldflags`

**Decision**: Declare three package-level variables (`Version`, `GitCommit`, `BuildDate`) in a new `cmd/version.go` file and populate them at build time with `-ldflags "-X ..."`. `GoVersion` is read at runtime via `runtime.Version()` and requires no injection.

**Rationale**: Standard Go pattern; zero runtime overhead; works identically for local builds (`make build`) and CI. The Makefile already has `VERSION` derived from `git describe --always`; extending it with `COMMIT` (→ `GitCommit`) and `DATE` (→ `BuildDate`) is trivial.

**Alternative considered**: Reading a `version.txt` file embedded via `//go:embed` — rejected because it requires an extra generated file and complicates the build graph.

### 2. `version` subcommand, not `--version` flag

**Decision**: Implement `pkgproxy version` as a Cobra subcommand rather than a root `--version` flag.

**Rationale**: Allows a clean multi-line output (version, commit, date) and is consistent with tools like `kubectl version` and `helm version`. Cobra's built-in `--version` flag only supports a plain string.

### 3. OCI labels via `ko build --image-label`

**Decision**: Pass `--image-label` flags directly in the `ko build` invocation inside the publish workflow, sourced from `github.*` context variables.

**Rationale**: No extra tooling needed; ko already supports this flag. Labels are baked into the image config and are visible via `docker inspect` and container registry UIs.

**Labels to set**:
| Label | Value |
|---|---|
| `org.opencontainers.image.source` | `${{ github.server_url }}/${{ github.repository }}` |
| `org.opencontainers.image.revision` | `${{ github.sha }}` |
| `org.opencontainers.image.version` | `$(git describe --always)` — matches `pkgproxy version` output |
| `org.opencontainers.image.created` | `$(date -u +%Y-%m-%dT%H:%M:%SZ)` |
| `org.opencontainers.image.title` | `pkgproxy` (static) |
| `org.opencontainers.image.vendor` | `ganto` (static) |
| `org.opencontainers.image.licenses` | `Apache-2.0` (static) |
| `org.opencontainers.image.description` | `Caching forward proxy for Linux package repositories` (static) |

**Annotations to set** (via `--image-annotation`, on the OCI manifest):
| Annotation | Value |
|---|---|
| `org.opencontainers.image.source` | `${{ github.server_url }}/${{ github.repository }}` |
| `org.opencontainers.image.revision` | `${{ github.sha }}` |

**Rationale for annotations**: cosign attaches attestations (e.g. SBOM, provenance) to the OCI manifest by digest. Tools verifying those attestations look for `org.opencontainers.image.source` and `org.opencontainers.image.revision` on the manifest itself — not inside the image config — to confirm the artefact's origin. Setting them as manifest annotations (in addition to image config labels) ensures `cosign verify-attestation` and policy engines can resolve provenance without pulling the image layers.

### 4. Release workflow separate from publish workflow

**Decision**: Create a new `.github/workflows/release.yaml` triggered on `push: tags: ['v*']`. The existing `publish.yaml` remains triggered on `push: branches: ['main']` for continuous image publishing from `main`.

**Rationale**: Keeps concerns separated — `main` pushes produce a rolling `latest` image; tag pushes produce a versioned release image and a GitHub Release. Avoids conditional logic inside a single workflow.

**Changelog**: A `CHANGELOG.md` following [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/) is kept at the repository root. Unreleased changes accumulate under an `[Unreleased]` section. A rule added to `CLAUDE.md` instructs Claude to update this section whenever a user-facing change is made to the codebase, removing the need for manual tracking. Before pushing a tag, the author manually promotes the `[Unreleased]` section to a versioned entry (e.g. `[v0.1.0] - 2026-03-17`) and commits that change. The tag is then pushed, triggering the release workflow.

**Changelog extraction**: Version tags and `CHANGELOG.md` section headers both use the `v` prefix (e.g. tag `v0.1.0` → section `## [v0.1.0]`), so no prefix stripping is needed. Extraction uses `awk` to print lines between the matching version header and the next `## [` header (exclusive):
```bash
awk "/^## \[${GITHUB_REF_NAME}\]/{found=1; next} found && /^## \[/{exit} found" CHANGELOG.md > release-notes.md
```

## Risks / Trade-offs

- **`git describe` produces dirty suffixes** (e.g. `v0.1.0-3-gabcdef`) for non-tag commits → acceptable for local dev; release builds are always on a clean tag.
- **`CHANGELOG.md` entries depend on Claude being invoked for changes** → mitigated by the `CLAUDE.md` rule; entries are added incrementally as part of each implementation session rather than in a pre-release rush.

## Migration Plan

1. Merge the Go changes (`cmd/version.go`, Makefile ldflag wiring).
2. Merge the workflow changes (`publish.yaml` label/annotation additions, new `release.yaml`).
3. Prepare `CHANGELOG.md` for the first release (promote `[Unreleased]` to `[v0.1.0]`) and push the tag to validate the release workflow end-to-end.
4. No rollback concerns — additive changes only; existing `serve` behaviour is unchanged.
