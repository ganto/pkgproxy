## Context

pkgproxy is distributed both as a source checkout (local `go run` / `make run`) and as a container image built by ko. The ko image already bundles `configs/pkgproxy.yaml` via the `kodata → configs` symlink; ko places that file at `/var/run/ko/pkgproxy.yaml` inside the image and sets `KO_DATA_PATH=/var/run/ko`. However, ko does not allow customising the image's `CMD`, `ENTRYPOINT`, or additional runtime `ENV` — every ko image has `ENTRYPOINT=/ko-app/pkgproxy`, no `CMD`, and only the `KO_DATA_PATH` env var. The current CLI assumes a user explicitly types `serve --config …`, which is acceptable for local use but fails the container "just run it" expectation.

Two coupled changes make the image self-contained without introducing a second image-build tool and without touching `.ko.yaml`:

1. A default subcommand so `podman run IMG` (which invokes the entrypoint with no arguments) dispatches `serve`.
2. An additional fallback in the config-file search so `serve` finds the bundled `pkgproxy.yaml` at `$KO_DATA_PATH/pkgproxy.yaml` without requiring the user to pass `--config`.

## Goals / Non-Goals

**Goals:**
- `podman run -p 8080:8080 ghcr.io/ganto/pkgproxy serve --host 0.0.0.0` starts a functional proxy using the bundled default config with no bind mount. (Note: `--host` must be supplied explicitly because `serve` defaults to `localhost`; changing that default is out of scope here.)
- Existing invocation patterns (`pkgproxy serve …`, `pkgproxy version`, `pkgproxy --help`, explicit `--config`, `$PKGPROXY_CONFIG`) continue to behave exactly as today.
- Local development (`make run`, `go run .`, `bin/pkgproxy serve`) is unaffected.
- No ko-specific path strings hard-coded anywhere in the Go source. The image-bundled location is discovered via the `KO_DATA_PATH` env var that ko itself sets — if the env var is absent (outside a ko image), the new lookup step is skipped entirely.

**Non-Goals:**
- Changing the container image's `--host` default (currently `localhost`; users must pass `--host 0.0.0.0` for port-mapped container access), cache directory defaults, or any other server-side defaults.
- Replacing ko with a Containerfile or mutating the image after `ko build` (evaluated and rejected — see Decisions).
- Supporting `/etc/pkgproxy/pkgproxy.yaml` or other FHS-style search paths. Can be added later under a separate proposal if needed.
- Changing the cache-directory behavior (`--cachedir` defaults to relative `cache`, which resolves under `/ko-app/cache` in the image and silently writes to the overlay FS if no volume is mounted). Acknowledged as a separate rough edge.

## Decisions

### D1. Default-subcommand mechanic: argv shim in `Execute()`

When `len(os.Args) == 1` (program invoked with zero user-supplied arguments), prepend `"serve"` to `os.Args` before calling the root command's `Execute()`. Cobra then parses a normal `pkgproxy serve` invocation.

**Alternatives considered:**
- _Move `RunE` onto the root command and drop `MinimumNArgs(1)`._ Rejected: conflates root-command help output with `serve`-specific flags, making `--help` noisy and `version`'s relative positioning awkward. Also breaks `cobra`'s child-command flag parsing for `serve`'s own flags when they'd need to appear on root too.
- _Use `cobra.Command.SilenceUsage` + a custom `PersistentPreRunE` that no-ops when no args._ Rejected: doesn't change `MinimumNArgs` semantics, so Cobra still errors before any custom code runs.

The argv shim is the narrowest possible change: it runs before Cobra sees anything, leaves `MinimumNArgs(1)` untouched as a guard for genuinely malformed invocations that somehow bypass the shim, and keeps `pkgproxy --help`, `pkgproxy version`, `pkgproxy serve --help`, etc. byte-identical to today.

**Edge case considered:** `pkgproxy --debug` (flag but no subcommand). Before this change it errors; after this change it still errors. The shim only fires when `len(os.Args) == 1` — a flag alone doesn't trigger the default. This keeps the rule trivially simple and avoids the ambiguity of "is this flag meant for serve, or for root?". If an extension is ever desired, it can be layered on later without breaking anything.

### D2. Config-discovery order: `./pkgproxy.yaml` wins over `$KO_DATA_PATH/pkgproxy.yaml`

Final order when no explicit `--config` flag and no `$PKGPROXY_CONFIG` env var:

```
1. ./pkgproxy.yaml       ← if the file exists
2. $KO_DATA_PATH/pkgproxy.yaml  ← if $KO_DATA_PATH is set
```

If neither path yields a file, `LoadConfig` returns the existing not-found error against the last path tried (`$KO_DATA_PATH/...` if set, otherwise `./pkgproxy.yaml`).

**Alternatives considered:**
- _`$KO_DATA_PATH` ahead of `./pkgproxy.yaml`._ Rejected: a stray `KO_DATA_PATH` in a developer shell would shadow the local checkout's config. "Local file wins" matches the least-surprise rule.
- _Walk a richer list (`/etc/pkgproxy/…`, `$XDG_CONFIG_HOME/pkgproxy/…`)._ Deferred. Keeps this change focused on the container use case that motivated it.

**"Exists" check:** Step 1 must be conditional on the file actually existing — otherwise `./pkgproxy.yaml` would always "match" and step 2 would be unreachable. Use `os.Stat` with `errors.Is(err, os.ErrNotExist)` to distinguish missing-file from other errors; other stat errors propagate.

### D3. `PKGPROXY_CONFIG` precedence vs. the default-path fallback

Today `initConfig` only consults `$PKGPROXY_CONFIG` when `configPath == defaultConfigPath`, i.e. the `--config` flag wasn't explicitly set. That coupling is preserved. The new discovery logic only runs when *both* the flag and env var are unset. Explicit user intent always wins.

### D4. No ko-awareness in production code beyond the env-var name

The string `"KO_DATA_PATH"` appears in `cmd/root.go`. That's the only ko-specific identifier in the app. The path literal `/var/run/ko` never appears — it's always read from the env var. If ko ever changes its data path, or if a different packaging tool sets `KO_DATA_PATH` differently, we follow automatically.

## Risks / Trade-offs

- **Behavioural change for existing image users** → a few users may rely on `podman run IMG` currently printing the Cobra "requires at least 1 arg" error as a smoke test. Those invocations will now start a server on port 8080 against the bundled config. Mitigation: CHANGELOG entry flags the default-subcommand behavior explicitly.
- **Silent shadowing between local and bundled configs** → if a developer somehow has `KO_DATA_PATH` set in their shell _and_ no `./pkgproxy.yaml` in CWD, the bundled config from a ko build cache could be loaded. Mitigation: the order chosen makes this essentially impossible in practice — the only way to hit it is to explicitly export `KO_DATA_PATH`, and by that point the user is opting into ko semantics.
- **Container-side error semantics change** → previously the image's entry-point error message named `./pkgproxy.yaml` as the missing file; now the final not-found error may name `$KO_DATA_PATH/pkgproxy.yaml`. Acceptable: the new message points at the actual location the image expected.

## Migration Plan

Pure additive behavior; no migration needed. Existing commands continue to work. Users reading old README snippets will find them still correct. The new shorter invocation is documented alongside for users adopting fresh installs.
