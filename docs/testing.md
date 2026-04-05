# Testing Guide

This document describes the testing strategy, how to run tests locally, and how CI/CD works for prolm.

---

## Running Tests Locally

```bash
# Run all tests (no race detector)
make test

# Run all tests with race detector — required before opening a PR
go test -race -count=1 ./...

# Run static analysis
go vet ./...
make lint        # runs staticcheck (must be installed: go install honnef.co/go/tools/cmd/staticcheck@latest)

# Run tests for a single package
go test -race -count=1 ./internal/installer/...

# Run a specific test
go test -race -run TestSafeExtract ./internal/installer/...
```

`-count=1` disables Go's test result cache so tests always execute fresh.

---

## CI/CD Pipeline

The workflow is defined in [.github/workflows/ci.yml](../.github/workflows/ci.yml) and runs on:

- Every pull request targeting `main`
- Every push to `main`

### Jobs

| Job | Runs on | What it does |
|-----|---------|-------------|
| `test (ubuntu-latest)` | Ubuntu | `go vet`, `staticcheck`, `go test -race` |
| `test (macos-latest)` | macOS | Same — catches OS-specific filesystem behaviour |
| `build` | Ubuntu | `go build`, uploads binary artifact (runs only if both test jobs pass) |

The `build` job depends on `test`. A PR cannot be merged if any test job fails.

---

## Test Inventory

### `cmd/`

| File | What it covers |
|------|---------------|
| `version_test.go` | `version` and `help` command output via subprocess |

### `internal/atomicfile/`

| File | What it covers |
|------|---------------|
| `atomicfile_test.go` | Atomic write-and-rename, permissions, concurrent safety |

### `internal/httputil/`

| File | What it covers |
|------|---------------|
| `retry_test.go` | Exponential backoff, `Retry-After` header, context cancellation, exhaustion |
| `localhost_test.go` | Rejection of non-localhost URLs in dev helpers |

### `internal/lockfile/`

| File | What it covers |
|------|---------------|
| `lockfile_test.go` | Load/save round-trip, atomic writes, Git conflict marker detection, signature fields |

### `internal/manifest/`

| File | What it covers |
|------|---------------|
| `manifest_test.go` | Discover, load, save, TOML round-trip, section ordering |
| `validate_test.go` | Required fields, version constraint syntax, runtime config, dependency names |

### `internal/registry/`

| File | What it covers |
|------|---------------|
| `cache_test.go` | ETag flow, 304 handling, atomic cache writes |
| `swi_test.go` | HTML parsing, package search, version lists, rate-limit (429), HTTPS enforcement |

### `internal/installer/`

| File | What it covers |
|------|---------------|
| `store_test.go` | Directory layout, installation tracking, file-lock, stale lock detection |
| `unpack_test.go` | Tarball extraction, path traversal, symlink escape, device files, size/count limits |
| `verify_test.go` | SHA-256 checksum verification, mismatch rejection, truncated files |
| `fetch_test.go` | HTTP download, size limit enforcement, atomic writes, timeout, rate limiting |
| `installer_test.go` | Full install workflow, yanked version handling, deterministic install order |

### `internal/ui/`

| File | What it covers |
|------|---------------|
| `printer_test.go` | Info/Warn/Error/Success output, JSON mode, colour suppression, spinners |

### `pkg/prolfile/`

| File | What it covers |
|------|---------------|
| `types_test.go` | `ProlFile` / `LockEntry` TOML round-trip, namespace support |

---

## Adversarial & Security Tests

The following test cases exist specifically to verify that hostile inputs are rejected.
They are in `internal/installer/unpack_test.go` unless noted.

| Attack | Test example |
|--------|-------------|
| Path traversal via `..` | Entry `../../.bashrc` |
| Absolute path in tarball | Entry `/etc/passwd` |
| Symlink escaping destDir | Symlink whose target resolves outside the store directory |
| Chained symlinks | A → B → C where C exits destDir |
| Tar bomb (decompression bomb) | Large decompressed content from a small archive |
| Excessive file count | Tarball containing 100 000 empty files |
| Device / special files | Block device, char device, FIFO entries |
| Null bytes in filenames | Paths containing `\x00` |
| Checksum mismatch | Bit-flipped content, truncated download (`verify_test.go`) |
| Git conflict markers in lockfile | `<<<<<<<` / `=======` / `>>>>>>>` in TOML (`lockfile_test.go`) |

---

## Conventions

- Use `t.TempDir()` for all filesystem tests — cleaned up automatically.
- Use table-driven tests (`[]struct{ name, ... }`) for logic with many input variants.
- Unit tests mock the `Registry` interface; integration tests use `testdata/` fixtures.
- Internal packages return `error`; only the `cmd/` layer calls `os.Exit`.
- See [CLAUDE.md](../CLAUDE.md) §10 for the full Go + Cobra + Viper conventions.

---

## Fuzz Targets (Future)

The following fuzz targets are planned (see CLAUDE.md §11):

- `internal/resolver` — random version constraint strings
- `internal/installer/unpack.go` — crafted tarball entry paths
- `internal/manifest` — malformed TOML input
- `internal/lockfile` — corrupted lockfile content
