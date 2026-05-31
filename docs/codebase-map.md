# Codebase Map

This file is the entry point for the `prolm` codebase map. It maps components,
workflows, code paths, and project references at the level that is useful
before reading individual symbols.

The graph is a navigation layer, not a generated call graph. Start with the
component overview when locating code. Read the implementation and use source
search when behavior needs verification.

## Map

- [Project Component Overview](component-overview.md) is the path lookup map for the current tree.
- [CLI Surface](cli-surface.md) maps Cobra commands to the subsystems they orchestrate.
- [Dependency Install Flow](dependency-install-flow.md) maps dependency resolution, fetch, verification,
  local store writes, and lockfile sync.
- [Project Metadata Flow](project-metadata-flow.md) maps manifest, lockfile, and public metadata
  types.
- [Runtime Test And Check Flow](runtime-test-check-flow.md) maps SWI runtime invocation, test execution,
  and static checks.
- [Scaffolding And UX](scaffolding-ux.md) maps project creation, initialization, scanner input,
  and CLI output helpers.

## Project references

- [AGENTS.md](../AGENTS.md) is the short agent entry point and reference index.
- [Project Files](project-files.md), [Security Rules](security.md), and
  [Package Manager Design](package-manager-design.md) hold focused format,
  trust, and dependency-design references.
- [Documentation Index](index.md) names the canonical role of each document.
- [ROADMAP.md](../ROADMAP.md) tracks implementation phases and accepted work.
- [TESTING.md](../TESTING.md) defines the testing strategy.
- [Command Central](command-central.md) tracks the current and planned
  command surface.
- [Flow Diagrams](flow-diagrams.md) contains concise ASCII command and
  version-resolution flows.

## Map Use

Use this map to walk nearby components before reading individual symbols. Follow
workflow notes when a task crosses packages. Follow the component overview when
the task starts with "where does this live?"
