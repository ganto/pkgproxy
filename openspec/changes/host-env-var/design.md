## Context

The previous `container-default-config` change (archived 2026-04-19) made `serve` the default subcommand and taught `initConfig` to find the bundled `pkgproxy.yaml` via `KO_DATA_PATH`. The remaining rough edge is the listen address: `--host` defaults to `localhost`, which is unreachable through a port-mapped container, so users still have to type `serve --host 0.0.0.0`.

The binary doesn't currently read any env var for the listen address. Adding `PKGPROXY_HOST` gives operators a way to control the listen address via the environment — useful both for `podman run -e PKGPROXY_HOST=0.0.0.0 …` and for any orchestrator that sets env vars at deploy time.

The existing CLI already has two flag→env precedents:

- `--public-host` ↔ `PKGPROXY_PUBLIC_HOST` — uses empty-string-as-unset because the flag has no default.
- `--config` ↔ `PKGPROXY_CONFIG` — uses value-equals-default because the flag has a meaningful default.

`--host` falls into the same category as `--config`. Mirroring `PKGPROXY_CONFIG`'s value-equals-default trick would carry forward its known edge case (a user typing `--host localhost` would be indistinguishable from "no flag passed"). Cobra exposes a cleaner primitive — `cmd.Flag(name).Changed` — that distinguishes "user typed the default" from "default applied". This change adopts that primitive.

## Goals / Non-Goals

**Goals:**
- `podman run -p 8080:8080 -e PKGPROXY_HOST=0.0.0.0 ghcr.io/ganto/pkgproxy` starts a working server reachable through the port mapping without passing CLI arguments.
- `podman run -p 8080:8080 -e PKGPROXY_HOST=127.0.0.1 ghcr.io/ganto/pkgproxy` lets the operator narrow the listen address without rebuilding the image or passing CLI arguments.
- Local development (`make run`, `go run .`, `bin/pkgproxy serve`) keeps `localhost` as the listen default — no change to muscle memory, no surprise binding to a public interface.
- Explicit `--host` always wins, including when it equals the built-in default. `--host localhost` means "really listen on localhost".

**Non-Goals:**
- Adding `PKGPROXY_PORT`. Symmetric and harmless, but the 8080 default already works in containers; YAGNI.
- Changing the binary's CLI default for `--host` or baking `0.0.0.0` into the published image config. Operators set the listen address via the env var at runtime.
- Migrating `PKGPROXY_CONFIG` to the same `Flag.Changed` pattern. Worth doing later for consistency, but out of scope here.
- FHS-style env-var conventions (`PKGPROXY_LISTEN_ADDRESS`, `PKGPROXY_BIND_HOST`, etc.). The chosen name `PKGPROXY_HOST` is symmetric with the flag and matches the project's existing naming.

## Decisions

### D1. Detect "user passed --host" via `cmd.Flag("host").Changed`

The resolution helper takes either the Cobra command (so it can call `cmd.Flag("host").Changed`) or a pre-extracted `bool changed` argument. The latter is preferred because it makes the helper trivially unit-testable without constructing a Cobra command.

```go
// in cmd/serve.go
const hostEnvVar = "PKGPROXY_HOST"

func resolveListenHost(flagChanged bool, flagValue, envValue string) string {
    if flagChanged {
        return flagValue
    }
    if envValue != "" {
        return envValue
    }
    return defaultAddress
}
```

Wiring lives in `PersistentPreRunE` (which already calls `initConfig`):

```go
PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
    listenAddress = resolveListenHost(
        cmd.Flag("host").Changed,
        listenAddress,
        os.Getenv(hostEnvVar),
    )
    return initConfig()
},
```

`startServer` keeps reading `listenAddress` unchanged.

**Alternatives considered:**
- _Mirror `PKGPROXY_CONFIG`'s value-equals-default trick._ Rejected. Carries forward the same edge case the user explicitly chose to avoid for new code: `--host localhost` would silently be overridden by `PKGPROXY_HOST=192.168.0.10`, which is the wrong behavior.
- _Use viper for full env-var binding._ Rejected. Two flag→env mappings (this one and the existing `PKGPROXY_PUBLIC_HOST`/`PKGPROXY_CONFIG`) don't justify a new dependency.

### D2. Empty-string env var is treated as "unset"

`os.Getenv` returns `""` for unset and for explicitly-set-to-empty. Both are treated as "fall through to the default". Distinguishing them with `os.LookupEnv` would let a user "unset" the env var by passing `-e PKGPROXY_HOST=` — but that's never useful here (an empty listen address would fail to bind), and the simpler rule keeps parity with `resolvePublicAddr`.

### D3. The CLI default stays `localhost`; operators set the listen address at runtime

`defaultAddress` in `cmd/serve.go` is unchanged. Local builds and `make run` keep their current behavior. Container users who need the server reachable on all interfaces pass `-e PKGPROXY_HOST=0.0.0.0` at `podman run` time — no Go-side conditional logic, no image-baked default.

This is deliberate: developers running `bin/pkgproxy serve` from a checkout never accidentally bind to a public interface, and the published image carries no opinionated default for the listen address.

## Risks / Trade-offs

- **`cmd.Flag("host").Changed` is package-global** → `listenAddress` is set as a package-level `var`, and `Changed` reflects what Cobra parsed *for the current invocation*. In tests that reuse the same root command across calls, `Changed` could leak between iterations. → Mitigation: write the unit tests against `resolveListenHost(changed, flagValue, envValue)` directly so the helper has no Cobra dependency. End-to-end behavior of `PersistentPreRunE` is exercised through normal command runs, not table-driven tests.
- **Inconsistency with `PKGPROXY_CONFIG`** → `--config` keeps the value-equals-default heuristic; `--host` uses `Flag.Changed`. Two patterns in the same package is mildly ugly. → Mitigation: documented in this design as a deliberate one-way ratchet — new code uses the better primitive; the existing `PKGPROXY_CONFIG` can be migrated later under a separate change.
