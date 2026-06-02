# prolm

`prolm` is a project manager and build tool for Prolog projects. It provides
the missing project layer around manifests, reproducible lockfiles, dependency
installation, runtime invocation, tests, and checks.

The current implementation targets SWI-Prolog first. GNU Prolog and Scryer
Prolog are part of the runtime model, but broader runtime support is still on
the roadmap.

## Status

`prolm` is in early MVP development. It is useful for trying the current
workflow, validating the package model, and contributing to the tool itself.
Expect the CLI and metadata format to keep tightening before a stable release.

Current project commands include:

- `prolm new` and `prolm init` for project scaffolding
- `prolm install`, `prolm add`, and `prolm remove` for dependencies
- `prolm run`, `prolm test`, and `prolm check` for runtime workflows
- `prolm env` for local environment diagnostics

Planned registry-era commands such as `search`, `info`, `publish`, `login`,
`logout`, and `owner` are documented but not implemented yet.

See [Command Central](docs/command-central.md) for the full command contract
and implementation status.

## Installation

From source:

```bash
git clone https://github.com/SAhmadi/prolm.git
cd prolm
go build -o prolm .
```

Then either run `./prolm` directly or move the binary onto your `PATH`.

Requirements:

- Go 1.25 or newer for development
- SWI-Prolog for the current runtime-backed commands
- `staticcheck` for the full local CI target

## Quick Start

Create and run a project:

```bash
prolm new hello
cd hello
prolm install
prolm run
prolm test
prolm check
```

Initialize an existing project:

```bash
cd existing-prolog-project
prolm init
prolm install
```

## Project Files

`prolm` uses two project files:

- `Prolfile.toml` is the user-authored manifest.
- `Prolfile.lock` records selected package versions, sources, checksums, and
  resolved dependency metadata.

Commit both files for applications. The lockfile is generated metadata, but it
is part of reproducible installs and should not be edited by hand.

The format reference lives in [Project Files](docs/project-files.md).

## Security Model

Package-manager code has a broad trust boundary. `prolm` treats registry
metadata, manifests, lockfiles, archives, and package source as untrusted
input.

Current security invariants include:

- SHA-256 verification before archive unpacking
- HTTPS-only registry and download inputs
- traversal, link, file-type, and resource-limit defenses during extraction
- atomic lockfile writes
- no dependency install script execution
- no credential logging or project metadata credential storage

See [Security Rules](docs/security.md) and [SECURITY.md](SECURITY.md) before
changing package acquisition, archive extraction, credentials, registry logic,
or lockfile persistence.

## Development

Run the local CI target before opening a pull request:

```bash
make ci
```

That target runs:

```bash
go vet ./...
staticcheck ./...
go test -race -count=1 ./...
```

Narrower commands:

```bash
make test
make vet
make lint
go test ./cmd -run TestSmoke_EndToEndPhase1Acceptance -count=1
```

See [TESTING.md](TESTING.md) for test ownership, CI behavior, and adversarial
security coverage.

## Contributor Navigation

Start with [Documentation Index](docs/index.md), then read the smallest focused
reference for the change. For command behavior use
[Command Central](docs/command-central.md); for code paths use
[Project Component Overview](docs/component-overview.md); for manifests and
lockfiles use [Project Files](docs/project-files.md); for package acquisition
or trust boundaries use [Security Rules](docs/security.md) and
[Package Manager Design](docs/package-manager-design.md).

## Documentation Map

- [Documentation Index](docs/index.md)
- [Command Central](docs/command-central.md)
- [Codebase Map](docs/codebase-map.md)
- [Project Component Overview](docs/component-overview.md)
- [Project Files](docs/project-files.md)
- [Security Rules](docs/security.md)
- [Package Manager Design](docs/package-manager-design.md)
- [Flow Diagrams](docs/flow-diagrams.md)
- [Roadmap](ROADMAP.md)
- [Changelog](CHANGELOG.md)

## Contributing

Contributions should keep Cobra command files thin and put business logic under
`internal/`. Public metadata types live in `pkg/prolfile/`.

Security-sensitive changes need focused tests for hostile inputs, not only happy
paths. Use the existing package boundaries and test style unless the change
explicitly requires a new structure.

## License

MIT. See [LICENSE](LICENSE).
