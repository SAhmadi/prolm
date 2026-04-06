# BUGS.md — prolm Known Issues & Technical Debt

> Open issues are tracked on GitHub Issues.
> This file serves as an index and counter registry so IDs never repeat.

---

## ID Counters

Keep these counters up to date when opening new issues. Always increment —
never reuse a number, even after an issue is closed.

| Prefix    | Meaning                          | Next ID |
|-----------|----------------------------------|---------|
| BUG       | Correctness bugs                 | BUG-012 |
| SEC       | Security issues                  | SEC-017 |
| DRY       | Code duplication / DRY violation | DRY-003 |
| QUALITY   | Code quality / tech debt         | QUALITY-018 |
| ERR       | Error handling issues            | ERR-003 |

---

## Open Issues

| ID | Severity | GitHub Issue | Phase |
|----|----------|--------------|-------|
| QUALITY-012 | Low    | [QUALITY-012 — syscall.Exec is Unix-only in SWIRuntime.Exec](https://github.com/SAhmadi/prolm/issues/10) | Before 4.7 |

---

## Resolved Issues

| ID | Severity | GitHub Issue | Resolution | Commit | Date |
|----|----------|--------------|------------|--------|------|
| QUALITY-015 | Low | [#14 — install command missing --frozen / --offline / --no-verify flags](https://github.com/SAhmadi/prolm/issues/14) | Registered all three flags as Phase 2 stubs; passing them returns a clear "not yet implemented" error instead of cobra's "unknown flag" | 714fd21 | 2026-04-06 |
| QUALITY-016 | Low | [#15 — runInstall always rewrites Prolfile.lock even when content is unchanged](https://github.com/SAhmadi/prolm/issues/15) | Added `lockfile.Encode()` helper; `runInstall` now compares encoded bytes against on-disk content and skips `Save` when identical | 714fd21 | 2026-04-06 |
| QUALITY-017 | Low | [#16 — install seams are package globals without sync](https://github.com/SAhmadi/prolm/issues/16) | Replaced `newRegistry`/`installOpts` package vars with an `installRunner` struct; tests use `newTestRunner` + `withTestRunner` instead of mutating globals | 714fd21 | 2026-04-06 |
