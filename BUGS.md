# BUGS.md — prolm Known Issues & Technical Debt

> This file tracks bugs, security concerns, DRY violations, and code-quality
> issues identified during code review. Items are grouped by severity.
> Resolve before the sub-phase that first needs the fix is declared complete.

---

## Critical / Security

### BUG-005 — `Retry-After` header parsed as integer seconds only

**Severity:** Low — incomplete RFC compliance  
**Affects:** `internal/registry/swi.go`  
**Found in phase:** 1.5 review

RFC 9110 §10.2.4 allows `Retry-After` to be either an integer (delay seconds)
**or** an HTTP-date (e.g. `Retry-After: Wed, 21 Oct 2025 07:28:00 GMT`).
The current implementation silently ignores HTTP-date values, falling back to
the exponential backoff default, which may be much shorter than the server asks for.

**Fix:** After `strconv.Atoi` fails, attempt to parse the value with
`http.ParseTime` and derive the delay from `time.Until(parsed)`. Return 0 if both
parsing methods fail.

---

## DRY Violations

### DRY-001 — Atomic write pattern duplicated across three packages

**Severity:** Medium — maintenance risk  
**Affects:**
- `internal/manifest/manifest.go:104-111`
- `internal/lockfile/lockfile.go:121-129`
- `internal/registry/cache.go:102-111` (already has an `atomicWrite` helper, but it is package-private)

The write-to-tmp-then-rename pattern (SEC-8) is implemented three times. If a
bug is found in one (e.g. missing cleanup of the `.tmp` on rename failure),
it must be fixed in all three independently.

**Fix:** Extract to a shared `internal/atomicfile` package:
```go
// Package atomicfile provides atomic file write (SEC-8).
package atomicfile

import "os"

// Write atomically writes data to path by writing to a .tmp file first,
// then renaming into place. A crash mid-write never leaves a partial file.
func Write(path string, data []byte, perm os.FileMode) error {
    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, perm); err != nil {
        return err
    }
    if err := os.Rename(tmp, path); err != nil {
        os.Remove(tmp) // best-effort
        return err
    }
    return nil
}
```

Then replace the inline patterns in `manifest.go`, `lockfile.go`, and `cache.go`.

---

### DRY-002 — Recursive DOM walking duplicated in `swi.go`

**Severity:** Low — readability / maintenance  
**Affects:** `internal/registry/swi.go`

The same recursive `walk` closure pattern appears independently in:
- `parsePackList`
- `findHrefs`
- `findElements`

Each closure captures different state but the traversal skeleton is identical.

**Fix:** Extract a generic DOM walker:
```go
// walkDOM calls fn on every node in the subtree rooted at n.
// If fn returns false, traversal stops.
func walkDOM(n *html.Node, fn func(*html.Node) bool) {
    if !fn(n) {
        return
    }
    for c := n.FirstChild; c != nil; c = c.NextSibling {
        walkDOM(c, fn)
    }
}
```

Then rewrite `findHrefs` and `findElements` in terms of `walkDOM`.

---

## Code Quality

### QUALITY-004 — Null byte checks missing on non-critical string fields

**Severity:** Low — defence-in-depth gap  
**Affects:** `internal/manifest/validate.go`

`containsNullByte` is applied to `name`, `entry`, and dependency keys, but not
to free-text fields: `Authors`, `Description`, `License`, `Homepage`, `Scripts`
values. A null byte in a TOML string value is technically invalid per the TOML
spec, and BurntSushi/toml will likely reject it at parse time, but explicit
rejection with a clear error message would be more user-friendly and resilient
against future parser changes.

**Fix:** Add null byte validation for all string fields in `Validate()`, or add
a general-purpose `validateStringField` helper that checks for null bytes and
applies a max length. Apply it to all `Package` string fields.

---

## Tracking

| ID | Severity | Status | Phase |
|----|----------|--------|-------|
| BUG-001 | High | Fixed | Before 1.6 |
| BUG-002 | Medium | Fixed | Before 1.6 |
| BUG-003 | Medium | Fixed | Before 1.6 |
| BUG-004 | Medium | Fixed | Before 1.6 |
| BUG-005 | Low | Open | Phase 2 |
| BUG-006 | Low | Fixed | Before 1.6 |
| DRY-001 | Medium | Open | Phase 2 |
| DRY-002 | Low | Open | Phase 2 |
| QUALITY-001 | Low | Fixed (go mod tidy) | Immediately |
| QUALITY-002 | Trivial | Fixed | Immediately |
| QUALITY-003 | Low | Fixed | Before 1.15 |
| QUALITY-004 | Low | Open | Phase 2 |
| DOC-001 | Low | Fixed (swi.go + ROADMAP) | Immediately |
| DOC-002 | Low | Fixed (swi.go comments) | Immediately |
