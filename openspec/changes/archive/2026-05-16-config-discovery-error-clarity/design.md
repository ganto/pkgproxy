## Context

`cmd/root.go`'s `resolveConfigPath` performs an ordered lookup for the repository configuration file when neither `--config` nor `PKGPROXY_CONFIG` is set:

1. `./pkgproxy.yaml`
2. `$KO_DATA_PATH/pkgproxy.yaml` (only attempted if `KO_DATA_PATH` is non-empty)

Today the function returns the joined `$KO_DATA_PATH/pkgproxy.yaml` path unconditionally when the local default is missing — it does not stat the ko candidate. `initConfig` then calls `LoadConfig(configPath)`, and any failure surfaces in a wrapped error that names only the single, last-tried path. An operator who unknowingly inherited `KO_DATA_PATH` from a parent shell sees an error pointing at a directory they may not even know about, with no indication that the local default was tried first.

The existing capability spec (`openspec/specs/config-discovery/spec.md`) explicitly codifies the current behavior in its "KO_DATA_PATH set but no file present" scenario, so this change must update that spec.

## Goals / Non-Goals

**Goals:**

- Operators see every candidate path the resolver attempted when the lookup fails, in the order it tried them.
- The `$KO_DATA_PATH` candidate is verified (stat'd) before being treated as the path to load — silent fall-through to the local default avoids reporting a confusing single-path error.
- No change to success-path behavior, precedence rules, or public surface (CLI flags, env vars, README, config schema).

**Non-Goals:**

- Reworking how `--config` / `PKGPROXY_CONFIG` precedence is evaluated.
- Adding new lookup locations (e.g. `$XDG_CONFIG_HOME`, `/etc/pkgproxy/`).
- Changing what `LoadConfig` itself reports when the file exists but is malformed.

## Decisions

### Decision 1: Surface all attempted candidates in the failure error, not just the last one

The wrapped error from `initConfig` becomes:

```
unable to load configuration; tried: ./pkgproxy.yaml, /var/run/ko/pkgproxy.yaml: <underlying error>
```

`resolveConfigPath` is reshaped to return both the path to load and the list of candidates it considered. `initConfig` includes that list in its wrapped error.

**Alternative considered:** keep `resolveConfigPath`'s `(string, error)` signature and have `initConfig` re-derive the candidate list. Rejected — duplicating the lookup logic in two places invites them drifting apart, and the resolver already knows the answer.

**Alternative considered:** include the candidate list only when `len(candidates) > 1`. Rejected — the cost of always listing is one path in single-candidate cases, which is clearer than a conditional format.

### Decision 2: Stat the `$KO_DATA_PATH` candidate before returning it

If `$KO_DATA_PATH/pkgproxy.yaml` does not exist as a regular file, `resolveConfigPath` returns `defaultConfigPath` as the path to load (matching what would happen if `KO_DATA_PATH` were unset). The candidate list still records both paths that were checked, so the eventual error message remains informative.

**Alternative considered:** return the (unverified) ko path and let `LoadConfig` fail. Rejected — pairs poorly with Decision 1, because the resolver should pick the most-likely-to-succeed path when both candidates miss; reporting `./pkgproxy.yaml` is more meaningful to an operator running locally.

### Decision 3: Update the existing `config-discovery` capability spec, not introduce a new one

This is a refinement of an already-shipped capability. The spec delta uses `MODIFIED Requirements` for the lookup requirement (the failure-mode clause changes) and revises the `KO_DATA_PATH set but no file present` scenario in place. No new requirement is introduced.

## Risks / Trade-offs

- **Risk:** existing test `"KO_DATA_PATH set but no file returns ko path"` in `cmd/config_test.go` pins the old behavior and will fail.
  - **Mitigation:** rewrite that case to assert the new return value (`defaultConfigPath`) and add a new test asserting the wrapped error from `initConfig` enumerates both candidates.
- **Risk:** operators parsing the error string in scripts (unlikely, but possible) break.
  - **Mitigation:** the change is documented in `CHANGELOG.md` under `[Unreleased]`; the surface is a CLI error message, not a contract.
- **Trade-off:** the resolver now returns a small struct/tuple instead of a single string, marginally complicating the call site. Worth it for the single source of truth on candidate paths.

## Migration Plan

No migration needed — the change is a pure error-message and fallback-path refinement. No data, configuration, or persisted state is affected. Rollback is a single `git revert`.
