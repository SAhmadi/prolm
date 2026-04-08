# TESTING.md — prolm Testing Guide

> Full reference for running tests locally, simulating CI, and writing new tests.
> **Always run `make ci` before opening a Pull Request.**

---

## Table of Contents

1. [Run CI locally before every PR](#1-run-ci-locally-before-every-pr)
2. [Individual test commands](#2-individual-test-commands)
3. [Running the full GitHub Actions workflow locally with `act`](#3-running-the-full-github-actions-workflow-locally-with-act)
4. [Test structure](#4-test-structure)
5. [Testing strategy by layer](#5-testing-strategy-by-layer)
6. [Adversarial testing](#6-adversarial-testing)
7. [Runtime compatibility testing](#7-runtime-compatibility-testing)
8. [Writing new tests — conventions](#8-writing-new-tests--conventions)

---

## 1. Run CI locally before every PR

The GitHub CI runs `go vet`, `staticcheck`, and `go test -race -count=1`. The `make ci`
target runs the exact same steps in the exact same order, so you catch failures before
pushing.

```bash
make ci
```

This is equivalent to:

```bash
go vet ./...
staticcheck ./...
go test -race -count=1 ./...
```

**Install `staticcheck` once** (if not already installed):

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

Common things `staticcheck` catches that `go vet` does not:
- Capitalized error strings (ST1005) — e.g. `"Prolfile.lock not found"` must be `"prolfile.lock not found"`
- Unused exported symbols, deprecated API usage, redundant code

---

## 2. Individual test commands

```bash
# All unit tests (fast, no race detector)
make test
# or: go test ./...

# All tests with race detector (what CI runs)
make test-race
# or: go test -race -count=1 ./...

# Vet only
make vet

# Staticcheck (lint) only
make lint

# Single package
go test ./internal/installer/...

# Single test function
go test ./cmd/... -run TestRunCommand_MissingLockfile -v

# With verbose output
go test ./... -v

# Fuzz a target (run for at least 30s)
go test ./internal/installer -fuzz FuzzSafeExtract -fuzztime 30s
go test ./internal/resolver  -fuzz FuzzConstraint   -fuzztime 30s
```

---

## 3. Running the full GitHub Actions workflow locally with `act`

[`act`](https://github.com/nektos/act) runs your `.github/workflows/*.yml` files inside
Docker containers that mirror the GitHub Actions runner environment. This lets you catch
environment-specific failures before pushing.

### Install

```bash
# macOS
brew install act

# Linux (via release binary)
curl -s https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
```

`act` requires Docker to be running.

### Usage

```bash
# Run the full CI workflow (all jobs, all matrix OS entries)
act pull_request

# Run only the test job
act pull_request -j test

# Simulate a push to main
act push

# List all available jobs
act --list

# Dry-run (show what would run without executing)
act pull_request --dryrun
```

> **Note:** GitHub-hosted runners (`ubuntu-latest`, `macos-latest`) are mapped to Docker
> images. The default medium image (`catthehacker/ubuntu:act-latest`) covers most Go
> projects. If a step fails only inside `act` and not locally, check whether the Docker
> image has a missing system dependency.

### Recommended `.actrc` (place in repo root or `~/.actrc`)

```
-P ubuntu-latest=catthehacker/ubuntu:act-latest
--container-architecture linux/amd64
```

---

## 4. Test structure

```
prolm/
├── internal/
│   ├── manifest/        manifest_test.go    — unit: Load, Save, Validate
│   ├── lockfile/        lockfile_test.go    — unit: round-trip, atomic write, conflict markers
│   ├── resolver/        resolver_test.go    — unit: MVS, diamond dep, circular dep, yanked
│   │                    semver_test.go      — unit + fuzz: constraint parsing
│   ├── registry/        *_test.go           — unit: httptest mock server, cache hit/miss
│   ├── installer/       verify_test.go      — unit + adversarial: checksum match/mismatch
│   │                    unpack_test.go      — adversarial: path traversal, symlink escape, tar bomb
│   │                    fetch_test.go       — unit: HTTPS enforcement, timeout, 429 backoff
│   │                    store_test.go       — unit: permissions, file lock
│   │                    installer_test.go   — integration: full flow with mocked registry
│   ├── runtime/         runtime_test.go     — unit: arg assembly, version parsing, not-found error
│   ├── scaffold/        scaffold_test.go    — unit: file creation in t.TempDir(), name validation
│   ├── testrunner/      runner_test.go      — unit: discovery, PlUnit output parsing, exit codes
│   └── checker/         checker_test.go     — unit: warning detection, --strict mode
├── cmd/                 *_test.go           — integration: subprocess tests via os/exec
├── pkg/prolfile/        types_test.go       — unit: struct construction, serialisation round-trip
└── testdata/                                — fixture projects for integration tests
    ├── valid-project/
    ├── no-lockfile/
    ├── circular-deps/
    ├── yanked-version/
    └── bad-checksum/
```

---

## 5. Testing strategy by layer

| Layer              | Type                | Location                             |
|--------------------|---------------------|--------------------------------------|
| Manifest parsing   | Unit                | `internal/manifest/manifest_test.go` |
| Semver constraints | Unit + fuzz         | `internal/resolver/semver_test.go`   |
| MVS resolution     | Unit (table-driven) | `internal/resolver/resolver_test.go` |
| Checksum verify    | Unit                | `internal/installer/verify_test.go`  |
| Path traversal     | Unit (adversarial)  | `internal/installer/unpack_test.go`  |
| Runtime detection  | Unit                | `internal/runtime/runtime_test.go`   |
| Full install flow  | Integration         | `testdata/` fixtures                 |
| CLI commands       | Integration         | Subprocess tests via `os/exec`       |

**Fuzz targets (critical — run before major releases):**

```bash
go test ./internal/resolver  -fuzz FuzzConstraint        -fuzztime 60s
go test ./internal/installer -fuzz FuzzSafeExtract        -fuzztime 60s
go test ./internal/manifest  -fuzz FuzzProlfileToml       -fuzztime 60s
go test ./internal/lockfile  -fuzz FuzzLockfileContent    -fuzztime 60s
```

---

## 6. Adversarial testing

All security-critical code paths must have adversarial test cases that verify correct
rejection of hostile inputs. These are **not optional** — add them whenever you touch
`internal/installer/` or `internal/registry/`.

| Attack vector                | Test location                        | Examples                                          |
|------------------------------|--------------------------------------|---------------------------------------------------|
| Path traversal tarballs      | `internal/installer/unpack_test.go`  | `../../.bashrc`, `/etc/passwd`, `foo/../../../bar` |
| Symlink escape               | `internal/installer/unpack_test.go`  | Symlink to `/etc/shadow`, chained symlinks         |
| Oversized archives           | `internal/installer/unpack_test.go`  | 1 GB decompressed from 1 KB (tar bomb)             |
| Excessive file count         | `internal/installer/unpack_test.go`  | Tarball with 100,000 empty files                   |
| Invalid filenames            | `internal/installer/unpack_test.go`  | NULL bytes, UTF-8 edge cases, trailing dots/spaces |
| Device files / named pipes   | `internal/installer/unpack_test.go`  | Block device, char device, FIFO entries in tar     |
| Malformed TOML               | `internal/manifest/manifest_test.go` | Missing required fields, invalid types, huge values|
| Malformed registry responses | `internal/registry/*_test.go`        | Invalid JSON, missing fields, injection attempts   |
| Conflicted lockfile          | `internal/lockfile/lockfile_test.go` | Git conflict markers in TOML                       |
| Checksum mismatch            | `internal/installer/verify_test.go`  | Truncated file, bit-flipped content                |

---

## 7. Runtime compatibility testing

*(Phase 2+ — not yet active in CI)*

CI should test across multiple Prolog implementations and OS combinations to catch
runtime-specific incompatibilities early.

**Planned CI matrix:**

| Runtime       | Versions          | OS                |
|---------------|-------------------|-------------------|
| SWI-Prolog    | 9.0, 9.2, latest  | Linux, macOS      |
| GNU Prolog    | 1.5, latest       | Linux, macOS      |
| Scryer Prolog | 0.9, latest       | Linux, macOS      |
| (Windows)     | SWI only          | Windows (Phase 4) |

**What to test per runtime:**
- `prolm run` — entry point invocation, argument passing
- `prolm test` — PlUnit discovery and output parsing
- `prolm check` — static analysis warnings/errors
- `prolm repl` — interactive session startup
- Module loading — `use_module` syntax differences across runtimes
- Flag compatibility — runtime-specific flags in `[runtime.*]`

**Implementation notes:**
- Use GitHub Actions matrix strategy
- Cache Prolog runtime installs (swipl is ~200 MB compiled)
- Integration tests in `testdata/` should be runtime-portable where possible

---

## 8. Writing new tests — conventions

```go
// Always use t.TempDir() for filesystem tests — auto-cleaned on test exit
func TestInstall(t *testing.T) {
    dir := t.TempDir()
    // ...
}

// Table-driven tests for logic with many edge cases
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

// Mock the registry interface in unit tests; use testdata/ fixtures for integration tests
// Use httptest.NewServer() to mock HTTP registry responses — never hit the real network
```

**Error strings must be lowercase** (staticcheck ST1005 — this has broken CI twice):

```go
// bad — will fail staticcheck
return fmt.Errorf("Prolfile.lock not found at %s", path)

// good
return fmt.Errorf("prolfile.lock not found at %s", path)
```

**Wrap all errors** with `%w` and a context prefix (CLAUDE.md §10):

```go
// bad
return err

// good
return fmt.Errorf("loading manifest: %w", err)
```

**Test seams via structs, not package globals** (learned from QUALITY-015/016/017):

```go
// bad — mutable global, breaks parallel tests
var defaultRunner = realRunner

// good — inject via struct field, default to real impl
type myCmd struct {
    runner runnerFunc // set to mock in tests, nil means use real
}
```
