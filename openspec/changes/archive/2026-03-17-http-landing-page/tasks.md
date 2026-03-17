## 1. Landing Page Handler

- [x] 1.1 Create `pkg/pkgproxy/landing.go` with `LandingHandler` function that accepts the repository config and returns an Echo handler
- [x] 1.2 Render upstream mirror URLs as clickable `<a href>` links in the HTML template
- [x] 1.3 Replace generic suffix-type snippet detection with a per-repo-name lookup table using the exact snippets from the README; omit snippet entirely for unknown repo names
- [x] 1.4 Accept the resolved public address string in `LandingHandler` and use it in all config snippets instead of a placeholder

## 2. Public Host Configuration

- [x] 2.1 Add `--public-host` flag to the `serve` subcommand in `cmd/serve.go`
- [x] 2.2 Add `PKGPROXY_PUBLIC_HOST` environment variable support; CLI flag takes precedence over env var
- [x] 2.3 Implement address resolution: if public host is set use it verbatim (no listen port appended), otherwise fall back to `host:port` from listen configuration
- [x] 2.4 Pass the resolved public address to `LandingHandler` when registering the `GET /` route

## 3. Route Registration

- [x] 3.1 Register `GET /` in `cmd/serve.go` pointing to `LandingHandler`

## 4. Tests

- [x] 4.1 Verify landing page returns HTTP 200 with `Content-Type: text/html`
- [x] 4.2 Verify mirror URLs are rendered as `<a href>` anchor tags
- [x] 4.3 Verify known repo names show their correct README snippet with full path suffix
- [x] 4.4 Verify unknown repo names have no snippet rendered
- [x] 4.5 Verify `--public-host myproxy.lan` produces snippets with `myproxy.lan` and no port
- [x] 4.6 Verify `--public-host myproxy.lan:9090` produces snippets with `myproxy.lan:9090` verbatim
- [x] 4.7 Verify `PKGPROXY_PUBLIC_HOST` sets the address when no flag is given
- [x] 4.8 Verify CLI flag takes precedence over `PKGPROXY_PUBLIC_HOST` env var
- [x] 4.9 Verify default (no public host) uses `host:port` from listen configuration

## 5. README

- [x] 5.1 Add a CLI flags reference table for the `serve` subcommand to `README.md` covering all flags with defaults and descriptions
- [x] 5.2 Include `--public-host` and `PKGPROXY_PUBLIC_HOST` in the table with a description of their effect

## 6. Validation

- [x] 6.1 Run `make ci-check` and fix any lint or test failures
