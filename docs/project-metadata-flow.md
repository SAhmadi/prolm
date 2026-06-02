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

## Manifest movement

Commands either receive an explicit `--config` path or auto-discover
`Prolfile.toml` by walking upward from the working directory. The resolved
manifest path is kept with the parsed manifest so commands can locate the
sibling `Prolfile.lock` and resolve relative entry paths.

The normal manifest path is:

```text
discover/--config -> load TOML -> semantic validation -> command action
```

Metadata-mutating commands then save through the manifest writer so section
order and key order remain deterministic:

```text
command mutation -> validate intended state -> deterministic save
```

`prolm add` and `prolm remove` clone the loaded manifest before mutation.
`add` keeps the mutation in memory until install sync succeeds, then saves the
final constraint and can restore the original manifest if that save fails.
`remove` saves the deletion before sync and attempts to restore the original
manifest if sync fails.

## Command metadata behavior

| Command | Metadata behavior | Runtime required |
| --- | --- | --- |
| `new` | Creates a new project manifest, empty lockfile, and scaffold files. | No |
| `init` | Creates a manifest and empty lockfile in an existing directory; optional scan populates dependencies. | No |
| `add` | Mutates `[dependencies]`, syncs install state, then saves manifest and lock. | No Prolog runtime |
| `remove` | Removes from `[dependencies]`, syncs install state, then saves manifest and lock. | No Prolog runtime |
| `install` | Reads manifest intent, resolves/fetches packages, writes lockfile when needed. | No Prolog runtime |
| `run` | Reads package entry, scripts, runtime config, and exact locked deps. | Yes |
| `test` | Reads runtime config and exact locked deps before test discovery. | Yes |
| `check` | Reads runtime config and exact locked deps before source discovery. | Yes |
| `env` | Discovers manifest and reports project/store/runtime state. | Runtime detection only |

Metadata-only workflows should stay independent from Prolog runtime discovery
so users can create, inspect, and install project metadata before a runtime is
available.

## Lockfile movement

Lock freshness is based on a dependency-state hash, not file mtimes. The hash
covers `[dependencies]` and `[dev-dependencies]`, so direct dependency intent
changes make the lock stale.

The lockfile path is:

```text
load LF -> reject/handle conflict markers -> compare dependency hash
       -> sort package records -> encode deterministically -> atomic save
```

Package entries are sorted by name, empty locks omit `[[package]]` entries,
and saves go through the atomic lockfile writer. `prolm install` can detect
Git conflict markers in `Prolfile.lock`, warn, and re-resolve from the
manifest rather than trusting conflicted generated metadata.

## Current and future fields

Current manifest fields include `[meta]`, `[package]`, `[dependencies]`,
`[dev-dependencies]`, `[runtime.*]`, and `[scripts]`. Some fields are parsed
or serialized before every command uses them:

- `[dev-dependencies]` contributes to lock freshness today; command support
  for `add --dev` and `remove --dev` is planned.
- `[runtime.*]` supports runtime flags and minimum versions for runtime-backed
  commands.
- `[scripts]` supports `prolm run <script>` values in the restricted
  `prolm run <entry> [-- <args>]` shape.
- Path/workspace dependencies, explicit source overrides, namespaced package
  IDs, and authenticity/signature fields are future-facing package-manager
  metadata. Keep them parseable and documented, but do not treat them as fully
  implemented command behavior.

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
