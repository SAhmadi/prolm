# BUGS.md — prolm Known Issues & Technical Debt

> This file tracks bugs, security concerns, DRY violations, and code-quality
> issues identified during code review. Items are grouped by severity.
> Resolve before the sub-phase that first needs the fix is declared complete.

---

## Open Issues

### BUG-010 — TOCTOU race in stale lock removal (Medium)

**File:** `internal/installer/store.go:111-112`
**Found:** PR #6 review

Between `isLockStale(lockPath)` returning true and `os.Remove(lockPath)`, another
process can also detect staleness, remove the stale lock, create its own lock via
`O_EXCL`, and then the first process removes *that* lock and creates its own.
Result: two processes both believe they hold the exclusive lock, violating SEC-7.

**Fix:** Wrap stale-check + remove in a single atomic operation, or use
`os.Rename` to atomically replace the stale lock (rename to `.lock.stale`,
then create new `.lock` with `O_EXCL`, then remove `.lock.stale`).

---

### QUALITY-011 — Hardcoded 0 in ErrRetriesExhausted error message (Trivial)

**File:** `internal/httputil/retry.go:37`
**Found:** PR #6 review

```go
return fmt.Sprintf("rate limited; all %d retry attempts exhausted; retry after %s", 0, e.RetryAfter)
```

The `%d` always prints `0` instead of the actual retry count. `ErrRetriesExhausted`
does not store `MaxRetries`.

**Fix:** Add `MaxRetries int` field to `ErrRetriesExhausted` and set it in
`DoWithRetries()` at the point where the error is constructed (line 75).

---

### SEC-016 — validateSymlink does not use filepath.EvalSymlinks (Low)

**File:** `internal/installer/unpack.go:164-180`
**Found:** PR #6 review

CLAUDE.md SEC-13 specifies: *"Use filepath.EvalSymlinks() and verify the result
has destDir as a prefix."* The current `validateSymlink()` uses string-based path
resolution (`filepath.Join` + `filepath.Clean` + prefix check) instead of
`filepath.EvalSymlinks()`. The current implementation is functionally safe during
extraction (symlinks are being created, not followed), but does not provide
defense-in-depth against on-disk symlink chains as the spec intends.

**Fix:** After creating the symlink, call `filepath.EvalSymlinks()` on the
resolved path and verify the result still has `destDir` as a prefix. Handle
dangling symlinks gracefully (string-based check is sufficient if the target
does not yet exist on disk).

---

### QUALITY-012 — syscall.Exec is Unix-only in SWIRuntime.Exec (Low)

**File:** `internal/runtime/swi.go:9,202`
**Found:** Phase 1.7 implementation

`SWIRuntime.Exec()` uses `syscall.Exec` which is not available on Windows.
Phase 1 targets Linux and macOS only, so this is not blocking, but Phase 4
(Windows support) will need a build-tag split (`exec_unix.go` / `exec_windows.go`)
or an alternative using `os/exec.Command` + `os.Exit` on Windows.

**Fix:** When Windows support is added in Phase 4, split `Exec()` into
platform-specific files with build tags.

---

### QUALITY-013 — BuildRunArgs/BuildTestArgs do not escape single quotes in paths (Low)

**File:** `internal/runtime/swi.go:137-140,164-165,185`
**Found:** Phase 1.7 implementation

The `fmt.Sprintf("use_module('%s')", dep)` pattern wraps paths in Prolog
single quotes but does not escape single quotes within the path itself.
A dependency installed at a path containing `'` (e.g. `/home/o'brien/...`)
would produce malformed Prolog atoms.

**Fix:** Escape single quotes in paths before interpolation (replace `'` with
`''` per Prolog atom quoting rules), or use absolute paths that are validated
to not contain single quotes.

---

## Tracking (Open)

| ID | Severity | Status | Phase |
|----|----------|--------|-------|
| BUG-010 | Medium | Open | Before 2.2 |
| QUALITY-011 | Trivial | Open | Before 2.2 |
| SEC-016 | Low | Open | Before 2.2 |
| QUALITY-012 | Low | Open | Before 4.7 |
| QUALITY-013 | Low | Open | Before 1.15 |

## Tracking (Fixed)

| ID | Severity | Status | Phase |
|----|----------|--------|-------|
| BUG-001 | High | Fixed | Before 1.6 |
| BUG-002 | Medium | Fixed | Before 1.6 |
| BUG-003 | Medium | Fixed | Before 1.6 |
| BUG-004 | Medium | Fixed | Before 1.6 |
| BUG-005 | Low | Fixed | Before 1.6 |
| BUG-006 | Low | Fixed | Before 1.6 |
| BUG-007 | Medium | Fixed | Before 1.6 |
| BUG-008 | Low | Fixed | Before 1.6 |
| BUG-009 | Medium | Fixed | Before merge of PR #5 |
| DRY-001 | Medium | Fixed | Before 1.6 |
| DRY-002 | Low | Fixed | Before 1.6 |
| PERF-001 | Low | Fixed | Before 1.6 |
| QUALITY-001 | Low | Fixed (go mod tidy) | Immediately |
| QUALITY-002 | Trivial | Fixed | Immediately |
| QUALITY-003 | Low | Fixed | Before 1.15 |
| QUALITY-004 | Low | Fixed | Before 1.6 |
| QUALITY-005 | Trivial | Fixed | Before merge of PR #5 |
| DOC-001 | Low | Fixed (swi.go + ROADMAP) | Immediately |
| DOC-002 | Low | Fixed (swi.go comments) | Immediately |
| SEC-015 | Medium | Fixed | Before 1.15 |
| DRY-003 | Low | Fixed (internal/httputil) | Before 2.1 |
| DRY-004 | Trivial | Fixed (internal/httputil) | Before 2.1 |
| QUALITY-006 | Low | Fixed (store.go stale PID check) | Before 2.1 |
| QUALITY-007 | Trivial | Fixed (removed redundant versionCache write) | Before 2.1 |
| QUALITY-008 | Trivial | Fixed (EqualFold → == in verify.go) | Before 2.1 |
| QUALITY-009 | Low | Fixed (simplified installOne fast path) | Before 2.1 |
| QUALITY-010 | Low | Fixed (sort deps before iteration in Install) | Before 2.1 |
