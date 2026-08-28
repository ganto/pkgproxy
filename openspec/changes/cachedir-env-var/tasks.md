## 1. Resolver helper and wiring

- [ ] 1.1 Add `cachedirEnvVar = "PKGPROXY_CACHEDIR"` constant in `cmd/root.go` next to `configPathEnvVar` / `defaultDir`
- [ ] 1.2 Add `resolveCacheDir(flagChanged bool, flagValue, envValue string) string` helper in `cmd/root.go` implementing flag → env → default (`defaultDir`) precedence
- [ ] 1.3 In `cmd/serve.go` `newServeCommand()` `PersistentPreRunE`, call `cacheDir = resolveCacheDir(cmd.Flag("cachedir").Changed, cacheDir, os.Getenv(cachedirEnvVar))` before `initConfig()`

## 2. Unit tests

- [ ] 2.1 Add `TestResolveCacheDir` (in `cmd/root_test.go`, or `cmd/serve_test.go` if it fits the existing resolver tests better) mirroring the table-driven style of `TestResolveListenHost`
- [ ] 2.2 Cover: flag changed wins over env var; flag changed wins even when value equals default; env var used when flag unchanged; empty env var falls through to default; neither set returns `cache`
- [ ] 2.3 Run `go test ./cmd/... -run TestResolveCacheDir` and confirm all subtests pass

## 3. Documentation

- [ ] 3.1 In `README.md`, add `PKGPROXY_CACHEDIR` to the env-var column of the `--cachedir` row in the flags table
- [ ] 3.2 Add a concise (80–100 char) entry under `## [Unreleased]` → `### Added` in `CHANGELOG.md` for the new env var

## 4. Validation

- [ ] 4.1 Run `make ci-check` and confirm lint, govulncheck, and unit tests pass
- [ ] 4.2 Run `pre-commit run --all-files` and resolve any findings
- [ ] 4.3 Run `make e2e DISTRO=fedora` (at minimum) to confirm the e2e flow did not regress

## 5. Manual verification

Run from a clean shell so leftover env vars don't influence results.

- [ ] 5.1 Build: `make build`
- [ ] 5.2 Default (no flag, no env): `./bin/pkgproxy serve`, fetch a package, confirm it lands under `./cache/`
- [ ] 5.3 Env var overrides default: `PKGPROXY_CACHEDIR=/tmp/pp-cache ./bin/pkgproxy serve`, fetch a package, confirm it lands under `/tmp/pp-cache/`
- [ ] 5.4 Explicit flag wins over env: `PKGPROXY_CACHEDIR=/tmp/pp-cache ./bin/pkgproxy serve --cachedir ./cache`, confirm cache writes go to `./cache/` and `/tmp/pp-cache/` stays empty
- [ ] 5.5 Empty env var falls through: `PKGPROXY_CACHEDIR= ./bin/pkgproxy serve`, confirm same behavior as 5.2
