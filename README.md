# vault-tasks

[![Release](https://img.shields.io/github/v/release/totocaster/vault-tasks-obsidian-cli)](https://github.com/totocaster/vault-tasks-obsidian-cli/releases)
[![CI](https://github.com/totocaster/vault-tasks-obsidian-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/totocaster/vault-tasks-obsidian-cli/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/totocaster/vault-tasks-obsidian-cli)](https://go.dev/)
[![License](https://img.shields.io/github/license/totocaster/vault-tasks-obsidian-cli)](LICENSE)

`vault-tasks` is a read-only companion CLI for the [Vault Tasks Obsidian plugin](https://github.com/totocaster/vault-tasks-obsidian).

It reads the current vault, Obsidian app settings, and the plugin's saved settings, then renders the same grouped task view that the plugin shows inside Obsidian. The output is optimized for both humans and AI agents: readable by default, deterministic when you need JSON, and intentionally scoped to the plugin's worldview instead of inventing a separate task system.

## What It Does

- reads task data directly from Markdown notes in an Obsidian vault
- respects `.obsidian/plugins/vault-tasks-view/data.json`
- respects relevant `.obsidian/app.json` settings such as `readableLineLength`
- applies the plugin's note grouping, section grouping, pinned notes, task filtering, and sorting rules
- honors note-level frontmatter used by the plugin:
  - `deferred-until: YYYY-MM-DD`
  - `deffered-until: YYYY-MM-DD`
  - `hide-from-vault-tasks: true`
- can include plugin-style related-note backlink context
- can emit either human-readable text, summary text, or structured JSON

## What It Does Not Do

- it does not replace the [Vault Tasks Obsidian plugin](https://github.com/totocaster/vault-tasks-obsidian)
- it does not maintain its own task database
- it does not try to be a generic Markdown task manager
- it does not write task state back to notes in the current implementation

This tool is intentionally a companion CLI, not a standalone task product with a different mental model.

## Installation

### Homebrew

```bash
brew tap totocaster/tap
brew install vault-tasks
```

### Go install

```bash
go install github.com/totocaster/vault-tasks-obsidian-cli@latest
```

### From source

```bash
git clone https://github.com/totocaster/vault-tasks-obsidian-cli.git
cd vault-tasks-obsidian-cli
go build -o vault-tasks .
```

## Usage

If you run `vault-tasks` from inside a vault, it auto-discovers the vault root by looking for `.obsidian/`.

```bash
vault-tasks
vault-tasks --vault ~/Notes/totocaster
vault-tasks sections --vault ~/Notes/totocaster
vault-tasks settings --vault ~/Notes/totocaster
vault-tasks --vault ~/Notes/totocaster --filter all
vault-tasks --vault ~/Notes/totocaster --section "To Do"
vault-tasks --vault ~/Notes/totocaster --section none
vault-tasks --vault ~/Notes/totocaster --connections
vault-tasks --vault ~/Notes/totocaster --format json
vault-tasks --version
```

## Command Surface

```text
vault-tasks [show] [--filter pending|completed|all] [--section <heading>|none] [--connections|--no-connections] [--format view|summary|json] [--width readable|full] [--vault PATH]
vault-tasks sections [--filter pending|completed|all] [--vault PATH]
vault-tasks settings [--vault PATH]
vault-tasks version
```

## Output Modes

### Default view

The default output mirrors the plugin's grouped note view:

- note groups render as `##`
- section groups render as `###`
- tasks preserve Markdown checkbox text
- each task line includes a source path and line number

### Summary view

`--format summary` gives a compact snapshot that is useful for scripts, dashboards, and agents deciding what to inspect next.

### JSON view

`--format json` emits a structured snapshot including:

- effective filter and width
- active section filter
- visible groups and tasks
- effective settings loaded from the vault
- available section filters
- related-note backlinks
- scoped file counts

## Settings Compatibility

`vault-tasks` reads and applies the same settings vocabulary as the plugin, including:

- `defaultFilter`
- `includeFolders`
- `excludeFolders`
- `pinnedNotePaths`
- `persistSectionFilter`
- `savedSectionFilter`
- `showConnectionsByDefault`
- `showSectionHeadings`
- `pendingMode`
- `includeCancelledInCompleted`
- `noteSort`
- `sectionSort`
- `taskSort`

That means the CLI defaults should match what the user sees in the plugin, rather than forcing separate CLI-specific defaults.

## Development

```bash
gofmt -w .
go test ./...
go test -cover ./...
go build ./...
```

Build a local binary:

```bash
go build -o vault-tasks .
```

## Release Flow

This repo follows the same release pattern as other shipped Go CLIs in this workspace:

- CI runs on pushes and pull requests via `.github/workflows/ci.yml`
- a GitHub release is cut when a `v*` tag is pushed via `.github/workflows/release.yml`
- GoReleaser builds archives for macOS/Linux amd64 + arm64
- GoReleaser updates the `totocaster/homebrew-tap` formula using `HOMEBREW_TAP_TOKEN`

Release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Required GitHub secret:

- `HOMEBREW_TAP_TOKEN`

## Testing Notes

The codebase includes unit and fixture-based integration tests covering:

- settings normalization and option resolution
- folder-scope matching
- frontmatter parsing
- task and heading extraction
- section filtering and sorting
- rendered text and JSON output
- persisted section-filter behavior
- hidden and deferred note behavior
- root command help/version behavior

The tool was also smoke-tested against a real Obsidian vault during development.

## License

MIT © 2026 [Toto Tvalavadze](https://ttvl.co)
