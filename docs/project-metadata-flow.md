# Project Metadata Flow

Back to [Codebase Map](codebase-map.md) and [Project Component Overview](component-overview.md).

Project metadata crosses public data structures, `Prolfile.toml` parsing, and
`Prolfile.lock` persistence. These files sit underneath dependency commands,
runtime commands, and scaffolding.

## Metadata layers

- [`pkg/prolfile/types.go`](../pkg/prolfile/types.go) defines importable
  manifest, package, runtime config, dependency, lock meta, and lock entry
  types.
- [`internal/manifest/manifest.go`](../internal/manifest/manifest.go) loads,
  discovers, and saves `Prolfile.toml`.
- [`internal/manifest/validate.go`](../internal/manifest/validate.go)
  validates package identity, versions, entry paths, runtimes, dependencies,
  and scripts.
- [`internal/lockfile/lockfile.go`](../internal/lockfile/lockfile.go) loads,
  encodes, and saves `Prolfile.lock`.
- [`internal/lockfile/hash.go`](../internal/lockfile/hash.go) hashes
  dependency inputs for lockfile metadata.

## Workflow links

- [CLI Surface](cli-surface.md) loads metadata before commands act.
- [Dependency Install Flow](dependency-install-flow.md) reads manifest dependencies and writes lockfile
  package records.
- [Runtime Test And Check Flow](runtime-test-check-flow.md) uses package entry and runtime config data.
- [Scaffolding And UX](scaffolding-ux.md) creates the first manifest for new and initialized
  projects.

## Reference docs

- [Project Files](project-files.md) contains the Prolfile and lockfile format
  reference.
- [Security Rules](security.md) covers lockfile and archive trust
  invariants.
- [ROADMAP.md](../ROADMAP.md) records which metadata behaviors are implemented by phase.
