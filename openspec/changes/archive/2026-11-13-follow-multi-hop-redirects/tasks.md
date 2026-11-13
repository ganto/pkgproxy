## 1. Setup

- [x] 1.1 Create a branch `fix/follow-multi-hop-redirects` off `main`, leaving the unrelated uncommitted local changes untouched

## 2. Tests (red)

- [x] 2.1 Add unit test: a two-hop redirect chain (A → B → C) returns 200 and the `.rpm` file is cached
- [x] 2.2 Add unit test: a redirect loop on the only mirror returns 502, not 3xx
- [x] 2.3 Add unit test: mirror 1 exceeds the redirect limit and mirror 2 returns 200
- [x] 2.4 Add unit test: a TLS mirror (`httptest.NewTLSServer`, injected `Transport`) redirecting to `http` returns 502 and the `http` target is never requested
- [x] 2.5 Confirm the new tests fail (except where current behaviour already passes) and the existing redirect tests still pass

## 3. Implementation

- [x] 3.1 Add the `maxRedirects = 10` constant in `pkg/pkgproxy/proxy.go`
- [x] 3.2 Replace the single redirect `if` in `tryMirrors` with a loop bounded by `maxRedirects`, logging each hop
- [x] 3.3 Reject HTTPS → HTTP hops; on hop-limit or downgrade failure, close the response, set `rsp = nil` and `err`, log a warning, and move to the next mirror
- [x] 3.4 Update the `tryMirrors` doc comment to describe multi-hop following

## 4. Docs

- [x] 4.1 Update `docs/architecture.md`: up to 10 redirects per attempt, HTTPS → HTTP downgrades rejected
- [x] 4.2 Add `CHANGELOG.md` `[Unreleased]` entries: **Fixed** "Follow multi-hop upstream redirects, restoring package caching for Fedora COPR repositories" and **Changed** "Reject upstream redirects that downgrade from HTTPS to HTTP"

## 5. Verification

- [x] 5.1 Run `make ci-check` and confirm it passes
- [x] 5.2 Run `make e2e` and confirm all distros pass, including the four `*/COPR` subtests
- [x] 5.3 Request a Codex review of the diff and address findings
