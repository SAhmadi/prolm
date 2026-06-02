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
- `ERR`: fail with a user-facing error and hint where possible

## `prolm install`

```text
PF deps + existing LF
        |
        v
Stub flags set? -- yes --> ERR: --frozen/--offline/--no-verify Phase 2
        |
        no
        v
LF missing/stale hash? -- no --> Store complete? -- yes --> done
        | yes                         |
        v                             no
Resolve via Reg/source <--------------'
        |
        v
Cache hit? -- no --> Fetch HTTPS archive
        | yes            |
        '----------------'
               |
               v
        Verify SHA -- fail --> delete bad cache + ERR
               |
               v
        Safe unpack -- fail --> cleanup partial store + ERR
               |
               v
        Atomic LF write -- fail --> ERR
               |
               v
             done
```

See [Dependency Install Flow](dependency-install-flow.md), [Project Files](project-files.md),
and [Security Rules](security.md).

## `prolm add`

```text
Input name or URL
       |
       v
Normalize source
       |
       v
Resolve version/source -- fail --> manifest unchanged + ERR
       |
       v
Clone PF for save rollback
       |
       v
Mutate PF in memory
       |
       v
Run install sync -- fail --> on-disk PF unchanged + ERR
       |
       v
Save PF constraint -- fail --> attempt restore original PF + ERR
```

`prolm add` accepts SWI pack names, SWI listing URLs, and GitHub URLs today.
Future `--dev` and `--exact` behavior belongs in [Command Central](command-central.md).

## `prolm remove`

```text
Package name
      |
      v
Load PF
      |
      v
Declared in deps? -- no --> ERR with hint
      |
      yes
      v
Clone PF for rollback
      |
      v
Delete dependency
      |
      v
Save PF
      |
      v
Run install sync -- fail --> attempt restore original PF + ERR
      |
      v
LF synced
```

Removal currently targets `[dependencies]`. Future `--dev` behavior will cover
`[dev-dependencies]`.

## `prolm run`

```text
PF entry/scripts + RT config
          |
          v
Entry arg? -- script key --> script entry + script args after --
   |              |
   | path         v
   |          entry path
   v              |
[package].entry <-'
          |
          v
Read LF exact deps -- missing --> ERR: run prolm install
          |
          v
Verify Store paths -- missing --> ERR: run prolm install
          |
          v
Choose RT: --runtime > [package].runtime
          |
          v
Detect RT + min_version -- fail --> ERR
          |
          v
Build RT args: deps + entry + --goal + prolog args after --
          |
          v
Execute RT
```

Missing installed dependencies should explain that the user needs to run
`prolm install` first. See [Runtime Test And Check Flow](runtime-test-check-flow.md).

## `prolm test`

```text
PF + LF
  |
  v
Verify installed deps -- missing --> ERR: run prolm install
  |
  v
Choose/detect RT -- missing/version fail --> ERR
  |
  v
File arg? -- no --> discover **/*_test.pl and **/test_*.pl
  | yes             |
  v                 v
Selected files ----'
  |
  v
Apply --filter + --timeout
  |
  v
Run PlUnit
  |
  v
Report text or JSON
```

`--watch` and `--coverage` are planned flags.

## `prolm check`

```text
PF + LF
  |
  v
Verify installed deps -- missing --> ERR: run prolm install
  |
  v
Choose/detect RT -- missing/version fail --> ERR
  |
  v
Discover project .pl files
  |
  v
Load via RT with --timeout
  |
  v
Report warnings/errors as text or JSON
```

`prolm check` loads code in the runtime and is not a sandbox.

## `prolm env`

```text
Select runtime
      |
      v
Detect RT -- not found --> report not found
      |
      v
Discover PF -- absent --> report no project
      |
      v
Inspect Store
      |
      v
Print runtime, project, store, and environment summary
```

`prolm env` is diagnostic. It should not mutate project metadata.

## `prolm new`

```text
Name + template + runtime
        |
        v
Validate name/runtime -- fail --> ERR
        |
        v
Create project dir/files
        |
        v
Write PF + empty LF + .gitattributes
        |
        v
Render starter source, tests, README, .gitignore
```

Only the `app` template is live today. `library` and `cli` are planned.

## `prolm init`

```text
Current directory
       |
       v
Prolfile exists? -- yes --> ERR
       |
       no
       v
--yes? -- yes --> use defaults
  | no
  v
Prompt for metadata
       |
       v
--scan? -- yes --> scan .pl imports, skip built-in SWI libs
       |
       v
Write PF + empty LF
```

The scanner is a convenience pass, not a complete Prolog dependency analysis.

## Planned `prolm publish`

```text
PF package metadata
          |
          v
Clean tree? -- no --> ERR
          |
          v
Tag at HEAD? -- no --> ERR
          |
          v
Version unpublished? -- no --> ERR
          |
          v
LF committed + check pass? -- no --> ERR
          |
          v
Registry validation -- fail --> ERR
          |
          v
Build archive ---- --dry-run --> print planned upload
          |
          v
Load credentials -- fail --> ERR
          |
          v
HTTPS upload -- fail --> ERR
          |
          v
Registry indexes package
```

`prolm publish` is planned registry-era work, not a live command. See
[Command Central](command-central.md) for command status.

## Planned MVS Resolution

```text
Direct + transitive reqs
          |
          v
Circular path? -- yes --> ERR naming cycle
          |
          no
          v
Group reqs by package
          |
          v
Constraint conflict? -- yes --> ERR naming constraints
          |
          no
          v
MVS selects satisfying version
          |
          v
Yanked? -- fresh resolution skips; locked version warns
          |
          v
Reject "latest" lock value
          |
          v
Persist exact LF records: source, version, URL, checksum, deps
```

Guards include no silent major-version jumps, no `latest` in lockfiles, visible
yanked-version handling, and clear conflict errors. See
[Package Manager Design](package-manager-design.md).
