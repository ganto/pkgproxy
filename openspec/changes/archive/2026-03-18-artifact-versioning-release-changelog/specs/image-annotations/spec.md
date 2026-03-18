## ADDED Requirements

### Requirement: OCI manifest annotations are set for cosign attestation
The container image build step SHALL attach `org.opencontainers.image.source` and `org.opencontainers.image.revision` as OCI manifest annotations using `ko build --image-annotation`. These annotations are required so that cosign can anchor attestations (e.g. SBOM, provenance) to verifiable provenance metadata on the manifest. Tools such as `cosign verify-attestation` and OCI policy engines resolve provenance from manifest annotations, not from image config labels.

#### Scenario: Annotations present on published image manifest
- **WHEN** the publish workflow runs `ko build`
- **THEN** the OCI manifest SHALL contain `org.opencontainers.image.source` and `org.opencontainers.image.revision` annotations with non-empty values

#### Scenario: Revision annotation matches triggering commit
- **WHEN** a workflow is triggered by a git push
- **THEN** the `org.opencontainers.image.revision` manifest annotation SHALL equal `github.sha`
