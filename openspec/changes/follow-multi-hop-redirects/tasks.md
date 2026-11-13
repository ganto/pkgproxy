## 1. Setup

- [ ] 1.1 Create a branch `fix/follow-multi-hop-redirects` off `main`, leaving the unrelated uncommitted local changes untouched

## 2. Tests (red)

- [ ] 2.1 Add unit test: a two-hop redirect chain (A → B → C) returns 200 and the `.rpm` file is cached
- [ ] 2.2 Add unit test: a redirect loop on the only mirror returns 502, not 3xx
- [ ] 2.3 Add unit test: mirror 1 exceeds the redirect limit and mirror 2 returns 200
- [ ] 2.4 Add unit test: a TLS mirror (`httptest.NewTLSServer`, injected `Transport`) redirecting to `http` returns 502 and the `http` target is never requested
- [ ] 2.5 Confirm the new tests fail (except where current behaviour already passes) and the existing redirect tests still pass

## 3. Implementation

- [ ] 3.1 Add the `maxRedirects = 10` constant in `pkg/pkgproxy/proxy.go`
- [ ] 3.2 Replace the single redirect `if` in `tryMirrors` with a loop bounded by `maxRedirects`, logging each hop
- [ ] 3.3 Reject HTTPS → HTTP hops; on hop-limit or downgrade failure, close the response, set `rsp = nil` and `err`, log a warning, and move to the next mirror
- [ ] 3.4 Update the `tryMirrors` doc comment to describe multi-hop following

## 4. Docs

- [ ] 4.1 Update `docs/architecture.md`: up to 10 redirects per attempt, HTTPS → HTTP downgrades rejected
- [ ] 4.2 Add `CHANGELOG.md` `[Unreleased]` entries: **Fixed** "Follow multi-hop upstream redirects, restoring package caching for Fedora COPR repositories" and **Changed** "Reject upstream redirects that downgrade from HTTPS to HTTP"

## 5. Verification

- [ ] 5.1 Run `make ci-check` and confirm it passes
- [ ] 5.2 Run `make e2e` and confirm all distros pass, including the four `*/COPR` subtests
- [ ] 5.3 Request a Codex review of the diff and address findings
