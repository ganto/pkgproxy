## 1. Flag wiring

- [ ] 1.1 Add `var trustProxy string` and `const trustProxyEnvVar = "PKGPROXY_TRUST_PROXY"` to `cmd/serve.go`
- [ ] 1.2 Register `--trust-proxy` flag in `newServeCommand()` with empty default and a description referencing `none`, `loopback`, `private`, and CIDR/IP forms
- [ ] 1.3 Add `resolveTrustProxy(flagChanged bool, flagValue, envValue string) string` using the same flag→env→default pattern as `resolveListenHost`
- [ ] 1.4 Call `resolveTrustProxy` in `PersistentPreRunE` (after `listenAddress` resolution), pass result to `parseTrustProxy`, store the returned `echo.IPExtractor` in a package-level `var ipExtractor echo.IPExtractor`

## 2. Parser implementation

- [ ] 2.1 Implement `parseTrustProxy(value string) (echo.IPExtractor, error)` in `cmd/serve.go`:
  - Split on `,`, trim whitespace, lowercase, discard empties
  - Empty list → return `echo.ExtractIPDirect()`
  - `[none]` → return `echo.ExtractIPDirect()`
  - `none` present with other entries → error: "trust-proxy: 'none' cannot be combined with other entries"
  - `loopback` entry → append `echo.TrustLoopback(true)`
  - `private` entry → append `echo.TrustPrivateNet(true)`
  - Otherwise try `net.ParseCIDR`; if it fails try `net.ParseIP` and promote to `/32` (IPv4) or `/128` (IPv6); if both fail → error naming the bad token
  - Return `echo.ExtractIPFromXFFHeader(opts...)`

## 3. Server wiring and logging

- [ ] 3.1 Replace `cmd/serve.go:93` (`app.IPExtractor = echo.ExtractIPFromXFFHeader()`) with `app.IPExtractor = ipExtractor`
- [ ] 3.2 Update the comment at lines 91–92 to describe the opt-in model
- [ ] 3.3 Add a `slog.Info("trust-proxy", "value", ...)` log line in `startServer` near the existing startup log, showing the raw resolved trust-proxy string (or `"none"` when empty)

## 4. Tests

- [ ] 4.1 Add `TestResolveTrustProxy` in `cmd/serve_test.go` (table-driven, mirrors `TestResolveListenHost`): flag-changed wins; env-var used when flag absent; empty env falls through to empty default
- [ ] 4.2 Add `TestParseTrustProxy` in `cmd/serve_test.go` (table-driven), covering:
  - Empty string → `ExtractIPDirect` behavior (XFF ignored)
  - `"none"` → same as empty
  - `"loopback"` → loopback source honored, private source not trusted
  - `"private"` → private source honored, loopback source not trusted
  - `"10.0.0.0/8"` → matching source honored, non-matching not trusted
  - `"192.168.1.10"` (bare IPv4) → promoted to `/32`, only exact host trusted
  - `"::1"` (bare IPv6) → promoted to `/128`
  - `"loopback,10.0.0.0/8"` → both honored
  - `" loopback , 10.0.0.0/8 "` → whitespace tolerance, no error
  - `"LOOPBACK"` → case-insensitive, no error
  - `"none,loopback"` → error containing "cannot be combined"
  - `"garbage"` → error naming "garbage"
  - `"10.0.0.0/8,not-an-ip"` → error naming "not-an-ip"

## 5. Documentation

- [ ] 5.1 Add `--trust-proxy` row to the CLI flags table in `README.md` (after the `--public-host` row)
- [ ] 5.2 Add a `### Trusting X-Forwarded-For` subsection to `README.md` below the table with: why it's opt-in, common recipes (loopback / specific CIDR / private), and the container-bridge caveat
- [ ] 5.3 Add a combined entry to the `[Unreleased]` section of `CHANGELOG.md` covering the new flag and the breaking default-behavior change

## 6. Verification

- [ ] 6.1 Run `make ci-check` and confirm it passes (lint + govulncheck + unit tests)
- [ ] 6.2 Smoke-test default behavior: start server without `--trust-proxy`, `curl -H 'X-Forwarded-For: 1.2.3.4' http://localhost:8080/`, confirm access log shows `remote_ip` as the connecting IP, not `1.2.3.4`
- [ ] 6.3 Smoke-test opt-in: repeat with `PKGPROXY_TRUST_PROXY=loopback`, confirm log shows `remote_ip: 1.2.3.4`
- [ ] 6.4 Smoke-test fast-fail: start with `PKGPROXY_TRUST_PROXY=garbage`, confirm non-zero exit and error message naming "garbage"
- [ ] 6.5 Run `make e2e DISTRO=fedora` (or another distro) to confirm the default behavior change does not break existing e2e tests
