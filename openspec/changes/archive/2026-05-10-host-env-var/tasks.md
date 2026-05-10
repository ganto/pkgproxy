## 1. Resolver helper and wiring

- [x] 1.1 Add `hostEnvVar = "PKGPROXY_HOST"` constant in `cmd/serve.go` near `publicHostEnvVar`
- [x] 1.2 Add `resolveListenHost(flagChanged bool, flagValue, envValue string) string` helper in `cmd/serve.go` implementing the flag → env → default precedence
- [x] 1.3 Update `PersistentPreRunE` in `newServeCommand()` to call `resolveListenHost(cmd.Flag("host").Changed, listenAddress, os.Getenv(hostEnvVar))` and assign the result back to `listenAddress` before calling `initConfig()`

## 2. Unit tests

- [x] 2.1 Add `TestResolveListenHost` to `cmd/serve_test.go` mirroring the table-driven style of `TestResolvePublicAddr`
- [x] 2.2 Cover scenarios: flag changed wins over env var, flag changed wins even when value equals default, env var used when flag unchanged, empty env var falls through to default, neither set returns default
- [x] 2.3 Run `go test ./cmd/... -run TestResolveListenHost` and confirm all subtests pass

## 3. Documentation

- [x] 3.1 In `README.md`, add `PKGPROXY_HOST` to the `Env Variable` column of the `--host` row in the flags table
- [x] 3.2 In `README.md`, add a short sentence near the flags table noting that any env var listed in the table may be set in the environment instead of passing the flag
- [x] 3.3 In `README.md`, update the two container-run examples (the `podman run …` snippets near the top under "Run the code") to use `-e PKGPROXY_HOST=0.0.0.0` and drop the `serve --host 0.0.0.0` arguments
- [x] 3.4 Add a concise (80–100 char) entry under `## [Unreleased]` in `CHANGELOG.md` describing the new env var

## 4. Validation

- [x] 4.1 Run `make ci-check` and confirm lint, govulncheck, and unit tests all pass
- [x] 4.2 Run `pre-commit run --all-files` and resolve any findings
- [x] 4.3 Run `make e2e` (or at minimum one distro, e.g. `make e2e DISTRO=fedora`) to confirm nothing in the e2e flow regressed

## 5. Manual verification

Verify the headline UX goal end-to-end. Run from a clean shell so leftover env vars don't influence results.

- [x] 5.1 Build the binary: `make build`
- [x] 5.2 Default behavior (no flag, no env): `./bin/pkgproxy serve` in one shell, then in another run `ss -tlnp | grep 8080` (or `curl -sI http://127.0.0.1:8080/` and `curl -sI --connect-timeout 2 http://<host-LAN-ip>:8080/`). Expect a listener on `127.0.0.1:8080` only — connections from a non-loopback address must fail.
- [x] 5.3 Env var overrides default: `PKGPROXY_HOST=0.0.0.0 ./bin/pkgproxy serve`. Expect the listener to be on `0.0.0.0:8080` and a `curl` from a non-loopback address to succeed.
- [x] 5.4 Explicit flag wins over env: `PKGPROXY_HOST=0.0.0.0 ./bin/pkgproxy serve --host localhost`. Expect a listener on `127.0.0.1:8080` only (env var is ignored).
- [x] 5.5 Empty env var falls through: `PKGPROXY_HOST= ./bin/pkgproxy serve`. Expect the same listener as 5.2 (`127.0.0.1:8080` only).
- [x] 5.6 Container scenario (the original motivation): `make image-build`, then `podman run --rm -p 8080:8080 -e PKGPROXY_HOST=0.0.0.0 ko.local/pkgproxy:<tag>` (use the tag printed by `image-build`), and from the host run `curl -sI http://127.0.0.1:8080/`. Expect a `200 OK` for the landing page. Repeat the same `podman run` without `-e PKGPROXY_HOST=…` and confirm the curl now fails or hangs (server bound to container-local `localhost`, unreachable from the host).
