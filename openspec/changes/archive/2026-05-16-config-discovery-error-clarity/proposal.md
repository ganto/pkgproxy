## Why

When `KO_DATA_PATH` is set in the operator's environment but no `pkgproxy.yaml` exists under it, the config lookup currently fails with an error pointing at the (unverified) ko path — e.g. `unable to load configuration from /var/run/ko/pkgproxy.yaml: no such file or directory`. An operator running pkgproxy outside a container may have inherited `KO_DATA_PATH` from a parent shell and not realize the lookup ever attempted that path, leaving them confused about where the error originated and which file they were expected to provide.

## What Changes

- When the ordered config-file lookup exhausts all candidates without finding a readable file, the resulting error SHALL enumerate every candidate path that was tried (in order), not just the last one.
- The `LoadConfig` call site SHALL receive a candidate path that is known to exist where possible — `resolveConfigPath` will stat the `$KO_DATA_PATH/pkgproxy.yaml` candidate and only return it when it exists, otherwise fall through to the local default path for the final attempt.
- No CLI flags, env vars, or precedence rules change. Behavior is unchanged in the success cases (both for local checkouts and ko-built containers).

## Capabilities

### New Capabilities

_None — this change refines the existing `config-discovery` capability._

### Modified Capabilities

- `config-discovery`: the failure scenario where no config file is found is tightened — the error must name all attempted candidates rather than just the last one, and the `KO_DATA_PATH` candidate must be verified before it is returned as the path to load.

## Impact

- `cmd/root.go`: `resolveConfigPath` and `initConfig` (error-wrapping site)
- `cmd/config_test.go`: existing test case `"KO_DATA_PATH set but no file returns ko path"` will be revised, and a new case will assert the multi-candidate error message
- No changes to public CLI surface, configuration schema, README, or CHANGELOG beyond a one-line entry under `[Unreleased]`
