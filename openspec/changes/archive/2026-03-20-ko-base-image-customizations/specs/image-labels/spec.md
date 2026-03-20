## MODIFIED Requirements

### Requirement: OCI image labels are embedded at build time
The container image build step SHALL attach the following `org.opencontainers.image` labels to every published image using `ko build --image-label`, overriding any labels inherited from the Chainguard base image with the same keys:
- `org.opencontainers.image.source` — repository URL derived from `SOURCE_URL` (e.g., `https://github.com/ganto/pkgproxy`); only included when `SOURCE_URL` is set
- `org.opencontainers.image.revision` — full git commit SHA
- `org.opencontainers.image.version` — output of `git describe --tags --always`, matching the `pkgproxy version` output
- `org.opencontainers.image.created` — RFC 3339 UTC timestamp of the build
- `org.opencontainers.image.title` — static value `pkgproxy`
- `org.opencontainers.image.vendor` — static value `ganto`
- `org.opencontainers.image.licenses` — static value `Apache-2.0`
- `org.opencontainers.image.description` — static value `Caching forward proxy for Linux package repositories`
- `org.opencontainers.image.authors` — static value `Reto Gantenbein https://github.com/ganto`
- `org.opencontainers.image.url` — repository URL derived from `SOURCE_URL` (e.g., `https://github.com/ganto/pkgproxy`); only included when `SOURCE_URL` is set

The `source` and `url` labels are only required for images built via GitHub Actions and pushed to a registry. Local builds executed via Makefile without `SOURCE_URL` set will omit these two labels.

The final image SHALL NOT contain any labels with values referring to the Chainguard base image project (e.g., Chainguard vendor, Chainguard source URLs, or Chainguard image titles).

#### Scenario: All ten labels present on a tagged release build
- **WHEN** the release workflow runs `ko build` for tag `v0.1.0`
- **THEN** the resulting image config SHALL contain all ten labels with pkgproxy-specific values, for example:
  ```
  org.opencontainers.image.version     = v0.1.0
  org.opencontainers.image.revision    = abc1234def5678901234567890abcdef12345678
  org.opencontainers.image.created     = 2026-03-17T10:00:00Z
  org.opencontainers.image.source      = https://github.com/ganto/pkgproxy
  org.opencontainers.image.title       = pkgproxy
  org.opencontainers.image.vendor      = ganto
  org.opencontainers.image.licenses    = Apache-2.0
  org.opencontainers.image.description = Caching forward proxy for Linux package repositories
  org.opencontainers.image.authors     = Reto Gantenbein https://github.com/ganto
  org.opencontainers.image.url         = https://github.com/ganto/pkgproxy
  ```

#### Scenario: Labels present on an untagged revision build
- **WHEN** the publish workflow runs `ko build` for a push to `main` where the commit is not directly tagged
- **THEN** `org.opencontainers.image.version` SHALL be the `git describe --always` output (e.g. `v0.1.0-3-gabc1234`) and all other labels SHALL be non-empty with pkgproxy-specific values

#### Scenario: Revision matches triggering commit
- **WHEN** a workflow is triggered by a git push
- **THEN** `org.opencontainers.image.revision` SHALL equal `github.sha`

#### Scenario: Base image labels are overridden
- **WHEN** the Chainguard base image contains labels such as `org.opencontainers.image.title=static`, `org.opencontainers.image.vendor=Chainguard`, or `org.opencontainers.image.url` pointing to a Chainguard URL
- **THEN** the final published image SHALL have these labels replaced with pkgproxy-specific values (`title=pkgproxy`, `vendor=ganto`, `url=https://github.com/ganto/pkgproxy`)
