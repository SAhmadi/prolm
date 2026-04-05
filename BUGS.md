# BUGS.md — prolm Known Issues & Technical Debt

> This file tracks bugs, security concerns, DRY violations, and code-quality
> issues identified during code review. Items are grouped by severity.
> Resolve before the sub-phase that first needs the fix is declared complete.

---

## Open Issues

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

## Tracking (Open)

| ID | Severity | Status | Phase |
|----|----------|--------|-------|
| QUALITY-012 | Low | Open | Before 4.7 |

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
| BUG-010 | Medium | Fixed (atomic rename in store.go) | Before 2.2 |
| SEC-016 | Low | Fixed (EvalSymlinks check in unpack.go) | Before 2.2 |
| BUG-014 | Low | Fixed (runtime param in CheckMinVersion) | Before 2.5 |
| QUALITY-011 | Trivial | Fixed (MaxRetries field in ErrRetriesExhausted) | Before 2.2 |
| QUALITY-013 | Low | Fixed (escapePrologAtom helper in swi.go) | Before 1.15 |
| QUALITY-014 | Trivial | Fixed (factory renamed to NewRuntime) | Before 2.5 |
