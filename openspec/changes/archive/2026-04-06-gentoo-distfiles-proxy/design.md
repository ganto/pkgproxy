## Context

pkgproxy routes requests by stripping the first URL path segment as the repository name, then proxying the remainder to configured upstream mirrors. Cache candidacy is currently decided solely by file suffix (`IsCacheCandidate` in `cache.go`). Gentoo distfiles are content-addressed, permanent blobs with heterogeneous file extensions — the suffix model alone cannot represent "cache everything except a few metadata files".

## Goals / Non-Goals

**Goals:**
- Cache all Gentoo distfiles by default with a minimal exclude list for mirror-specific metadata.
- Introduce an `exclude` field that works independently of `"*"`, so operators can also exclude oversized individual files from any repo (e.g. `verylarge.rpm`).
- No changes to the proxy routing or transport layers — Gentoo fits the existing first-segment routing convention.

**Non-Goals:**
- Computing or validating the BLAKE2B path prefix — pkgproxy is a transparent proxy; path correctness is portage's responsibility.
- Caching `layout.conf` — excluded by default in the Gentoo config entry; no special-case code needed.
- Supporting `mirror://gentoo/` pseudo-URI scheme in ebuilds — handled transparently when portage resolves it to a real URL.

## Decisions

### 1. `"*"` wildcard in `suffixes` means "cache all"

**Decision:** A literal `"*"` entry in the `suffixes` list makes every proxied file a cache candidate, subject to `exclude` filtering.

**Alternatives considered:**
- `cache_all: true` boolean flag — adds a new top-level field and duplicates semantics already expressible via `suffixes`.
- Empty `suffixes` list means cache all — inverts current behavior (empty = cache nothing) and is surprising.
- `suffixes: ["*"]` is explicit, additive, and requires no validator changes.

**Edge case:** If `suffixes` contains both `"*"` and explicit entries (e.g. `["*", ".rpm"]`), the explicit entries are redundant. The config is accepted but `validateConfig` logs a warning naming the repository and the redundant suffixes. `IsCacheCandidate` treats this identically to `["*"]` alone.

### 2. `exclude` matches both exact filenames and suffixes

**Decision:** Each entry in `exclude` is tested against the filename as an exact match first, then as a suffix. This covers:
- Exact files: `layout.conf`, `timestamp.mirmon`, `timestamp.dev-local`
- Suffix-based: `.sig`, `.asc` if an operator wanted to exclude signatures

**Alternatives considered:**
- Separate `exclude_names` and `exclude_suffixes` fields — more explicit but adds config verbosity for a simple feature.
- Glob/regex patterns — more powerful but over-engineered for current needs; can be added later.

### 3. `exclude` is valid without `"*"` in suffixes

**Decision:** The `exclude` field is always applied, regardless of whether `"*"` is present. When no `"*"` is present, it acts as an override on top of suffix matching — useful for excluding a specific large file from an otherwise suffix-matched repo.

**Implementation:** `IsCacheCandidate` runs exclude check before suffix check. If any exclude entry matches, return false immediately.

### 4. Gentoo config uses init7 + Adfinis as primary Swiss mirrors

**Decision:** `mirror.init7.net` first, `pkg.adfinis-on-exoscale.ch` second, `distfiles.gentoo.org` as authoritative fallback.

### 5. E2e test bootstraps portage snapshot and uses emerge --fetchonly

**Decision:** Use `gentoo/stage3:latest`. The test script downloads `portage-latest.tar.xz` directly from `distfiles.gentoo.org` (bypassing the proxy — bootstrap only), unpacks it into `/var/db/repos/gentoo`, sets `GENTOO_MIRRORS` to pkgproxy, then runs `emerge --fetchonly app-text/tree`. This exercises the real portage fetch path including BLAKE2B path resolution.

**Alternatives considered:**
- Raw `wget` of a known distfile URL — simpler and faster, but doesn't validate that portage's mirror resolution works end-to-end through pkgproxy.

The test verifies:
1. `emerge --fetchonly app-text/tree` exits successfully with `GENTOO_MIRRORS` pointing at pkgproxy.
2. The tree source archive is cached on disk under `gentoo/distfiles/`.
3. `wget` of `distfiles/layout.conf` through the proxy succeeds but the file is NOT written to cache.

## Risks / Trade-offs

- **`"*"` caches everything including unexpected content** → Mitigated by the `exclude` list; operators can tune it.
- **Gentoo distfiles are large** → Cache disk usage is unbounded; this is an existing property of pkgproxy (no eviction). No change needed.
- **`portage-latest.tar.xz` snapshot download adds ~300 MB to each e2e test run** → Acceptable; Gentoo e2e tests are run manually on request, not in automated CI.
- **Mirror availability** → `distfiles.gentoo.org` as authoritative fallback ensures correctness.

## Open Questions

None — design is fully resolved by this document.
