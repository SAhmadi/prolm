# Dependency Install Flow

Back to [[Codebase Knowledge Graph]] and [[Project Component Overview]].

This workflow starts in the CLI dependency commands and crosses manifest,
registry, installer, local store, checksum, and lockfile code.

## Flow

1. [[CLI Surface]] loads `Prolfile.toml` and chooses dependency input.
2. [[Project Files And Metadata]] validates manifest and lockfile state.
3. [`internal/registry/`](../../internal/registry) queries the SWI pack index or
   uses a command-provided source override.
4. [`internal/installer/`](../../internal/installer) fetches archive bytes,
   verifies registry or lockfile checksums, unpacks safe content, and writes to
   the local store.
5. [`internal/lockfile/`](../../internal/lockfile) serializes deterministic
   lockfile metadata after sync.

## Core files

- [`cmd/install.go`](../../cmd/install.go) loads manifest and lockfile inputs for
  `prolm install`.
- [`cmd/deps_sync.go`](../../cmd/deps_sync.go) shares sync behavior with `add`
  and `remove`.
- [`internal/registry/registry.go`](../../internal/registry/registry.go)
  defines registry data and error contracts.
- [`internal/registry/swi.go`](../../internal/registry/swi.go) scrapes SWI pack
  index data and download URLs.
- [`internal/installer/installer.go`](../../internal/installer/installer.go)
  orchestrates deterministic install work.
- [`internal/installer/fetch.go`](../../internal/installer/fetch.go) downloads
  archives through bounded HTTP handling.
- [`internal/installer/verify.go`](../../internal/installer/verify.go) handles
  checksum computation and verification.
- [`internal/installer/unpack.go`](../../internal/installer/unpack.go) enforces
  archive extraction safety.
- [`internal/installer/store.go`](../../internal/installer/store.go) owns local
  store paths and write locking.

## Neighboring concerns

- [[Project Files And Metadata]] owns manifest and lockfile representation.
- [[CLI Surface]] owns dependency command input and output.
- [`internal/httputil/retry.go`](../../internal/httputil/retry.go) provides
  retry behavior shared by HTTP callers.
- [[docs/security|Security Rules]] captures checksum, extraction, and trust
  rules that matter here.
- [[docs/package-manager-design|Package Manager Design]] summarizes the
  dependency lifecycle and lockfile design constraints.
