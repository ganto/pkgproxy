## ADDED Requirements

### Requirement: Gentoo e2e test
The test suite SHALL include a Gentoo test function `TestGentoo` using a `docker.io/gentoo/stage3:latest` container. The test script SHALL:
1. Download the latest portage ebuild snapshot directly from `https://distfiles.gentoo.org/snapshots/portage-latest.tar.xz` (bypassing the proxy — this is bootstrap, not a distfile fetch).
2. Unpack the snapshot into `/var/db/repos/gentoo` inside the container.
3. Configure `GENTOO_MIRRORS` in `/etc/portage/make.conf` to point at the pkgproxy `gentoo` repository.
4. Run `emerge --fetchonly app-text/tree` to fetch the `tree` package sources through the proxy.
5. Fetch `http://<proxy>/gentoo/distfiles/layout.conf` via `wget` to exercise the negative cache path.

#### Scenario: emerge --fetchonly proxies and caches tree distfiles
- **WHEN** the Gentoo container runs `emerge --fetchonly app-text/tree` with `GENTOO_MIRRORS` pointing at pkgproxy
- **THEN** the command exits successfully and the tree source archive exists in the pkgproxy cache under the `gentoo/` subdirectory

#### Scenario: layout.conf is proxied but not cached
- **WHEN** the Gentoo container fetches `http://<proxy>/gentoo/distfiles/layout.conf` via `wget`
- **THEN** the request returns HTTP 200 and `layout.conf` does NOT exist in the pkgproxy cache under the `gentoo/` subdirectory
