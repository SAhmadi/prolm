# Documentation Index

This index names the canonical role of each project document. Use it when a
topic has both a codebase map and a reference spec.

| Need | Canonical document |
| --- | --- |
| Agent start point and guardrails | [AGENTS.md](../AGENTS.md) |
| Current and planned CLI command catalog | [Command Central](command-central.md) |
| Local checks, CI, test ownership, and testing conventions | [TESTING.md](../TESTING.md) |
| Implementation phases and future work sequencing | [ROADMAP.md](../ROADMAP.md) |
| Known-issue ID counters and issue index | [BUGS.md](../BUGS.md) |
| `Prolfile.toml` and `Prolfile.lock` reference | [Project Files](project-files.md) |
| Package acquisition and trust rules | [Security Rules](security.md) |
| Dependency, reproducibility, and runtime design context | [Package Manager Design](package-manager-design.md) |
| Code path and workflow navigation | [Codebase Knowledge Graph](codebase-knowledge-graph/Codebase%20Knowledge%20Graph.md) |
| Flow visualization entry point | [Flow Diagrams](flow-diagrams.md) and [Command Flow Map](codebase-knowledge-graph/Command%20Flow%20Map.canvas) |

## Ownership Notes

- Keep command status and planned command surface in Command Central.
- Keep source path maps and subsystem notes in the codebase knowledge graph.
- Keep flow visuals in Canvas and searchable behavior contracts in Markdown.
- Keep security and metadata rules in their focused reference docs rather than
  expanding `AGENTS.md` back into a full spec.
