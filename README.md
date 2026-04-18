# pkgproxy - A caching forward proxy for Linux package repositories

`pkgproxy` is a Web server that serves Linux packages for various repository
types (RPM, DEB, ...) from a local cache. It can be used as a central package
server in a local network. Packages not available in the local cache will be
fetched transparently from configurable upstream mirrors.

## Run the code

Build and run the code locally for testing:
```shell
PKGPROXY_CONFIG=./configs/pkgproxy.yaml go run github.com/ganto/pkgproxy serve
```

Run the application via a container engine (e.g. [Podman](https://podman.io/)):

```shell
podman run --rm -p 8080:8080 --volume ./cache:/ko-app/cache:z ghcr.io/ganto/pkgproxy serve --host 0.0.0.0
```

To use a custom `pkgproxy.yaml`, bind-mount it into the container:
```shell
podman run --rm -p 8080:8080 --volume ./cache:/ko-app/cache:z --volume ./pkgproxy.yaml:/ko-app/pkgproxy.yaml ghcr.io/ganto/pkgproxy serve --host 0.0.0.0
```

## Server Configuration

### CLI Flags

| Flag | Env Variable | Default | Description |
|------|--------------|---------|-------------|
| `--config, -c` | `PKGPROXY_CONFIG` | `./pkgproxy.yaml` | Path to the repository config file |
| `--cachedir` | | `cache` | Path to the local cache directory |
| `--host` | | `localhost` | Listen address |
| `--port` | | `8080` | Listen port |
| `--public-host` | `PKGPROXY_PUBLIC_HOST` | | Public hostname (or `host:port`) shown in landing page config snippets. When set, the listen port is not appended. Useful when running behind a reverse proxy. |
| `--debug` | | `false` | Enable debug logging |

## Repository Configuration

An example repository configuration can be found at [configs/pkgproxy.yaml](configs/pkgproxy.yaml).

Each repository supports the following options:

| Key | Required | Description |
|-----|----------|-------------|
| `suffixes` | yes | File suffixes that are eligible for caching (e.g. `.rpm`, `.deb`). Use `"*"` to cache all files. |
| `exclude` | no | List of file names to exclude from caching, even when they match a suffix. Useful with the `"*"` wildcard suffix. |
| `mirrors` | yes | Ordered list of upstream mirror URLs |
| `retries` | no | Number of attempts per mirror before moving to the next one (default: `1`) |

### Mirror retries

Some upstream mirrors (e.g. `download.fedoraproject.org`) act as redirectors that
send clients to a randomly selected mirror via HTTP 302. If the selected mirror is
temporarily unavailable and responds with a 5xx error, pkgproxy can automatically
retry the request to the same redirector, which will typically redirect to a
different, working mirror.

To enable this, set `retries` to a value greater than 1:

```yaml
repositories:
  fedora:
    suffixes:
      - .rpm
    mirrors:
      - https://download.fedoraproject.org/pub/fedora/linux/
    retries: 3
```

With `retries: 3`, pkgproxy will attempt each mirror up to 3 times before moving
on to the next one. An exponential backoff is applied between retry attempts
(1s, 2s, 4s, ...). Only 5xx (server error) responses trigger a retry — client
errors like 404 are returned immediately.

### Cache exclusions

When using the wildcard suffix `"*"` to cache all files, certain files (such as
metadata or timestamps) should not be cached because they change frequently. The
`exclude` option lets you list file names that will always be fetched from
upstream, bypassing the cache:

```yaml
repositories:
  gentoo:
    suffixes:
      - "*"
    exclude:
      - layout.conf
      - timestamp.mirmon
      - timestamp.dev-local
    mirrors:
      - https://distfiles.gentoo.org/
```

Files whose name matches an entry in the `exclude` list are served directly from
the upstream mirror without being stored in the local cache.

## Client Configuration

With the provided configuration a number of Linux distributions are handled. See below where and how the clients must be adjusted to use your instance of pkgproxy. Replace `<pkgproxy>` with the host name of the pkgproxy instance:

### Alma Linux

E.g. `/etc/yum.repos.d/almalinux-baseos.repo` (adjust other repositories accordingly):
```
[baseos]
# mirrorlist=https://mirrors.almalinux.org/mirrorlist/$releasever/baseos
baseurl=http://<pkgproxy>:8080/almalinux/$releasever/BaseOS/$basearch/os/
```

### Arch Linux

`/etc/pacman.d/mirrorlist`:
```
Server = http://<pkgproxy>:8080/archlinux/$repo/os/$arch
```

### Debian

E.g. Debian 13 Trixie: `/etc/apt/sources.list` (substitute your release codename):
```
deb http://<pkgproxy>:8080/debian           trixie            main contrib non-free non-free-firmware
deb http://<pkgproxy>:8080/debian           trixie-updates    main contrib non-free non-free-firmware
deb http://<pkgproxy>:8080/debian           trixie-backports  main contrib non-free non-free-firmware
deb http://<pkgproxy>:8080/debian-security  trixie-security   main contrib non-free non-free-firmware
```

### CentOS

- CentOS 7: `/etc/yum.repos.d/CentOS-Base.repo` (adjust other repositories accordingly):
```
[base]
# mirrorlist=http://mirrorlist.centos.org/?release=$releasever&arch=$basearch&repo=os&infra=$infra
baseurl=http://<pkgproxy>:8080/centos/$releasever/os/$basearch/
```

- CentOS Stream 8: `/etc/yum.repos.d/CentOS-Stream-BaseOS.repo` (adjust other repositories accordingly):
```
[baseos]
# mirrorlist=http://mirrorlist.centos.org/?release=$stream&arch=$basearch&repo=BaseOS&infra=$infra
baseurl=http://<pkgproxy>:8080/centos/$stream/BaseOS/$basearch/os/
```

- CentOS Stream 9: `/etc/yum.repos.d/centos.repo` (adjust other repositories accordingly):
```
[baseos]
# metalink=https://mirrors.centos.org/metalink?repo=centos-baseos-$stream&arch=$basearch&protocol=https,http
baseurl=http://<pkgproxy>:8080/centos-stream/$stream/BaseOS/$basearch/os/
```

### EPEL

Can be used for any type of RPM-based enterprise distribution. E.g. `/etc/yum.repos.d/epel.repo` (adjust other repositories accordingly):
```
[epel]
# metalink=https://mirrors.fedoraproject.org/metalink?repo=epel-$releasever&arch=$basearch
baseurl=http://<pkgproxy>:8080/epel/$releasever/Everything/$basearch/
```

### Gentoo

`/etc/portage/make.conf`:
```
GENTOO_MIRRORS="http://<pkgproxy>:8080/gentoo"
```

### Fedora

`/etc/yum.repos.d/fedora.repo` (adjust other repositories accordingly):
```
[fedora]
# metalink=https://mirrors.fedoraproject.org/metalink?repo=fedora-$releasever&arch=$basearch
baseurl=http://<pkgproxy>:8080/fedora/releases/$releasever/Everything/$basearch/os/
```

### Fedora COPR

`/etc/yum.repos.d/_copr:copr.fedorainfracloud.org:<user>:<repo>.repo` (replace `<user>` and `<repo>` with the corresponding [COPR](https://copr.fedorainfracloud.org/coprs/) repository):
```
[copr:copr.fedorainfracloud.org:<user>:<repo>]
# baseurl=https://download.copr.fedorainfracloud.org/results/<user>/<repo>/fedora-$releasever-$basearch/
baseurl=http://<pkgproxy>:8080/copr/<user>/<repo>/fedora-$releasever-$basearch/
```
For Enterprise distributions the URL suffix `epel-$releasever-$basearch` must be used.

### Rocky Linux

`/etc/yum.repos.d/rocky.repo` (adjust other repositories accordingly):
```
[baseos]
# mirrorlist=https://mirrors.rockylinux.org/mirrorlist?arch=$basearch&repo=BaseOS-$releasever$rltype
baseurl=http://<pkgproxy>:8080/rockylinux/$releasever/BaseOS/$basearch/os/
```

### Ubuntu

E.g. Ubuntu 24.04 Noble Numbat: `/etc/apt/sources.list` (substitute your release codename):
```
deb http://<pkgproxy>:8080/ubuntu           noble           main restricted universe multiverse
deb http://<pkgproxy>:8080/ubuntu           noble-updates   main restricted universe multiverse
deb http://<pkgproxy>:8080/ubuntu-security  noble-security  main restricted universe multiverse
```

## Testing

### End-to-End Tests

End-to-end tests validate pkgproxy against real package managers running in containers. They require either [Podman](https://podman.io/) or Docker.

Run all e2e tests:
```shell
make e2e
```

Run tests for a specific distribution:
```shell
make e2e DISTRO=fedora
```

Run tests for a specific distribution and release:
```shell
make e2e DISTRO=fedora RELEASE=42
```

Supported `DISTRO` values: `fedora`, `centos-stream`, `almalinux`, `rockylinux`, `debian`, `ubuntu`, `archlinux`, `gentoo`.

When adding support for a new Linux distribution, corresponding e2e tests should be added as well.

## Building the Container Image

Build a container image locally using [ko](https://ko.build/):

```shell
make image-build
```

This builds a single-platform image for your host architecture and loads it into the local container runtime via `ko.local`.

To use [Podman](https://podman.io/) instead of Docker, point `DOCKER_HOST` to the Podman socket:

```shell
DOCKER_HOST=unix://$(podman info --format '{{.Host.RemoteSocket.Path}}') make image-build
```

## Releasing

1. Rename the `[Unreleased]` section in `CHANGELOG.md` to `[v<version>] - <date>` and add a new empty `[Unreleased]` section above it.
2. Commit the changelog update.
3. Push a version tag (e.g. `git tag v0.1.0 && git push origin v0.1.0`).

The tag push triggers the release workflow which builds a versioned container image, signs it with cosign, and creates a GitHub Release with notes extracted from `CHANGELOG.md`.

## License

[Apache 2.0](https://spdx.org/licenses/Apache-2.0.html)

## Author Information

The [pkgproxy](https://github.com/ganto/pkgproxy) code was written and is maintained by:
- [Reto Gantenbein](https://linuxmonk.ch/) | [e-mail](mailto:reto.gantenbein@linuxmonk.ch) | [GitHub](https://github.com/ganto)
