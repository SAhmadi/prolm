# prolm Agent Guide

`prolm` is a project manager and build tool for Prolog, written in Go. It is
the missing `cargo` or `poetry` layer for Prolog projects: manifests,
reproducible lockfiles, dependency installation, runtime invocation, tests,
and checks.

This file is the short entry point for contributors and coding agents. Keep
detailed reference material in the linked docs so this file stays quick to
load at the start of a session.

## Start Here

Read the smallest reference that covers the task:

| Need | Reference |
| --- | --- |
| See documentation ownership first | [Documentation Index](docs/index.md) |
| Find the relevant code area first | [Project Component Overview](docs/codebase-knowledge-graph/Project%20Component%20Overview.md) |
| Check the current and planned CLI commands | [Command Central](docs/command-central.md) |
| Follow install, run, publish, or MVS flow | [Flow Diagrams](docs/flow-diagrams.md) and [Command Flow Map](docs/codebase-knowledge-graph/Command%20Flow%20Map.canvas) |
| Work on a specific CLI subsystem | [Codebase Knowledge Graph](docs/codebase-knowledge-graph/Codebase%20Knowledge%20Graph.md) |
| Change `Prolfile.toml`, lockfiles, or metadata | [Project Files](docs/project-files.md) |
| Change fetch, install, registry, or trust behavior | [Security Rules](docs/security.md) and [Package Manager Design](docs/package-manager-design.md) |
| Check implementation phase and open work | [Roadmap](ROADMAP.md) |
| Add or change tests | [Testing Guide](TESTING.md) |

Before searching for a component by hand, consult the component overview. Use
`rg` after that map is insufficient, may be stale, or symbol-level source
verification is needed.

## Product Shape

Core goals:

- Predictable: the same `Prolfile.lock` should select the same package bytes.
- Fast: prefer local cache and store state before network work.
- Safe: verify package bytes, validate input, and never run dependency install
  scripts.
- Friendly: errors should explain the next useful action.
- Composable: keep working with the existing SWI-Prolog pack ecosystem.

SWI-Prolog is the primary runtime today. GNU Prolog and Scryer Prolog remain
part of the intended runtime model; check the roadmap and command docs before
assuming support exists.

## Current Surface

The command backlog and command contract live together in
[Command Central](docs/command-central.md). Do not infer that a command is
implemented only because it appears in the catalog:

- Available project commands include `new`, `init`, `install`, `add`,
  `remove`, `run`, `test`, `check`, and `env`.
- `update` and `repl` are planned production-hardening commands.
- Registry-era commands such as `search`, `info`, `publish`, `yank`, `login`,
  `logout`, and `owner` are planned later.
- Some live commands expose stubbed or future flags. Check each command status
  and flag status before changing tests, help, or docs.

The flow reference keeps the high-level install, run, publish, and version
resolution diagrams that explain how those command groups fit together.

## Tech Stack

Use the repository's existing stack unless a task explicitly changes it:

| Concern | Default |
| --- | --- |
| CLI | Cobra |
| Configuration | Viper |
| TOML | `github.com/BurntSushi/toml` |
| Semver | `github.com/Masterminds/semver/v3` |
| HTTP, checksums, archives | Go standard library wrappers in `internal/` |
| Tests | Go `testing` and `testify` |
| CLI output | Existing `internal/ui` helpers |

## Code Boundaries

- Keep Cobra command files in `cmd/` thin: parse inputs and orchestrate
  package logic, but keep business logic under `internal/`.
- Public metadata types live in `pkg/prolfile/`.
- Prefer the existing manifest, lockfile, registry, installer, runtime,
  scaffold, test-runner, checker, and UI packages over new cross-cutting
  abstractions.
- Use the focused codebase notes for current paths. Do not copy a future
  directory tree back into this guide.

## Metadata Model

- `Prolfile.toml` is user-authored project intent.
- `Prolfile.lock` records selected versions, sources, checksums, and resolved
  dependency metadata for reproducible installs.
- Runtime commands must use exact installed dependency state or explain that
  the user should install first.
- Metadata-only commands should not require a Prolog runtime just to manage
  manifests or lockfiles.
- Read [Project Files](docs/project-files.md) before changing format rules or
  serialization behavior.

## Engineering Rules

- Go version target is 1.22+.
- Bind Viper to Cobra flags in `PersistentPreRunE`, not in command `init()`.
  Root setup must bind both local and persistent flags before command logic
  reads Viper values.
- Use `RunE` and returned errors in command handlers. Internal packages return
  errors; the CLI layer decides how to present them.
- Wrap errors with context using `%w`.
- Use argument arrays for runtime and subprocess execution. Do not build shell
  command strings from user or registry data.
- Test file names describe behavior, not implementation phases. Prefer names
  like `command_contract_test.go` or `runtime_detection_test.go`.

## Always-On Invariants

Read [Security Rules](docs/security.md) before changing package acquisition,
archive extraction, lockfile persistence, credentials, or registry behavior.
These invariants must survive every implementation:

- Verify SHA-256 before unpacking package archives, including cached archives.
- Extract archives with traversal, symlink, resource-limit, and file-type
  defenses.
- Require HTTPS for registry and download inputs.
- Never log credentials or store them in project metadata.
- Write lockfiles atomically and protect store writes from concurrent installs.
- Treat registry, lockfile, archive, and manifest inputs as untrusted.
- Never execute install scripts from downloaded packages.

## Reference Maintenance

- Keep current and future command descriptions in
  [Command Central](docs/command-central.md), with a status line for every
  command.
- Keep flow visuals in the [Command Flow Map](docs/codebase-knowledge-graph/Command%20Flow%20Map.canvas)
  and searchable flow navigation in [Flow Diagrams](docs/flow-diagrams.md).
- The Obsidian Canvas flow map is fine to keep as the visual source, but every
  flow it owns must remain discoverable through GitHub-readable Markdown,
  especially [Flow Diagrams](docs/flow-diagrams.md).
- Keep current path maps in the codebase knowledge graph rather than copying
  package trees into this file.
- Keep phase sequencing and open implementation work in
  [ROADMAP.md](ROADMAP.md).

## Verification

Scale tests with the change. Use package-level tests while iterating, then run
the broader check when the touched behavior warrants it.

Before opening a pull request, run:

```bash
make ci
```

`make ci` mirrors the repository CI checks described in [TESTING.md](TESTING.md).
