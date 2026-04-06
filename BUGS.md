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
| BUG-011     | Medium | [BUG-011 — Conflicted lockfile test enshrines wrong behavior; auto-heal is missing](https://github.com/SAhmadi/prolm/issues/12) | Phase 2 |
| SEC-016     | High   | [SEC-016 — Fresh install does not verify tarball against registry-supplied checksum](https://github.com/SAhmadi/prolm/issues/13) | Phase 1 |
| QUALITY-015 | Low    | [QUALITY-015 — install command missing --frozen / --offline / --no-verify flags claimed in PR description](https://github.com/SAhmadi/prolm/issues/14) | Phase 1 |
| QUALITY-016 | Low    | [QUALITY-016 — runInstall always rewrites Prolfile.lock even when content is unchanged](https://github.com/SAhmadi/prolm/issues/15) | Phase 1 |
| QUALITY-017 | Low    | [QUALITY-017 — cmd-package install seams (newRegistry, installOpts) are package globals without sync](https://github.com/SAhmadi/prolm/issues/16) | Phase 1 |
| ERR-002     | Low    | [ERR-002 — runInstall does not wrap errors from Discover, manifest.Load, lockfile.Load with context](https://github.com/SAhmadi/prolm/issues/17) | Phase 1 |
