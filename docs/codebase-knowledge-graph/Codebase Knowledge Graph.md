# Codebase Knowledge Graph

This note is the entry point for the `prolm` codebase knowledge graph. It maps
components, workflows, code paths, and project references at the level that is
useful before reading individual symbols.

The graph is a navigation layer, not a generated call graph. Start with the
component overview when locating code. Read the implementation and use source
search when behavior needs verification.

## Map

- [[Project Component Overview]] is the path lookup map for the current tree.
- [[CLI Surface]] maps Cobra commands to the subsystems they orchestrate.
- [[Dependency Install Flow]] maps dependency resolution, fetch, verification,
  local store writes, and lockfile sync.
- [[Project Files And Metadata]] maps manifest, lockfile, and public metadata
  types.
- [[Runtime Test And Check Flow]] maps SWI runtime invocation, test execution,
  and static checks.
- [[Scaffolding And UX]] maps project creation, initialization, scanner input,
  and CLI output helpers.

## Project references

- [[AGENTS]] is the short agent entry point and reference index.
- [[docs/project-files|Project Files]], [[docs/security|Security Rules]], and
  [[docs/package-manager-design|Package Manager Design]] hold focused format,
  trust, and dependency-design references.
- [[docs/index|Documentation Index]] names the canonical role of each document.
- [[ROADMAP]] tracks implementation phases and accepted work.
- [[TESTING]] defines the testing strategy.
- [[docs/command-central|command-central]] tracks the current and planned
  command surface.
- [[docs/flow-diagrams|Flow Diagrams]] indexes visual command flows.
- [[Command Flow Map]] is the Canvas view for command and version-resolution
  flows.

## Graph use

Open the repository root as the Obsidian vault so graph links can reach root
references such as [[AGENTS]], [[ROADMAP]], and [[TESTING]] as well as notes
under `docs/`. Use the local graph from this note to walk nearby components.
Follow workflow notes when a task crosses packages. Follow the component
overview when the task starts with "where does this live?"
