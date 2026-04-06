## Why

pkgproxy supports caching for RPM, DEB, and Arch-based distros but not Gentoo. Gentoo users who build many packages fetch large source tarballs (distfiles) repeatedly across machines; a local caching proxy reduces bandwidth and improves build times.

## What Changes

- Add `exclude` field to the `Repository` config type: a list of filenames or suffixes that are **never** cached, even when `suffixes` contains `"*"`.
- Add `"*"` wildcard support to the existing `suffixes` field: when present, all proxied files are cache candidates except those matching `exclude` entries.
- Add a `gentoo` repository entry to `configs/pkgproxy.yaml` using Swiss mirrors (init7, Adfinis/Exoscale) with `suffixes: ["*"]` and `exclude` covering mirror-specific metadata files.
- Add a Gentoo e2e test (`TestGentoo`) that fetches a distfile via the proxy from a `gentoo/stage3` container and asserts it is cached.

## Capabilities

### New Capabilities

- `gentoo-distfiles`: Proxy and cache Gentoo distfiles from configurable upstream mirrors, honoring the BLAKE2B hash-based directory layout (`distfiles/<xx>/<filename>`).
- `cache-exclude`: Per-repository `exclude` list that prevents specific filenames or suffixes from being cached, complementing the existing `suffixes` include list and enabling the `"*"` wildcard use case.

### Modified Capabilities

- `e2e-multi-distro`: Gentoo is added as a supported distro with a corresponding e2e test.

## Impact

- `pkg/pkgproxy/repository.go`: Add `Exclude []string` field to `Repository` struct; update `validateConfig` (no required validation, field is optional).
- `pkg/cache/cache.go`: Update `CacheConfig` to carry the exclude list; update `IsCacheCandidate` to handle `"*"` wildcard and exclude matching.
- `configs/pkgproxy.yaml`: Add `gentoo` repository entry.
- `test/e2e/e2e_test.go`: Add `TestGentoo`.
- `README.md` and landing page: Add Gentoo `make.conf` snippet.
- `CHANGELOG.md`: Document new features.
