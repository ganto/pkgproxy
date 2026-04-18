## Why

Running the published container image today requires users to explicitly pass `serve --host 0.0.0.0 --config /var/run/ko/pkgproxy.yaml` because (a) the root command rejects invocations without a subcommand and (b) the bundled config baked into the image by ko's `kodata` mechanism is not part of the binary's default search path. The advertised "just run the image" workflow therefore never works; every user ends up copying a long command from the README. The goal is `podman run -p 8080:8080 ghcr.io/ganto/pkgproxy` starting a working server out of the box.

## What Changes

- When the binary is invoked with no arguments at all, `serve` SHALL be dispatched as the default subcommand. Invocations with any other argument (including flags like `--help` or other subcommands like `version`) continue to behave as today.
- The config file search order SHALL be extended with a fallback to `$KO_DATA_PATH/pkgproxy.yaml` when the environment variable is set. The new order is: `--config` flag → `$PKGPROXY_CONFIG` → `./pkgproxy.yaml` (if it exists) → `$KO_DATA_PATH/pkgproxy.yaml` (if `KO_DATA_PATH` is set).
- Update `README.md` so that the "Run the code" section shows the simplified container invocation and retains the bind-mount variant as an explicit override.

## Capabilities

### New Capabilities
- `default-subcommand`: When invoked without arguments, the binary dispatches `serve`; all other invocation forms retain current semantics.
- `config-discovery`: Ordered lookup of the repository configuration file, including the `$KO_DATA_PATH` fallback so ko-built images are self-contained.

### Modified Capabilities
_None._

## Impact

- `cmd/root.go` — Extend `initConfig` with the `$KO_DATA_PATH` fallback; the existing `./pkgproxy.yaml` behaviour is preserved but becomes conditional on the file actually existing so the fallback is reachable.
- `cmd/root.go` or `main.go` — Insert `"serve"` into `os.Args` inside `Execute()` when `len(os.Args) == 1`, before `cobra.Command.Execute()` runs. `MinimumNArgs(1)` on the root command stays in place as a safety net for malformed invocations.
- `README.md` — Replace the two container example commands with the shorter form and a bind-mount override example; update the flags table note if needed.
- `CHANGELOG.md` — User-facing entry under `[Unreleased]`.
- No change to landing-page snippets (they describe client-side repo configuration, not server invocation).
- No change to `.ko.yaml`, the `kodata` symlink, or the Makefile `image-build` target.
- Published container images behave differently for users who previously ran `podman run IMG` expecting the help output: they will now start a server on port 8080. This is the intended behaviour and matches the documented "just run the image" use case.
