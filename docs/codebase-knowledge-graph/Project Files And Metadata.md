# Project Files And Metadata

Back to [[Codebase Knowledge Graph]] and [[Project Component Overview]].

Project metadata crosses public data structures, `Prolfile.toml` parsing, and
`Prolfile.lock` persistence. These files sit underneath dependency commands,
runtime commands, and scaffolding.

## Metadata layers

- [`pkg/prolfile/types.go`](../../pkg/prolfile/types.go) defines importable
  manifest, package, runtime config, dependency, lock meta, and lock entry
  types.
- [`internal/manifest/manifest.go`](../../internal/manifest/manifest.go) loads,
  discovers, and saves `Prolfile.toml`.
- [`internal/manifest/validate.go`](../../internal/manifest/validate.go)
  validates package identity, versions, entry paths, runtimes, dependencies,
  and scripts.
- [`internal/lockfile/lockfile.go`](../../internal/lockfile/lockfile.go) loads,
  encodes, and saves `Prolfile.lock`.
- [`internal/lockfile/hash.go`](../../internal/lockfile/hash.go) hashes
  dependency inputs for lockfile metadata.

## Workflow links

- [[CLI Surface]] loads metadata before commands act.
- [[Dependency Install Flow]] reads manifest dependencies and writes lockfile
  package records.
- [[Runtime Test And Check Flow]] uses package entry and runtime config data.
- [[Scaffolding And UX]] creates the first manifest for new and initialized
  projects.

## Reference docs

- [[docs/project-files|Project Files]] contains the Prolfile and lockfile format
  reference.
- [[docs/security|Security Rules]] covers lockfile and archive trust
  invariants.
- [[ROADMAP]] records which metadata behaviors are implemented by phase.
