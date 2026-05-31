# Flow Diagrams

This file is the canonical Markdown home for high-level `prolm` flows. Keep
the diagrams concise and link detailed behavior to the focused reference docs.

## Legend

- `PF`: `Prolfile.toml`
- `LF`: `Prolfile.lock`
- `Reg`: registry or SWI pack index
- `SHA`: SHA-256 archive verification
- `Store`: local package store
- `RT`: selected Prolog runtime
- `MVS`: minimum version selection

## `prolm install`

```text
PF deps
  |
  v
LF current? -- yes --> Store complete? -- yes --> done
  | no/stale             |
  v                      no
Resolve via Reg/source <-'
  |
  v
Cache hit? -- no --> Fetch archive
  | yes              |
  '------------------'
          |
          v
        SHA
          |
          v
     Safe unpack
          |
          v
   Store + atomic LF
```

See [Dependency Install Flow](dependency-install-flow.md), [Project Files](project-files.md),
and [Security Rules](security.md).

## `prolm run`

```text
PF entry/script + RT config
          |
          v
Read LF exact deps
          |
          v
Verify Store paths
          |
          v
Build RT argument array
          |
          v
Execute RT
```

Missing installed dependencies should explain that the user needs to run
`prolm install` first. See [Runtime Test And Check Flow](runtime-test-check-flow.md).

## Planned `prolm publish`

```text
PF package metadata
          |
          v
Pre-flight checks
          |
          v
Build archive ---- --dry-run --> print result
          |
          v
Load credentials
          |
          v
HTTPS upload to Reg
          |
          v
Registry validates + indexes
```

`prolm publish` is planned registry-era work, not a live command. See
[Command Central](command-central.md) for command status.

## Planned MVS Resolution

```text
Direct + transitive reqs
          |
          v
Group reqs by package
          |
          v
MVS selects satisfying version
          |
          v
Apply guards
          |
          v
Persist exact LF records
```

Guards include no silent major-version jumps, no `latest` in lockfiles, visible
yanked-version handling, and clear conflict errors. See
[Package Manager Design](package-manager-design.md).
