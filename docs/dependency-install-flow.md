# Dependency Install Flow

Back to [Codebase Map](codebase-map.md) and [Project Component Overview](component-overview.md).

This workflow starts in the CLI dependency commands and crosses manifest,
registry, installer, local store, checksum, and lockfile code.

## Flow

1. [CLI Surface](cli-surface.md) loads `Prolfile.toml` and chooses dependency input.
2. [Project Metadata Flow](project-metadata-flow.md) validates manifest and lockfile state.
3. [`internal/registry/`](../internal/registry) queries the SWI pack index or
   uses a command-provided source override.
4. [`internal/installer/`](../internal/installer) fetches archive bytes,
   verifies registry or lockfile checksums, unpacks safe content, and writes to
   the local store.
5. [`internal/lockfile/`](../internal/lockfile) serializes deterministic
   lockfile metadata after sync.

## Current behavior

`prolm install` is manifest and lock driven. It does not accept package names
as positional inputs. If `Prolfile.lock` is current and all locked package
paths exist in the local store, install is a fast no-op. If the lock is absent,
stale, conflicted, or references missing store paths, install resolves package
state from the registry or command-provided source and writes a refreshed lock.

`prolm add` and `prolm remove` share the same sync path after changing manifest
intent, but their save timing differs. `add` mutates the manifest in memory,
runs install sync, then saves the final dependency constraint; if sync fails,
the on-disk manifest is unchanged. `remove` saves the dependency deletion
before sync, then attempts to restore the original manifest if sync fails.

The `--frozen`, `--offline`, and `--no-verify` flags are registered today as
Phase 2 stubs. Using any of them should fail clearly instead of silently
changing install semantics.

## Failure paths and flags

| Area | Expected behavior | Security rules |
| --- | --- | --- |
| Registry/source URL | Reject invalid or non-HTTPS registry and download data. | SEC-4, SEC-9, SEC-14 |
| Cache | Store fetched archives for future offline/cache-reuse work; current installs refresh the cache path from the network when store state is missing. | SEC-1 |
| Fetch | Enforce HTTPS, timeouts, retry guidance, and archive size limits; write downloads to the cache path atomically. | SEC-4, SEC-10, SEC-12 |
| Verify | Delete bad cache entries and fail hard on checksum mismatch. | SEC-1 |
| Unpack | Reject traversal, unsafe links, special files, and resource-limit violations; clean partial output. | SEC-2, SEC-12, SEC-13 |
| Store | Hold the store lock while writing installed package state. | SEC-7 |
| Lockfile | Write atomically and never persist credentials. | SEC-5, SEC-8 |

`--offline` is planned to use local cache only, `--frozen` is planned to fail
when the lock would change, and `--no-verify` is planned as a dangerous
development-only bypass. Until those flags are implemented, the documented
behavior is a hard error.

## Core files

- [`cmd/install.go`](../cmd/install.go) loads manifest and lockfile inputs for
  `prolm install`.
- [`cmd/deps_sync.go`](../cmd/deps_sync.go) shares sync behavior with `add`
  and `remove`.
- [`internal/registry/registry.go`](../internal/registry/registry.go)
  defines registry data and error contracts.
- [`internal/registry/swi.go`](../internal/registry/swi.go) scrapes SWI pack
  index data and download URLs.
- [`internal/installer/installer.go`](../internal/installer/installer.go)
  orchestrates deterministic install work.
- [`internal/installer/fetch.go`](../internal/installer/fetch.go) downloads
  archives through bounded HTTP handling.
- [`internal/installer/verify.go`](../internal/installer/verify.go) handles
  checksum computation and verification.
- [`internal/installer/unpack.go`](../internal/installer/unpack.go) enforces
  archive extraction safety.
- [`internal/installer/store.go`](../internal/installer/store.go) owns local
  store paths and write locking.

## Neighboring concerns

- [Project Metadata Flow](project-metadata-flow.md) owns manifest and lockfile representation.
- [CLI Surface](cli-surface.md) owns dependency command input and output.
- [`internal/httputil/retry.go`](../internal/httputil/retry.go) provides
  retry behavior shared by HTTP callers.
- [Security Rules](security.md) captures checksum, extraction, and trust
  rules that matter here.
- [Package Manager Design](package-manager-design.md) summarizes the
  dependency lifecycle and lockfile design constraints.
