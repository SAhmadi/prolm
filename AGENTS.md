# CLAUDE.md — prolm Project Reference

> This file is the authoritative reference for the **prolm** project.
> Load it at the start of every Claude Code session: it contains architecture
> decisions, security rules, implementation phases, and the challenges that
> must be kept in mind at all times during development.

---

## Table of Contents

1. [What is prolm?](#1-what-is-prolm)
2. [Tech Stack](#2-tech-stack)
3. [Command Reference](#3-command-reference)
4. [Flow Diagrams](#4-flow-diagrams)
5. [Codebase Structure](#5-codebase-structure)
6. [Prolfile.toml & Prolfile.lock Reference](#6-prolfiletoml--prolfilelock-reference)
7. [Implementation Phases](#7-implementation-phases)
8. [Package Manager Challenges](#8-package-manager-challenges)
9. [Security Rules — Non-Negotiable](#9-security-rules--non-negotiable)
10. [Go + Cobra + Viper Conventions](#10-go--cobra--viper-conventions)
11. [Testing Strategy](TESTING.md)
12. [Open Decisions & Future Work](#12-open-decisions--future-work)

---

## 1. What is prolm?

`prolm` is a **project manager and build tool for Prolog**, written in Go.
It is the missing `cargo`/`poetry` of the Prolog world.

**Core problems it solves:**

- No standard way to declare, fetch, or pin dependencies across Prolog implementations
- Running a Prolog project requires knowing gnarly `swipl -g "use_module(...)"` invocations
- The SWI-Prolog pack system has no lockfile, no offline install, no CI-friendly workflow
- The ecosystem is fragmented across SWI-Prolog, GNU Prolog, and Scryer Prolog

**Design principles:**

- Predictable: same `Prolfile.lock` = same bytes on every machine
- Fast: local cache first, network only when necessary
- Safe: verify checksums before execution, refuse yanked versions
- Friendly: errors explain *what to do next*, not just what went wrong
- Composable: plays well with existing SWI `pack_install/1` ecosystem

**Target runtimes:** SWI-Prolog (primary), GNU Prolog, Scryer Prolog

---

## 2. Tech Stack

| Concern            | Library / Tool         | Reason                                              |
|--------------------|------------------------|-----------------------------------------------------|
| CLI framework      | `cobra`                | Battle-tested, sub-command model, auto help/docs    |
| Config / env vars  | `viper`                | TOML parsing, env var binding, config layering      |
| TOML               | `github.com/BurntSushi/toml` | Best Go TOML library, used by Cargo itself    |
| Semver             | `github.com/Masterminds/semver/v3` | Full semver 2.0 + constraint ranges     |
| HTTP client        | stdlib `net/http`      | Sufficient; wrap in internal `fetch` package        |
| Checksum           | stdlib `crypto/sha256` | SHA-256 for all tarball verification                |
| Archive            | stdlib `archive/tar` + `compress/gzip` | No external dep for unpack          |
| Testing            | `testing` + `testify` | Standard + assertion helpers                         |
| Logging            | `github.com/charmbracelet/log` | Pretty, levelled output for CLI             |
| Progress bars      | `github.com/schollz/progressbar/v3` | Download progress                      |
| Colour output      | `github.com/fatih/color` | Terminal colour, respects NO_COLOR env var        |

**Go version:** 1.22+ (use `go.work` for workspace support from day one)

**Important Cobra + Viper wiring note:**
Bind Viper to Cobra flags in `PersistentPreRunE`, not `init()`, to avoid
global state issues in tests. Always call `viper.BindPFlags(cmd.Flags())`.

---

## 3. Command Reference

### Global flags (all commands)

```
--config string     Path to Prolfile.toml (default: auto-discover upward)
--runtime string    Override runtime: swi | gnu | scryer
--verbose           Enable verbose/debug output
--no-color          Disable colour output (also respects NO_COLOR env var)
--json              Machine-readable JSON output (for CI/tooling integration)
```

---

### prolm new

Creates a new project in a new directory.

```
prolm new <name> [flags]

Flags:
  --template string   Project template: app (available now) | library (Phase 2) | cli (Phase 2)  (default: app)
  --runtime string    Target runtime: swi | gnu | scryer      (default: swi)

Examples:
  prolm new my-expert-system
  prolm new my-app --template app
  prolm new my-app --runtime scryer
```

**What it creates (app template):**
```
<name>/
├── Prolfile.toml
├── Prolfile.lock       (empty, committed to git)
├── .gitignore
├── README.md
├── src/
│   └── main.pl
└── tests/
    └── main_test.pl
```

---

### prolm completion

Generates shell-completion scripts for supported shells.

```
prolm completion [bash|zsh|fish|powershell]

Examples:
  prolm completion zsh
  prolm completion bash
```

**Current status:** command exists today. Setup and discoverability for macOS and Linux should be polished in Phase 1.5.

---

### prolm init

Initialises prolm in an existing directory (interactive).

```
prolm init [flags]

Flags:
  --yes               Accept all defaults non-interactively
  --scan              Scan .pl files and auto-detect use_module deps (default: true)

Examples:
  prolm init
  prolm init --yes
```

---

### prolm install

Resolves, fetches, and installs all dependencies.

```
prolm install [flags]

Flags:
  --frozen            Fail if Prolfile.lock would change (for CI)
  --offline           Use local cache only, no network requests
  --no-verify         Skip checksum verification (DANGEROUS — dev only)

Examples:
  prolm install
  prolm install --frozen          # CI usage — lock must already exist and be up to date
  prolm install --offline         # air-gapped / no network
```

**Behaviour:**
- If `Prolfile.lock` exists and is up to date: fast path, just verify local store
- If `Prolfile.lock` is absent or stale: resolve → fetch → write lock
- Never overwrites an existing lock without the user's `prolm update`
- Follows a cargo-style project flow: `prolm add` / `prolm remove` mutate dependencies, while `prolm install` syncs from manifest+lock
- `prolm install` is manifest-driven in MVP/Phase 1.5 (no package-name/URL positional install and no separate `prolm uninstall`)

---

### prolm add

Adds a dependency and re-runs install.

```
prolm add <pack> [version-constraint] [flags]

Flags:
  --dev               Add to [dev-dependencies] instead of [dependencies]
  --exact             Pin to exact version (=x.y.z) rather than ^x.y.z

Examples:
  prolm add clpfd
  prolm add clpfd "^1.4"
  prolm add plunit --dev
  prolm add prosqlite --exact
```

**Current status:** planned command. The first pre-registry implementation path is expected to land in Phase 1.5 using `prolm add <name|url>`.

---

### prolm remove

Removes a dependency from Prolfile.toml and updates the lock.

```
prolm remove <pack> [flags]

Examples:
  prolm remove clpfd
```

**Current status:** planned command, paired with `prolm add` for Phase 1.5 dependency lifecycle symmetry. `prolm remove` is the canonical removal command (no separate `prolm uninstall`).

---

### prolm update

Updates one or all dependencies to the latest allowed by semver constraints.

```
prolm update [pack] [flags]

Flags:
  --breaking          Allow updates that cross a major version boundary (careful)

Examples:
  prolm update              # update all deps within their semver constraints
  prolm update clpfd        # update only clpfd
```

---

### prolm run

Invokes the Prolog runtime with all dependencies loaded.

```
prolm run [entry] [flags] [-- <prolog-args>]

Flags:
  --runtime string    Override runtime for this invocation
  --goal string       Prolog goal to call instead of entry predicate (default: main)

Examples:
  prolm run                          # runs entry from Prolfile.toml
  prolm run src/other.pl             # run a specific file
  prolm run diagnose                 # run a named [scripts] entry
  prolm run -- --mode interactive    # pass args to your Prolog main/1
  prolm run --runtime scryer         # override runtime
```

---

### prolm test

Discovers and runs PlUnit tests with formatted output.

```
prolm test [file] [flags]

Flags:
  --filter string     Only run tests whose name matches this substring
  --verbose           Print each passing test (default: only failures)
  --watch             Re-run on file change
  --timeout duration  Per-test timeout (default: 30s)
  --coverage          Emit predicate coverage report (Phase 3+)

Examples:
  prolm test
  prolm test tests/rules_test.pl
  prolm test --filter negation
  prolm test --watch
```

**Test discovery:** walks project for `**/*_test.pl` and `**/test_*.pl`.
Exits with code `1` on any failure (CI-safe).

---

### prolm check

Static analysis — loads code without executing goals.

```
prolm check [flags]

Flags:
  --strict            Treat warnings as errors

Examples:
  prolm check
```

Reports: undefined predicates, singleton variables, missing imports,
deprecated predicates, missing module declarations.

---

### prolm repl

Starts an interactive Prolog REPL with all project deps loaded.

```
prolm repl [flags]

Examples:
  prolm repl
  prolm repl --runtime gnu
```

---

### prolm env

Prints environment info for debugging.

```
prolm env

Output:
  prolm      1.0.0
  runtime    swi  →  /usr/local/bin/swipl  (9.2.1)  ✓
  store      ~/.prolm/store/  (12 packs installed)
  project    my-expert-system 0.1.0

  Runtimes on PATH:
    swipl      9.2.1  ✓  (active)
    gprolog    1.5.0  ✓
    scryer     0.9.3  ✓
```

---

### prolm search *(Phase 3 — requires registry)*

```
prolm search <term> [flags]

Flags:
  --tag string        Filter by tag
  --author string     Filter by author/org
  --runtime string    Only show packs compatible with this runtime
  --sort string       Sort by: relevance | downloads | updated  (default: relevance)
  --limit int         Max results (default: 20)

Examples:
  prolm search clp
  prolm search json --runtime scryer
  prolm search --tag constraint --sort downloads
```

---

### prolm info *(Phase 3)*

```
prolm info <pack>[@version]

Examples:
  prolm info clpfd
  prolm info clpfd@1.4.2
```

---

### prolm publish *(Phase 3 — requires registry)*

```
prolm publish [flags]

Flags:
  --dry-run           Validate + build tarball, do not upload
  --tag string        Publish under a dist-tag (default: latest)
  --access string     public | private (default: public)

Examples:
  prolm publish --dry-run
  prolm publish
  prolm publish --tag beta
```

**Pre-flight checks (hard failures):**
1. `Prolfile.toml` version not already published on registry
2. Git tag `v<version>` exists and HEAD is on it
3. Working tree is clean (no uncommitted changes)
4. `prolm check` passes with no errors
5. `Prolfile.lock` is committed and up to date

---

### prolm yank *(Phase 3)*

```
prolm yank <version> [flags]

Flags:
  --reason string     Explain why (shown to users, highly recommended)
  --undo              Un-yank a previously yanked version

Examples:
  prolm yank 1.4.2 --reason "breaks clpb compat, use 1.4.3"
```

---

### prolm login / logout *(Phase 3)*

```
prolm login [flags]

Flags:
  --token string      Provide token directly (for CI, non-interactive)

prolm logout
```

---

### prolm owner *(Phase 3)*

```
prolm owner list <pack>
prolm owner add <pack> <username>
prolm owner remove <pack> <username>
```

---

## 4. Flow Diagrams

### prolm install — full lifecycle

```
User runs: prolm install
              │
              ▼
  ┌─────────────────────┐
  │  Read Prolfile.toml │
  │  Parse [deps] +     │
  │  [dev-deps]         │
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐      ┌─────────────────────┐
  │  Prolfile.lock      │─yes─▶│  Verify local store │
  │  exists & current?  │      │  All packs present? │
  └──────────┬──────────┘      └──────────┬──────────┘
             │ no                          │ yes
             ▼                            ▼
  ┌─────────────────────┐      ┌─────────────────────┐
  │  Resolve versions   │      │  Done — fast path   │
  │  Query registry /   │      │  (no network)       │
  │  SWI index / GitHub │      └─────────────────────┘
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │  For each pack:     │
  │  In local cache?    │─yes─▶ skip download
  └──────────┬──────────┘
             │ no
             ▼
  ┌─────────────────────┐
  │  Download tarball   │
  │  ~/.prolm/cache/    │
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │  Verify SHA-256     │◀── ALWAYS, even from cache
  │  checksum           │
  └──────────┬──────────┘
             │ mismatch → ABORT, delete bad file, error
             ▼
  ┌─────────────────────┐
  │  Unpack to          │
  │  ~/.prolm/store/    │
  │  <name>/<version>/  │
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │  Write Prolfile.lock│
  │  (name, version,    │
  │  source, checksum)  │
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │  Done               │
  └─────────────────────┘
```

---

### prolm run — invocation assembly

```
User runs: prolm run [entry]
              │
              ▼
  ┌─────────────────────┐
  │  Read Prolfile.toml │
  │  entry, runtime,    │
  │  [runtime.swi] flags│
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │  Read Prolfile.lock │
  │  Get exact versions │
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │  For each dep:      │
  │  Verify pack exists │
  │  in ~/.prolm/store/ │◀── if missing: "run prolm install first"
  └──────────┬──────────┘
             │
             ▼
  ┌──────────────────────────────────────────────────────┐
  │  Assemble swipl invocation:                          │
  │                                                      │
  │  swipl -O                               (from flags) │
  │    --stack-limit=2g                     (from flags) │
  │    -g "use_module('/path/clpfd/...')"   (dep 1)      │
  │    -g "use_module('/path/prosqlite/...')"(dep 2)     │
  │    -g "use_module('src/main')"          (entry)      │
  │    -g "main"                            (goal)       │
  │    -t halt                              (exit after) │
  └──────────┬───────────────────────────────────────────┘
             │
             ▼
  ┌─────────────────────┐
  │  exec() — replace   │
  │  current process    │
  │  with swipl         │
  └─────────────────────┘
```

---

### prolm publish — pre-flight + upload

```
User runs: prolm publish
              │
              ▼
  ┌─────────────────────────────────────┐
  │  Pre-flight checks (all must pass)  │
  │                                     │
  │  1. Prolfile.toml valid             │
  │  2. version not on registry         │
  │  3. git tag v<version> exists       │
  │  4. HEAD == tag commit              │
  │  5. git working tree is clean       │
  │  6. prolm check passes              │
  │  7. Prolfile.lock committed         │
  └──────────┬──────────────────────────┘
             │ any fail → abort with clear message
             ▼
  ┌─────────────────────┐
  │  Build tarball      │
  │  .pl + Prolfile.toml│
  │  + README.md        │
  └──────────┬──────────┘
             │
    --dry-run? ──yes──▶ print manifest, exit 0
             │
             ▼
  ┌─────────────────────┐
  │  Load token from    │
  │  ~/.prolm/credentials│
  └──────────┬──────────┘
             │ missing → "run prolm login first"
             ▼
  ┌─────────────────────┐
  │  POST tarball to    │
  │  registry API       │
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │  Registry validates │
  │  + stores to R2/S3  │
  │  + indexes metadata │
  └──────────┬──────────┘
             │
             ▼
  ┌─────────────────────┐
  │  Published ✓        │
  │  https://registry.. │
  └─────────────────────┘
```

---

### Version resolution (MVS — Minimum Version Selection)

```
  Prolfile.toml declares:
    clpfd   ^1.4        ←── "I need at least 1.4, compatible up to <2.0"
    http    ^7.0
    clpfd's Prolfile.toml also declares:
      http  ^7.1        ←── transitive dep, wants >=7.1

  MVS algorithm:
    Collect all version requirements for each package
    across the entire dependency graph.

    clpfd: [^1.4]         → resolve to 1.4.3  (latest in range)
    http:  [^7.0, ^7.1]   → resolve to 7.1.2  (minimum satisfying ALL constraints)

    Rule: take the MAXIMUM of the minimum requirements.
    Never silently upgrade to a new major version.
    Never take latest if it doesn't satisfy all constraints.
```

---

## 5. Codebase Structure

```
prolm/
│
├── main.go                          ← entry point: calls cmd.Execute()
│
├── cmd/                             ← Cobra command definitions (thin layer only)
│   ├── root.go                      ← root command, global flags, PersistentPreRunE
│   ├── new.go                       ← prolm new
│   ├── init.go                      ← prolm init
│   ├── install.go                   ← prolm install
│   ├── add.go                       ← prolm add
│   ├── remove.go                    ← prolm remove
│   ├── update.go                    ← prolm update
│   ├── run.go                       ← prolm run
│   ├── test.go                      ← prolm test
│   ├── check.go                     ← prolm check
│   ├── repl.go                      ← prolm repl
│   ├── env.go                       ← prolm env
│   ├── search.go                    ← prolm search  (Phase 3)
│   ├── info.go                      ← prolm info    (Phase 3)
│   ├── publish.go                   ← prolm publish (Phase 3)
│   ├── yank.go                      ← prolm yank    (Phase 3)
│   ├── login.go                     ← prolm login   (Phase 3)
│   └── owner.go                     ← prolm owner   (Phase 3)
│
├── internal/                        ← all business logic (not importable externally)
│   │
│   ├── manifest/                    ← Prolfile.toml read/write/validate
│   │   ├── manifest.go              ← ProlFile struct, Load(), Save()
│   │   ├── validate.go              ← semantic validation (not just schema)
│   │   └── manifest_test.go
│   │
│   ├── lockfile/                    ← Prolfile.lock read/write
│   │   ├── lockfile.go              ← LockFile struct, Load(), Save(), IsUpToDate()
│   │   └── lockfile_test.go
│   │
│   ├── resolver/                    ← dependency resolution
│   │   ├── resolver.go              ← main Resolve() entry point
│   │   ├── mvs.go                   ← Minimum Version Selection algorithm
│   │   ├── graph.go                 ← dependency graph construction
│   │   ├── semver.go                ← semver helpers wrapping Masterminds/semver
│   │   └── resolver_test.go        ← critical: many edge case tests here
│   │
│   ├── registry/                    ← registry sources (interface + implementations)
│   │   ├── registry.go              ← Registry interface
│   │   ├── swi.go                   ← SWI-Prolog pack index scraper
│   │   ├── github.go                ← GitHub API search (topic:swi-prolog)
│   │   ├── prolm.go                 ← prolm registry API client (Phase 3)
│   │   └── cache.go                 ← HTTP response cache (ETags, 304s)
│   │
│   ├── installer/                   ← fetch, verify, unpack
│   │   ├── installer.go             ← Install() orchestrator
│   │   ├── fetch.go                 ← download with progress bar
│   │   ├── verify.go                ← SHA-256 checksum verification
│   │   ├── unpack.go                ← tar.gz extraction (path traversal safe)
│   │   ├── store.go                 ← local store at ~/.prolm/store/
│   │   └── installer_test.go
│   │
│   ├── runtime/                     ← Prolog runtime detection + invocation
│   │   ├── runtime.go               ← Runtime interface: Detect(), BuildArgs(), Exec()
│   │   ├── swi.go                   ← SWI-Prolog implementation
│   │   ├── gnu.go                   ← GNU Prolog implementation
│   │   ├── scryer.go                ← Scryer Prolog implementation
│   │   └── runtime_test.go
│   │
│   ├── scaffold/                    ← prolm new / prolm init
│   │   ├── scaffold.go              ← NewProject(), InitProject()
│   │   ├── scanner.go               ← scan .pl files for use_module deps
│   │   ├── templates/               ← embedded via go:embed
│   │   │   ├── app/
│   │   │   │   ├── Prolfile.toml.tmpl
│   │   │   │   ├── main.pl.tmpl
│   │   │   │   ├── main_test.pl.tmpl
│   │   │   │   ├── .gitignore.tmpl
│   │   │   │   └── README.md.tmpl
│   │   │   ├── library/
│   │   │   │   ├── Prolfile.toml.tmpl
│   │   │   │   ├── pack.pl.tmpl
│   │   │   │   └── ...
│   │   │   └── cli/
│   │   │       └── ...
│   │   └── scaffold_test.go
│   │
│   ├── testrunner/                  ← prolm test
│   │   ├── runner.go                ← discover files, invoke swipl, parse output
│   │   ├── discover.go              ← find *_test.pl files
│   │   ├── parser.go                ← parse PlUnit TAP-like output
│   │   ├── reporter.go              ← format output (colour, JSON)
│   │   └── runner_test.go
│   │
│   ├── checker/                     ← prolm check (static analysis)
│   │   ├── checker.go
│   │   └── checker_test.go
│   │
│   ├── auth/                        ← credentials store (Phase 3)
│   │   ├── auth.go                  ← Load/Save token from ~/.prolm/credentials
│   │   └── auth_test.go
│   │
│   └── ui/                          ← shared output helpers
│       ├── printer.go               ← Info(), Success(), Warn(), Error(), Fatal()
│       ├── spinner.go               ← progress spinners
│       └── table.go                 ← tabular output (search results, etc.)
│
├── pkg/                             ← public types importable by external tools
│   └── prolfile/
│       ├── types.go                 ← ProlFile, Dependency, LockEntry — public structs
│       └── types_test.go
│
├── testdata/                        ← fixture projects for integration tests
│   ├── valid-project/
│   ├── no-lockfile/
│   ├── circular-deps/               ← must error gracefully
│   ├── yanked-version/
│   └── bad-checksum/                ← must abort
│
├── Prolfile.toml                    ← prolm manages itself (dogfooding)
├── Prolfile.lock
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 6. Prolfile.toml & Prolfile.lock Reference

### Prolfile.toml

```toml
[meta]
prolfile_version = 1                 # format version — increment on breaking changes
min_prolm_version = "0.1.0"         # minimum prolm CLI version required

[package]
name        = "my-expert-system"
version     = "0.1.0"
description = "A diagnostic expert system for network faults"
authors     = ["Ada Lovelace <ada@example.com>"]
license     = "MIT"                  # SPDX identifier
homepage    = "https://github.com/ada/my-expert-system"
entry       = "src/main.pl"        # entrypoint for `prolm run`
runtime     = "swi"                # swi | gnu | scryer

[dependencies]
clpfd       = "^1.4"
prosqlite   = "^0.9"
http        = "^7.0"

[dev-dependencies]
plunit      = "*"                  # only for `prolm test`

[runtime.swi]
min_version = "9.0"                # fail fast if wrong SWI version
flags       = ["-O", "--stack-limit=2g"]

[runtime.scryer]
min_version = "0.9"

[scripts]
start       = "prolm run src/main.pl"
diagnose    = "prolm run src/diagnose.pl -- --mode interactive"
```

### Version constraint syntax

| Syntax     | Meaning                                  |
|------------|------------------------------------------|
| `"^1.4"`   | `>=1.4.0, <2.0.0`  (compatible)          |
| `"~1.4.2"` | `>=1.4.2, <1.5.0`  (patch-level only)    |
| `">=1.4"`  | at least 1.4, no upper bound             |
| `"=1.4.3"` | exact pin                                |
| `"*"`      | any version                              |
| `"1.4.3"`  | treated as `^1.4.3` (following npm rule) |

### Prolfile.lock

```toml
# This file is auto-generated by prolm. Do not edit by hand.
# Commit this file to version control for reproducible installs.

[meta]
lock_version  = 1
prolfile_hash = "sha256:e3b0c442..."   # hash of [dependencies] + [dev-dependencies]

[[package]]
name         = "clpfd"
version      = "1.4.3"
source       = "swi-pack-index"        # swi-pack-index | github | prolm-registry
url          = "https://..."
checksum     = "sha256:a3f2c1d9..."    # SHA-256 of the downloaded tarball
dependencies = ["http@7.1.2"]          # resolved transitive deps (for inspection)

[[package]]
name         = "prosqlite"
version      = "0.9.11"
source       = "swi-pack-index"
url          = "https://..."
checksum     = "sha256:b8e91d4f..."
dependencies = []

[[package]]
name         = "http"
version      = "7.1.2"
source       = "swi-pack-index"
url          = "https://..."
checksum     = "sha256:f1d2e3a4..."
dependencies = []
```

**Lock rules:**
- Always commit `Prolfile.lock` to version control
- Never edit by hand
- `prolm install --frozen` fails if the lock would change (use in CI)
- Lockfile format must be forward-compatible — add fields, never remove
- `[[package]]` entries MUST be sorted by name (deterministic output)
- `dependencies` field enables `prolm tree` and easier debugging
- `[meta] prolfile_hash` enables staleness detection without mtime
- Install order is deterministic: topological sort of dependency graph,
  ties broken alphabetically by package name

---

## 7. Implementation Phases

### Phase 1 — "Works for me" (MVP)
**Goal:** prolm is useful for the author's own Prolog projects.

**Scope:**
- `prolm new` (app template only)
- `prolm init` (with dep scanning)
- `prolm install` (SWI pack index only, no GitHub fallback yet)
- `prolm run`
- `prolm test` (PlUnit wrapper, discovery, coloured output)
- `prolm check`
- `prolm env`
- Prolfile.toml + Prolfile.lock read/write
- SHA-256 checksum verification
- Local store at `~/.prolm/store/`
- SWI-Prolog only (no GNU, no Scryer)

**Done when:**
```bash
prolm new hello && cd hello
prolm install          # fetches plunit
prolm run              # prints hello
prolm test             # 1 passed
prolm check            # no warnings
```

**NOT in Phase 1:**
- Semver resolution (use exact versions from SWI index)
- Lock staleness detection
- Lockfile --frozen flag
- Any registry beyond SWI pack index
- GNU / Scryer support

---

### Phase 2 — "Production hardening"
**Goal:** Correct, safe, robust enough for other teams to use.

**Scope:**
- Full MVS dependency resolution with semver constraint solving
- Lockfile staleness detection (`prolm install` detects drift)
- `--frozen` flag for CI
- Dependency command hardening: full `prolm add` / `prolm remove` flag support and `prolm update`
- `prolm repl`
- GNU Prolog runtime support
- Scryer Prolog runtime support
- `prolm new --template library` and `--template cli`
- Circular dependency detection (hard error)
- `prolm env` fully implemented
- `--offline` mode
- GitHub fallback registry (search `topic:swi-prolog`)
- Proper error messages with actionable next steps
- `prolm test --watch`
- Integration test suite against `testdata/` fixtures

---

### Phase 3 — "Registry"
**Goal:** A centralised prolm registry at `registry.prolm.dev`.

**Scope:**
- `prolm search`, `prolm info`
- `prolm login`, `prolm logout`
- `prolm publish` (with full pre-flight checks)
- `prolm yank`
- `prolm owner`
- Registry backend: Go API server + Cloudflare R2 storage + Supabase metadata DB
- Registry frontend: search/browse website
- Dist-tags (latest, beta, etc.)
- Download counters
- Dependents tracking

**Registry infrastructure:**
- Object storage: Cloudflare R2 (zero egress fees — critical)
- Metadata DB: Supabase (PostgreSQL) free tier covers early scale
- CDN: Cloudflare (in front of R2, automatic)
- API server: deployed on Fly.io or Railway
- Auth: GitHub OAuth + API token model
- Estimated cost at Prolog ecosystem scale: ~$0–10/month

---

### Phase 4 — "Ecosystem"
**Goal:** Make prolm the standard across all Prolog implementations.

**Scope:**
- VSCode extension integration (LSP + prolm)
- GitHub Actions `setup-prolm` action
- `prolm doc` — generate HTML docs from pldoc comments
- `prolm bench` — benchmarking harness
- Multi-pack workspaces (`workspace.toml`)
- `prolm audit` — check for yanked or vulnerable versions
- SWI-Prolog pack compatibility bridge (`pack.pl` auto-generation)
- Windows support (currently best-effort)

---

## 8. Package Manager Challenges

These are known hard problems. Every implementation decision must consider them.

### 8.1 Semver Edge Cases

```
Problem: Semver is simple in theory, chaotic in practice.

Known pitfalls:
  - Pre-release versions: 1.4.0-beta.1 < 1.4.0
    ^1.4 should NOT match 1.4.0-beta.1 (npm learned this the hard way)
    Only include pre-releases if the user explicitly requests them.

  - Build metadata: 1.4.3+build.2 == 1.4.3 for comparison purposes
    Strip build metadata before any comparison.

  - "v" prefix: git tags are often "v1.4.3", semver strings are "1.4.3"
    Normalise on ingest — always strip leading "v".

  - Zero major versions: 0.x.y has different compat rules
    ^0.4.2 means >=0.4.2 <0.5.0 (NOT <1.0.0)
    Many package managers get this wrong.

  - Constraint intersection: "^1.4" AND "^1.3" from two different deps
    Must correctly compute intersection: >=1.4.0 <2.0.0

  - "Latest" is not a version
    Never store "latest" in a lockfile. Resolve it to a real version at lock time.
```

**Rule:** Use `github.com/Masterminds/semver/v3` and do not write custom semver
comparison logic. Add fuzz tests for the resolver.

---

### 8.2 Dependency Resolution

```
Problem: Finding a valid set of package versions satisfying all constraints
         across the whole transitive dependency graph is NP-complete in general.

Our approach: MVS (Minimum Version Selection), same as Go modules.

MVS properties:
  - Always deterministic: same input → same output
  - Always selects the MINIMUM version satisfying all constraints
  - No backtracking needed (unlike npm's SAT solver)
  - Naturally produces reproducible builds

Hard cases to test:
  - Diamond dependency: A→B^1.0, A→C^1.0, B→D^1.0, C→D^1.2
    Must resolve D to 1.2 (max of minimums), not cause conflict.

  - Conflict: A requires D^1.0 (<2.0), B requires D^2.0 (>=2.0)
    Must fail with clear message naming both A and B.

  - Yanked version in range: best version is yanked
    Must skip yanked versions and pick next best.
    Must warn user that a yanked version was skipped.

  - Circular dependency: A→B, B→A
    Must detect cycle and abort with the cycle path in the error message.
    Implement cycle detection in graph.go using DFS with colour marking.

  - Missing package: dep listed in Prolfile.toml not in any registry
    Must fail immediately with the package name and tried sources.

  - Version does not exist: ^1.4 but only 1.3.x published
    Must fail with clear message, not silently use 1.3.x.
```

---

### 8.3 Lockfile Correctness

```
Problem: The lockfile is the contract for reproducibility. Any bug here
         breaks the "same bytes on every machine" guarantee.

Rules:
  - Lockfile is the source of truth for install, run, and test
  - Prolfile.toml drives add/update/remove
  - These two must never silently diverge

Staleness detection (Phase 2):
  Compute a hash of all [dependencies] + [dev-dependencies] entries.
  Store this hash in Prolfile.lock as [meta] prolfile_hash = "..."
  On `prolm install`, re-compute hash. If different → lock is stale → re-resolve.

  DO NOT use file mtime for staleness. mtimes are unreliable (git checkout,
  docker builds, CI caches all reset mtimes in surprising ways).

  Use content hash only.

Atomicity:
  Write lockfile atomically: write to Prolfile.lock.tmp, then rename.
  A crash mid-write must never leave a partial lockfile.
  In Go: use os.Rename() which is atomic on POSIX systems.
  On Windows: use github.com/natefinish/go-atomicfilewriter or equivalent.
```

---

### 8.4 Network & Caching

```
Problem: Network requests are slow, flaky, and may not be available.

Rules:
  - Cache HTTP responses using ETags and Last-Modified / 304 Not Modified
  - Cache location: ~/.prolm/cache/ (separate from store/)
  - Tarballs in cache are never re-downloaded if checksum matches
  - Respect HTTP Retry-After headers
  - Timeout all HTTP requests (default 30s per request, configurable)
  - --offline mode: no network, cache only, fail if pack not cached

Cache layout:
  ~/.prolm/
  ├── cache/                        ← raw downloaded tarballs (can be deleted)
  │   └── <name>/<version>.tar.gz
  ├── store/                        ← unpacked packs (rebuilt from cache)
  │   └── <name>/<version>/
  └── credentials                   ← API tokens (chmod 600)

Never store credentials in cache or store directories.
```

---

### 8.5 Path Traversal in Tarball Extraction

```
Problem: A malicious tarball can contain paths like ../../.bashrc
         and overwrite arbitrary files during extraction.
         This is a critical security vulnerability, not a theoretical one.
         npm was affected by this in 2019.

Mitigation (MUST implement in unpack.go):
  For every entry in the tarball:
    1. Compute the clean join of destDir + entry.Name
    2. Call filepath.Clean() on the result
    3. Verify it has destDir as a prefix (use strings.HasPrefix)
    4. If not → ABORT extraction, delete partial output, return error

  // Safe extraction — implement exactly this pattern:
  func safeExtract(destDir, entryPath string) (string, error) {
      dest := filepath.Join(destDir, entryPath)
      dest = filepath.Clean(dest)
      if !strings.HasPrefix(dest, filepath.Clean(destDir)+string(os.PathSeparator)) {
          return "", fmt.Errorf("illegal path in tarball: %s", entryPath)
      }
      return dest, nil
  }

Also reject:
  - Absolute paths in tarballs (/etc/passwd)
  - Symlinks pointing outside the destination directory
  - Hardlinks to sensitive files
  - Device files, named pipes, sockets
```

---

### 8.6 Checksum Verification

```
Problem: A compromised CDN or registry could serve a modified tarball.

Rules:
  - ALWAYS verify SHA-256 after download, before unpacking
  - ALWAYS verify against the checksum stored in Prolfile.lock
  - On mismatch: delete the downloaded file, return a hard error, never proceed
  - Store checksums as "sha256:<hex>" in lockfile (prefix makes algorithm explicit)
  - If registry returns a checksum in its API response, verify against THAT
    before writing to lock (prevents TOCTOU between fetch and lock write)

  On first install (no lock yet):
    Fetch tarball → compute checksum → write to lock
  On subsequent install (lock exists):
    Read expected checksum from lock → fetch tarball → verify → fail if mismatch
```

---

### 8.7 Concurrent Installs

```
Problem: Two `prolm install` processes running simultaneously in the same
         project can corrupt the local store or produce incorrect lockfiles.

Mitigation:
  - Take a file lock on ~/.prolm/store/.lock before any write operations
  - Use github.com/gofrs/flock or equivalent
  - Lock must be released on exit, including on panic (defer)
  - If lock cannot be acquired within 30s → error with "another prolm process
    is running, wait for it to finish"
  - Never take the lock for read-only operations (install --frozen, run, test)
```

---

### 8.8 Yanked Versions

```
Problem: A yanked version is known-bad but must not break existing locked installs.

Behaviour:
  - Resolution: skip yanked versions when computing fresh resolutions
  - Locked installs: still install yanked versions if they are in Prolfile.lock
    (the user explicitly locked to this version; do not silently change it)
  - Warn loudly if a locked version is yanked:
    "Warning: clpfd@1.4.2 is yanked: 'breaks clpb compat, use 1.4.3'
     Run `prolm update clpfd` to move to a newer version."
  - Never silently upgrade away from a yanked locked version
  - --frozen installs still succeed on yanked locked versions (with warning)
```

---

### 8.9 Token & Credentials Security

```
Problem: API tokens for prolm publish must be stored and transmitted securely.

Rules:
  - Store token in ~/.prolm/credentials (TOML format)
  - Set file permissions to 0600 on creation (owner read/write only)
  - Never log tokens — check all log/debug output paths
  - Never include tokens in error messages
  - Never store tokens in Prolfile.toml or Prolfile.lock
  - Accept PROLM_TOKEN env var for CI (takes precedence over credentials file)
  - Tokens transmitted only over HTTPS — reject HTTP registry URLs
  - Rotate: `prolm login` invalidates and replaces the existing token
```

---

### 8.10 Supply Chain Integrity

```
Problem: Package managers are high-value targets for supply chain attacks
         (see: event-stream, ua-parser-js, colors.js in npm ecosystem).

Mitigations:
  - Checksums in lockfile (see 8.6)
  - Immutable versions (see 8.3) — once published, a version is frozen
  - Yank mechanism (see 8.8) — bad versions can be flagged, not silently replaced
  - Owner model — only authorised users can publish to a pack name (Phase 3)
  - Audit command (Phase 4) — `prolm audit` checks for known-yanked deps
  - Consider: Sigstore/cosign signing for tarballs (Phase 4, future)
  - Never auto-update without user consent — `prolm update` is explicit
  - Never run install scripts from packages (unlike npm `postinstall`)
    Prolog packs are source-only; there is no need for install scripts.
    If a pack ships an install script, prolm ignores it.
```

---

### 8.11 Package Authenticity & Signing

```
Problem: Checksums verify integrity (the bytes haven't changed) but not
         authenticity (who published the package). A compromised registry
         can serve malicious packages with valid checksums.

Current state:
  Phase 1-2 rely on checksum verification only. This is necessary but
  not sufficient for full supply chain trust.

Future mitigations (Phase 4+):
  - Sigstore / cosign signing for tarballs
  - Publisher identity binding (e.g. GitHub org ↔ package namespace)
  - Trusted publisher mode (allowlist of verified publishers)
  - Transparency log for all publish events (append-only, auditable)

Rule:
  A valid checksum alone is not sufficient to establish trust.
  Design all data structures (lockfile, registry API) to accommodate
  a future `signature` field from day one, even if it is optional initially.
```

---

### 8.12 Package Namespacing

```
Problem: A global flat namespace does not scale and creates security risks.

Risks:
  - Name collisions: common names like "json", "http", "utils" will conflict
  - Typosquatting attacks: "htp", "clpdfd", "httpp" — slight misspellings
    that trick users into installing malicious packages
  - Name squatting: reserving popular names with empty packages

Solution:
  Use namespaced packages: <owner>/<package>

  Examples:
    swi/http
    ada/clpfd
    community/json-parser

Rules:
  - Prolfile.toml must support namespaced identifiers in [dependencies]
  - Prolfile.lock must store the full namespaced identifier
  - Registry must enforce ownership of namespaces
  - Reserved org prefixes: swi/*, gnu/*, scryer/* for official packs
  - Implement Levenshtein distance check on publish to flag potential
    typosquatting (e.g. "htp" vs "http")

CRITICAL: Even if Phase 1 ignores namespaces, all data structures
(ProlFile struct, LockEntry struct, registry API) must support them
from day one. Retrofitting namespaces is extremely painful.
```

---

### 8.13 Module & Namespace Isolation

```
Problem: All dependencies load into a single Prolog runtime process.
         Multiple packages may define conflicting predicates, operators,
         or module names. Unlike compiled languages, Prolog has no
         linker to catch these at build time.

Risks:
  - Two packages export the same predicate name → silent override
  - Custom operator definitions (op/3) in deps change parsing semantics
  - term_expansion/goal_expansion hooks in deps rewrite host code

Rules:
  - All packages MUST use Prolog module declarations (:- module(Name, Exports))
  - No implicit global predicate leakage — unexported predicates stay private
  - prolm check should warn on packages that define predicates without a module
  - prolm check should flag packages that define term_expansion or
    goal_expansion (see 8.22)

Open questions:
  - Can two versions of the same package coexist in one runtime?
    (Answer: generally no in Prolog — this is a hard constraint of MVS)
  - Should prolm enforce module naming conventions (e.g. module name
    must match package name)?
```

---

### 8.14 Reproducibility Guarantees

```
Problem: "Same lockfile = same bytes" is the promise, but its scope
         must be clearly defined to avoid false expectations.

Guaranteed:
  - Same Prolfile.lock → same tarball bytes downloaded
  - Same tarball bytes → same source files unpacked
  - Deterministic dependency resolution (MVS)
  - Deterministic install order (sorted, stable graph traversal)

NOT guaranteed (non-goals):
  - Cross-OS binary reproducibility (Prolog is source-only, but
    C extensions in some SWI packs may compile differently)
  - Cross-runtime equivalence (SWI, GNU, and Scryer have different
    built-in predicates and module systems)
  - Cross-runtime-version equivalence (SWI 9.0 vs 9.2 may behave
    differently on the same source code)

Rationale:
  Prolog implementations differ fundamentally. prolm guarantees
  source-level reproducibility, not behavioural equivalence.
  The [runtime.swi] min_version field helps narrow this gap.
```

---

### 8.15 Runtime Execution Model

```
Problem: `prolm run` and `prolm test` execute arbitrary Prolog code.
         Users must understand the trust boundary.

Important:
  prolm does NOT sandbox execution in any way.

  When you run `prolm run`, the Prolog runtime has full access to:
    - The filesystem (read, write, delete)
    - The network (HTTP requests, sockets)
    - Environment variables (including secrets)
    - Subprocesses (shell_exec, process_create)

  prolm does NOT:
    - Restrict filesystem access
    - Restrict network access
    - Limit memory or CPU usage (beyond runtime flags)
    - Isolate dependencies from each other at runtime

User responsibility:
  Only run trusted projects and dependencies.
  For untrusted code, use external sandboxing (containers, VMs, seccomp).

Future consideration:
  A `prolm run --sandbox` mode using OS-level sandboxing (seccomp,
  pledge/unveil, App Sandbox) could be explored in Phase 4+.
```

---

### 8.16 Workspaces & Local Dependencies

```
Problem: Real-world Prolog projects often span multiple packages
         developed in tandem. Without local dependency support,
         developers must publish to a registry just to test changes
         across packages.

Phase 2 — Path dependencies:
  [dependencies]
  my-lib = { path = "../my-lib" }

  Rules:
    - Path dependencies are resolved locally, not from any registry
    - Path dependencies are NOT written to Prolfile.lock (they are mutable)
    - prolm install verifies the path exists and contains a valid Prolfile.toml
    - prolm publish rejects projects with path dependencies
      (must be converted to versioned deps before publishing)

Phase 4 — Workspaces:
  workspace.toml at the repo root, listing member packages:

  [workspace]
  members = ["core", "plugins/*", "tools/cli"]

  Rules:
    - Shared dependency resolution across all workspace members
    - Single Prolfile.lock at the workspace root
    - `prolm test --workspace` runs tests for all members
    - Similar to Go workspaces and Cargo workspaces
```

---

### 8.17 Prolfile Format Versioning

```
Problem: The Prolfile.toml format will evolve. Without explicit versioning,
         old prolm versions may silently misparse new format features,
         and new versions may break on old files.

Solution:
  Add a required version field to Prolfile.toml:

  [meta]
  prolfile_version = 1

Rules:
  - prolm MUST check prolfile_version before parsing
  - If the version is higher than supported → hard error with message:
    "This Prolfile.toml requires prolm version X or newer.
     You are running prolm Y. Run `prolm self-update` to upgrade."
  - Increment prolfile_version only on breaking format changes
  - Maintain backward compatibility within the same version number
  - Version 1 = Phase 1 format (everything documented in this file)

Lockfile versioning:
  Similarly, add to Prolfile.lock:

  [meta]
  lock_version = 1
  prolfile_hash = "sha256:..."
```

---

### 8.18 Plugin System

```
Problem: Users and organisations will need to extend prolm without
         forking the core (custom registries, resolvers, commands).

Goal:
  Allow extensibility without modifying core prolm.

Approach (Phase 4+):
  Subprocess-based plugins (recommended over Go plugin system):
    - Plugins are standalone executables named `prolm-<name>`
    - prolm discovers them on $PATH
    - Communication via stdin/stdout JSON protocol
    - Plugin lifecycle: init → execute → result
    - No shared memory, no ABI compatibility issues

  Use cases:
    - Custom private registries (prolm-registry-artifactory)
    - Custom resolvers (prolm-resolver-nix)
    - Additional commands (prolm-format, prolm-lint)
    - IDE integrations

  Inspiration: git's plugin model, cargo's custom subcommands

Rules:
  - Plugins must not bypass security rules (SEC-1 through SEC-14)
  - Core commands cannot be overridden by plugins
  - Plugin errors must not crash prolm — isolate failures
```

---

### 8.19 Lockfile Merge Conflict Resolution

```
Problem: When two branches add different dependencies, Prolfile.lock
         will have TOML merge conflicts. Manually resolving these is
         error-prone and a top friction point in Cargo and npm workflows.

         Git cannot auto-merge lockfiles because the format is not
         line-independent — version resolution is holistic.

Solution:
  `prolm install` should detect and auto-heal lockfile conflicts:

  1. On load, scan Prolfile.lock for Git conflict markers
     (<<<<<<<, =======, >>>>>>>)
  2. If found: discard the conflicted lock, re-resolve from Prolfile.toml
  3. Write a clean Prolfile.lock
  4. Print: "Lockfile had merge conflicts — re-resolved from Prolfile.toml"

  Additional measures:
    - Add a .gitattributes recommendation in `prolm new` output:
      Prolfile.lock merge=ours
      (prefer "ours" on conflict, then `prolm install` fixes it)
    - `prolm lock repair` as an explicit command for this workflow
```

---

### 8.20 Cache Garbage Collection

```
Problem: ~/.prolm/cache/ and ~/.prolm/store/ grow unboundedly.
         Over time, old versions accumulate and consume significant
         disk space, especially on CI machines.

Solution:
  prolm cache clean [flags]

  Flags:
    --older-than duration   Remove entries older than this (default: 90d)
    --max-size string       Remove oldest entries until under this limit (e.g. 1GB)
    --dry-run               Show what would be deleted without deleting

  prolm cache list
    Show disk usage per package and total.

Rules:
  - Never auto-delete cache without user action
  - Never delete entries for versions currently referenced in any
    Prolfile.lock in the current project
  - Store last-accessed timestamp per cache entry for LRU eviction
  - CI environments should use `prolm cache clean --older-than 7d`
    in their pipeline cleanup steps
```

---

### 8.21 Network Proxy & Corporate Environments

```
Problem: Users behind corporate proxies, university firewalls, or
         air-gapped environments cannot use prolm without proxy support.
         This is a blocker for enterprise and academic adoption.

Rules:
  - Respect standard proxy environment variables:
    HTTP_PROXY, HTTPS_PROXY, NO_PROXY (Go stdlib net/http does this
    automatically via http.ProxyFromEnvironment — verify it is used)
  - Support custom CA certificate bundles:
    PROLM_CA_BUNDLE or SSL_CERT_FILE environment variable
    Append custom certs to the default system pool
  - Support .netrc for authentication to private registries
  - --offline mode (already specified) covers air-gapped environments
  - Document proxy setup in `prolm env` output:
    proxy: HTTPS_PROXY=http://proxy.corp:8080

Implementation note:
  Go's net/http handles HTTP_PROXY/HTTPS_PROXY automatically.
  For custom CA bundles, use crypto/x509.SystemCertPool() + AppendCertsFromPEM().
```

---

### 8.22 Prolog-Specific Hook Abuse

```
Problem: Prolog has powerful metaprogramming hooks that can rewrite
         code at load time. A malicious or careless package can use
         these to silently alter the behaviour of other packages
         or the host project.

Dangerous hooks:
  - term_expansion/2: rewrites terms (clauses) at load time
    A package defining term_expansion can inject, modify, or delete
    clauses from any module that loads after it.
  - goal_expansion/2: rewrites goals at compile time
    Can replace a call to safe_predicate/1 with malicious_code/0.
  - op/3 (operator definitions): changes parsing precedence and associativity
    A package that redefines standard operators can make valid Prolog
    parse differently or fail silently.

Mitigations:
  - `prolm check` should scan dependencies for term_expansion,
    goal_expansion, and op/3 definitions and flag them as warnings
  - `prolm check --strict` should treat these as errors
  - Future: `prolm audit` should report which packages use these hooks
  - Document this risk prominently — most Prolog users are unaware

Note:
  Legitimate uses of these hooks exist (e.g. DCG notation, test frameworks).
  The goal is awareness and auditability, not blanket prohibition.
```

---

### 8.23 Package Deprecation

```
Problem: Deprecation is distinct from yanking.
  - Yanking: a specific VERSION is known-bad, skip it in resolution
  - Deprecation: an entire PACKAGE is superseded, point to replacement

Examples:
  "clpfd" is deprecated in favour of "clpz" (SWI-Prolog 9+)
  "old-http" is deprecated, use "swi/http" instead

Registry metadata:
  {
    "deprecated": true,
    "deprecated_message": "Use clpz instead. See https://..."
    "successor": "clpz"
  }

Behaviour:
  - `prolm install` warns on deprecated deps:
    "Warning: clpfd is deprecated: Use clpz instead."
  - `prolm add <deprecated-pack>` warns but allows it
  - `prolm search` marks deprecated packages clearly
  - Deprecated packages remain installable (unlike yanked versions)
  - `prolm update` can suggest successor packages
```

---

### 8.24 License Compliance

```
Problem: The Prolfile.toml has a `license` field but nothing checks
         license compatibility across the dependency tree. A GPL
         dependency in an MIT project creates legal issues that
         most developers won't notice until too late.

Solution (Phase 3+):
  prolm license [flags]

  Flags:
    --tree          Show license for every transitive dependency
    --deny string   Fail if any dep uses this license (e.g. --deny GPL-3.0)
    --allow string  Only allow these licenses (allowlist mode)

  Output:
    my-project (MIT)
    ├── clpfd 1.4.3 (BSD-2-Clause) ✓
    ├── prosqlite 0.9.11 (MIT) ✓
    └── http 7.1.2 (SWI-Prolog license) ⚠ review

Rules:
  - License field in Prolfile.toml should use SPDX identifiers
  - Registry should validate SPDX format on publish
  - `prolm publish` should warn if license field is missing
  - Inspiration: cargo-deny, license-checker (npm)
```

---

### 8.25 Self-Update & CLI Version Compatibility

```
Problem: No strategy for updating prolm itself, and no way for
         projects to require a minimum prolm version. An outdated
         CLI may silently mishandle new Prolfile features.

Solution:
  1. min_prolm_version field in Prolfile.toml:

     [meta]
     prolfile_version = 1
     min_prolm_version = "0.5.0"

     On every command, check this field. If the running prolm version
     is lower → hard error:
       "This project requires prolm >= 0.5.0 (you have 0.3.1).
        Run `prolm self-update` or visit https://prolm.dev/install"

  2. Self-update mechanism (Phase 2+):

     prolm self-update [flags]

     Flags:
       --check    Only check, do not update
       --version  Update to a specific version

     Implementation: download the appropriate binary from GitHub Releases
     and replace the current executable. Verify checksum of the new binary.

  3. Version check on startup (opt-in):
     If enabled, check GitHub Releases API for a newer version (at most
     once per 24 hours, cached). Print a one-line notice, never block.
```

---

### 8.26 Telemetry & Privacy

```
Problem: Users and organisations need to know whether prolm phones
         home. The spec is currently silent on this.

Decision: NO telemetry by default.

Rules:
  - The prolm CLI MUST NOT send any usage data, analytics, or
    crash reports without explicit opt-in
  - No tracking of commands run, packages installed, or error types
  - Registry-side download counters are acceptable (server logs)
  - If telemetry is ever added, it MUST be:
    - Opt-in only (not opt-out)
    - Controllable via `prolm config telemetry off`
    - Fully documented: what is sent, where, how often
    - Disabled when PROLM_NO_TELEMETRY=1 or DO_NOT_TRACK=1 is set
  - `prolm env` should display telemetry status

Rationale:
  Trust is the foundation of a package manager. Surprise data collection
  destroys trust faster than any feature can build it.
```

---

### 8.27 Graceful Degradation

```
Problem: Not all commands require a Prolog runtime, but prolm may
         fail early if it cannot detect one. Users should be able to
         manage manifests and lockfiles without a runtime installed.

Commands that do NOT require a Prolog runtime:
  prolm new, prolm init, prolm install, prolm add, prolm remove,
  prolm update, prolm env, prolm search, prolm info, prolm publish,
  prolm login, prolm logout, prolm cache, prolm license

Commands that DO require a Prolog runtime:
  prolm run, prolm test, prolm check, prolm repl

Rules:
  - Metadata-only commands must never fail due to missing runtime
  - Runtime-required commands must provide clear errors:
    "swipl not found on PATH.
     Install SWI-Prolog: https://www.swi-prolog.org/download/stable
     Or set PROLM_RUNTIME_PATH=/path/to/swipl"
  - `prolm env` should show runtime status without failing:
    "runtime: swi → not found ✗"
  - Runtime detection should check both PATH and common install locations
    (/usr/local/bin, /opt/homebrew/bin, snap paths, etc.)
```

---

### 8.28 Deterministic TOML Serialization

```
Problem: Without deterministic key ordering, every `prolm add` or
         `prolm update` produces noisy diffs that make code review
         harder and clutter git history.

Rules for Prolfile.toml (when prolm writes it):
  - Sections in fixed order: [meta], [package], [dependencies],
    [dev-dependencies], [runtime.*], [scripts]
  - Keys within each section: alphabetically sorted
  - Preserve user comments where possible (use a TOML library that
    supports comment-preserving round-trips, or document that comments
    may be lost on `prolm add`)

Rules for Prolfile.lock:
  - [meta] section first
  - [[package]] entries sorted by name, then by version
  - Keys within each [[package]]: name, version, source, url,
    checksum, dependencies (fixed order, not alphabetical)

Implementation:
  Use a TOML encoder that supports ordered output. If BurntSushi/toml
  does not preserve order, write a custom serializer for lockfile output.
  The lockfile is machine-generated — comment preservation is not needed.
```

---

## 9. Security Rules — Non-Negotiable

The following rules apply to **every line of code** in this project.
They are not optional and must be checked in every code review.

```
SEC-1   Verify SHA-256 checksum on every tarball, every time.
        No exceptions. Even from local cache.

SEC-2   Safe tarball extraction only. See 8.5.
        The safeExtract() function must be used for every tar entry.

SEC-3   No install scripts. Never execute code from a downloaded pack
        during install. Prolog is interpreted at runtime, not install time.

SEC-4   HTTPS only for all registry and download URLs.
        Reject any http:// URL in registry responses.

SEC-5   Credentials never logged, never in error messages, never in
        Prolfile.toml or Prolfile.lock.

SEC-6   File permissions: credentials file = 0600, store = 0755.
        Check permissions on load, fix or abort if wrong.

SEC-7   File lock on ~/.prolm/store/.lock for all write operations.
        Prevents concurrent install corruption.

SEC-8   Atomic lockfile writes only (write tmp, then rename).

SEC-9   Input validation on all registry API responses.
        Never trust server-side data without schema validation.
        A compromised registry should not be able to trigger
        path traversal or RCE via malformed API responses.

SEC-10  Rate limiting: respect 429 responses with exponential backoff.
        Do not hammer the registry or GitHub API.

SEC-11  No implicit cross-registry resolution.
        If multiple registries provide the same package name, FAIL
        with a clear error naming both sources. Require explicit
        source selection in Prolfile.toml:
          company-utils = { version = "^1.2", source = "internal" }
        This prevents dependency confusion attacks where a public
        package shadows a private one (or vice versa).

SEC-12  Enforce tarball and extraction limits.
        Max tarball size: 50 MB (configurable via PROLM_MAX_TARBALL_SIZE)
        Max extracted size: 200 MB
        Max file count: 10,000 files
        Max single file size: 20 MB
        On any violation: abort extraction, delete partial output,
        return hard error. This prevents zip/tar bomb attacks that
        exhaust disk space or memory.

SEC-13  Symlink resolution must stay within destination directory.
        After resolving all symlinks, the final path must remain
        inside destDir. Reject:
          - Symlinks whose target resolves outside destDir
          - Symlink chains (A→B→C) where any intermediate or final
            target escapes destDir
          - Relative symlinks that traverse above destDir via ../
        Use filepath.EvalSymlinks() and verify the result has
        destDir as a prefix. This hardens SEC-2 against symlink-
        based path traversal bypasses.

SEC-14  Lockfile source URL validation.
        All URLs in Prolfile.lock must match an expected registry
        origin from the known registry list. Reject URLs pointing
        to unexpected domains. Optionally enforce a domain allowlist
        via PROLM_ALLOWED_HOSTS. This prevents lockfile tampering
        attacks where URLs are modified to point to attacker-controlled
        servers that serve tarballs with valid checksums but different
        (malicious) content from a re-published version.
```

---

## 10. Go + Cobra + Viper Conventions

### Command structure pattern

```go
// cmd/install.go — thin command layer, all logic in internal/
var installCmd = &cobra.Command{
    Use:   "install",
    Short: "Install all dependencies from Prolfile.toml",
    RunE:  runInstall,  // Use RunE, not Run — always return errors
}

func init() {
    rootCmd.AddCommand(installCmd)
    installCmd.Flags().Bool("frozen", false, "Fail if Prolfile.lock would change")
    installCmd.Flags().Bool("offline", false, "Use local cache only")
}

func runInstall(cmd *cobra.Command, args []string) error {
    frozen, _ := cmd.Flags().GetBool("frozen")
    offline, _ := cmd.Flags().GetBool("offline")

    cfg, err := manifest.Load(viper.GetString("config"))
    if err != nil {
        return fmt.Errorf("reading Prolfile.toml: %w", err)
    }

    return installer.Install(cfg, installer.Options{
        Frozen:  frozen,
        Offline: offline,
    })
}
```

### Viper binding in root

```go
// cmd/root.go
func init() {
    rootCmd.PersistentFlags().String("config", "", "path to Prolfile.toml")
    rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")
    rootCmd.PersistentFlags().Bool("no-color", false, "disable colour")
    rootCmd.PersistentFlags().Bool("json", false, "JSON output")
}

// PersistentPreRunE fires before every command
var rootCmd = &cobra.Command{
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        viper.BindPFlags(cmd.Flags())
        viper.BindPFlags(cmd.PersistentFlags())
        viper.AutomaticEnv()
        viper.SetEnvPrefix("PROLM")  // PROLM_TOKEN, PROLM_VERBOSE, etc.
        return nil
    },
}
```

### Error handling conventions

```go
// Always wrap errors with context using %w
return fmt.Errorf("installing clpfd: %w", err)

// User-facing errors via ui package, not fmt.Println
ui.Error("clpfd@1.4.3 not found in any registry")
ui.Hint("Run `prolm search clpfd` to find available versions")

// Fatal only in main, never in internal packages
// internal packages return errors, cmd layer decides to fatal

// Exit codes:
//   0 = success
//   1 = user error (bad args, pack not found, etc.)
//   2 = internal error (bug, unexpected state)
//   3 = network error (registry unavailable)
```

### Testing conventions

```go
// Use t.TempDir() for all file system tests — auto-cleaned
func TestInstall(t *testing.T) {
    dir := t.TempDir()
    // ...
}

// Table-driven tests for resolver edge cases
var resolveTests = []struct {
    name    string
    deps    map[string]string
    want    map[string]string
    wantErr string
}{
    {"diamond dep", ...},
    {"circular dep", ...},
    {"yanked version skipped", ...},
}

// Integration tests use testdata/ fixtures
// Unit tests mock the registry interface
```

---

## 11. Testing Strategy

See **[TESTING.md](TESTING.md)** for the full testing reference: unit tests, integration
tests, adversarial security tests, fuzz targets, runtime compatibility matrix, test
writing conventions, and instructions for running CI locally.

**Always run `make ci` locally before opening a Pull Request.** This mirrors the GitHub
Actions workflow exactly (`go vet` → `staticcheck` → `go test -race -count=1`) and
catches failures — including staticcheck issues that have broken CI in the past — before
they reach the remote.

```bash
make ci   # vet + staticcheck + test -race — run this before every PR
```

---

## 12. Open Decisions & Future Work

| Decision                                | Status      | Notes                                       |
|-----------------------------------------|-------------|---------------------------------------------|
| Registry domain name                    | Open        | registry.prolm.dev ?                        |
| Registry auth provider                  | Open        | GitHub OAuth recommended                    |
| Tarball signing (Sigstore)              | Future      | Phase 4+ — see 8.11                         |
| Windows support                         | Best-effort | Atomic rename behaviour differs on Windows  |
| Private registries                      | Future      | For enterprise users                        |
| Workspace support (monorepo)            | Phase 4     | workspace.toml — see 8.16                   |
| `prolm doc` — pldoc HTML generation     | Phase 4     | Wrap SWI-Prolog pldoc tools                 |
| SWI pack bridge (`pack.pl` generation)  | Phase 2     | Backward compat with `pack_install/1`       |
| `prolm audit` — yanked dep scanning     | Phase 4     | Inspired by `npm audit`                     |
| Dependency vulnerability DB             | Future      | Would need community buy-in                 |
| Cross-implementation test matrix        | Phase 3     | Run `prolm test` on SWI + GNU + Scryer — see 11.2 |
| Package namespacing strategy            | Phase 3     | `<owner>/<package>` — design for it now, see 8.12  |
| Plugin architecture                     | Phase 4     | Subprocess-based, git-style — see 8.18      |
| Telemetry policy                        | Decided     | No telemetry by default — see 8.26          |
| License compliance tooling              | Phase 3     | `prolm license` command — see 8.24          |
| Package deprecation model               | Phase 3     | Distinct from yanking — see 8.23            |
| Self-update mechanism                   | Phase 2     | `prolm self-update` — see 8.25             |
| Lockfile merge conflict handling        | Phase 2     | Auto-heal on `prolm install` — see 8.19     |
| Cache garbage collection                | Phase 2     | `prolm cache clean` — see 8.20             |
| Prolog hook abuse detection             | Phase 2     | `prolm check` flags term_expansion — see 8.22 |
| TOML comment preservation               | Open        | Need library support — see 8.28             |

---
