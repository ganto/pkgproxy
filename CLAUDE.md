# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

pkgproxy is a caching forward proxy for Linux package repositories, written in Go.

## Commands

```bash
make build                                               # build binary → bin/pkgproxy
make ci-check                                            # lint + govulncheck + tests
make run                                                 # run locally with debug logging
go test -v -race ./pkg/pkgproxy/ -run TestName          # run a single test
make e2e                                                 # run all e2e tests (requires podman or docker)
make e2e DISTRO=fedora                                   # run e2e tests for a specific distro
make e2e DISTRO=fedora RELEASE=42                        # run e2e tests for a specific distro and release
```

## Pre-commit

Run pre-commit hooks directly with:

```bash
pre-commit run --all-files
pre-commit run codespell --all-files
```

## Rules

- Run `make ci-check` before committing.
- Do not delete failing tests.
- Update the `[Unreleased]` section of `CHANGELOG.md` for every user-facing change made to the codebase. Entries must be concise (80–100 characters), written for pkgproxy users, and omit internal implementation details.
- Before pushing a release tag: rename `[Unreleased]` to `[v<version>] - <date>`, add a new empty `[Unreleased]` section above it, and commit.
- E2e tests must pass before a feature is considered complete.
- Adding support for a new Linux distribution requires adding corresponding e2e tests.
- Changes to client config snippets (sources.list, .repo files) must be replicated in the landing page snippets and in README.md.
- New or changed configuration options (CLI flags, repository config keys) must be documented in README.md.

## OpenSpec

Run openspec commands via npx, but first source nvm:

```bash
bash -c 'source ~/.config/bash/rc.d/nvm.sh && npx openspec <command>'
```

Common commands:
```bash
bash -c 'source ~/.config/bash/rc.d/nvm.sh && npx openspec new change "<name>"'
bash -c 'source ~/.config/bash/rc.d/nvm.sh && npx openspec status --change "<name>" --json'
bash -c 'source ~/.config/bash/rc.d/nvm.sh && npx openspec instructions <artifact> --change "<name>" --json'
```

## Docs

- [Architecture](docs/architecture.md)
- [Testing](docs/testing.md)
