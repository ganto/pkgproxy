## 1. Version Command (Go)

- [ ] 1.1 Add `Version` and `GitCommit` variables with `unknown` defaults to `cmd/version.go`; `BuildDate` SHALL default to empty string and fall back to `time.Now().UTC()` at runtime when the ldflag was not injected
- [ ] 1.2 Implement `newVersionCommand()` Cobra subcommand that prints version, commit, date, and Go compiler version (`runtime.Version()`)
- [ ] 1.3 Register `newVersionCommand()` in `NewRootCommand()` in `cmd/root.go`
- [ ] 1.4 Move `cobra.OnInitialize(initConfig)` out of `cmd/root.go` and into a `PersistentPreRunE` on the `serve` command so that `pkgproxy version` does not require a config file to be present
- [ ] 1.5 Add an `slog.Info("starting pkgproxy", ...)` call in `startServer()` in `cmd/serve.go` immediately after the logger is initialised, logging `version`, `gitCommit`, `goVersion`, and `buildDate`

## 2. Makefile Ldflag Wiring

- [ ] 2.1 Add `COMMIT` and `DATE` variables to the Makefile (via `git rev-parse --short HEAD` and `date -u`)
- [ ] 2.2 Set `GO_BUILD_ARGS_EXTRA` to pass `-ldflags "-X ..."` for `Version`, `GitCommit`, `BuildDate` in the `ci-build` target
- [ ] 2.3 Verify `make build && bin/pkgproxy version` outputs correct values

## 3. OCI Image Labels and Annotations in Publish Workflow

- [ ] 3.1 Add a step to compute `VERSION=$(git describe --always)` as an environment variable before the `ko build` step in `.github/workflows/publish.yaml`
- [ ] 3.2 Add `--image-label` flags for `source`, `revision`, `version`, `created`, `title`, `vendor`, `licenses`, `description` to the `ko build` invocation in `.github/workflows/publish.yaml`
- [ ] 3.3 Add `--image-annotation` flags for `source` and `revision` to the same `ko build` invocation in `.github/workflows/publish.yaml`

## 4. Changelog

- [x] 4.1 Create `CHANGELOG.md` at the repository root following Keep a Changelog 1.1.0 format with an initial `[Unreleased]` section
- [ ] 4.2 Add a rule to `CLAUDE.md` instructing Claude to update the `[Unreleased]` section of `CHANGELOG.md` for every user-facing change made to the codebase
- [ ] 4.3 Document the release preparation step in `CLAUDE.md` and `README.md`: before pushing a tag the author SHALL rename `[Unreleased]` to `[v<version>] - <date>`, add a new empty `[Unreleased]` above it, and commit

## 5. Release Workflow

- [ ] 5.1 Create `.github/workflows/release.yaml` triggered on `push: tags: ['v*']` with permissions `contents: write`, `packages: write`, `id-token: write`
- [ ] 5.2 Add job steps: checkout, compute `VERSION=$(git describe --always)`, setup-go, install cosign, setup ko, `ko build --bare --tags ${GITHUB_REF_NAME}` with OCI labels/annotations, cosign sign
- [ ] 5.3 Add step to extract the `## [v<tag>]` section from `CHANGELOG.md` into a temp file using `awk` with `GITHUB_REF_NAME` directly (no `v`-stripping needed)
- [ ] 5.4 Add step to create GitHub Release via `gh release create --notes-file <extracted-notes>`
