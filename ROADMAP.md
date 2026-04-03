# prolm Implementation Roadmap

> Step-by-step build plan for **prolm** — the Prolog project manager.
> Each sub-phase is a logical unit of work. Complete them in order (dependencies noted).
> Derived from [CLAUDE.md](CLAUDE.md).

---

## Phase 0: Project Bootstrap

### 0.1 — Go module and directory skeleton

- [x] Run `go mod init` with chosen module path
- [x] Create directory tree per CLAUDE.md section 5:
  - `cmd/`
  - `internal/manifest/`, `internal/lockfile/`, `internal/resolver/`, `internal/registry/`, `internal/installer/`, `internal/runtime/`, `internal/scaffold/`, `internal/testrunner/`, `internal/checker/`, `internal/auth/`, `internal/ui/`
  - `pkg/prolfile/`
  - `testdata/valid-project/`, `testdata/no-lockfile/`, `testdata/circular-deps/`, `testdata/yanked-version/`, `testdata/bad-checksum/`
- [x] Create `main.go` entry point (calls `cmd.Execute()`)
- [x] Add placeholder `doc.go` in each internal package so `go build ./...` works
- [x] Install dependencies: `cobra`, `viper`, `BurntSushi/toml`, `Masterminds/semver/v3`, `charmbracelet/log`, `schollz/progressbar/v3`, `fatih/color`, `stretchr/testify`
- [x] Create `Makefile` with targets: `build`, `test`, `vet`, `lint`, `run`
- [x] Verify `go build ./...` and `go test ./...` pass

**Testing:** Compilation succeeds. `go vet ./...` clean.

### 0.2 — Root Cobra command and global flags

- [x] Implement `cmd/root.go`:
  - `PersistentPreRunE` binding Viper flags, `viper.AutomaticEnv()`, env prefix `PROLM`
  - Global persistent flags: `--config`, `--runtime`, `--verbose`, `--no-color`, `--json`
  - Respect `NO_COLOR` env var
- [x] Implement `cmd/version.go` — prints hardcoded version (e.g. `0.1.0-dev`)
- [x] Verify `go run . version` and `go run . --help` work

**Testing:** Smoke test. Unit test for `Execute()` returning nil on `version`. ✓

---

## Phase 1: "Works for me" (MVP)

> **Done when:** `prolm new hello && cd hello && prolm install && prolm run && prolm test && prolm check` all succeed.

### 1.1 — Public types (`pkg/prolfile/`)

*Depends on: 0.1*

- [x] Define `ProlFile` struct matching Prolfile.toml schema (CLAUDE.md section 6):
  - `Meta` — `ProlfileVersion int`, `MinProlmVersion string`
  - `Package` — `Name`, `Version`, `Description`, `Authors []string`, `License`, `Homepage`, `Entry`, `Runtime`
  - `Dependencies map[string]string`
  - `DevDependencies map[string]string`
  - `RuntimeConfig` map for `[runtime.swi]` etc. (`Flags []string`, `MinVersion string`)
  - `Scripts map[string]string`
- [x] Define `LockFile` struct:
  - `Meta` — `LockVersion int`, `ProlfileHash string`
  - `Packages []LockEntry` (sorted by name)
- [x] Define `LockEntry` struct:
  - `Name`, `Version`, `Source`, `URL`, `Checksum string`
  - `Dependencies []string`
  - `Signature string` (empty for now — future-proofing per 8.11)
- [x] Define `Dependency` struct for richer deps (Phase 2 path deps, source override)
- [x] Support namespaced identifiers (`owner/name`) from day one (per 8.12)

**Testing:** Unit tests — struct construction, zero values, basic serialization round-trip.
**Security:** `Signature` field placeholder addresses future signing (8.11).

### 1.2 — UI helpers (`internal/ui/`)

*Depends on: 0.2 (needs global flags)*

- [x] `printer.go` — `Info()`, `Success()`, `Warn()`, `Error()`, `Fatal()`, `Hint()`
  - Respect `--no-color` / `NO_COLOR` via `fatih/color`
  - Respect `--json` flag for structured output
- [x] `spinner.go` — wrap `schollz/progressbar` for download progress
  - `StartSpinner(msg)` / `StopSpinner()`

**Testing:** `Info` writes to buffer. `NO_COLOR` suppresses ANSI codes.

### 1.3 — Manifest read/write (`internal/manifest/`)

*Depends on: 1.1*

- [x] `manifest.go` — `Load(path) (*ProlFile, error)`:
  - Auto-discover `Prolfile.toml` by walking upward from cwd if path is empty
  - Check `prolfile_version` — hard error if higher than supported (per 8.17)
  - Check `min_prolm_version` — hard error if current version is lower (per 8.25)
- [x] `manifest.go` — `Save(path, pf) error`:
  - Deterministic key ordering (per 8.28)
  - Section order: `[meta]`, `[package]`, `[dependencies]`, `[dev-dependencies]`, `[runtime.*]`, `[scripts]`
  - Keys within sections: alphabetically sorted
- [x] `validate.go` — `Validate(pf) error`:
  - `name` non-empty, valid identifier (alphanumeric + hyphens)
  - `version` valid semver
  - `entry` is a `.pl` file path
  - `runtime` one of `swi`, `gnu`, `scryer`
  - Dependency version strings parseable by `Masterminds/semver`

**Testing:** Table-driven tests — valid round-trip, missing fields, unknown prolfile_version, malformed TOML.
**Security:** SEC-9 (input validation on all parsed fields).

### 1.4 — Lockfile read/write (`internal/lockfile/`)

*Depends on: 1.1*

- [x] `lockfile.go` — `Load(path) (*LockFile, error)`:
  - Missing file returns `nil` (not an error — means first install)
  - Validate `lock_version` field
- [x] `lockfile.go` — `Save(path, lf) error`:
  - **Atomic writes** (SEC-8): write to `Prolfile.lock.tmp`, then `os.Rename()`
  - `[[package]]` entries sorted by name
  - Keys in fixed order: `name`, `version`, `source`, `url`, `checksum`, `dependencies`
- [x] `IsEmpty(lf) bool`

**Testing:** Round-trip identity, temp file cleanup, missing file returns nil, corrupted TOML errors, Git conflict markers produce clear error.
**Security:** SEC-8 (atomic writes).

### 1.5 — SWI-Prolog pack index registry client (`internal/registry/`)

*Depends on: 1.1*

- [x] `registry.go` — define `Registry` interface:
  - `Search(name) ([]PackageVersion, error)`
  - `Versions(name) ([]PackageVersion, error)`
  - `DownloadURL(name, version) (string, error)`
- [x] Define `PackageVersion` struct: `Name`, `Version`, `URL`, `Checksum`, `Dependencies []string`, `Yanked bool`
- [x] `swi.go` — `SWIRegistry` querying the SWI-Prolog pack index:
  - HTTPS only (SEC-4) — reject any HTTP URL
  - HTTP timeout: 30s per request
  - Validate all response fields (SEC-9)
- [x] `cache.go` — HTTP response caching:
  - ETags / `If-None-Match` / 304 handling (per 8.4)
  - Cache to `~/.prolm/cache/registry/`
- [x] Respect `429 Too Many Requests` with exponential backoff (SEC-10)

**Implementation note:** The SWI pack index has no public JSON API. Phase 1 uses
HTML scraping of `/pack/list` and `/pack/list?p=<name>` via `golang.org/x/net/html`.
If SWI changes their HTML structure the parse functions in `swi.go` must be updated.
See BUGS.md for known issues deferred to later phases.

**Testing:** Unit tests with `httptest.Server` mocking SWI responses. Malformed response handling. HTTP URL rejection. Timeout handling. Cache hit/miss.
**Security:** SEC-4, SEC-9, SEC-10.

### 1.6 — Installer core: fetch, verify, unpack, store (`internal/installer/`)

*Depends on: 1.1, 1.2, 1.3, 1.4, 1.5*

This is the most security-critical sub-phase.

- [ ] `fetch.go` — `Fetch(url, destPath) error`:
  - Progress bar via UI helpers
  - HTTPS only (SEC-4)
  - Max tarball size: 50 MB, configurable via `PROLM_MAX_TARBALL_SIZE` (SEC-12)
  - Respect 429 with backoff (SEC-10)
- [ ] `verify.go` — `Verify(filePath, expectedChecksum) error`:
  - Compute SHA-256, compare to expected (SEC-1)
  - On mismatch: **delete the file**, return hard error
  - Checksum format: `sha256:<hex>`
- [ ] `verify.go` — `ComputeChecksum(filePath) (string, error)`
- [ ] `unpack.go` — `Unpack(tarballPath, destDir) error`:
  - Implement `safeExtract()` exactly as in CLAUDE.md 8.5 (SEC-2)
  - Reject absolute paths
  - Reject symlinks pointing outside destDir (SEC-13)
  - Reject hardlinks to files outside destDir
  - Reject device files, named pipes, sockets
  - Max extracted size: 200 MB (SEC-12)
  - Max file count: 10,000 files (SEC-12)
  - Max single file size: 20 MB (SEC-12)
  - On any violation: delete partial extraction, return hard error
- [ ] `store.go` — `Store` struct managing `~/.prolm/store/`:
  - `PackPath(name, version) string`
  - `IsInstalled(name, version) bool`
  - `EnsureDir()` — create store dir with permissions 0755 (SEC-6)
  - File lock on `~/.prolm/store/.lock` for write ops (SEC-7)
- [ ] `installer.go` — `Install(manifest, lock, opts) (*LockFile, error)`:
  - Full install flow per CLAUDE.md section 4 flow diagram
  - For each dep: check lock -> check store -> fetch -> verify -> unpack
  - Validate lockfile URLs against known registry origins (SEC-14)
  - Never execute code from downloaded packs (SEC-3)
  - Write lockfile atomically (SEC-8)

**Testing (adversarial — per 11.1):**
- `verify_test.go`: checksum match, mismatch (deletes file), empty file, truncated file, bit-flipped content
- `unpack_test.go`: path traversal (`../../.bashrc`, `/etc/passwd`), symlink escape, tar bomb, excessive file count, device files, invalid filenames (NULL bytes)
- `fetch_test.go`: HTTP rejection, timeout, 429 backoff
- `store_test.go`: directory creation, permissions, concurrent access with file lock
- `installer_test.go`: full flow with mocked registry

**Security:** SEC-1, SEC-2, SEC-3, SEC-4, SEC-6, SEC-7, SEC-8, SEC-9, SEC-10, SEC-12, SEC-13, SEC-14.

### 1.7 — SWI-Prolog runtime detection and invocation (`internal/runtime/`)

*Depends on: 1.1*

- [ ] `runtime.go` — define `Runtime` interface:
  - `Name() string`
  - `Detect() (*RuntimeInfo, error)`
  - `BuildRunArgs(entry, deps, flags, goal) []string`
  - `BuildTestArgs(testFiles, deps, flags) []string`
  - `BuildCheckArgs(files, deps) []string`
  - `Exec(args) error`
- [ ] `RuntimeInfo` struct: `Path string`, `Version string`
- [ ] `Detect(name) (Runtime, error)` — factory; Phase 1 supports only `swi`
- [ ] `swi.go` — `SWIRuntime`:
  - `Detect()`: search PATH + common locations (`/usr/local/bin`, `/opt/homebrew/bin`, snap)
  - Run `swipl --version`, parse version
  - If not found: clear error with install URL (per 8.27)
  - If below `[runtime.swi].min_version`: error with version info
  - `BuildRunArgs()`: assemble invocation per CLAUDE.md section 4 run flow diagram
  - `Exec()`: `syscall.Exec` to replace process (or `os/exec` when capturing output)

**Testing:** Mock-based tests for argument assembly. Version parsing. Graceful error when swipl not found.
**Security:** Use `exec.Command` with argument arrays, never string concatenation.

### 1.8 — Scaffold: `prolm new` (`internal/scaffold/`, `cmd/new.go`)

*Depends on: 1.3*

- [ ] Create embedded templates via `go:embed` in `internal/scaffold/templates/app/`:
  - `Prolfile.toml.tmpl`
  - `main.pl.tmpl` — `main :- write('Hello, world!'), nl.`
  - `main_test.pl.tmpl` — minimal PlUnit test
  - `.gitignore.tmpl`
  - `README.md.tmpl`
- [ ] `scaffold.go` — `NewProject(name, template, runtime) error`:
  - Validate name (no path separators, no `..`)
  - Create `<name>/`, `src/`, `tests/`
  - Render templates
  - Create empty `Prolfile.lock`
  - Run `git init` if git available
  - Add `.gitattributes` with `Prolfile.lock merge=ours` (per 8.19)
  - Print success message with next steps
- [ ] `cmd/new.go` — Cobra command:
  - `--template` (default: `app`, only option in Phase 1)
  - `--runtime` (default: `swi`, only option in Phase 1)

**Testing:** `NewProject` creates all expected files in `t.TempDir()`. Prolfile.toml parses back. Name validation rejects `../evil`.

### 1.9 — Scaffold: `prolm init` (`cmd/init.go`)

*Depends on: 1.3, 1.8*

- [ ] `scanner.go` — `ScanDeps(dir) (map[string]string, error)`:
  - Walk `.pl` files, find `:- use_module(library(X))` patterns
  - Return map of detected dep names to `"*"`
- [ ] `scaffold.go` — `InitProject(dir, scan, yes) error`:
  - Error if `Prolfile.toml` already exists
  - If `--scan`: run `ScanDeps`, pre-populate `[dependencies]`
  - If `--yes`: use directory name as project name + defaults
  - Otherwise: prompt interactively
  - Write `Prolfile.toml` and empty `Prolfile.lock`
- [ ] `cmd/init.go` — Cobra command with `--yes` and `--scan` flags

**Testing:** Scanner against known `.pl` files. Init in empty dir. Init with existing Prolfile.toml errors. `--yes` mode.

### 1.10 — `prolm install` command (`cmd/install.go`)

*Depends on: 1.3, 1.4, 1.5, 1.6*

- [ ] Cobra command (no `--frozen`/`--offline` in Phase 1)
- [ ] Flow:
  1. `manifest.Load()` — find and parse Prolfile.toml
  2. `lockfile.Load()` — try to load Prolfile.lock (may be nil)
  3. If lock exists and all packages in store: done (fast path)
  4. If lock nil or packages missing: query `SWIRegistry` for each dep
  5. Call `installer.Install()` to fetch/verify/unpack
  6. Write lockfile via `lockfile.Save()`
  7. Print summary: "Installed N packages in Xs"

**Testing:** Integration test with `testdata/valid-project/`. Fast path. Fresh install. Mocked SWI registry.
**Security:** All installer security rules apply transitively.

### 1.11 — `prolm run` command (`cmd/run.go`)

*Depends on: 1.3, 1.4, 1.7*

- [ ] Cobra command with `--runtime` (override) and `--goal` (default: `main`)
- [ ] Support positional args: `prolm run`, `prolm run src/other.pl`, `prolm run scriptname`
- [ ] Support `--` separator for passing args to Prolog
- [ ] Flow:
  1. Load manifest + lockfile
  2. Verify all locked deps are in store (error: "Run `prolm install` first")
  3. Resolve entry point from args, `[package].entry`, or `[scripts]`
  4. Detect runtime, build args, exec

**Testing:** Argument assembly. Error when swipl not found. Error when deps not installed.

### 1.12 — `prolm test` command (`internal/testrunner/`, `cmd/test.go`)

*Depends on: 1.7*

- [ ] `discover.go` — `Discover(projectDir) ([]string, error)`:
  - Find `**/*_test.pl` and `**/test_*.pl`
- [ ] `runner.go` — `Run(files, deps, runtime, opts) (*TestResult, error)`:
  - Invoke swipl with PlUnit test files
  - Capture stdout/stderr, pass to parser
  - `opts`: `Filter string`, `Verbose bool`, `Timeout time.Duration` (default 30s)
- [ ] `parser.go` — parse PlUnit output into structured `TestResult`:
  - `TotalTests`, `Passed`, `Failed`, `Errors`, individual test statuses
- [ ] `reporter.go` — format with colour:
  - Green checkmark passed, red X failed
  - Summary: "N passed, M failed (Xs)"
  - Respect `--no-color` and `--json`
  - Exit code 1 on any failure (CI-safe)
- [ ] `cmd/test.go` — Cobra command with `--filter`, `--verbose`, `--timeout`

**Testing:** Parser against sample PlUnit output. Discovery finds correct files. Reporter output. Exit code 1 on failure. `--filter` works.

### 1.13 — `prolm check` command (`internal/checker/`, `cmd/check.go`)

*Depends on: 1.7*

- [ ] `checker.go` — `Check(projectDir, runtime, deps, strict) (*CheckResult, error)`:
  - Load `.pl` files via swipl in check mode
  - Capture warnings: undefined predicates, singleton variables, missing imports, deprecated predicates, missing module declarations
  - Parse swipl warning output
  - If `--strict`: treat warnings as errors
- [ ] `cmd/check.go` — Cobra command with `--strict`
  - Exit code 1 on errors (or warnings in strict mode)

**Testing:** Known warning-producing Prolog files. `--strict` elevates warnings. Clean project exits 0.

### 1.14 — `prolm env` command (`cmd/env.go`)

*Depends on: 1.3, 1.7*

- [ ] Print diagnostic info:
  - prolm version
  - Runtime: `swi -> /path/to/swipl (version) [check/X]`
  - Store location + pack count
  - Project info (if in a project)
  - Proxy status: `HTTP_PROXY`, `HTTPS_PROXY` (per 8.21)
- [ ] Graceful when no swipl or no project (per 8.27)

**Testing:** Output format. Graceful handling when swipl/project missing.

### 1.15 — End-to-end smoke test

*Depends on: all of Phase 1*

- [ ] Create `testdata/valid-project/` fixture: complete Prolfile.toml, `src/main.pl`, `tests/main_test.pl`
- [ ] Write e2e integration test running the acceptance scenario:
  ```
  prolm new hello && cd hello && prolm install && prolm run && prolm test && prolm check
  ```
  via `os/exec` (requires swipl installed)
- [ ] Run full `go test ./...` and `go test -race ./...`
- [ ] Verify `go vet ./...` clean
- [ ] Run adversarial unpack tests one final time

**Phase 1 is complete when the e2e scenario passes.**

---

## Phase 2: "Production hardening"

### 2.1 — Semver resolution and MVS algorithm

- [ ] `internal/resolver/semver.go` — helpers wrapping `Masterminds/semver`: parse `^`, `~`, `>=`, `=`, `*` constraints (per CLAUDE.md section 6)
- [ ] `internal/resolver/graph.go` — build dependency graph from manifest + transitive deps
- [ ] `internal/resolver/mvs.go` — Minimum Version Selection:
  - For each package, take maximum of minimum requirements
  - Never silently cross a major version boundary
- [ ] `internal/resolver/resolver.go` — `Resolve(manifest, registries) (*LockFile, error)`
- [ ] Circular dependency detection via DFS with color marking — hard error with cycle path
- [ ] Pre-release handling: `^1.4` does NOT match `1.4.0-beta.1` (per 8.1)
- [ ] Zero major: `^0.4.2` means `>=0.4.2, <0.5.0` (per 8.1)
- [ ] Strip `v` prefix on ingest (per 8.1)
- [ ] Fuzz tests for semver constraint parsing

**Testing (table-driven):** Diamond dep, conflict (clear error naming both dependents), yanked version skip, circular dep, missing package, version-not-exist.
**Security:** SEC-9.

### 2.2 — Lockfile staleness detection and `--frozen`

- [ ] Compute `prolfile_hash` as SHA-256 of serialized `[dependencies]` + `[dev-dependencies]`
- [ ] `lockfile.IsUpToDate(manifest, lock)` — compare hash to `[meta].prolfile_hash`
- [ ] On `prolm install`: if stale, re-resolve from manifest
- [ ] `--frozen` flag: error if lock would change
- [ ] Lockfile merge conflict auto-heal: detect `<<<<<<<` markers, discard, re-resolve (per 8.19)

**Security:** SEC-8.

### 2.3 — `prolm add`, `prolm remove`, `prolm update`

- [ ] `cmd/add.go` — add dep to Prolfile.toml, re-resolve, re-install, write lock
  - `--dev`, `--exact` flags
- [ ] `cmd/remove.go` — remove from Prolfile.toml, re-resolve, write lock
- [ ] `cmd/update.go` — update one or all deps within semver constraints
  - `--breaking` allows major version bumps
- [ ] Deterministic TOML serialization in all writes (per 8.28)

### 2.4 — `prolm repl`

- [ ] `cmd/repl.go` — interactive swipl session with all deps loaded
- [ ] No `-t halt` — keep REPL alive

### 2.5 — GNU Prolog and Scryer Prolog support

- [ ] `internal/runtime/gnu.go` — `Runtime` interface for `gprolog`
- [ ] `internal/runtime/scryer.go` — `Runtime` interface for `scryer-prolog`
- [ ] Update `prolm env` to show all detected runtimes
- [ ] Update `--runtime` flag handling across all commands

### 2.6 — Additional templates

- [ ] `internal/scaffold/templates/library/` — library template (exports module, no `main.pl`)
- [ ] `internal/scaffold/templates/cli/` — CLI template (argument parsing, `main/1` with argv)
- [ ] Update `prolm new --template` to accept `library` and `cli`

### 2.7 — GitHub fallback registry

- [ ] `internal/registry/github.go` — search GitHub API for `topic:swi-prolog` repos
  - GitHub releases for version listing and tarball downloads
  - HTTPS only (SEC-4), validate responses (SEC-9), rate limits (SEC-10)
- [ ] Registry priority: SWI pack index first, GitHub fallback second
- [ ] Same package in both registries: require explicit `source` in Prolfile.toml (SEC-11)

### 2.8 — `--offline` mode

- [ ] Thread `offline` flag through to registry and installer
- [ ] Skip all network, use only local cache
- [ ] Clear error if required package not in cache

### 2.9 — `prolm test --watch`

- [ ] File watcher on `src/` and `tests/` (e.g. `fsnotify`)
- [ ] On change: re-run test suite
- [ ] Debounce rapid changes (100ms)

### 2.10 — Error messages and polish

- [ ] Audit every error path — does it explain what to do next?
- [ ] Use `ui.Hint()` consistently across all commands
- [ ] Examples: "swipl not found... Install at: ...", "Lockfile out of date. Run `prolm install`..."

### 2.11 — Integration test suite

- [ ] Tests against all `testdata/` fixtures: `valid-project/`, `no-lockfile/`, `circular-deps/`, `yanked-version/`, `bad-checksum/`
- [ ] Subprocess tests: build binary, invoke via `os/exec`, assert stdout/stderr/exit code
- [ ] CI pipeline: `go test -race ./...`, `go vet`, `staticcheck`
- [ ] Begin runtime compatibility matrix (SWI 9.0, 9.2, latest on Linux + macOS)

---

## Phase 3: "Registry"

### 3.1 — Auth and credentials (`internal/auth/`)

- [ ] `auth.go` — Load/Save token from `~/.prolm/credentials` (TOML)
  - File permissions 0600 (SEC-6)
  - `PROLM_TOKEN` env var takes precedence
- [ ] `cmd/login.go` — prompt for token or accept `--token` flag
- [ ] `cmd/logout.go` — delete credentials file
- [ ] Tokens only over HTTPS (SEC-4), never logged or in error messages (SEC-5)

### 3.2 — Registry backend

- [ ] Go API server (separate repo or `registry/` directory)
- [ ] Endpoints: publish, versions, download, search, yank, owner management
- [ ] Storage: Cloudflare R2 for tarballs
- [ ] Metadata: Supabase (PostgreSQL)
- [ ] Auth: GitHub OAuth + API token model
- [ ] Immutable versions — once published, cannot be replaced
- [ ] Validate SPDX license on publish

### 3.3 — Registry client (`internal/registry/prolm.go`)

- [ ] Implement `Registry` interface for prolm registry API
- [ ] Authenticated endpoints use token from `auth.Load()`
- [ ] Search, versions, download, publish, yank, owner operations

### 3.4 — CLI commands: search, info, publish, yank, owner

- [ ] `cmd/search.go` — `prolm search <term>` with `--tag`, `--author`, `--runtime`, `--sort`, `--limit`
  - Tabular output via `internal/ui/table.go`
- [ ] `cmd/info.go` — `prolm info <pack>[@version]`
- [ ] `cmd/publish.go` — pre-flight checks (all 5 from CLAUDE.md), build tarball, upload, `--dry-run`
- [ ] `cmd/yank.go` — `prolm yank <version> --reason "..."`, `--undo` to un-yank
- [ ] `cmd/owner.go` — `prolm owner list/add/remove`

### 3.5 — Registry frontend

- [ ] Search/browse website at `registry.prolm.dev`
- [ ] Package detail pages: README, versions, dependencies, download counts
- [ ] Dist-tags display, dependents tracking

### 3.6 — Advanced registry features

- [ ] Dist-tags (latest, beta, etc.)
- [ ] Download counters (server-side)
- [ ] Dependents tracking
- [ ] Package deprecation model (per 8.23)
- [ ] Typosquatting detection on publish (Levenshtein distance, per 8.12)
- [ ] License compliance: `prolm license` command with `--tree`, `--deny`, `--allow` (per 8.24)

---

## Phase 4: "Ecosystem"

### 4.1 — VSCode extension

- [ ] Language server integration
- [ ] prolm command palette integration
- [ ] Test runner integration with VS Code test explorer
- [ ] Prolfile.toml syntax highlighting and validation

### 4.2 — GitHub Actions

- [ ] `setup-prolm` action: install prolm + optional Prolog runtime
- [ ] Cache `~/.prolm/store/` and `~/.prolm/cache/` between CI runs
- [ ] Example workflow templates

### 4.3 — `prolm doc`

- [ ] Parse pldoc comments from `.pl` files
- [ ] Generate HTML documentation
- [ ] Wrap SWI-Prolog pldoc tools or standalone parser

### 4.4 — `prolm bench`

- [ ] Benchmarking harness for Prolog predicates
- [ ] Statistical analysis (mean, median, stddev, outlier detection)
- [ ] Regression detection between runs

### 4.5 — Multi-pack workspaces

- [ ] `workspace.toml` at repo root (per 8.16)
- [ ] Shared dependency resolution across members
- [ ] Single `Prolfile.lock` at workspace root
- [ ] `prolm test --workspace` runs all members

### 4.6 — `prolm audit`

- [ ] Check all locked deps against yanked/deprecated status
- [ ] Exit code 1 if issues found (CI-safe)

### 4.7 — Windows support

- [ ] Test file path handling with backslashes
- [ ] Atomic rename on Windows (`go-atomicfilewriter`)
- [ ] swipl detection on Windows (Program Files, chocolatey, etc.)
- [ ] Add Windows to CI matrix

### 4.8 — Plugin system

- [ ] Subprocess-based plugins named `prolm-<name>` on PATH (per 8.18)
- [ ] JSON protocol over stdin/stdout
- [ ] Plugin discovery and help integration
- [ ] Core commands cannot be overridden
- [ ] Plugin errors isolated from prolm core

### 4.9 — Additional features

- [ ] `prolm self-update` — download new binary from GitHub Releases, verify checksum (per 8.25)
- [ ] `prolm cache clean` — garbage collection with `--older-than`, `--max-size`, `--dry-run` (per 8.20)
- [ ] Prolog hook abuse detection in `prolm check`: flag `term_expansion`, `goal_expansion`, `op/3` (per 8.22)
- [ ] SWI pack compatibility bridge: `pack.pl` auto-generation
- [ ] Path dependencies: `my-lib = { path = "../my-lib" }` (per 8.16)

---

## Security Rules Cross-Reference

| Rule | Description | First applies |
|------|-------------|---------------|
| SEC-1 | SHA-256 checksum on every tarball, every time | 1.6 |
| SEC-2 | Safe tarball extraction (`safeExtract()`) | 1.6 |
| SEC-3 | No install scripts — never execute code during install | 1.6 |
| SEC-4 | HTTPS only for all registry/download URLs | 1.5 |
| SEC-5 | Credentials never logged or in error messages | 3.1 |
| SEC-6 | File permissions: credentials 0600, store 0755 | 1.6 |
| SEC-7 | File lock on store for write operations | 1.6 |
| SEC-8 | Atomic lockfile writes (write tmp, rename) | 1.4 |
| SEC-9 | Input validation on all registry API responses | 1.5 |
| SEC-10 | Rate limiting: respect 429 with exponential backoff | 1.5 |
| SEC-11 | No implicit cross-registry resolution | 2.7 |
| SEC-12 | Tarball and extraction size/count limits | 1.6 |
| SEC-13 | Symlink resolution stays within destDir | 1.6 |
| SEC-14 | Lockfile source URL validation against known origins | 1.6 |

---

## Dependency Order (Phase 1 Critical Path)

```
0.1 Bootstrap ─── 0.2 Root command
                     │
          ┌──────────┼──────────┐
          │          │          │
        1.1 Types  1.2 UI    1.7 Runtime
          │                     │
     ┌────┼────┐           ┌───┴───┐
     │    │    │           │       │
   1.3  1.4  1.5       1.12    1.13
 Manifest Lock Registry  test   check
     │    │    │
     └────┼────┘
          │
        1.6 Installer
          │
     ┌────┼────────┐
     │    │        │
   1.8  1.10    1.11
   new  install  run
     │
   1.9
   init
          │
        1.14 env
          │
        1.15 E2E smoke test
```

Parallel opportunities:
- 1.1 and 1.2 can run in parallel
- 1.3 and 1.4 can run in parallel (both depend on 1.1)
- 1.5 can run in parallel with 1.3/1.4
- 1.7 can start as soon as 1.1 is done
- 1.8/1.9 can run in parallel with 1.10-1.14 once underlying packages exist
