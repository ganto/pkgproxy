## Context

`--cachedir` is a persistent flag declared on the **root** command (`cmd/root.go:43`), defaulting
to `cache`. Only the `serve` subcommand reads the resolved value — `startServer` passes `cacheDir`
into `newEchoApp` (`cmd/serve.go:239`). No env var currently influences it.

The project has three flag↔env precedents, split across two patterns:

- **`Flag.Changed` pattern** (newer, preferred): `--host` ↔ `PKGPROXY_HOST` and
  `--trust-proxy` ↔ `PKGPROXY_TRUST_PROXY`, both resolved in `serve`'s `PersistentPreRunE` via
  `resolveListenHost` / `resolveTrustProxy` helpers that take a `flagChanged bool`.
- **value-equals-default heuristic** (older): `--config` ↔ `PKGPROXY_CONFIG`, which treats
  `configPath == defaultConfigPath` as "not set". Its known edge case: `--config ./pkgproxy.yaml`
  is indistinguishable from omitting the flag.

The `host-env-var` change (archived 2026-05-10) established the `Flag.Changed` approach as a
one-way ratchet for new code. This change follows it.

## Goals / Non-Goals

**Goals:**
- `podman run … -e PKGPROXY_CACHEDIR=/var/cache/pkgproxy ghcr.io/ganto/pkgproxy` uses that path
  for the cache without appending `serve --cachedir …` to the entrypoint.
- Explicit `--cachedir` always wins, including `--cachedir cache` (the built-in default value).
- Local development (`make run`, `bin/pkgproxy serve`) keeps `cache` as the default — no change
  to existing behavior when neither flag nor env var is set.

**Non-Goals:**
- Migrating `PKGPROXY_CONFIG` to the `Flag.Changed` pattern (separate change).
- Adding a `cachedir` key to the repository YAML config or the landing page snippets.
- Creating or validating the directory, or changing how `FileCache` treats the path.
- Baking a `PKGPROXY_CACHEDIR` default into the published image config.

## Decisions

### D1. Resolve in `serve`'s `PersistentPreRunE` via `cmd.Flag("cachedir").Changed`

`--cachedir` is an inherited persistent flag on `serve`, so `cmd.Flag("cachedir")` resolves it
from within the `serve` command's `PersistentPreRunE`. Add the helper and constant to
`cmd/root.go` (where `cacheDir`, `defaultDir`, and the `configPathEnvVar` constant already live),
and do the wiring in `cmd/serve.go` alongside the existing `resolveListenHost` /
`resolveTrustProxy` calls:

```go
// cmd/root.go
const cachedirEnvVar = "PKGPROXY_CACHEDIR"

func resolveCacheDir(flagChanged bool, flagValue, envValue string) string {
    if flagChanged {
        return flagValue
    }
    if envValue != "" {
        return envValue
    }
    return defaultDir
}
```

```go
// cmd/serve.go — PersistentPreRunE, before initConfig()
cacheDir = resolveCacheDir(cmd.Flag("cachedir").Changed, cacheDir, os.Getenv(cachedirEnvVar))
```

`startServer` continues to read the package-level `cacheDir` unchanged.

**Alternatives considered:**
- _value-equals-default heuristic (mirror `PKGPROXY_CONFIG`)._ Rejected — carries forward the
  edge case the project has already decided to stop propagating.
- _Add a `PersistentPreRunE` to the root command._ Rejected — the root command has none today,
  `serve` already has one that does exactly this kind of resolution, and `serve` is the only
  consumer of `cacheDir`.
- _viper for env binding._ Rejected — one more mapping doesn't justify a dependency.

### D2. Empty-string env var is treated as "unset"

`os.Getenv` returns `""` for both unset and explicitly-empty. Both fall through to the default.
An empty cache path is never useful, and this matches `resolveListenHost` / `resolveTrustProxy`.

### D3. Built-in default stays `cache`

`defaultDir` in `cmd/root.go` is unchanged. No Go-side conditional logic; container users set the
path at `podman run` time.

## Risks / Trade-offs

- **Two resolution patterns coexist in `cmd/`** (`Flag.Changed` for host/trust-proxy/cachedir,
  value-equals-default for config) → documented deliberate ratchet; `PKGPROXY_CONFIG` migrates
  later under its own change.
- **`Flag.Changed` on a package-global var can leak between command reruns in tests** → mitigate
  by unit-testing `resolveCacheDir(changed, flagValue, envValue)` directly, with no Cobra
  dependency, exactly as `TestResolveListenHost` does.
