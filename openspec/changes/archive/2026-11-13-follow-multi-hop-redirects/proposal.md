## Why

Fedora COPR now serves downloads through a two-hop redirect chain (`download.copr.fedorainfracloud.org` → `packages.redhat.com` Pulp → S3). pkgproxy follows only one redirect per mirror attempt, so it passes the second `302` through to the client. The client then downloads directly from upstream and nothing is cached. The `*/COPR` e2e subtests fail on Fedora, CentOS Stream, AlmaLinux, and Rocky Linux.

## What Changes

- Follow up to 10 upstream redirects (301/302/303/307/308) per mirror attempt instead of one.
- Reject any redirect hop that downgrades from HTTPS to HTTP. **BREAKING** for mirrors that currently redirect from `https` to plain `http`: the mirror is now treated as failed.
- When a mirror's redirect chain exceeds the limit or is rejected, mark that mirror as failed and try the next one. If every mirror fails, respond with 502 Bad Gateway. The proxy never passes a 3xx response to the client.

## Capabilities

### New Capabilities
- `upstream-redirects`: how the proxy follows upstream redirects per mirror attempt: hop limit, downgrade protection, and failure handling.

### Modified Capabilities

## Impact

- Code: `tryMirrors` in `pkg/pkgproxy/proxy.go`. Unit tests in `pkg/pkgproxy/proxy_test.go`.
- Docs: `docs/architecture.md` (mirror/redirect paragraph), `CHANGELOG.md` `[Unreleased]`.
- No new configuration options, no dependency changes. README is unchanged.
- The existing COPR e2e subtests serve as the regression check.
