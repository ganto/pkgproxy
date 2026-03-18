## Requirements

### Requirement: `pkgproxy version` subcommand prints build metadata
The binary SHALL expose a `version` Cobra subcommand that prints `Version`, `GitCommit`, `GoVersion`, and `BuildDate` to stdout in that order. `Version` and `GitCommit` SHALL be injected at build time via `-ldflags "-X ..."` and default to `unknown` when not set. `BuildDate` SHALL be injected via ldflags for CI/release builds; when the ldflag is absent the binary SHALL fall back to `time.Now().UTC()` so that a real timestamp is always printed. `GoVersion` SHALL be read at runtime via `runtime.Version()` and requires no ldflag injection.

#### Scenario: Version output on tagged release build
- **WHEN** the binary is built with ldflags for a tagged commit (e.g. `v0.1.0`)
- **THEN** `pkgproxy version` SHALL print all four values, for example:
  ```
  Version:    v0.1.0
  GitCommit:  abc1234
  GoVersion:  go1.24.1
  BuildDate:  2026-03-17T10:00:00Z
  ```

#### Scenario: Version output on an untagged revision build
- **WHEN** the binary is built with ldflags for a commit that is not directly tagged
- **THEN** `pkgproxy version` SHALL print the `git describe` suffix form for version, for example:
  ```
  Version:    v0.1.0-3-gabc1234
  GitCommit:  abc1234
  GoVersion:  go1.24.1
  BuildDate:  2026-03-17T14:32:00Z
  ```

#### Scenario: Fallback values for development builds
- **WHEN** the binary is built without ldflags (e.g. `go run .`)
- **THEN** `pkgproxy version` SHALL print `unknown` for `Version` and `GitCommit`
- **THEN** `GoVersion` SHALL show the compiler version from `runtime.Version()`
- **THEN** `BuildDate` SHALL show the current time at invocation (RFC 3339 UTC) rather than a placeholder

### Requirement: `serve` command logs version information at startup
The `serve` command SHALL emit a structured `info` log line immediately after the logger is initialised, before any other output. The log line SHALL include the same four fields as the `version` subcommand output: `Version`, `GitCommit`, `GoVersion`, and `BuildDate`.

#### Scenario: Version info logged on server start
- **WHEN** `pkgproxy serve` is executed
- **THEN** the first log line SHALL be at `INFO` level with message `starting pkgproxy` and structured fields, for example:
  ```json
  {"level":"INFO","msg":"starting pkgproxy","version":"v0.1.0","gitCommit":"abc1234","goVersion":"go1.24.1","buildDate":"2026-03-17T10:00:00Z"}
  ```

### Requirement: Makefile passes version ldflags during build
The `ci-build` Makefile target SHALL pass `-ldflags` to inject `Version` (from `git describe --always`), `GitCommit` (from `git rev-parse --short HEAD`), and `BuildDate` (current UTC timestamp) into the binary.

#### Scenario: Binary built via `make build` reports version
- **WHEN** `make build` is run in a git repository with at least one commit
- **THEN** `bin/pkgproxy version` SHALL output a non-empty version string matching `git describe --always`
