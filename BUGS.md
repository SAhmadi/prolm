# BUGS.md — prolm Known Issues & Technical Debt

> This file tracks bugs, security concerns, DRY violations, and code-quality
> issues identified during code review. Items are grouped by severity.
> Resolve before the sub-phase that first needs the fix is declared complete.

---

## Open Issues

### DRY-003 — Duplicated retry/backoff logic between registry and installer

**Severity:** Low  
**Phase:** Before 2.1  
**File:** `internal/installer/fetch.go`, `internal/registry/swi.go`

The exponential backoff + jitter retry logic for HTTP 429 responses is duplicated
between `doFetchWithRetries()` in `fetch.go` and `doRequestWithRetries()` in `swi.go`.
The installer variant is simpler (no ETag/304 caching), but the core pattern is the same.

**Fix:** Extract a shared `internal/httputil/` package with a generic
`DoWithRetries(ctx, client, req) (*http.Response, error)` function.

### DRY-004 — Duplicated `isLocalhostURL` helper

**Severity:** Trivial  
**Phase:** Before 2.1  
**File:** `internal/installer/fetch.go`, `internal/installer/installer.go`

The localhost URL check is duplicated between `fetch.go` (for HTTPS bypass in tests)
and `installer.go` (for SEC-14 URL validation). Both are trivial 3-line functions.

**Fix:** Extract to a shared helper, or export from the registry package.

### QUALITY-006 — Store file lock does not detect stale locks

**Severity:** Low  
**Phase:** Before 2.1  
**File:** `internal/installer/store.go`

The `O_CREATE|O_EXCL` file lock writes the PID to the lock file but does not check
whether the PID is still running. If prolm crashes without releasing the lock, the
user must manually delete `~/.prolm/store/.lock`. Phase 2 should add stale lock
detection (check if the PID is alive via `os.FindProcess` + signal 0).

---

## Tracking

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
| DRY-003 | Low | Open | Before 2.1 |
| DRY-004 | Trivial | Open | Before 2.1 |
| QUALITY-006 | Low | Open | Before 2.1 |
