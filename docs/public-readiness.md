# Public Repository Readiness

Use this checklist before making the repository public or merging public-facing
readiness work into `main`.

## Required Before Public Visibility

- [ ] README explains project status, install path, quick start, security
      model, and contribution expectations.
- [ ] LICENSE is present and matches the intended project license.
- [ ] SECURITY.md explains how to report vulnerabilities privately.
- [ ] CI passes on the branch that will become public.
- [ ] Tracked files are free of secrets, local credentials, private notes, and
      vendor-specific agent memory.
- [ ] Public docs use neutral "agent" language for coding assistants and do
      not require a specific LLM vendor.
- [ ] Release workflow avoids publishing from private-only assumptions.
- [ ] Dependabot or equivalent dependency update automation is enabled for Go
      modules and GitHub Actions.
- [ ] Branch protection or a repository ruleset protects `main`.

## Suggested `main` Ruleset

For a small early-stage project, protect `main` with a GitHub repository
ruleset named `protect-main`:

- Target branch: `main`
- Restrict deletions: enabled
- Require a pull request before merging: enabled
- Required approvals: 1
- Dismiss stale pull request approvals when new commits are pushed: enabled
- Require status checks to pass: enabled
- Required checks: CI jobs from `.github/workflows/ci.yml`
- Require branches to be up to date before merging: enabled when checks are
  stable enough for the extra friction
- Block force pushes: enabled
- Bypass list: repository administrators only, and only for emergencies

This is worth doing before the repository is public. It prevents accidental
direct pushes, keeps CI as the merge gate, and makes the project safer for
outside contributions.

## `gh` CLI Repair

If the GitHub CLI reports an invalid token, repair local auth before using it
for issue, pull request, or ruleset work:

```bash
gh auth logout -h github.com -u SAhmadi
gh auth login -h github.com -p https -w
gh auth status
```

The login flow needs an interactive browser or device-code confirmation. Agent
sessions should not ask users to paste tokens into chat, logs, or tracked files.
