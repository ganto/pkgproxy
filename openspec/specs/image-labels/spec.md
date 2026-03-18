## Requirements

### Requirement: OCI image labels are embedded at build time
The container image build step SHALL attach the following `org.opencontainers.image` labels to every published image using `ko build --image-label`:
- `org.opencontainers.image.source` — repository URL
- `org.opencontainers.image.revision` — full git commit SHA
- `org.opencontainers.image.version` — output of `git describe --always`, matching the `pkgproxy version` output
- `org.opencontainers.image.created` — RFC 3339 UTC timestamp of the build
- `org.opencontainers.image.title` — static value `pkgproxy`
- `org.opencontainers.image.vendor` — static value `ganto`
- `org.opencontainers.image.licenses` — static value `Apache-2.0`
- `org.opencontainers.image.description` — static value `Caching forward proxy for Linux package repositories`

#### Scenario: Labels present on a tagged release build
- **WHEN** the release workflow runs `ko build` for tag `v0.1.0`
- **THEN** the resulting image config SHALL contain all eight labels, for example:
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

#### Scenario: Labels present on an untagged revision build
- **WHEN** the publish workflow runs `ko build` for a push to `main` where the commit is not directly tagged
- **THEN** `org.opencontainers.image.version` SHALL be the `git describe --always` output (e.g. `v0.1.0-3-gabc1234`) and all other labels SHALL be non-empty, for example:
  ```
  org.opencontainers.image.version   = v0.1.0-3-gabc1234
  org.opencontainers.image.revision  = abc1234def5678901234567890abcdef12345678
  ```

#### Scenario: Revision matches triggering commit
- **WHEN** a workflow is triggered by a git push
- **THEN** `org.opencontainers.image.revision` SHALL equal `github.sha`
