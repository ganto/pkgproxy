## Why

The cache directory can only be set with the `--cachedir` flag. Every other operator-facing
setting (`--config`, `--host`, `--public-host`, `--trust-proxy`) has a `PKGPROXY_*` environment
variable so it can be configured in a container or an orchestrator without rewriting the command
line. `--cachedir` is the last gap: a containerized pkgproxy that stores its cache on a mounted
volume at a non-default path still needs a CLI argument appended to the default entrypoint.

## What Changes

- Add a `PKGPROXY_CACHEDIR` environment variable consulted by `serve` whenever the user has not
  explicitly passed `--cachedir`. Resolution chain: `--cachedir` flag (when set by the user) →
  `PKGPROXY_CACHEDIR` (when set and non-empty) → built-in default `cache`.
- Detect explicit user input with Cobra's `cmd.Flag("cachedir").Changed`, matching the
  `PKGPROXY_HOST` / `PKGPROXY_TRUST_PROXY` precedent rather than the value-equals-default
  heuristic still used by `PKGPROXY_CONFIG`. This keeps `--cachedir cache` distinguishable from
  "no flag passed".
- An empty or unset `PKGPROXY_CACHEDIR` is treated as "no env-var input" and falls through to the
  next step.
- Update the README.md flags table to fill in the env-var column for the `--cachedir` row.
- Add a `[Unreleased]` CHANGELOG entry.
- No new dependency (no viper); no change to the built-in default; no `.ko.yaml` change.

## Capabilities

### New Capabilities
- `cache-directory-config`: How `serve` resolves the local cache directory path from the
  `--cachedir` flag, the `PKGPROXY_CACHEDIR` environment variable, and the built-in default.

### Modified Capabilities
_None._ This change is purely additive; no existing spec's requirements change.

## Impact

- `cmd/root.go` — Add a `cachedirEnvVar = "PKGPROXY_CACHEDIR"` constant near `configPathEnvVar`
  and the `defaultDir` constant, and a `resolveCacheDir(flagChanged bool, flagValue, envValue string) string`
  helper implementing flag → env → default precedence.
- `cmd/serve.go` — In `newServeCommand()`'s `PersistentPreRunE`, call
  `resolveCacheDir(cmd.Flag("cachedir").Changed, cacheDir, os.Getenv(cachedirEnvVar))` and assign
  the result back to `cacheDir` before `startServer` reads it. `--cachedir` is an inherited
  persistent flag on the root command, so `cmd.Flag("cachedir")` resolves from `serve`.
- `cmd/root_test.go` (or `cmd/serve_test.go`) — Add `TestResolveCacheDir` mirroring the
  table-driven style of `TestResolveListenHost`.
- `README.md` — Flags table: add `PKGPROXY_CACHEDIR` to the env-var column of the `--cachedir` row.
- `CHANGELOG.md` — One concise `[Unreleased]` entry.
- No changes to landing-page snippets (they describe client-side repo config, not server
  invocation) or e2e tests (they invoke `serve` with explicit flags).
