# BUGS.md — prolm Known Issues & Technical Debt

> Open issues are tracked on GitHub Issues.
> This file serves as an index and counter registry so IDs never repeat.

---

## ID Counters

Keep these counters up to date when opening new issues. Always increment —
never reuse a number, even after an issue is closed.

| Prefix    | Meaning                          | Next ID |
|-----------|----------------------------------|---------|
| BUG       | Correctness bugs                 | BUG-014 |
| SEC       | Security issues                  | SEC-017 |
| DRY       | Code duplication / DRY violation | DRY-005 |
| QUALITY   | Code quality / tech debt         | QUALITY-023 |
| ERR       | Error handling issues            | ERR-004 |

---

## Open Issues

| ID | Severity | GitHub Issue | Phase |
|----|----------|--------------|-------|
| QUALITY-012 | Low    | [QUALITY-012 — syscall.Exec is Unix-only in SWIRuntime.Exec](https://github.com/SAhmadi/prolm/issues/10) | Before 4.7 |
| BUG-012 | Medium | [BUG-012 — testrunner parser regex under-reports PlUnit failures with non-word characters](https://github.com/SAhmadi/prolm/issues/24) | 1.12 |
| BUG-013 | Medium | [BUG-013 — testrunner runner swallows exec error whenever any stderr byte is present](https://github.com/SAhmadi/prolm/issues/25) | 1.12 |
| QUALITY-020 | Low | [QUALITY-020 — verifyStoreDeps takes an unused *prolfile.ProlFile parameter](https://github.com/SAhmadi/prolm/issues/26) | 1.12 |
| QUALITY-021 | Low | [QUALITY-021 — local variable 'exec' shadows imported os/exec package in runner.go](https://github.com/SAhmadi/prolm/issues/27) | 1.12 |
| QUALITY-022 | Low | [QUALITY-022 — Dead workaround '_ = testrunner.Options{}' in cmd/test_test.go](https://github.com/SAhmadi/prolm/issues/28) | 1.12 |
| DRY-004 | Low | [DRY-004 — NO_COLOR resolution duplicated in cmd/test.go instead of going through ui](https://github.com/SAhmadi/prolm/issues/29) | 1.12 |

