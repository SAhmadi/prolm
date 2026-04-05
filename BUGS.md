# BUGS.md — prolm Known Issues & Technical Debt

> This file tracks bugs, security concerns, DRY violations, and code-quality
> issues identified during code review. Items are grouped by severity.
> Resolve before the sub-phase that first needs the fix is declared complete.

---

## Open Issues

### BUG-015 — `--runtime` flag not registered as local flag on `newCmd` (Medium)

**File:** `cmd/new.go:30-32`
**Found:** PR #8 review

`newCmd` reads the `runtime` flag via `cmd.Flags().GetString("runtime")` and the
`Example` text shows `--runtime scryer`, but `init()` only registers `--template`.
The flag silently falls through to the global persistent `--runtime` from `rootCmd`,
which Cobra merges into `cmd.Flags()`. This causes three problems:

1. Help output shows `--runtime` under "Global Flags" instead of local flags, confusing users.
2. The global `--runtime` has a different semantic (override runtime for execution) vs. here
   where it sets the project's default runtime in `Prolfile.toml`.
3. CLAUDE.md Section 3 specifies `--runtime` as a flag of `prolm new` specifically.

**Fix:** Add `newCmd.Flags().String("runtime", "swi", "target runtime: swi | gnu | scryer")`
in `init()` and remove the `if runtime == ""` fallback in `RunE` (the flag default handles it).

---

### TEST-001 — `TestNewProject_CleanupOnError` does not test cleanup path (Medium)

**File:** `internal/scaffold/scaffold_test.go:195-218`
**Found:** PR #8 review

The test pre-creates a `blocker` directory so `NewProject` fails with "directory already
exists" — *before* the `created = true` line is reached. The cleanup defer never fires.
The test passes but does not verify its stated purpose.

**Fix:** Inject a failure after directory creation (e.g. make `manifest.Save` fail by
making the directory read-only, or introduce a test hook) so the cleanup defer actually
executes and can be verified.

---

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

### TEST-002 — No happy-path command-layer test for `prolm new` (Low)

**File:** `cmd/new_test.go:27-42`
**Found:** PR #8 review

`TestExecute_New_InvalidName` tests error paths through the command layer, but there is
no test that invokes `prolm new <valid-name>` in a temp directory and verifies the happy
path end-to-end at the command level (as opposed to the scaffold unit tests which cover it).

**Fix:** Add a command-layer happy-path test (e.g. `prolm new valid-name` in a temp dir).

---

### ERR-001 — Validation errors in scaffold bypass `ui.Error()` convention (Low)

**File:** `internal/scaffold/scaffold.go:59,62`
**Found:** PR #8 review

Template and runtime validation errors use `fmt.Errorf` directly. Per CLAUDE.md Section 10,
user-facing errors should use `ui.Error()` + `ui.Hint()`. The scaffold package already
imports `ui` and uses it for success/hint messages (lines 131–132), but validation errors
bubble up through Cobra's default error printer instead.

**Fix:** Align validation error paths with the `ui.Error` + `ui.Hint` convention used
elsewhere in the package.

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
| BUG-015 | Medium | Open | Before merge of PR #8 |
| TEST-001 | Medium | Open | Before merge of PR #8 |
| QUALITY-015 | Low | Open | Before merge of PR #8 |
| TEST-002 | Low | Open | Before merge of PR #8 |
| ERR-001 | Low | Open | Before 2.0 |
| DRY-005 | Low | Open | Before 2.5 |
| QUALITY-012 | Low | Open | Before 4.7 |
