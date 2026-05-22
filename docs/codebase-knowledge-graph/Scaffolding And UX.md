# Scaffolding And UX

Back to [[Codebase Knowledge Graph]] and [[Project Component Overview]].

Scaffolding is where a user first meets the project shape. It creates project
files, optionally scans Prolog imports during init, and relies on shared CLI
output helpers for readable feedback.

## Project creation

- [`cmd/new.go`](../../cmd/new.go) defines `prolm new`.
- [`cmd/init.go`](../../cmd/init.go) defines `prolm init`.
- [`internal/scaffold/scaffold.go`](../../internal/scaffold/scaffold.go) creates
  new projects and initializes existing directories.
- [`internal/scaffold/prompt.go`](../../internal/scaffold/prompt.go) handles
  init prompts.
- [`internal/scaffold/scanner.go`](../../internal/scaffold/scanner.go) scans
  Prolog source for external dependencies.
- [`internal/scaffold/templates/app/`](../../internal/scaffold/templates/app)
  contains the embedded app template files.

## Shared user output

- [`internal/ui/printer.go`](../../internal/ui/printer.go) owns regular and JSON
  messages plus color handling.
- [`internal/ui/spinner.go`](../../internal/ui/spinner.go) owns progress and
  download status helpers.
- [[CLI Surface]] keeps command definitions thin around this logic.

## Neighboring notes

- [[Project Files And Metadata]] explains the manifest emitted by scaffolding.
- [[Dependency Install Flow]] consumes scanned or declared dependencies.
- [[Runtime Test And Check Flow]] runs the starter source and tests created by
  the app template.

