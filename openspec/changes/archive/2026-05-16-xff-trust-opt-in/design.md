## Context

`cmd/serve.go:93` installs `echo.ExtractIPFromXFFHeader()` with no `TrustOption` arguments. Echo v5.1.1's defaults then trust loopback (`127.0.0.0/8`, `::1`), link-local (`169.254.0.0/16`, `fe80::/10`), and all RFC1918/RFC4193 private ranges. In container bridge deployments the direct connecting peer is the bridge gateway (e.g. `172.17.0.1`) — inside the trusted private range — so any client can set `X-Forwarded-For` to an arbitrary IP that `c.RealIP()` at `cmd/serve.go:123` will reflect in the `remote_ip` access-log field.

`cmd/serve.go` already follows a hand-rolled flag → env-var → default precedence pattern for `--host` and `--public-host`, with no viper dependency. Echo v5.1.1 exports the full API surface needed: `TrustLoopback(bool)`, `TrustPrivateNet(bool)`, `TrustIPRange(*net.IPNet)`, `ExtractIPFromXFFHeader(...TrustOption)`, and `ExtractIPDirect()`.

## Goals / Non-Goals

**Goals:**
- Replace implicit echo XFF trust with an explicit, operator-controlled opt-in.
- Default to no XFF trust (`ExtractIPDirect`) — the server's behavior is safe out of the box.
- Provide ergonomic keyword shortcuts (`loopback`, `private`) for the two most common deployment topologies.
- Fail startup fast with a clear, actionable error on misconfiguration.
- Remain consistent with the existing flag/env-var/resolver pattern in `cmd/serve.go`.

**Non-Goals:**
- `X-Real-IP` header support (separate concern; not requested).
- Automatic trust defaulting based on the listen address value.
- Per-repository or per-path trust configuration.
- A `linklocal` keyword (link-local addresses are not a real HTTP service-connection scenario for pkgproxy).
- Any change to cache, mirror selection, or forwarding logic.

## Decisions

### 1. Strict-by-default (no XFF trust unless configured)

**Chosen:** When `--trust-proxy` is unset or evaluates to empty/`none`, install `echo.ExtractIPDirect()`. Echo's implicit defaults are never applied.

**Alternatives considered:**
- *Infer trust from `--host`*: loopback trust when `--host=localhost`; private trust for all other bind addresses; no trust for public-IP binds. Rejected — coupling two orthogonal flags makes the contract surprising (changing `--host` silently changes trust), and `0.0.0.0` is ambiguous (binds everything but is the common container default).
- *Keep echo's defaults, add the flag as an override*: existing deployments keep working, new operators learn they can tighten trust. Rejected — the current default is the bug; a non-breaking fix leaves the original problem in place indefinitely.

### 2. Keyword tokens alongside literal CIDRs/IPs

**Chosen:** Accept `loopback` (→ `echo.TrustLoopback(true)`) and `private` (→ `echo.TrustPrivateNet(true)`) as convenience keywords. Literal CIDRs and bare IPs (promoted to `/32`/`/128`) are accepted alongside or instead.

**Rationale:** The two most common reverse-proxy topologies (same-host, LAN) map exactly to `loopback` and `private`. Without keywords, operators must either look up the RFC1918 ranges or supply `127.0.0.0/8,::1/128`. Keywords reduce friction while preserving the option of tighter control via exact CIDRs. `linklocal` is omitted because IPv4 169.254/16 and IPv6 fe80::/10 do not arise in normal service-to-service communication.

### 3. `StringVar` (comma-separated), not `StringSliceVar`

**Chosen:** A single `StringVar` flag whose value is comma-separated.

**Rationale:** Consistent with `--host`, `--public-host`, and other flags in this codebase. `StringSliceVar` introduces subtle cobra behavior (shell splitting, quoting) and its env-var representation must still be a delimited string, so it adds complexity without benefit. The custom parser trims whitespace and handles both `a,b` and `a, b` forms.

### 4. Package-level `ipExtractor` var set in `PersistentPreRunE`

**Chosen:** Resolve and parse `--trust-proxy` inside `PersistentPreRunE` (where `listenAddress` and `initConfig` are also resolved), storing the resulting `echo.IPExtractor` in a package-level variable consumed by `startServer`.

**Rationale:** Matches the pattern used for `listenAddress`. Keeps `startServer` free of resolution logic and makes the resolver and parser individually testable without running the full command.

### 5. `none` as explicit no-trust keyword; empty value treated as `none`

**Chosen:** `--trust-proxy=none` installs `ExtractIPDirect`, identical to unset. Empty string (unset flag, empty env var) has the same effect. Mixing `none` with any other entry is a startup error.

**Rationale:** An explicit `none` lets operators document intent in deployment config rather than relying on an absent flag. Treating empty as `none` prevents accidents from an env var that is set but blank in shell scripts.

## Risks / Trade-offs

- **Breaking behavior change** → Mitigated by a prominent `CHANGELOG.md` entry and a startup log line showing the resolved trust mode. Operators behind a reverse proxy need one-time migration (`PKGPROXY_TRUST_PROXY=private` or a specific CIDR).
- **`private` keyword retains the original container-bridge risk** → By design; the operator who sets `private` accepts the lenient posture. The key improvement is that this is now an explicit choice rather than a silent default.
- **Bare IP promotion could surprise users** → `10.0.0.5` becomes `10.0.0.5/32`; trusting only that exact host may be unexpected if the operator meant a subnet. Startup log shows the resolved mode string, which helps diagnose.

## Migration Plan

| Deployment topology | Required action |
|---------------------|-----------------|
| No reverse proxy | None — default (no XFF) is correct. |
| Same-host reverse proxy (nginx/caddy on localhost) | Add `PKGPROXY_TRUST_PROXY=loopback`. |
| LAN reverse proxy (different host, private network) | Add `PKGPROXY_TRUST_PROXY=<proxy-ip>/32` (preferred) or `PKGPROXY_TRUST_PROXY=private` (lenient). |
| Container, no reverse proxy | None — default prevents any XFF spoofing. |

Rollback: revert the flag definition and IPExtractor wiring in `cmd/serve.go`; echo's defaults re-apply. No data migration involved.
