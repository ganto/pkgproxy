# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

pkgproxy is a caching forward proxy for Linux package repositories, written in Go.

## Commands

```bash
make build                                               # build binary → bin/pkgproxy
make ci-check                                            # lint + govulncheck + tests
make run                                                 # run locally with debug logging
go test -v -race ./pkg/pkgproxy/ -run TestName          # run a single test
```

## Rules

- Run `make ci-check` before committing.
- Do not delete failing tests.

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
