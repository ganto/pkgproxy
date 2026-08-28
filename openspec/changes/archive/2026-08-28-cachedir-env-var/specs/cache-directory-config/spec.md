## ADDED Requirements

### Requirement: Cache directory is resolved from flag, then env var, then default
The `serve` subcommand SHALL resolve the local cache directory path using the following ordered
precedence, producing the value passed to the caching proxy as its cache base path:

1. The value of `--cachedir` when the user explicitly passed the flag on the command line
   (detected via Cobra's `cmd.Flag("cachedir").Changed` returning `true`).
2. The value of the `PKGPROXY_CACHEDIR` environment variable when it is set to a non-empty string.
3. The built-in default `cache`.

An empty `PKGPROXY_CACHEDIR` (set but empty, or unset) SHALL be treated as "no env-var input" and
SHALL fall through to step 3. This change does not create, validate, or otherwise alter treatment
of the resolved path beyond selecting its value.

#### Scenario: Explicit `--cachedir` overrides everything
- **WHEN** the binary is started with `serve --cachedir /data/cache` and `PKGPROXY_CACHEDIR=/other` is set
- **THEN** the caching proxy SHALL use `/data/cache` as its cache base path

#### Scenario: Explicit `--cachedir cache` is honored
- **WHEN** the binary is started with `serve --cachedir cache` and `PKGPROXY_CACHEDIR=/data/cache` is set
- **THEN** the caching proxy SHALL use `cache` as its cache base path
- **AND** the env var SHALL NOT override the explicit flag value, even though it equals the built-in default

#### Scenario: `PKGPROXY_CACHEDIR` is used when the flag is absent
- **WHEN** the binary is started with `serve` (no `--cachedir`) and `PKGPROXY_CACHEDIR=/var/cache/pkgproxy` is set
- **THEN** the caching proxy SHALL use `/var/cache/pkgproxy` as its cache base path

#### Scenario: Empty `PKGPROXY_CACHEDIR` falls through to default
- **WHEN** the binary is started with `serve` (no `--cachedir`) and `PKGPROXY_CACHEDIR=` (set but empty)
- **THEN** the caching proxy SHALL use `cache` as its cache base path

#### Scenario: Neither flag nor env var produces the built-in default
- **WHEN** the binary is started with `serve` (no `--cachedir`) and `PKGPROXY_CACHEDIR` is unset
- **THEN** the caching proxy SHALL use `cache` as its cache base path
