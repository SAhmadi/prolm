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
| DRY       | Code duplication / DRY violation | DRY-004 |
| QUALITY   | Code quality / tech debt         | QUALITY-019 |
| ERR       | Error handling issues            | ERR-003 |

---

## Open Issues

| ID | Severity | GitHub Issue | Phase |
|----|----------|--------------|-------|
| QUALITY-012 | Low    | [QUALITY-012 — syscall.Exec is Unix-only in SWIRuntime.Exec](https://github.com/SAhmadi/prolm/issues/10) | Before 4.7 |
| QUALITY-018 | Medium | [QUALITY-018 — prolm run does not yet execute named [scripts] entries](https://github.com/SAhmadi/prolm/issues/18) | Phase 2 |
| DRY-003     | Low    | [DRY-003 — manifest discover+load boilerplate duplicated across cmd/](https://github.com/SAhmadi/prolm/issues/19) | Phase 1 |

