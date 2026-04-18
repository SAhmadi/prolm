# BUGS.md — prolm Known Issues & Technical Debt

> Open issues are tracked on GitHub Issues.
> This file serves as an index and counter registry so IDs never repeat.

---

## ID Counters

Keep these counters up to date when opening new issues. Always increment —
never reuse a number, even after an issue is closed.

| Prefix    | Meaning                          | Next ID |
|-----------|----------------------------------|---------|
| BUG       | Correctness bugs                 | BUG-023 |
| SEC       | Security issues                  | SEC-019 |
| DRY       | Code duplication / DRY violation | DRY-007 |
| QUALITY   | Code quality / tech debt         | QUALITY-032 |
| ERR       | Error handling issues            | ERR-005 |

---

## Open Issues

| ID | Severity | GitHub Issue | Phase |
|----|----------|--------------|-------|
| BUG-022 | Medium | [BUG-022 — docs and contract tests still use built-in clpfd as add/remove example](https://github.com/SAhmadi/prolm/issues/58) | Before 2.3 |
| QUALITY-012 | Low    | [QUALITY-012 — syscall.Exec is Unix-only in SWIRuntime.Exec](https://github.com/SAhmadi/prolm/issues/10) | Before 4.7 |
| QUALITY-030 | Low | [QUALITY-030 — `prolm new --help` still presents `--runtime` only as a global flag](https://github.com/SAhmadi/prolm/issues/59) | Before 1.5.5 |
| QUALITY-031 | Medium | [QUALITY-031 — generated lockfiles leave `prolfile_hash` empty and encode zero packages as `package = []`](https://github.com/SAhmadi/prolm/issues/60) | Before 2.1 |
