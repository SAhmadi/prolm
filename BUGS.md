# BUGS.md — prolm Known Issues & Technical Debt

> Open issues are tracked on GitHub Issues.
> This file serves as an index and counter registry so IDs never repeat.

---

## ID Counters

Keep these counters up to date when opening new issues. Always increment —
never reuse a number, even after an issue is closed.

| Prefix    | Meaning                          | Next ID |
|-----------|----------------------------------|---------|
| BUG       | Correctness bugs                 | BUG-026 |
| SEC       | Security issues                  | SEC-020 |
| DRY       | Code duplication / DRY violation | DRY-007 |
| QUALITY   | Code quality / tech debt         | QUALITY-033 |
| ERR       | Error handling issues            | ERR-005 |

---

## Open Issues

| ID | Severity | GitHub Issue | Phase |
|----|----------|--------------|-------|
| QUALITY-012 | Low    | [QUALITY-012 — syscall.Exec is Unix-only in SWIRuntime.Exec](https://github.com/SAhmadi/prolm/issues/10) | Before 4.7 |
