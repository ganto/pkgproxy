## ADDED Requirements

### Requirement: Listen host is resolved from flag, then env var, then default
The `serve` subcommand SHALL resolve the listen host using the following ordered precedence, with each step producing the value used by the HTTP server's listen socket:

1. The value of `--host` when the user explicitly passed the flag on the command line (detected via Cobra's `cmd.Flag("host").Changed` returning `true`).
2. The value of the `PKGPROXY_HOST` environment variable when it is set to a non-empty string.
3. The built-in default `localhost`.

An empty `PKGPROXY_HOST` (set but empty, or unset) SHALL be treated as "no env-var input" and SHALL fall through to step 3. The listen port resolution is unaffected by this change and continues to come exclusively from the `--port` flag and its built-in default.

#### Scenario: Explicit `--host` overrides everything
- **WHEN** the binary is started with `serve --host 192.168.10.4` and `PKGPROXY_HOST=10.0.0.1` is set
- **THEN** the HTTP server SHALL bind to `192.168.10.4:8080`

#### Scenario: Explicit `--host localhost` is honored
- **WHEN** the binary is started with `serve --host localhost` and `PKGPROXY_HOST=0.0.0.0` is set
- **THEN** the HTTP server SHALL bind to `localhost:8080`
- **AND** the env var SHALL NOT override the explicit flag value, even though it equals the built-in default

#### Scenario: `PKGPROXY_HOST` is used when the flag is absent
- **WHEN** the binary is started with `serve` (no `--host`) and `PKGPROXY_HOST=0.0.0.0` is set
- **THEN** the HTTP server SHALL bind to `0.0.0.0:8080`

#### Scenario: Empty `PKGPROXY_HOST` falls through to default
- **WHEN** the binary is started with `serve` (no `--host`) and `PKGPROXY_HOST=` (set but empty)
- **THEN** the HTTP server SHALL bind to `localhost:8080`

#### Scenario: Neither flag nor env var produces the built-in default
- **WHEN** the binary is started with `serve` (no `--host`) and `PKGPROXY_HOST` is unset
- **THEN** the HTTP server SHALL bind to `localhost:8080`
