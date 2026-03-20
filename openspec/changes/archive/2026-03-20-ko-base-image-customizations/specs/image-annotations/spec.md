## MODIFIED Requirements

### Requirement: OCI manifest annotations are consistent with image labels
The container image build step SHALL attach the following `org.opencontainers.image` annotations to the OCI manifest using `ko build --image-annotation`, mirroring all label values and overriding any annotations inherited from the Chainguard base image with the same keys:
- `org.opencontainers.image.source` — repository URL derived from `SOURCE_URL` (e.g., `https://github.com/ganto/pkgproxy`); only included when `SOURCE_URL` is set
- `org.opencontainers.image.revision` — full git commit SHA
- `org.opencontainers.image.version` — output of `git describe --tags --always`
- `org.opencontainers.image.created` — RFC 3339 UTC timestamp of the build
- `org.opencontainers.image.title` — static value `pkgproxy`
- `org.opencontainers.image.vendor` — static value `ganto`
- `org.opencontainers.image.licenses` — static value `Apache-2.0`
- `org.opencontainers.image.description` — static value `Caching forward proxy for Linux package repositories`
- `org.opencontainers.image.authors` — static value `Reto Gantenbein https://github.com/ganto`
- `org.opencontainers.image.url` — repository URL derived from `SOURCE_URL` (e.g., `https://github.com/ganto/pkgproxy`); only included when `SOURCE_URL` is set

The `source` and `url` annotations are only required for images built via GitHub Actions and pushed to a registry. Local builds executed via Makefile without `SOURCE_URL` set will omit these two annotations.

Annotations are required so that cosign can anchor attestations (e.g. SBOM, provenance) to verifiable provenance metadata on the manifest, and so that tools inspecting either image config labels or manifest annotations see coherent, pkgproxy-specific metadata. The final manifest SHALL NOT contain any annotations with values referring to the Chainguard base image project.

#### Scenario: All annotations present on published image manifest
- **WHEN** the publish workflow runs `ko build`
- **THEN** the OCI manifest SHALL contain all ten `org.opencontainers.image` annotations with pkgproxy-specific values, matching their corresponding label values

#### Scenario: Revision annotation matches triggering commit
- **WHEN** a workflow is triggered by a git push
- **THEN** the `org.opencontainers.image.revision` manifest annotation SHALL equal `github.sha`

#### Scenario: Base image annotations are overridden
- **WHEN** the Chainguard base image contains annotations such as `org.opencontainers.image.created`, `org.opencontainers.image.title`, `org.opencontainers.image.vendor`, or `org.opencontainers.image.url` with Chainguard-specific values
- **THEN** the final published image manifest SHALL have all these annotations replaced with pkgproxy-specific values

#### Scenario: Labels and annotations are consistent
- **WHEN** the publish workflow produces a container image
- **THEN** the value of each `org.opencontainers.image.*` annotation SHALL match the value of the corresponding label for all ten keys
