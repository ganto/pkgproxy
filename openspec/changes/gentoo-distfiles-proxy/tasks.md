## 1. Cache exclude feature

- [ ] 1.1 Add `Exclude []string` field to `Repository` struct in `pkg/pkgproxy/repository.go`; in `validateConfig`, if a repository's `suffixes` list contains `"*"` alongside other entries, log a `slog.Warn` naming the repository and the redundant suffixes
- [ ] 1.2 Add `Exclude []string` field to `CacheConfig` in `pkg/cache/cache.go`
- [ ] 1.3 Pass `Exclude` from `Repository` into `CacheConfig` when constructing upstreams in `proxy.go`
- [ ] 1.4 Update `IsCacheCandidate` in `cache.go` to: run exclude check first (exact name + suffix), then handle `"*"` wildcard, then existing suffix logic
- [ ] 1.5 Add unit tests for `IsCacheCandidate` covering: wildcard match, exclude exact name, exclude suffix, exclude overrides wildcard, exclude overrides explicit suffix, no exclude field
- [ ] 1.6 Add unit test for `validateConfig` covering: wildcard with redundant explicit suffixes emits a warning and returns no error

## 2. Gentoo repository config

- [ ] 2.1 Add `gentoo` entry to `configs/pkgproxy.yaml` with `suffixes: ["*"]`, `exclude: [layout.conf, timestamp.mirmon, timestamp.dev-local]`, and mirrors: `mirror.init7.net`, `pkg.adfinis-on-exoscale.ch`, `distfiles.gentoo.org`

## 3. E2e test

- [ ] 3.1 Add `assertNotCached` helper to `test/e2e/e2e_test.go` that asserts no file matching a given name exists anywhere under a cache subdirectory
- [ ] 3.2 Write `test/e2e/test-gentoo.sh` shell script that: downloads `portage-latest.tar.xz` directly from `distfiles.gentoo.org`, unpacks it to `/var/db/repos/gentoo`, sets `GENTOO_MIRRORS` in `make.conf` to point at pkgproxy, runs `emerge --fetchonly app-text/tree`, then fetches `distfiles/layout.conf` via `wget` through the proxy
- [ ] 3.3 Add `TestGentoo` to `test/e2e/e2e_test.go` using `docker.io/gentoo/stage3:latest`, mounting the script, asserting tree source archive is cached under `gentoo/distfiles/`, and asserting `layout.conf` is NOT cached using `assertNotCached`

## 3b. Makefile

- [ ] 3b.1 Add `gentoo → TestGentoo` mapping to the `distroToTest` macro in `Makefile` so `make e2e DISTRO=gentoo` works; add `gentoo` to the error message's list of valid values

## 4. Documentation

- [ ] 4.1 Add Gentoo `make.conf` snippet to `README.md`
- [ ] 4.2 Add Gentoo `make.conf` snippet to the HTTP landing page (`pkg/pkgproxy/landing.go` or template)
- [ ] 4.3 Update `CHANGELOG.md` `[Unreleased]` section with new features
