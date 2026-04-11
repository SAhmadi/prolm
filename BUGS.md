# BUGS.md — prolm Known Issues & Technical Debt

> Open issues are tracked on GitHub Issues.
> This file serves as an index and counter registry so IDs never repeat.

---

## ID Counters

Keep these counters up to date when opening new issues. Always increment —
never reuse a number, even after an issue is closed.

| Prefix    | Meaning                          | Next ID |
|-----------|----------------------------------|---------|
| BUG       | Correctness bugs                 | BUG-020 |
| SEC       | Security issues                  | SEC-019 |
| DRY       | Code duplication / DRY violation | DRY-007 |
| QUALITY   | Code quality / tech debt         | QUALITY-029 |
| ERR       | Error handling issues            | ERR-005 |

---

## Open Issues

| ID | Severity | GitHub Issue | Phase |
|----|----------|--------------|-------|
| QUALITY-012 | Low    | [QUALITY-012 — syscall.Exec is Unix-only in SWIRuntime.Exec](https://github.com/SAhmadi/prolm/issues/10) | Before 4.7 |
| BUG-018 | High | [BUG-018 — `prolm add` cannot resolve known SWI packs such as `prosqlite`](https://github.com/SAhmadi/prolm/issues/51) | Phase 1.5 |
| BUG-019 | Medium | [BUG-019 — `init --scan` treats built-in `clpfd` as external dependency](https://github.com/SAhmadi/prolm/issues/53) | Phase 1 |
| QUALITY-028 | Low | [QUALITY-028 — `check --help` duplicates timeout default text](https://github.com/SAhmadi/prolm/issues/52) | Phase 1.5 |
