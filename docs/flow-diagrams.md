# Flow Diagrams

The visual source for high-level command and resolution flows is the Obsidian
Canvas [Command Flow Map](codebase-knowledge-graph/Command%20Flow%20Map.canvas).
Open the repository root as the Obsidian vault so the canvas file and its note
links resolve alongside the codebase knowledge graph.

The Canvas currently groups:

- `prolm install` lifecycle
- `prolm run` invocation assembly
- `prolm publish` pre-flight and upload path
- MVS/version-resolution flow

Add future command and subsystem flows to that Canvas while it remains readable.
If a region becomes too dense, split the detailed flow into a dedicated Canvas
and keep this index linked to it.

Textual contracts stay in Markdown:

- [Command Central](command-central.md) defines current and planned CLI surface.
- [Project Files](project-files.md) defines manifest and lockfile formats.
- [Security Rules](security.md) defines trust and package-acquisition
  invariants.
- [Package Manager Design](package-manager-design.md) defines dependency and
  runtime design context.
