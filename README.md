# pkgproxy - A caching reverse proxy for Linux package repositories

`pkgproxy` is a Web server that serves Linux packages for various repository
types (RPM, DEB, ...) from a local cache. It can be used as a central package
server in a local network. Packages not available in the local cache will be
fetched transparently from configurable upstream mirrors.

An example repository configuration can be found at [config/pkgproxy.yaml](config/pkgproxy.yaml).

## Run the code

```shell
PKGPROXY_CONFIG=./configs/pkgproxy.yaml go run github.com/ganto/pkgproxy serve
```

## License

[Apache 2.0](https://spdx.org/licenses/Apache-2.0.html)

## Author Information

The [pkgproxy](https://github.com/ganto/pkgproxy) code was written and is maintained by:
- [Reto Gantenbein](https://linuxmonk.ch/) | [e-mail](mailto:reto.gantenbein@linuxmonk.ch) | [GitHub](https://github.com/ganto)
