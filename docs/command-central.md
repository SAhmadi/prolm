# prolm Command Central

This is the command catalog for `prolm`. It records the live CLI surface and
the planned commands that still need implementation so command work does not
disappear into roadmap prose.

## Status Legend

- `Available`: implemented and part of the live CLI today.
- `Phase 1.5`: planned for CLI UX and command-contract hardening.
- `Phase 2`: planned for production hardening.
- `Phase 3`: planned for registry work.
- `Stubbed`: visible today but intentionally not implemented yet.
- `Not implemented`: documented future surface with no live command yet.

Status lines describe the command as a whole. Individual flags may have a
different status and are annotated where that matters.

## Root Command

### `prolm`

**Status:** `Available`

```text
prolm [command]
```

Global flags:

```text
--config string     Path to Prolfile.toml (default: auto-discover upward)
--runtime string    Override runtime: swi | gnu | scryer
--verbose           Enable verbose/debug output
--no-color          Disable colour output (also respects NO_COLOR env var)
--json              Machine-readable JSON output (for CI/tooling integration)
-h, --help          Show help
```

Current behavior:

- With no subcommand, prints root help.
- With only global flags and no subcommand, prints a short guidance line then
  root help.
- Global flags can be combined with help, for example
  `prolm --no-color --help`.

## Project Setup

### `prolm new`

**Status:** `Available`

Creates a new project in a new directory.

```text
prolm new <name> [flags]

Flags:
  --template string   Project template: app (available now) |
                      library (Phase 2) | cli (Phase 2) (default: app)
  --runtime string    Target runtime: swi | gnu | scryer (default: swi)

Examples:
  prolm new my-expert-system
  prolm new my-app --template app
  prolm new my-app --runtime scryer
```

Flag precedence:

- Local `prolm new --runtime` overrides global `--runtime` when both are set.
- If neither is set, `prolm new` defaults to `swi`.

The app template creates:

```text
<name>/
|-- Prolfile.toml
|-- Prolfile.lock
|-- .gitignore
|-- README.md
|-- src/
|   `-- main.pl
`-- tests/
    `-- main_test.pl
```

Expected output: success summary and next-step hint. Missing project names
should return a clear argument error and usage text.

### `prolm init`

**Status:** `Available`

Initializes `prolm` in an existing directory.

```text
prolm init [flags]

Flags:
  --yes               Accept all defaults non-interactively
  --scan              Scan .pl files and auto-detect use_module deps
                      (default: true)

Examples:
  prolm init
  prolm init --yes
```

Expected output: success summary, optional dependency-scan summary, and a
next-step hint.

## Dependency Management

### `prolm install`

**Status:** `Available`

Resolves, fetches, and installs dependencies declared in `Prolfile.toml`.
The command remains manifest and lock driven: it does not accept package names
or URLs as positional install inputs.

```text
prolm install [flags]

Flags:
  --frozen            Fail if Prolfile.lock would change
  --offline           Use local cache only, no network requests
  --no-verify         Skip checksum verification (DANGEROUS - dev only)

Examples:
  prolm install
  prolm install --frozen
  prolm install --offline
```

Flag status:

- `--frozen`: `Stubbed`
- `--offline`: `Stubbed`
- `--no-verify`: `Stubbed`

Behavior:

- If `Prolfile.lock` exists and is current, verify local store state first.
- If the lock is absent or stale, resolve, fetch, and write lock state.
- Dependency mutation belongs to `prolm add`, `prolm remove`, and
  `prolm update`.
- Empty generated locks omit package entries instead of writing
  `package = []`.
- Generated lockfiles include non-empty dependency-state hashes in
  `[meta].prolfile_hash`.
- There is no separate `prolm uninstall` command for project dependencies in
  this workflow.

### `prolm add`

**Status:** `Available`

Adds a dependency and re-runs install.

```text
prolm add <name|url>

Examples:
  prolm add aop
  prolm add 'https://www.swi-prolog.org/pack/list?p=aop'
  prolm add https://github.com/hargettp/aop
```

Current behavior:

- Accepts SWI package names, direct SWI URLs, and GitHub URLs.
- Updates `Prolfile.toml`, installs the dependency, and writes or updates
  `Prolfile.lock`.
- SWI listing URLs normalize to package-name installs.
- GitHub repository URLs resolve to the latest stable semver tag or release;
  there is no HEAD fallback.
- If no stable semver tag or release exists, add fails with guidance to use an
  explicit tagged archive URL.
- When a stable resolved version exists, the manifest persists
  `^<resolved-version>`; otherwise it keeps `*` with a warning.
- Lockfile SHA-256 remains authoritative. If an upstream checksum cannot be
  byte-for-byte cross-verified for a derived archive URL, `prolm` warns and
  proceeds with lockfile pinning.

Future flags:

- `--dev`: `Phase 2`
- `--exact`: `Phase 2`

### `prolm remove`

**Status:** `Available`

Removes a dependency from `Prolfile.toml` and refreshes lock/install state.

```text
prolm remove <pack> [flags]

Examples:
  prolm remove aop
```

Current behavior:

- Removes from `[dependencies]` by default.
- Returns a clear error and hint when the package is not declared.
- Pairs with `prolm add`; `remove` is the canonical dependency-removal
  command.

Future behavior: once `--dev` exists, support removal from
`[dev-dependencies]`.

### `prolm update`

**Status:** `Phase 2`

Updates one or all dependencies to the latest allowed by semver constraints.

```text
prolm update [pack] [flags]

Flags:
  --breaking          Allow updates that cross a major version boundary

Examples:
  prolm update
  prolm update aop
```

## Runtime And Analysis

### `prolm run`

**Status:** `Available`

Invokes the Prolog runtime with project dependencies loaded.

```text
prolm run [entry] [flags] [-- <prolog-args>]

Flags:
  --runtime string    Override runtime for this invocation
  --goal string       Prolog goal to call instead of entry predicate
                      (default: main)

Examples:
  prolm run
  prolm run src/other.pl
  prolm run diagnose
  prolm run -- --mode interactive
  prolm run --runtime scryer
```

### `prolm test`

**Status:** `Available`

Discovers and runs PlUnit tests with formatted output.

```text
prolm test [file] [flags]

Flags:
  --filter string     Only run tests whose name matches this substring
  --verbose           Print each passing test
  --watch             Re-run on file change
  --timeout duration  Per-test timeout (default: 30s)
  --coverage          Emit predicate coverage report

Examples:
  prolm test
  prolm test tests/rules_test.pl
  prolm test --filter negation
  prolm test --watch
```

Flag status:

- `--filter`: `Available`
- `--verbose`: `Available`
- `--timeout`: `Available`
- `--watch`: `Phase 2`
- `--coverage`: `Not implemented`

Test discovery walks project files matching `**/*_test.pl` and
`**/test_*.pl`. The command exits non-zero on test failure.

### `prolm check`

**Status:** `Available`

Static analysis that loads code without executing goals.

```text
prolm check [flags]

Flags:
  --strict            Treat warnings as errors
  --no-deps           Skip dependency inputs
  --timeout duration  Limit runtime check execution

Examples:
  prolm check
```

Reports should cover undefined predicates, singleton variables, missing
imports, deprecated predicates, and missing module declarations as that
analysis becomes available.

### `prolm repl`

**Status:** `Phase 2`

Starts an interactive Prolog REPL with all project dependencies loaded.

```text
prolm repl [flags]

Examples:
  prolm repl
  prolm repl --runtime gnu
```

### `prolm env`

**Status:** `Available`

Prints environment information for debugging.

```text
prolm env
```

Expected output includes `prolm` version, runtime status, local store
location, project status, and proxy status. The intended full output shape is:

```text
prolm      1.0.0
runtime    swi -> /usr/local/bin/swipl  (9.2.1)
store      ~/.prolm/store/  (12 packs installed)
project    my-expert-system 0.1.0

Runtimes on PATH:
  swipl      9.2.1
  gprolog    1.5.0
  scryer     0.9.3
```

## Shell And Help

### `prolm completion`

**Status:** `Available`

Generates shell-completion scripts for supported shells.

```text
prolm completion [bash|zsh|fish|powershell]

Examples:
  prolm completion zsh
  prolm completion bash
  prolm completion fish
```

Setup has been validated for macOS `zsh`, Linux `bash`, and Linux `zsh`.

Setup platforms:

- macOS (zsh)
- Linux (bash)
- Linux (zsh)

Windows completion setup is deferred to the Windows support phase.

### `prolm version`

**Status:** `Available`

Prints the current `prolm` version.

### `prolm help`

**Status:** `Available`

```text
prolm help [command]
```

Keep help text aligned with implementation status and keep global-flag behavior
consistent with the root command.

## Registry-Era Commands

### `prolm search`

**Status:** `Phase 3`

Requires registry support.

```text
prolm search <term> [flags]

Flags:
  --tag string        Filter by tag
  --author string     Filter by author/org
  --runtime string    Only show packs compatible with this runtime
  --sort string       Sort by: relevance | downloads | updated
                      (default: relevance)
  --limit int         Max results (default: 20)

Examples:
  prolm search clp
  prolm search json --runtime scryer
  prolm search --tag constraint --sort downloads
```

### `prolm info`

**Status:** `Phase 3`

```text
prolm info <pack>[@version]

Examples:
  prolm info aop
  prolm info aop@0.0.9
```

### `prolm publish`

**Status:** `Phase 3`

Requires registry support.

```text
prolm publish [flags]

Flags:
  --dry-run           Validate and build tarball, do not upload
  --tag string        Publish under a dist-tag (default: latest)
  --access string     public | private

Examples:
  prolm publish --dry-run
  prolm publish
  prolm publish --tag beta
```

Pre-flight checks are hard failures:

1. `Prolfile.toml` version is not already published on the registry.
2. Git tag `v<version>` exists and HEAD is on it.
3. Working tree is clean.
4. `prolm check` passes with no errors.
5. `Prolfile.lock` is committed and up to date.

### `prolm yank`

**Status:** `Phase 3`

```text
prolm yank <version> [flags]

Flags:
  --reason string     Explain why users should avoid the version
  --undo              Un-yank a previously yanked version

Examples:
  prolm yank 1.4.2 --reason "breaks clpb compat, use 1.4.3"
```

### `prolm login`

**Status:** `Phase 3`

```text
prolm login [flags]

Flags:
  --token string      Provide token directly for CI/non-interactive use
```

### `prolm logout`

**Status:** `Phase 3`

```text
prolm logout
```

### `prolm owner`

**Status:** `Phase 3`

```text
prolm owner list <pack>
prolm owner add <pack> <username>
prolm owner remove <pack> <username>
```
