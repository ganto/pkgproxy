## ADDED Requirements

### Requirement: `serve` is dispatched when the binary is invoked with no arguments
The pkgproxy CLI SHALL dispatch the `serve` subcommand when the binary is invoked with no user-supplied arguments (i.e. `len(os.Args) == 1`). The dispatch SHALL be implemented as a pre-Cobra shim that prepends the literal string `"serve"` to `os.Args` before `cobra.Command.Execute()` is called. Any other invocation form SHALL be passed to Cobra unchanged; in particular, invocations that include at least one argument (whether a subcommand, a flag, or a positional value) SHALL retain their current behavior, including the `MinimumNArgs(1)` error when a flag appears without a subcommand.

#### Scenario: Container start with no arguments runs serve
- **WHEN** the binary is executed with `os.Args == ["pkgproxy"]` (e.g. `podman run ghcr.io/ganto/pkgproxy`)
- **THEN** the CLI SHALL behave as if `pkgproxy serve` had been invoked, starting the HTTP server using the configured defaults and discovered configuration file

#### Scenario: `--help` still shows help
- **WHEN** the binary is executed as `pkgproxy --help` or `pkgproxy -h`
- **THEN** Cobra SHALL print the root command help output and exit 0 (no server is started)

#### Scenario: `version` subcommand still works
- **WHEN** the binary is executed as `pkgproxy version`
- **THEN** the `version` subcommand SHALL run as defined by the `version-command` spec (no server is started and no `serve` dispatch occurs)

#### Scenario: Explicit `serve` invocation is unchanged
- **WHEN** the binary is executed as `pkgproxy serve --host 0.0.0.0 --debug`
- **THEN** the shim SHALL NOT modify `os.Args`, and Cobra SHALL parse the invocation as `serve` with the given flags

#### Scenario: Flag without a subcommand still errors
- **WHEN** the binary is executed as `pkgproxy --debug` (a flag with no subcommand)
- **THEN** the shim SHALL NOT fire (because `len(os.Args) > 1`), and Cobra SHALL return its standard "requires at least 1 arg(s)" error
