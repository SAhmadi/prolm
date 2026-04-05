# BUGS.md — prolm Known Issues & Technical Debt

> This file tracks bugs, security concerns, DRY violations, and code-quality
> issues identified during code review. Items are grouped by severity.
> Resolve before the sub-phase that first needs the fix is declared complete.

---

## Open Issues

### QUALITY-012 — `syscall.Exec` is Unix-only in `SWIRuntime.Exec` (Low)

**File:** `internal/runtime/swi.go:9,202`
**Found:** Phase 1.7 implementation

`SWIRuntime.Exec()` uses `syscall.Exec` which is not available on Windows.
Phase 1 targets Linux and macOS only, so this is not blocking, but Phase 4
(Windows support) will need a build-tag split (`exec_unix.go` / `exec_windows.go`)
or an alternative using `os/exec.Command` + `os.Exit` on Windows.

**Fix:** When Windows support is added in Phase 4, split `Exec()` into
platform-specific files with build tags.

### QUALITY-013 — `ScanDeps` does not handle multi-line `use_module` directives (Low)

**File:** `internal/scaffold/scanner.go`
**Found:** Phase 1.9 implementation

The scanner uses a line-by-line regex to match `:- use_module(library(...)).`
directives. Prolog allows these to span multiple lines:

```prolog
:- use_module(
    library(clpfd)
).
```

This pattern will not be detected. The scanner is best-effort and a hint is
shown to the user to review the generated `[dependencies]`, so this is
acceptable for Phase 1.

**Fix:** Implement a multi-line-aware scanner that buffers directive text
across lines until the closing `).` is found. Consider in Phase 2.

---

### QUALITY-014 — `swiBuiltins` list is hardcoded and may drift from SWI-Prolog releases (Low)

**File:** `internal/scaffold/scanner.go`
**Found:** Phase 1.9 implementation

The set of SWI-Prolog built-in libraries excluded from dependency scanning
is hardcoded. As SWI-Prolog adds or removes standard libraries across
releases, this list may become stale. False positives (including a built-in
as a dep) are low-risk since `prolm install` will fail to resolve it,
prompting the user to remove it. False negatives (excluding a real external
dep) are higher-risk but unlikely since external packs rarely shadow
built-in names.

**Fix:** Consider querying `swipl` to list installed libraries at scan time
(requires runtime to be installed). Alternatively, maintain a versioned
built-in list per SWI-Prolog release. Low priority.

## Tracking (Open)

| ID | Severity | Status | Phase |
|----|----------|--------|-------|
| QUALITY-012 | Low | Open | Before 4.7 |
| QUALITY-013 | Low | Open | Phase 2+ |
| QUALITY-014 | Low | Open | Phase 2+ |
