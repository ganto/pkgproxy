## Why

`cmd/serve.go` installs `echo.ExtractIPFromXFFHeader()` with no trust options, so echo trusts loopback, link-local, and all RFC1918/RFC4193 private ranges by default. In a typical container deployment (`podman run -p 8080:8080`), every external request arrives via the bridge gateway in private address space — meaning any client can inject an `X-Forwarded-For` header and have its value reflected in the `remote_ip` access-log field. The impact today is log-integrity only, but the implicit trust should be replaced with an explicit, opt-in mechanism before `RealIP()` ever feeds an authorization decision.

## What Changes

- Add a `--trust-proxy` CLI flag on the `serve` subcommand with a `PKGPROXY_TRUST_PROXY` environment variable fallback.
- The flag accepts a comma-separated list whose entries are one of: `none`, `loopback`, `private`, a CIDR (e.g. `10.0.0.0/8`), or a bare IP (auto-promoted to `/32` for IPv4 or `/128` for IPv6).
- When the flag is unset, empty, or set to `none`, the server SHALL install `echo.ExtractIPDirect()` — the `X-Forwarded-For` header is ignored entirely.
- When the flag carries any other valid value, the server SHALL install `echo.ExtractIPFromXFFHeader(...)` configured with **only** the operator-supplied trust options. Echo's implicit defaults (loopback / link-local / private) SHALL NOT apply.
- Mixing `none` with any other entry SHALL be an error at startup.
- Invalid entries (unrecognized keyword, malformed CIDR, unparseable IP) SHALL fail startup with a clear error naming the offending token.
- The resolved trust mode SHALL be logged once at startup so operators can confirm what was applied.
- **BREAKING**: Deployments that today rely on echo's implicit private-net trust to populate `remote_ip` from `X-Forwarded-For` will see that field switch to the direct connecting peer until they set `PKGPROXY_TRUST_PROXY` (commonly `private` for LAN reverse proxies, `loopback` for same-host reverse proxies, or a specific CIDR for tightest control).

## Capabilities

### New Capabilities
- `trusted-proxies-config`: Resolution and parsing of the `--trust-proxy` flag / `PKGPROXY_TRUST_PROXY` env var, and selection of the corresponding `echo.IPExtractor` for the HTTP server.

### Modified Capabilities
- _(none — no existing spec covers XFF/IPExtractor behavior; this is a purely additive capability.)_

## Impact

- **Code**: `cmd/serve.go` (flag registration, resolver, parser, `IPExtractor` wiring, startup log line), `cmd/serve_test.go` (table-driven tests for the resolver and parser).
- **Docs**: `README.md` (new row in the CLI flags table plus a short subsection explaining the keywords and common recipes), `CHANGELOG.md` (Unreleased entry calling out both the new flag and the default-behavior change).
- **Runtime behavior**: Default access-log `remote_ip` field changes for operators behind a reverse proxy until they opt in. No effect on cache, mirror selection, or forwarding logic.
- **Dependencies**: No new third-party dependencies; uses `echo.TrustLoopback`, `echo.TrustPrivateNet`, and `echo.TrustIPRange` already exported by `github.com/labstack/echo/v5` v5.1.1.
