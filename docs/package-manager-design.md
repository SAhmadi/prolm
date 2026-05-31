# Package Manager Design

This note holds design context that should inform dependency, runtime, and
metadata work without making [AGENTS.md](../AGENTS.md) a full spec.

## Design Topic Index

The roadmap still uses these compact topic numbers when a checklist item needs
to point at a design concern:

| Topic | Concern |
| --- | --- |
| 8.1 | Semver edge cases |
| 8.2 | Dependency resolution |
| 8.3 | Lockfile correctness |
| 8.4 | Network and caching |
| 8.5 | Archive path traversal |
| 8.6 | Checksum verification |
| 8.7 | Concurrent installs |
| 8.8 | Yanked versions |
| 8.9 | Token and credential safety |
| 8.10 | Supply chain integrity |
| 8.11 | Package authenticity and future signing |
| 8.12 | Package namespacing and typo risk |
| 8.13 | Module and namespace isolation |
| 8.14 | Reproducibility guarantees |
| 8.15 | Runtime execution model |
| 8.16 | Workspaces and local dependencies |
| 8.17 | Prolfile format versioning |
| 8.18 | Plugin model |
| 8.19 | Lockfile merge conflicts |
| 8.20 | Cache garbage collection |
| 8.21 | Proxy and corporate network support |
| 8.22 | Prolog hook abuse |
| 8.23 | Package deprecation |
| 8.24 | License compliance |
| 8.25 | CLI self-update and minimum version |
| 8.26 | Telemetry and privacy |
| 8.27 | Graceful degradation without a runtime |
| 8.28 | Deterministic TOML serialization |

Topics with an `SEC-*` rule are governed by [Security Rules](security.md).

## Dependency Lifecycle

The current and planned command contract belongs in
[Command Central](command-central.md). The design intent is:

1. `Prolfile.toml` declares dependency intent.
2. A lockfile captures exact source, version, checksum, and dependency records.
3. Install prefers lockfile and local state when they are current.
4. Network fetches are verified before archive extraction.
5. The local store receives unpacked package source for runtime commands.

`prolm add` and `prolm remove` mutate manifest intent and sync the lock.
`prolm install` stays manifest and lock driven rather than accepting ad hoc
package positional inputs.

High-level install, run, planned publish, and planned version-resolution
diagrams live in [Flow Diagrams](flow-diagrams.md). The runtime receives
project and dependency code; `prolm` does not sandbox that execution.

## Resolution And Lockfiles

- MVS is the intended resolution model: requirements are collected across the
  dependency graph and the selected version must satisfy all constraints
  deterministically.
- Never store `latest` in a lockfile. Normalize real versions before locking.
- Fresh resolution skips yanked versions; locked yanked versions remain
  reproducible and visible through warnings.
- Detect circular dependency paths and constraint conflicts with errors naming
  the relevant packages.
- Same lockfile should select the same package bytes. It does not promise
  cross-runtime behavioral equivalence or equal native-extension builds across
  operating systems.

Semver handling is easy to get subtly wrong. Use the library named in
[Project Files](project-files.md), and test prereleases, build metadata,
leading `v` tags, zero-major ranges, and constraint intersection.

## Prolog-Specific Risks

- Dependencies share one Prolog runtime. Module names, predicates, operators,
  and expansion hooks can interfere with other code.
- `term_expansion`, `goal_expansion`, and custom operators deserve checker and
  audit visibility. Legitimate uses exist, so warnings and strict-mode policy
  matter.
- Packages should declare modules; checker work should make missing module
  boundaries visible.
- Running untrusted projects or packages requires external sandboxing such as a
  container or VM.

## Caching And Network

- Cache downloaded data separately from the unpacked store.
- Reuse cached archives only after checksum verification.
- Honor standard proxy behavior and explicit offline workflows.
- Time out network operations and respect retry guidance.
- Credentials live outside the cache and store.

## Future-Facing Constraints

- Keep metadata structures ready for namespaced package identifiers such as
  `<owner>/<package>`.
- Keep lock and registry data ready for future authenticity/signature fields.
- Path dependencies and workspaces must remain visibly mutable and should not
  erode publish reproducibility.
- Future plugins must not bypass [Security Rules](security.md).
- License, deprecation, audit, self-update, cache cleanup, private registry,
  and workspace work belong in the roadmap rather than in the agent entry
  document.
