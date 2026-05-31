# Runtime Test And Check Flow

Back to [Codebase Map](codebase-map.md) and [Project Component Overview](component-overview.md).

Run, test, and check commands share runtime detection and installed dependency
paths. Test and check work then split into separate discovery, parsing, and
reporting packages.

## Runtime center

- [`internal/runtime/runtime.go`](../internal/runtime/runtime.go) defines the
  runtime abstraction and runtime errors.
- [`internal/runtime/swi.go`](../internal/runtime/swi.go) detects SWI-Prolog
  and builds run, test, and check invocation arguments.
- [`cmd/run.go`](../cmd/run.go) resolves project entry or script input and
  executes runtime invocation.

## Test path

- [`cmd/test.go`](../cmd/test.go) loads runtime, dependencies, and test flags.
- [`internal/testrunner/discover.go`](../internal/testrunner/discover.go)
  finds Prolog test files.
- [`internal/testrunner/runner.go`](../internal/testrunner/runner.go) runs
  tests through the runtime with timeout and filter handling.
- [`internal/testrunner/parser.go`](../internal/testrunner/parser.go) parses
  test output.
- [`internal/testrunner/reporter.go`](../internal/testrunner/reporter.go)
  renders test results.

## Check path

- [`cmd/check.go`](../cmd/check.go) loads runtime, dependencies, and static
  check flags.
- [`internal/checker/discover.go`](../internal/checker/discover.go) finds
  source files for checking.
- [`internal/checker/checker.go`](../internal/checker/checker.go) runs the
  static check invocation.
- [`internal/checker/parser.go`](../internal/checker/parser.go) parses
  diagnostics.
- [`internal/checker/reporter.go`](../internal/checker/reporter.go) reports
  warnings and errors.

## Neighboring notes

- [Project Metadata Flow](project-metadata-flow.md) supplies runtime config and entry metadata.
- [Dependency Install Flow](dependency-install-flow.md) populates installed dependency paths before these
  commands need them.
- [CLI Surface](cli-surface.md) owns command wiring and output contracts.

