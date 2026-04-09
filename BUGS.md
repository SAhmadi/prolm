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
| SEC-017     | Medium | [SEC-017 — prolm check executes dependency code without sandboxing or user warning](https://github.com/SAhmadi/prolm/issues/35) | Phase 1 |
| DRY-005     | Medium | [DRY-005 — cmd/check.go reimplements internal/checker reporter (entire reporter package is dead code)](https://github.com/SAhmadi/prolm/issues/36) | Phase 1 |
| QUALITY-023 | Low-Med | [QUALITY-023 — cmd/check.go: hand-rolled JSON uses fmt %q which is not JSON-safe](https://github.com/SAhmadi/prolm/issues/37) | Phase 1 |
| QUALITY-024 | Low    | [QUALITY-024 — cmd/check.go: strict-mode exit decision duplicated with checker.failing](https://github.com/SAhmadi/prolm/issues/38) | Phase 1 |
| QUALITY-025 | Low    | [QUALITY-025 — prolm check missing --timeout flag (inconsistent with prolm test)](https://github.com/SAhmadi/prolm/issues/39) | Phase 1 |
| ERR-004     | Medium | [ERR-004 — checker.Check silently swallows exec errors (binary missing, ctx deadline)](https://github.com/SAhmadi/prolm/issues/40) | Phase 1 |
| QUALITY-026 | Cosmetic | [QUALITY-026 — PR #31 description claims it adds prolm test command (false)](https://github.com/SAhmadi/prolm/issues/41) | Phase 1 |

