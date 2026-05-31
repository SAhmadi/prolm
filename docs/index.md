# Documentation Index

This index names the canonical role of each project document. Use it when a
topic has both a codebase map and a reference spec.

| Need | Canonical document |
| --- | --- |
| Agent start point and guardrails | [AGENTS.md](../AGENTS.md) |
| Current and planned CLI command catalog | [Command Central](command-central.md) |
| Local checks, CI, test ownership, and testing conventions | [TESTING.md](../TESTING.md) |
| Release tagging, GitHub Releases, and binary verification | [Release Runbook](releases.md) |
| User-facing release history | [CHANGELOG.md](../CHANGELOG.md) |
| Implementation phases and future work sequencing | [ROADMAP.md](../ROADMAP.md) |
| `Prolfile.toml` and `Prolfile.lock` reference | [Project Files](project-files.md) |
| Package acquisition and trust rules | [Security Rules](security.md) |
| Dependency, reproducibility, and runtime design context | [Package Manager Design](package-manager-design.md) |
| Code path and workflow navigation | [Codebase Map](codebase-map.md) |
| Current code path lookup table | [Project Component Overview](component-overview.md) |
| Flow visualization entry point | [Flow Diagrams](flow-diagrams.md) |

## Ownership Notes

- Use normal Markdown links so GitHub, editors, and language models can ingest
  docs without editor-specific syntax.
- Keep command status and planned command surface in Command Central.
- Keep source path maps and subsystem notes in the codebase map.
- Keep concise ASCII flow diagrams and searchable behavior contracts in
  Markdown.
- Keep security and metadata rules in their focused reference docs rather than
  expanding `AGENTS.md` back into a full spec.
- Keep implementation sequencing and open work in [ROADMAP.md](../ROADMAP.md).
- Keep security invariants in [Security Rules](security.md).
