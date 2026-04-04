# BUGS.md — prolm Known Issues & Technical Debt

> This file tracks bugs, security concerns, DRY violations, and code-quality
> issues identified during code review. Items are grouped by severity.
> Resolve before the sub-phase that first needs the fix is declared complete.

---

## Open Issues

### QUALITY-006 — Store file lock does not detect stale locks

**Severity:** Low  
**Phase:** Before 2.1  
**File:** `internal/installer/store.go`

The `O_CREATE|O_EXCL` file lock writes the PID to the lock file but does not check
whether the PID is still running. If prolm crashes without releasing the lock, the
user must manually delete `~/.prolm/store/.lock`. Phase 2 should add stale lock
detection (check if the PID is alive via `os.FindProcess` + signal 0).

### QUALITY-007 — `cachedVersions` double-writes to versionCache

**Severity:** Trivial  
**Phase:** Before 2.1  
**File:** `internal/registry/swi.go`

After the BUG-009 fix, `Versions()` now writes to `r.versionCache` before
returning. `cachedVersions()` calls `Versions()` on a cache miss and then
writes the same result to `r.versionCache` again. The second write is
redundant (idempotent) but makes the code harder to reason about.

**Fix:** Remove the redundant cache write in `cachedVersions()` — it is now
fully handled by `Versions()`.

### QUALITY-008 — `Verify()` uses `EqualFold` but regex enforces lowercase

**Severity:** Trivial  
**Phase:** Before 2.1  
**File:** `internal/installer/verify.go`

`checksumRe` enforces lowercase hex `[0-9a-f]{64}`, so an uppercase checksum
is rejected at the format validation stage before comparison. The subsequent
`strings.EqualFold` comparison at line 58 is therefore unreachable for
mixed-case inputs. The `EqualFold` is misleading — a plain `==` would be
clearer since both sides are guaranteed lowercase by this point.

**Fix:** Replace `strings.EqualFold(got, expectedChecksum)` with `got == expectedChecksum`.

### QUALITY-009 — Fast-path verifies cached tarball for already-installed packs

**Severity:** Low  
**Phase:** Before 2.1  
**File:** `internal/installer/installer.go`

`installOne()` lines 90-95: when a package is already in the lock AND already
installed in the store, the code still attempts to verify the cached tarball
checksum. This is unnecessary I/O (the store already contains the unpacked
files). Worse, if the cached tarball was deleted (cache is ephemeral), the
`os.Stat` fails and the code falls through to the "already installed" return —
but the control flow is confusing and could mask real issues.

**Fix:** On the fast path (lock entry exists + pack in store), validate the
lock URL (SEC-14) and return immediately. Skip tarball cache verification.

### QUALITY-010 — Non-deterministic dependency installation order

**Severity:** Low  
**Phase:** Before 2.1  
**File:** `internal/installer/installer.go`

`Install()` iterates over `allDeps()` which returns a `map[string]string`. Go
map iteration order is randomized, so packages are fetched/installed in a
different order on each run. While the final lockfile is sorted, the
non-deterministic install order means: (a) progress output varies between runs,
(b) if one package download fails, the set of already-installed packages is
unpredictable, making retries less efficient.

**Fix:** Sort dependency names before iterating. This aligns with CLAUDE.md
section 6 which specifies "Install order is deterministic: topological sort of
dependency graph, ties broken alphabetically by package name."

---

## Tracking (Open)

| ID | Severity | Status | Phase |
|----|----------|--------|-------|
| QUALITY-006 | Low | Open | Before 2.1 |
| QUALITY-007 | Trivial | Open | Before 2.1 |
| QUALITY-008 | Trivial | Open | Before 2.1 |
| QUALITY-009 | Low | Open | Before 2.1 |
| QUALITY-010 | Low | Open | Before 2.1 |

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
