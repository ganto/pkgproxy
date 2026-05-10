## Why

The published container image still requires users to type `serve --host 0.0.0.0` because the `--host` flag defaults to `localhost`, which is unreachable through a port-mapped container. This change adds a `PKGPROXY_HOST` environment variable so operators can control the listen address via the environment without rewriting the command line. The previous `container-default-config` change closed half the gap by making `serve` the default subcommand and finding the bundled config; this change adds the missing env-var hook for the listen address.

## What Changes

- Add a new `PKGPROXY_HOST` environment variable that is consulted by `serve` whenever the user has not explicitly set `--host` on the command line. Resolution chain: `--host` flag (when set by the user) → `PKGPROXY_HOST` (when set and non-empty) → built-in default `localhost`.
- Use Cobra's `cmd.Flag("host").Changed` to detect explicit user input rather than the value-equals-default heuristic used by `PKGPROXY_CONFIG`. This makes `--host localhost` distinguishable from "no flag passed".
- Update README.md flags table to document the new env var, add a short note that env-var entries in the table can substitute for the corresponding flag, and replace the `serve --host 0.0.0.0` argument in the existing container-run examples with `-e PKGPROXY_HOST=0.0.0.0` so the env-var path becomes the recommended way to make a containerized pkgproxy reachable.
- Add a `[Unreleased]` CHANGELOG entry.
- `PKGPROXY_PORT` is **not** introduced. The 8080 default is acceptable in containers and the smaller surface is preferred.

## Capabilities

### New Capabilities
- `listen-address-config`: How `serve` resolves its listen address from the `--host` flag, the `PKGPROXY_HOST` environment variable, and the built-in default.

### Modified Capabilities
_None._ The existing `default-subcommand` and `config-discovery` specs are unaffected; this change is purely additive.

## Impact

- `cmd/serve.go` — Add `hostEnvVar = "PKGPROXY_HOST"` constant and a `resolveListenHost` helper. Wire it into `PersistentPreRunE` so `listenAddress` is finalized before `startServer` reads it for `sc.Address`.
- `cmd/serve_test.go` — Add tests for `resolveListenHost` mirroring the style of `TestResolvePublicAddr`, covering the flag-changed-vs-env precedence matrix and the empty-env-string case.
- `README.md` — Flags table adds `PKGPROXY_HOST` to the `--host` row, a short note clarifies that listed env vars are interchangeable with their flags, and the two `podman run` examples (lines 18 and 23) drop `serve --host 0.0.0.0` in favor of `-e PKGPROXY_HOST=0.0.0.0`.
- `CHANGELOG.md` — One concise `[Unreleased]` entry.
- No changes to `.ko.yaml`.
- No changes to landing-page snippets (those describe client-side repo config, not server invocation).
- No changes to e2e tests; they invoke `serve` with explicit flags.
