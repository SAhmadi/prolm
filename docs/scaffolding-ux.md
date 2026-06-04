# Scaffolding And UX

Back to [Codebase Map](codebase-map.md) and [Project Component Overview](component-overview.md).

Scaffolding is where a user first meets the project shape. It creates project
files, optionally scans Prolog imports during init, and relies on shared CLI
output helpers for readable feedback.

## `new` vs `init`

`prolm new <name>` creates a new directory. It validates the project name,
chooses the template and runtime, writes `Prolfile.toml`, creates an empty
`Prolfile.lock`, adds `.gitattributes` for generated lockfile merge handling,
and renders starter source, tests, README, and `.gitignore`. The `app`
template is live today; `library` and `cli` templates are planned.

`prolm init` works in an existing directory. It refuses to overwrite an
existing `Prolfile.toml`, then either prompts for metadata or uses defaults
when `--yes` is set. Defaults are derived from the current directory where
possible, with SWI as the default runtime.

Both commands are metadata/scaffold workflows. They should not require a
Prolog runtime to be installed.

## Dependency scan

`prolm init --scan` scans `.pl` files for `use_module(library(...))` imports
and pre-populates dependencies for names that look external. It skips common
built-in SWI libraries such as `clpfd` and `lists` so generated manifests do
not ask users to install packages they already get from the runtime.

The scan is intentionally limited. It does not fully evaluate Prolog terms,
conditional loading, custom library paths, or runtime-specific module systems.
Output should remind users to review `[dependencies]`.

## Project creation

- [`cmd/new.go`](../cmd/new.go) defines `prolm new`.
- [`cmd/init.go`](../cmd/init.go) defines `prolm init`.
- [`internal/scaffold/scaffold.go`](../internal/scaffold/scaffold.go) creates
  new projects and initializes existing directories.
- [`internal/scaffold/prompt.go`](../internal/scaffold/prompt.go) handles
  init prompts.
- [`internal/scaffold/scanner.go`](../internal/scaffold/scanner.go) scans
  Prolog source for external dependencies.
- [`internal/scaffold/templates/app/`](../internal/scaffold/templates/app)
  contains the embedded app template files.

## Shared user output

- [`internal/ui/printer.go`](../internal/ui/printer.go) owns regular and JSON
  messages plus color handling.
- [`internal/ui/spinner.go`](../internal/ui/spinner.go) owns progress and
  download status helpers.
- [CLI Surface](cli-surface.md) keeps command definitions thin around this logic.

## Neighboring notes

- [Project Metadata Flow](project-metadata-flow.md) explains the manifest emitted by scaffolding.
- [Dependency Install Flow](dependency-install-flow.md) consumes scanned or declared dependencies.
- [Runtime Test And Check Flow](runtime-test-check-flow.md) runs the starter source and tests created by
  the app template.
