## 1. Resolver reshape

- [ ] 1.1 Change `resolveConfigPath` in `cmd/root.go` to return both the path to load and the ordered list of candidate paths it considered (e.g. signature `func resolveConfigPath() (path string, candidates []string, err error)`).
- [ ] 1.2 Stat the `$KO_DATA_PATH/pkgproxy.yaml` candidate before returning it as the path to load; on miss, return `defaultConfigPath` as the path to load while keeping both entries in the `candidates` slice.
- [ ] 1.3 Keep the existing fall-through behavior when `./pkgproxy.yaml` is present-but-not-a-regular-file (current `info.Mode().IsRegular()` check).

## 2. Error-wrapping update

- [ ] 2.1 Update `initConfig` in `cmd/root.go` to receive the `candidates` slice from `resolveConfigPath` and include it in the wrapped error of the form `unable to load configuration; tried: <path1>, <path2>: %w` (only emit the multi-candidate form when the ordered lookup ran — explicit `--config` and `PKGPROXY_CONFIG` paths continue to use a single-path error).

## 3. Tests

- [ ] 3.1 Rewrite the `"KO_DATA_PATH set but no file returns ko path"` case in `cmd/config_test.go` (rename appropriately) to assert that `resolveConfigPath` returns `defaultConfigPath` as the load path and a two-element candidate slice (`./pkgproxy.yaml`, `$KO_DATA_PATH/pkgproxy.yaml`).
- [ ] 3.2 Update the other `TestResolveConfigPath` cases to assert the new return signature (including `candidates` content) without changing their underlying intent.
- [ ] 3.3 Add a `TestInitConfig` case asserting the wrapped error contains `tried: ./pkgproxy.yaml, <ko-path>/pkgproxy.yaml` when the ordered lookup exhausts both candidates.
- [ ] 3.4 Add a `TestInitConfig` case asserting the wrapped error contains only the single attempted path when `--config` or `PKGPROXY_CONFIG` is used (no `tried:` prefix needed for that path).

## 4. Verification

- [ ] 4.1 Run `make ci-check` and confirm lint, govulncheck, and unit tests all pass.
- [ ] 4.2 Run `pre-commit run --all-files` to catch formatting and codespell issues.
- [ ] 4.3 Manually verify the new error format by running the built binary in a tempdir with `KO_DATA_PATH=/tmp/no-such-dir` and confirming the operator-facing message names both candidate paths.

## 5. Docs and changelog

- [ ] 5.1 Add a single concise entry under `[Unreleased]` in `CHANGELOG.md` (80–100 chars), framed for pkgproxy users — e.g. that config-file errors now list every default path that was attempted.
- [ ] 5.2 Confirm no README changes are required (no CLI flags, env vars, or config keys changed).
