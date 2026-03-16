# OpenSpec

[OpenSpec](https://openspec.dev/) is a spec-driven planning layer for AI coding agents. It generates proposals, technical designs, and implementation tasks before code is written, and stores specifications alongside the repository for persistent context across sessions.

## Installation

Install as a project-local dependency:

```bash
npm install @fission-ai/openspec@latest
```

Initialize OpenSpec in the repository:

```bash
npx openspec init
```

## Usage

### CLI commands

| Command | Description |
|---------|-------------|
| `npx openspec init` | Initialize OpenSpec in the project |
| `npx openspec config profile` | Select a workflow profile |
| `npx openspec update` | Refresh agent instructions |

### Slash commands (Claude Code)

| Command | Description |
|---------|-------------|
| `/opsx:propose` | Create a new change proposal with specs |
| `/opsx:apply` | Implement tasks from the current change |
| `/opsx:archive` | Archive a completed change |
| `/opsx:new` | Start a new change |
| `/opsx:continue` | Resume work on an existing change |
| `/opsx:verify` | Validate completed work |
| `/opsx:sync` | Synchronize changes |

## Removal

```bash
npm uninstall @fission-ai/openspec
```
