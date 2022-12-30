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

- Using the default repository configuration:
```shell
podman run --rm -p 8080:8080 --volume ./cache:/ko-app/cache:z ghcr.io/ganto/pkgproxy serve --host 0.0.0.0 --config /var/run/ko/pkgproxy.yaml
```
- Mounting your own local `pkgproxy.yaml`:
```shell
podman run --rm -p 8080:8080 --volume ./cache:/ko-app/cache:z --volume ./pkgproxy.yaml:/ko-app/pkgproxy.yaml ghcr.io/ganto/pkgproxy serve --host 0.0.0.0 --config /ko-app/pkgproxy.yaml
```

## Repository Configuration

An example repository configuration can be found at [configs/pkgproxy.yaml](configs/pkgproxy.yaml).

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

E.g. Debian 11 Bullseye: `/etc/apt/sources.list`:
```
deb http://<pkgproxy>:8080/debian           bullseye            main contrib non-free
deb http://<pkgproxy>:8080/debian           bullseye-updates    main contrib non-free
deb http://<pkgproxy>:8080/debian           bullseye-backports  main contrib non-free
deb http://<pkgproxy>:8080/debian-security  bullseye-security   main contrib non-free
```
Adapt configuration accordingly for other Debian releases.

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
#mirrorlist=http://mirrorlist.centos.org/?release=$stream&arch=$basearch&repo=BaseOS&infra=$infra
baseurl=http://<pkgproxy>:8080/centos/$stream/BaseOS/$basearch/os/
```

- CentOS Stream 9: `/etc/yum.repos.d/centos.repo` (adjust other repositories accordingly):
```
[baseos]
# metalink=https://mirrors.centos.org/metalink?repo=centos-baseos-$stream&arch=$basearch&protocol=https,http
baseurl=http://merkur.oasis.home:8080/centos-stream/$stream/BaseOS/$basearch/os/
```

### EPEL

Can be used for any type of RPM-based enterprise distribution. E.g. `/etc/yum.repos.d/epel.repo` (adjust other repositories accordingly):
```
[epel]
# metalink=https://mirrors.fedoraproject.org/metalink?repo=epel-$releasever&arch=$basearch
baseurl=http://<pkgproxy>:8080/epel/$releasever/Everything/$basearch/
```

### Fedora

`/etc/yum.repos.d/fedora.repo` (adjust other repositories accordingly):
```
[fedora]
# metalink=https://mirrors.fedoraproject.org/metalink?repo=fedora-$releasever&arch=$basearch
baseurl=http://<pkgproxy>:8080/fedora/releases/$releasever/Everything/$basearch/os/
```

### Rocky Linux

`/etc/yum.repos.d/rocky.repo` (adjust other repositories accordingly):
```
[baseos]
# mirrorlist=https://mirrors.rockylinux.org/mirrorlist?arch=$basearch&repo=BaseOS-$releasever$rltype
baseurl=http://<pkgproxy>:8080/rocky/$releasever/BaseOS/$basearch/os/
```

### Ubuntu

E.g. Ubuntu 22.04 Jammy Jellyfish: `/etc/apt/sources.list`:
```
deb http://<pkgproxy>:8080/ubuntu  jammy           main restricted universe multiverse
deb http://<pkgproxy>:8080/ubuntu  jammy-updates   main restricted universe multiverse
deb http://<pkgproxy>:8080/ubuntu  jammy-security  main restricted universe multiverse
```

## License

[Apache 2.0](https://spdx.org/licenses/Apache-2.0.html)

## Author Information

The [pkgproxy](https://github.com/ganto/pkgproxy) code was written and is maintained by:
- [Reto Gantenbein](https://linuxmonk.ch/) | [e-mail](mailto:reto.gantenbein@linuxmonk.ch) | [GitHub](https://github.com/ganto)
