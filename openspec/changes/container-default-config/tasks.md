## 1. Default-subcommand shim

- [ ] 1.1 In `cmd/root.go` (or `main.go`), add a helper called from `Execute()` that prepends `"serve"` to `os.Args` when `len(os.Args) == 1`
- [ ] 1.2 Add unit tests in a new `cmd/root_test.go` covering: zero args → serve inserted; `--help` → unchanged; `version` → unchanged; explicit `serve` → unchanged; bare flag like `--debug` → unchanged (shim does not fire). Test the shim helper directly rather than going through `Execute()`.

## 2. Config-discovery fallback

- [ ] 2.1 Extract the default-path resolution logic in `cmd/root.go` `initConfig()` into a small helper that implements the ordered lookup (`./pkgproxy.yaml` exists → use it; else if `$KO_DATA_PATH` set → `$KO_DATA_PATH/pkgproxy.yaml`; else → `./pkgproxy.yaml` as before, so the missing-file error message remains stable)
- [ ] 2.2 Use `os.Stat` with `errors.Is(err, os.ErrNotExist)` to distinguish missing from unreadable; non-NotExist stat errors propagate
- [ ] 2.3 Preserve existing precedence: explicit `--config` flag wins over `$PKGPROXY_CONFIG` wins over the ordered lookup
- [ ] 2.4 Add unit tests for all scenarios in `specs/config-discovery/spec.md` in a new `cmd/config_test.go`, testing the extracted helper directly. Use `t.TempDir()` for file presence/absence and `t.Setenv()` for env vars. Scenarios: local wins over ko; ko fallback; explicit flag bypass; env-var bypass; both missing; `KO_DATA_PATH` set but no file.

## 3. Documentation

- [ ] 3.1 Update `README.md` "Run the code" section: replace the long podman invocations with `podman run --rm -p 8080:8080 ghcr.io/ganto/pkgproxy` as the primary example; keep a bind-mount override example for users supplying their own config
- [ ] 3.2 Verify the flags table in `README.md` still reads correctly (no change to flag defaults themselves)
- [ ] 3.3 Grep `pkg/pkgproxy/landing.go` and landing-page templates for any hard-coded container invocation examples; update if found, otherwise record that none exist
- [ ] 3.4 Add a `[Unreleased]` entry in `CHANGELOG.md` (80–100 chars): e.g. "container image now runs `serve` by default and loads bundled config from `$KO_DATA_PATH`"

## 4. End-to-end verification

- [ ] 4.1 Build the image locally with `make image-build` and run `podman run --rm -p 8080:8080 ko.local/pkgproxy:<tag>`; verify a GET to `/` returns the landing page and a cache-passthrough GET to a small package URL succeeds
- [ ] 4.2 Run `podman run --rm -p 8080:8080 --volume $PWD/myconfig.yaml:/ko-app/pkgproxy.yaml ko.local/pkgproxy:<tag> serve --config /ko-app/pkgproxy.yaml` and verify the user-supplied config is honored (override path still works)
- [ ] 4.3 Run `podman run --rm ko.local/pkgproxy:<tag> --help` and verify help output is printed, no server starts
- [ ] 4.4 Run `podman run --rm ko.local/pkgproxy:<tag> version` and verify version output is printed
- [ ] 4.5 Run the existing e2e suite: `make e2e` — must pass unchanged (the tests invoke `pkgproxy serve --config configs/pkgproxy.yaml` explicitly, so no behavior change is expected)

## 5. CI-check and release prep

- [ ] 5.1 `make ci-check` passes (lint + govulncheck + tests)
- [ ] 5.2 Manually review the CHANGELOG entry for length and user-facing phrasing
