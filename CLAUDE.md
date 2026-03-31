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
11. [Testing Strategy](#11-testing-strategy)
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
  --template string   Project template: app | library | cli  (default: app)
  --runtime string    Target runtime: swi | gnu | scryer      (default: swi)

Examples:
  prolm new my-expert-system
  prolm new my-library --template library
  prolm new my-cli --template cli --runtime scryer
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

---

### prolm remove

Removes a dependency from Prolfile.toml and updates the lock.

```
prolm remove <pack> [flags]

Examples:
  prolm remove clpfd
```

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
[package]
name        = "my-expert-system"
version     = "0.1.0"
description = "A diagnostic expert system for network faults"
authors     = ["Ada Lovelace <ada@example.com>"]
license     = "MIT"
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

[[package]]
name     = "clpfd"
version  = "1.4.3"
source   = "swi-pack-index"          # swi-pack-index | github | prolm-registry
url      = "https://..."
checksum = "sha256:a3f2c1d9..."       # SHA-256 of the downloaded tarball

[[package]]
name     = "prosqlite"
version  = "0.9.11"
source   = "swi-pack-index"
url      = "https://..."
checksum = "sha256:b8e91d4f..."
```

**Lock rules:**
- Always commit `Prolfile.lock` to version control
- Never edit by hand
- `prolm install --frozen` fails if the lock would change (use in CI)
- Lockfile format must be forward-compatible — add fields, never remove

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
- `prolm add` / `prolm remove` / `prolm update`
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

| Layer              | Type               | Location                             |
|--------------------|--------------------|--------------------------------------|
| Manifest parsing   | Unit               | `internal/manifest/manifest_test.go` |
| Semver constraints | Unit + fuzz        | `internal/resolver/semver_test.go`   |
| MVS resolution     | Unit (table-driven)| `internal/resolver/resolver_test.go` |
| Checksum verify    | Unit               | `internal/installer/verify_test.go`  |
| Path traversal     | Unit (adversarial) | `internal/installer/unpack_test.go`  |
| Runtime detection  | Unit               | `internal/runtime/runtime_test.go`   |
| Full install flow  | Integration        | `testdata/` fixtures                 |
| CLI commands       | Integration        | Subprocess tests via `os/exec`       |

**Fuzz targets (critical):**
- `internal/resolver` — fuzz with random version constraint strings
- `internal/installer/unpack.go` — fuzz with crafted tarball paths

**CI must run:**
```bash
go test ./...                          # all unit tests
go test -race ./...                    # race detector (concurrent install)
go vet ./...
staticcheck ./...
```

---

## 12. Open Decisions & Future Work

| Decision                                | Status      | Notes                                       |
|-----------------------------------------|-------------|---------------------------------------------|
| Registry domain name                    | Open        | registry.prolm.dev ?                        |
| Registry auth provider                  | Open        | GitHub OAuth recommended                    |
| Tarball signing (Sigstore)              | Future      | Phase 4 consideration                       |
| Windows support                         | Best-effort | Atomic rename behaviour differs on Windows  |
| Private registries                      | Future      | For enterprise users                        |
| Workspace support (monorepo)            | Phase 4     | workspace.toml, similar to Go workspaces    |
| `prolm doc` — pldoc HTML generation     | Phase 4     | Wrap SWI-Prolog pldoc tools                 |
| SWI pack bridge (`pack.pl` generation)  | Phase 2     | Backward compat with `pack_install/1`       |
| `prolm audit` — yanked dep scanning     | Phase 4     | Inspired by `npm audit`                     |
| Dependency vulnerability DB             | Future      | Would need community buy-in                 |
| Cross-implementation test matrix        | Phase 3     | Run `prolm test` on SWI + GNU + Scryer      |

---
