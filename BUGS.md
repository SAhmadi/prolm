# BUGS.md — prolm Known Issues & Technical Debt

> This file tracks bugs, security concerns, DRY violations, and code-quality
> issues identified during code review. Items are grouped by severity.
> Resolve before the sub-phase that first needs the fix is declared complete.

---

## Open Issues

### QUALITY-015 — `Prolfile.toml.tmpl` is dead template code (Low)

**File:** `internal/scaffold/templates/app/Prolfile.toml.tmpl`
**Found:** PR #8 review

This file is embedded via `go:embed all:templates/app` but is never referenced in
`templateMapping` and never rendered. It ships as dead code in the binary. `Prolfile.toml`
is correctly generated via `manifest.Save()` for deterministic serialization (Section 8.28).

**Fix:** Either remove the file and add a comment in `scaffold.go` explaining that
`Prolfile.toml` is generated via `manifest.Save()`, or rename it to
`Prolfile.toml.example` to make its documentation-only purpose explicit.

---

### DRY-005 — Duplicate `validRuntimes` map in scaffold and manifest (Low)

**File:** `internal/scaffold/scaffold.go:28`, `internal/manifest/validate.go:24`
**Found:** Phase 1.8 implementation

`validRuntimes` is defined as an unexported `map[string]bool` in both
`internal/manifest/validate.go` and `internal/scaffold/scaffold.go`.
If a new runtime is added, both maps must be updated in lockstep.

**Fix:** Export the canonical map from `manifest` (or add a
`ValidateRuntime(name string) error` helper like `ValidateName`), then
use it in the scaffold package. Defer to Phase 2.5 when GNU/Scryer
runtimes are added.

---

### QUALITY-012 — `syscall.Exec` is Unix-only in `SWIRuntime.Exec` (Low)

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
| QUALITY-015 | Low | Open | Before merge of PR #8 |
| DRY-005 | Low | Open | Before 2.5 |
| QUALITY-012 | Low | Open | Before 4.7 |
