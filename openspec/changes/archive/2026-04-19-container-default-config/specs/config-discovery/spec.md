## ADDED Requirements

### Requirement: Repository configuration is discovered via an ordered lookup
When neither the `--config`/`-c` flag nor the `PKGPROXY_CONFIG` environment variable is set, the CLI SHALL resolve the repository configuration file by trying the following paths in order and using the first one that exists as a regular readable file:

1. `./pkgproxy.yaml` (relative to the process working directory)
2. `$KO_DATA_PATH/pkgproxy.yaml` — only attempted if the `KO_DATA_PATH` environment variable is set to a non-empty value

If neither path yields a readable file, the CLI SHALL fail with the existing "unable to load configuration from ..." error, naming the last path that was attempted.

Explicit user input SHALL always take precedence over the ordered lookup: when `--config`/`-c` is passed with a value other than the built-in default, or when `PKGPROXY_CONFIG` is set, that path is used directly and the ordered lookup is not consulted.

#### Scenario: Local source checkout uses `./pkgproxy.yaml`
- **WHEN** the binary is started with no `--config` flag, no `PKGPROXY_CONFIG` env var, and `./pkgproxy.yaml` exists in the working directory
- **THEN** the CLI SHALL load configuration from `./pkgproxy.yaml`
- **AND** the value of `KO_DATA_PATH` SHALL NOT affect the outcome

#### Scenario: Ko-built container uses the bundled config
- **WHEN** the binary is started in a ko-built image (working directory `/ko-app`, `KO_DATA_PATH=/var/run/ko`, `pkgproxy.yaml` present at `/var/run/ko/pkgproxy.yaml` and not at `./pkgproxy.yaml`) with no `--config` flag and no `PKGPROXY_CONFIG` env var
- **THEN** the CLI SHALL load configuration from `/var/run/ko/pkgproxy.yaml`

#### Scenario: Local file wins over ko-bundled fallback
- **WHEN** the binary is started with both `./pkgproxy.yaml` present and `KO_DATA_PATH` set to a directory containing a different `pkgproxy.yaml`, with no `--config` flag and no `PKGPROXY_CONFIG` env var
- **THEN** the CLI SHALL load configuration from `./pkgproxy.yaml`

#### Scenario: Explicit `--config` bypasses the lookup
- **WHEN** the binary is started with `--config /custom/path.yaml`
- **THEN** the CLI SHALL load configuration from `/custom/path.yaml` regardless of whether `./pkgproxy.yaml` or `$KO_DATA_PATH/pkgproxy.yaml` exist

#### Scenario: `PKGPROXY_CONFIG` bypasses the lookup
- **WHEN** the binary is started with `PKGPROXY_CONFIG=/custom/path.yaml` and no `--config` flag
- **THEN** the CLI SHALL load configuration from `/custom/path.yaml` regardless of whether `./pkgproxy.yaml` or `$KO_DATA_PATH/pkgproxy.yaml` exist

#### Scenario: All default paths missing produces a clear error
- **WHEN** the binary is started with no `--config` flag, no `PKGPROXY_CONFIG` env var, `./pkgproxy.yaml` absent, and `KO_DATA_PATH` unset
- **THEN** the CLI SHALL exit with an error of the form `unable to load configuration from ./pkgproxy.yaml: ...`

#### Scenario: `KO_DATA_PATH` set but no file present
- **WHEN** the binary is started with no `--config` flag, no `PKGPROXY_CONFIG` env var, `./pkgproxy.yaml` absent, and `KO_DATA_PATH` set to a directory that does not contain `pkgproxy.yaml`
- **THEN** the CLI SHALL exit with an error naming `$KO_DATA_PATH/pkgproxy.yaml` as the path it attempted to load
