## ADDED Requirements

### Requirement: Trust proxy is resolved from flag, then env var, then empty default
The `serve` subcommand SHALL resolve the trust-proxy value using the following ordered precedence:

1. The value of `--trust-proxy` when the user explicitly passed the flag on the command line (detected via Cobra's `cmd.Flag("trust-proxy").Changed` returning `true`).
2. The value of the `PKGPROXY_TRUST_PROXY` environment variable when it is set to a non-empty string.
3. The built-in default: empty string (no trust).

An empty `PKGPROXY_TRUST_PROXY` (set but empty, or unset) SHALL be treated as "no env-var input" and SHALL fall through to step 3.

#### Scenario: Explicit flag overrides env var
- **WHEN** the binary is started with `serve --trust-proxy=loopback` and `PKGPROXY_TRUST_PROXY=private` is set
- **THEN** only loopback addresses SHALL be trusted for X-Forwarded-For

#### Scenario: Env var used when flag is absent
- **WHEN** the binary is started with `serve` (no `--trust-proxy`) and `PKGPROXY_TRUST_PROXY=private` is set
- **THEN** RFC1918 and RFC4193 private ranges SHALL be trusted for X-Forwarded-For

#### Scenario: Empty env var falls through to default
- **WHEN** the binary is started with `serve` (no `--trust-proxy`) and `PKGPROXY_TRUST_PROXY=` (set but empty)
- **THEN** X-Forwarded-For SHALL be ignored (direct IP extraction)

#### Scenario: Neither flag nor env var produces no-trust default
- **WHEN** the binary is started with `serve` (no `--trust-proxy`) and `PKGPROXY_TRUST_PROXY` is unset
- **THEN** X-Forwarded-For SHALL be ignored (direct IP extraction)

### Requirement: XFF trust is disabled by default
When `--trust-proxy` resolves to an empty string or the value `none`, the server SHALL install `echo.ExtractIPDirect()` as the IP extractor. The `X-Forwarded-For` header SHALL be ignored entirely; `c.RealIP()` SHALL return the direct network peer's IP address.

#### Scenario: Default behavior ignores XFF
- **WHEN** `--trust-proxy` is unset and a request arrives with `X-Forwarded-For: 1.2.3.4` from `127.0.0.1`
- **THEN** the access-log `remote_ip` field SHALL be `127.0.0.1`

#### Scenario: Explicit `none` also disables XFF
- **WHEN** `--trust-proxy=none` and a request arrives with `X-Forwarded-For: 1.2.3.4` from `127.0.0.1`
- **THEN** the access-log `remote_ip` field SHALL be `127.0.0.1`

### Requirement: `loopback` keyword trusts loopback addresses
When the resolved value contains the entry `loopback`, the server SHALL apply `echo.TrustLoopback(true)` so that `X-Forwarded-For` headers arriving from `127.0.0.0/8` and `::1` are honored.

#### Scenario: Loopback source with XFF surfaced
- **WHEN** `--trust-proxy=loopback` and a request arrives from `127.0.0.1` with `X-Forwarded-For: 203.0.113.5`
- **THEN** the access-log `remote_ip` field SHALL be `203.0.113.5`

#### Scenario: Non-loopback source not trusted for XFF
- **WHEN** `--trust-proxy=loopback` and a request arrives from `10.0.0.1` with `X-Forwarded-For: 203.0.113.5`
- **THEN** the access-log `remote_ip` field SHALL be `10.0.0.1` (the direct peer, not the XFF value)

### Requirement: `private` keyword trusts RFC1918 and RFC4193 private ranges
When the resolved value contains the entry `private`, the server SHALL apply `echo.TrustPrivateNet(true)` so that `X-Forwarded-For` headers arriving from RFC1918 (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) and RFC4193 (`fc00::/7`) ranges are honored. Loopback addresses are NOT implied by `private` alone.

#### Scenario: Private-range source with XFF surfaced
- **WHEN** `--trust-proxy=private` and a request arrives from `10.0.0.1` with `X-Forwarded-For: 203.0.113.5`
- **THEN** the access-log `remote_ip` field SHALL be `203.0.113.5`

#### Scenario: Loopback source not trusted with private-only setting
- **WHEN** `--trust-proxy=private` and a request arrives from `127.0.0.1` with `X-Forwarded-For: 203.0.113.5`
- **THEN** the access-log `remote_ip` field SHALL be `127.0.0.1` (loopback not in private ranges)

### Requirement: Literal CIDR or bare IP trusts exactly that range
When the resolved value contains a CIDR notation entry (e.g. `10.0.0.5/32`, `192.168.0.0/24`) or a bare IP address (auto-promoted to `/32` for IPv4, `/128` for IPv6), the server SHALL apply `echo.TrustIPRange(...)` for that range only.

#### Scenario: Exact-host CIDR trusts only that host
- **WHEN** `--trust-proxy=10.0.0.5/32` and a request arrives from `10.0.0.5` with `X-Forwarded-For: 203.0.113.5`
- **THEN** the access-log `remote_ip` field SHALL be `203.0.113.5`

#### Scenario: Sibling in the subnet not trusted if not in the CIDR
- **WHEN** `--trust-proxy=10.0.0.5/32` and a request arrives from `10.0.0.6` with `X-Forwarded-For: 203.0.113.5`
- **THEN** the access-log `remote_ip` field SHALL be `10.0.0.6`

#### Scenario: Bare IP is promoted to single-host CIDR
- **WHEN** `--trust-proxy=192.168.1.10` (no mask) and a request arrives from `192.168.1.10` with `X-Forwarded-For: 203.0.113.5`
- **THEN** the access-log `remote_ip` field SHALL be `203.0.113.5`

#### Scenario: Bare IPv6 is promoted to /128
- **WHEN** `--trust-proxy=::1` (no mask) and a request arrives from `::1` with `X-Forwarded-For: 203.0.113.5`
- **THEN** the access-log `remote_ip` field SHALL be `203.0.113.5`

### Requirement: Multiple entries may be combined (comma-separated)
The value of `--trust-proxy` SHALL be split on commas; whitespace around each entry SHALL be trimmed; each entry is evaluated independently. Keywords and literal CIDRs/IPs may be mixed freely, provided `none` is not present.

#### Scenario: Keyword and CIDR combined
- **WHEN** `--trust-proxy=loopback,10.0.0.0/8` and a request arrives from `10.5.5.5` with `X-Forwarded-For: 203.0.113.5`
- **THEN** the access-log `remote_ip` field SHALL be `203.0.113.5`

#### Scenario: Whitespace around entries is tolerated
- **WHEN** `--trust-proxy= loopback , 10.0.0.0/8 ` (spaces around commas)
- **THEN** the server SHALL start without error

### Requirement: `none` may not be combined with other entries
When the resolved value contains `none` alongside any other non-empty entry, the server SHALL fail at startup with an error stating that `none` cannot be combined with other trust entries.

#### Scenario: `none` combined with keyword causes startup failure
- **WHEN** `PKGPROXY_TRUST_PROXY=none,loopback`
- **THEN** the server SHALL exit before accepting connections with a non-zero status and an error message referencing `none`

### Requirement: Unrecognized or malformed entries cause startup failure
Any entry that is not a recognized keyword (`none`, `loopback`, `private`), a valid CIDR, or a parseable IP address SHALL cause the server to fail at startup. The error message SHALL name the offending token.

#### Scenario: Unrecognized keyword causes startup failure
- **WHEN** `--trust-proxy=garbage`
- **THEN** the server SHALL exit with a non-zero status and an error message containing `garbage`

#### Scenario: Malformed CIDR causes startup failure
- **WHEN** `--trust-proxy=10.0.0.0/8,not-an-ip`
- **THEN** the server SHALL exit with a non-zero status and an error message containing `not-an-ip`

### Requirement: Resolved trust mode is logged at startup
The server SHALL emit a structured log line at `INFO` level during startup that records the effective trust-proxy setting (the raw resolved string, or `none` when unset/empty). This line SHALL appear before the server begins accepting connections.

#### Scenario: Trust mode logged when set
- **WHEN** `--trust-proxy=loopback` and the server starts successfully
- **THEN** a startup log line SHALL contain the key `trust-proxy` with value `loopback`

#### Scenario: Trust mode logged as `none` by default
- **WHEN** `--trust-proxy` is unset and the server starts successfully
- **THEN** a startup log line SHALL contain the key `trust-proxy` with value `none`
