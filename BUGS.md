# BUGS.md — prolm Known Issues & Technical Debt

> Open issues are tracked on GitHub Issues.
> This file serves as an index and counter registry so IDs never repeat.

---

## ID Counters

Keep these counters up to date when opening new issues. Always increment —
never reuse a number, even after an issue is closed.

| Prefix    | Meaning                          | Next ID |
|-----------|----------------------------------|---------|
| BUG       | Correctness bugs                 | BUG-017 |
| SEC       | Security issues                  | SEC-018 |
| DRY       | Code duplication / DRY violation | DRY-006 |
| QUALITY   | Code quality / tech debt         | QUALITY-027 |
| ERR       | Error handling issues            | ERR-005 |

---

## Open Issues

| ID | Severity | GitHub Issue | Phase |
|----|----------|--------------|-------|
| QUALITY-012 | Low    | [QUALITY-012 — syscall.Exec is Unix-only in SWIRuntime.Exec](https://github.com/SAhmadi/prolm/issues/10) | Before 4.7 |
| BUG-014     | Medium | [BUG-014 — cmd/check.go: fragile manifestPath string slicing breaks --config](https://github.com/SAhmadi/prolm/issues/32) | Phase 1 |
| BUG-015     | Medium | [BUG-015 — cmd/check.go: ignores [runtime.swi].flags from manifest](https://github.com/SAhmadi/prolm/issues/33) | Phase 1 |
| BUG-016     | Medium-High | [BUG-016 — checker: load_files/1 executes initialization directives during prolm check](https://github.com/SAhmadi/prolm/issues/34) | Phase 1 |
| QUALITY-025 | Low    | [QUALITY-025 — prolm check missing --timeout flag (inconsistent with prolm test)](https://github.com/SAhmadi/prolm/issues/39) | Phase 1 |
| QUALITY-026 | Cosmetic | [QUALITY-026 — PR #31 description claims it adds prolm test command (false)](https://github.com/SAhmadi/prolm/issues/41) | Phase 1 |

