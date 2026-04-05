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

### BUG-015 — `InitProject` interactive mode does not validate version or entry inputs (Medium)

**File:** `internal/scaffold/scaffold.go:261-263`
**Found:** PR #9 review

In interactive mode, `InitProject` validates `name` (via `manifest.ValidateName`) and
`runtime` (via `manifest.ValidateRuntime`), but accepts `version` and `entry` verbatim
from user input. Since `manifest.Save()` does not call `Validate()`, invalid values
(e.g. `"not-a-version"` for version, `"/etc/passwd"` or `"../../evil.pl"` for entry)
are written to Prolfile.toml. Subsequent `manifest.Load()` calls will fail with
confusing validation errors.

**Fix:** Either export `ValidateVersion`/`ValidateEntry` from the manifest package
(matching the pattern of `ValidateName`/`ValidateRuntime`) and call them in the
interactive prompts, or call `manifest.Validate(pf)` on the assembled ProlFile
before passing it to `manifest.Save()`.

---

### QUALITY-015 — `ScanDeps` does not check `bufio.Scanner` error after scan loop (Low)

**File:** `internal/scaffold/scanner.go:82-109`
**Found:** PR #9 review

After the `for scanner.Scan()` loop in `ScanDeps`, `scanner.Err()` is never checked.
If the file read encounters an I/O error mid-scan, partial results are silently used
without any indication of the truncated read.

**Fix:** Add `if err := scanner.Err(); err != nil { ... }` after the scan loop.
Since the scanner is best-effort, logging a warning is sufficient.

---

### QUALITY-016 — `InitProject` writes Prolfile.lock non-atomically (Low)

**File:** `internal/scaffold/scaffold.go:304`
**Found:** PR #9 review

Prolfile.toml is correctly written via `manifest.Save()` which uses `atomicfile.Write`
(SEC-8 compliant), but Prolfile.lock is written with plain `os.WriteFile`. While an
empty lock is unlikely to cause corruption, this is inconsistent with the project's
atomic write policy.

**Fix:** Use `atomicfile.Write` for Prolfile.lock as well.

---

### QUALITY-017 — `ScanDeps` walks hidden directories (`.git`, etc.) (Low)

**File:** `internal/scaffold/scanner.go:64-73`
**Found:** PR #9 review

`ScanDeps` does not skip hidden directories (`.git`, `.hg`, `node_modules`, etc.)
when walking the file tree. For large repositories, walking `.git/` is wasteful and
could produce false positives from vendored or checked-in `.pl` files.

**Fix:** Add early `return filepath.SkipDir` for directories whose name starts with `.`.

---

### QUALITY-018 — `Stdin` package-level mutable variable in prompt.go (Low)

**File:** `internal/scaffold/prompt.go:15`
**Found:** PR #9 review

The `Stdin` variable is a shared mutable package-level global used for test injection.
Tests properly save/restore it, but parallel test execution could race on it.

**Fix:** Consider passing an `io.Reader` parameter to `InitProject` (or a config struct)
instead of using a package-level variable. Low priority since current tests are not
parallel on this variable.

---

## Tracking (Open)

| ID | Severity | Status | Phase |
|----|----------|--------|-------|
| QUALITY-012 | Low | Open | Before 4.7 |
| QUALITY-013 | Low | Open | Phase 2+ |
| QUALITY-014 | Low | Open | Phase 2+ |
| BUG-015 | Medium | Open | PR #9 |
| QUALITY-015 | Low | Open | PR #9 |
| QUALITY-016 | Low | Open | PR #9 |
| QUALITY-017 | Low | Open | PR #9 |
| QUALITY-018 | Low | Open | PR #9 |
