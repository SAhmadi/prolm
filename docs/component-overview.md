# Project Component Overview

Back to [Codebase Map](codebase-map.md).

Use this note as the first path map for the current `prolm` codebase. It is a
component-level lookup aid. Verify behavior in source and use `rg` when this
overview does not answer a symbol-level question.

Tests are listed next to the code they protect. Descriptions match the current
tree; future work belongs in [ROADMAP.md](../ROADMAP.md) and focused references
linked from [AGENTS.md](../AGENTS.md).

## Start here

| Component | First map | Code area |
| --- | --- | --- |
| CLI commands and command helpers | [CLI Surface](cli-surface.md) | [`cmd/`](../cmd) |
| Public metadata types | [Project Metadata Flow](project-metadata-flow.md) | [`pkg/prolfile/`](../pkg/prolfile) |
| Manifest and lockfile persistence | [Project Metadata Flow](project-metadata-flow.md) | [`internal/manifest/`](../internal/manifest), [`internal/lockfile/`](../internal/lockfile) |
| Registry, fetch, verification, and store | [Dependency Install Flow](dependency-install-flow.md) | [`internal/registry/`](../internal/registry), [`internal/installer/`](../internal/installer) |
| Runtime, run, test, and check | [Runtime Test And Check Flow](runtime-test-check-flow.md) | [`internal/runtime/`](../internal/runtime), [`internal/testrunner/`](../internal/testrunner), [`internal/checker/`](../internal/checker) |
| Project creation and CLI feedback | [Scaffolding And UX](scaffolding-ux.md) | [`internal/scaffold/`](../internal/scaffold), [`internal/ui/`](../internal/ui) |

## Maintenance

Keep this manual map current when source files move, components are added, or
subsystem ownership changes. After editing it, verify that Markdown file links
still resolve and that the current `cmd/`, `internal/`, `pkg/`, and `testdata/`
files remain represented below.

## CLI entrypoints and command orchestration

Related note: [CLI Surface](cli-surface.md).

| Path | Responsibility |
| --- | --- |
| [`main.go`](../main.go) | Process entry point that calls `cmd.Execute()`. |
| [`cmd/doc.go`](../cmd/doc.go) | Package overview for Cobra command definitions. |
| [`cmd/root.go`](../cmd/root.go) | Root command, global flags, execution, and usage error handling. |
| [`cmd/version.go`](../cmd/version.go) | `version` command registration and version output. |
| [`cmd/new.go`](../cmd/new.go) | `new` command wiring for scaffold creation. |
| [`cmd/init.go`](../cmd/init.go) | `init` command wiring for existing directories. |
| [`cmd/install.go`](../cmd/install.go) | `install` command orchestration around manifest, registry, installer, and lockfile. |
| [`cmd/add.go`](../cmd/add.go) | `add` input parsing, manifest mutation, sync, and saved dependency constraint. |
| [`cmd/add_github.go`](../cmd/add_github.go) | GitHub stable tag and release resolution for `add`. |
| [`cmd/remove.go`](../cmd/remove.go) | `remove` dependency mutation and sync orchestration. |
| [`cmd/deps_sync.go`](../cmd/deps_sync.go) | Shared dependency sync path and source override registry wrapper. |
| [`cmd/manifest_helpers.go`](../cmd/manifest_helpers.go) | Shared command manifest loading helper. |
| [`cmd/manifest_clone.go`](../cmd/manifest_clone.go) | Clones manifest data for rollback-safe mutation. |
| [`cmd/store_helpers.go`](../cmd/store_helpers.go) | Verifies installed dependency paths and resolves module paths in store. |
| [`cmd/run.go`](../cmd/run.go) | `run` command, script entry parsing, and runtime execution setup. |
| [`cmd/test.go`](../cmd/test.go) | `test` command, runtime dependency loading, and runner invocation. |
| [`cmd/check.go`](../cmd/check.go) | `check` command, source discovery, and checker invocation. |
| [`cmd/env.go`](../cmd/env.go) | `env` command output for runtime, store, project, and environment details. |
| [`cmd/command_contract_test.go`](../cmd/command_contract_test.go) | Command surface and help contract tests. |
| [`cmd/smoke_test.go`](../cmd/smoke_test.go) | Basic CLI smoke coverage. |
| [`cmd/version_test.go`](../cmd/version_test.go) | Version command tests. |
| [`cmd/new_test.go`](../cmd/new_test.go) | New command tests. |
| [`cmd/init_test.go`](../cmd/init_test.go) | Init command tests. |
| [`cmd/install_test.go`](../cmd/install_test.go) | Install command tests. |
| [`cmd/add_github_test.go`](../cmd/add_github_test.go) | Add and GitHub dependency input tests. |
| [`cmd/run_test.go`](../cmd/run_test.go) | Run command and script entry tests. |
| [`cmd/test_test.go`](../cmd/test_test.go) | Test command wiring tests. |
| [`cmd/check_test.go`](../cmd/check_test.go) | Check command wiring tests. |
| [`cmd/env_test.go`](../cmd/env_test.go) | Env command rendering tests. |
| [`cmd/store_helpers_test.go`](../cmd/store_helpers_test.go) | Installed module path helper tests. |

## Public Prolfile and lockfile types

Related note: [Project Metadata Flow](project-metadata-flow.md).

| Path | Responsibility |
| --- | --- |
| [`pkg/prolfile/doc.go`](../pkg/prolfile/doc.go) | Package overview for public metadata types. |
| [`pkg/prolfile/types.go`](../pkg/prolfile/types.go) | Public Prolfile, lockfile, runtime config, and dependency structs and format versions. |
| [`pkg/prolfile/types_test.go`](../pkg/prolfile/types_test.go) | Serialization and type behavior tests for public metadata. |

## Manifest and lockfile persistence

Related note: [Project Metadata Flow](project-metadata-flow.md).

| Path | Responsibility |
| --- | --- |
| [`internal/manifest/manifest.go`](../internal/manifest/manifest.go) | Manifest discovery, load, version checks, and deterministic save. |
| [`internal/manifest/validate.go`](../internal/manifest/validate.go) | Semantic validation for manifest fields and dependency sections. |
| [`internal/manifest/manifest_test.go`](../internal/manifest/manifest_test.go) | Manifest load and save tests. |
| [`internal/manifest/validate_test.go`](../internal/manifest/validate_test.go) | Manifest validation tests. |
| [`internal/lockfile/lockfile.go`](../internal/lockfile/lockfile.go) | Lockfile load, conflict detection, encode, and save logic. |
| [`internal/lockfile/hash.go`](../internal/lockfile/hash.go) | Hash input generation for manifest dependency state. |
| [`internal/lockfile/lockfile_test.go`](../internal/lockfile/lockfile_test.go) | Lockfile persistence and validation tests. |

## Registry and HTTP support

Related note: [Dependency Install Flow](dependency-install-flow.md).

| Path | Responsibility |
| --- | --- |
| [`internal/registry/doc.go`](../internal/registry/doc.go) | Package overview for registry sources. |
| [`internal/registry/registry.go`](../internal/registry/registry.go) | Registry interface, package version shape, validation, and registry errors. |
| [`internal/registry/swi.go`](../internal/registry/swi.go) | SWI pack index fetch, cache integration, HTML parsing, and archive URL normalization. |
| [`internal/registry/cache.go`](../internal/registry/cache.go) | Registry response cache keyed by request URL. |
| [`internal/registry/swi_test.go`](../internal/registry/swi_test.go) | SWI registry parsing and request tests. |
| [`internal/registry/cache_test.go`](../internal/registry/cache_test.go) | Registry cache tests. |
| [`internal/httputil/retry.go`](../internal/httputil/retry.go) | Reusable HTTP retry and Retry-After handling. |
| [`internal/httputil/localhost.go`](../internal/httputil/localhost.go) | Localhost URL detection for test and HTTP safety paths. |
| [`internal/httputil/retry_test.go`](../internal/httputil/retry_test.go) | Retry behavior tests. |
| [`internal/httputil/localhost_test.go`](../internal/httputil/localhost_test.go) | Localhost URL tests. |

## Installer and local store

Related note: [Dependency Install Flow](dependency-install-flow.md).

| Path | Responsibility |
| --- | --- |
| [`internal/installer/doc.go`](../internal/installer/doc.go) | Package overview for fetch, verification, unpack, and store work. |
| [`internal/installer/installer.go`](../internal/installer/installer.go) | Deterministic install orchestration for manifest dependencies. |
| [`internal/installer/fetch.go`](../internal/installer/fetch.go) | Archive downloading, size limits, retries, and GitHub tag archive fallback. |
| [`internal/installer/verify.go`](../internal/installer/verify.go) | SHA checksum computation and verification. |
| [`internal/installer/unpack.go`](../internal/installer/unpack.go) | Safe tar.gz and zip extraction with traversal and resource limits. |
| [`internal/installer/store.go`](../internal/installer/store.go) | Store paths, install checks, store directory setup, and write lock. |
| [`internal/installer/installer_test.go`](../internal/installer/installer_test.go) | End-to-end installer behavior tests with mocked sources. |
| [`internal/installer/fetch_test.go`](../internal/installer/fetch_test.go) | Fetch validation and retry tests. |
| [`internal/installer/verify_test.go`](../internal/installer/verify_test.go) | Checksum verification tests. |
| [`internal/installer/unpack_test.go`](../internal/installer/unpack_test.go) | Archive safety tests. |
| [`internal/installer/store_test.go`](../internal/installer/store_test.go) | Store path and lock tests. |

## Runtime execution

Related note: [Runtime Test And Check Flow](runtime-test-check-flow.md).

| Path | Responsibility |
| --- | --- |
| [`internal/runtime/doc.go`](../internal/runtime/doc.go) | Package overview for Prolog runtime support. |
| [`internal/runtime/runtime.go`](../internal/runtime/runtime.go) | Runtime interface, factory, minimum version checks, and runtime errors. |
| [`internal/runtime/swi.go`](../internal/runtime/swi.go) | SWI binary detection, argument construction, and execution. |
| [`internal/runtime/runtime_test.go`](../internal/runtime/runtime_test.go) | Runtime detection and argument tests. |

## Scaffold, init, and templates

Related note: [Scaffolding And UX](scaffolding-ux.md).

| Path | Responsibility |
| --- | --- |
| [`internal/scaffold/doc.go`](../internal/scaffold/doc.go) | Package overview for project creation. |
| [`internal/scaffold/scaffold.go`](../internal/scaffold/scaffold.go) | New project creation, init flow, template render, and git init helper. |
| [`internal/scaffold/prompt.go`](../internal/scaffold/prompt.go) | Prompt handling for interactive init defaults. |
| [`internal/scaffold/scanner.go`](../internal/scaffold/scanner.go) | Scans Prolog imports for external dependency names. |
| [`internal/scaffold/templates/app/README.md.tmpl`](../internal/scaffold/templates/app/README.md.tmpl) | App scaffold README template. |
| [`internal/scaffold/templates/app/.gitignore.tmpl`](../internal/scaffold/templates/app/.gitignore.tmpl) | App scaffold ignore-file template. |
| [`internal/scaffold/templates/app/main.pl.tmpl`](../internal/scaffold/templates/app/main.pl.tmpl) | App scaffold source entry template. |
| [`internal/scaffold/templates/app/main_test.pl.tmpl`](../internal/scaffold/templates/app/main_test.pl.tmpl) | App scaffold test template. |
| [`internal/scaffold/scaffold_test.go`](../internal/scaffold/scaffold_test.go) | New project scaffold tests. |
| [`internal/scaffold/init_test.go`](../internal/scaffold/init_test.go) | Existing directory init tests. |
| [`internal/scaffold/scanner_test.go`](../internal/scaffold/scanner_test.go) | Dependency scanner tests. |

## Test runner and checker

Related note: [Runtime Test And Check Flow](runtime-test-check-flow.md).

| Path | Responsibility |
| --- | --- |
| [`internal/testrunner/doc.go`](../internal/testrunner/doc.go) | Package overview for `prolm test`. |
| [`internal/testrunner/discover.go`](../internal/testrunner/discover.go) | Discovers Prolog test files in a project. |
| [`internal/testrunner/runner.go`](../internal/testrunner/runner.go) | Runs PlUnit through the selected runtime. |
| [`internal/testrunner/parser.go`](../internal/testrunner/parser.go) | Parses PlUnit output into structured results. |
| [`internal/testrunner/reporter.go`](../internal/testrunner/reporter.go) | Formats test output for users and JSON mode. |
| [`internal/testrunner/discover_test.go`](../internal/testrunner/discover_test.go) | Test discovery tests. |
| [`internal/testrunner/runner_test.go`](../internal/testrunner/runner_test.go) | Runner behavior tests. |
| [`internal/testrunner/parser_test.go`](../internal/testrunner/parser_test.go) | Test parser tests. |
| [`internal/testrunner/reporter_test.go`](../internal/testrunner/reporter_test.go) | Test reporting tests. |
| [`internal/checker/doc.go`](../internal/checker/doc.go) | Package overview for `prolm check`. |
| [`internal/checker/discover.go`](../internal/checker/discover.go) | Discovers source files for static checks. |
| [`internal/checker/checker.go`](../internal/checker/checker.go) | Runs static runtime checks with timeout handling. |
| [`internal/checker/parser.go`](../internal/checker/parser.go) | Parses checker diagnostics. |
| [`internal/checker/reporter.go`](../internal/checker/reporter.go) | Reports checker diagnostics and strict failure state. |
| [`internal/checker/discover_test.go`](../internal/checker/discover_test.go) | Checker discovery tests. |
| [`internal/checker/checker_test.go`](../internal/checker/checker_test.go) | Checker execution tests. |
| [`internal/checker/parser_test.go`](../internal/checker/parser_test.go) | Checker parser tests. |
| [`internal/checker/reporter_test.go`](../internal/checker/reporter_test.go) | Checker reporting tests. |

## UI and output helpers

Related notes: [CLI Surface](cli-surface.md) and [Scaffolding And UX](scaffolding-ux.md).

| Path | Responsibility |
| --- | --- |
| [`internal/ui/doc.go`](../internal/ui/doc.go) | Package overview for shared CLI output helpers. |
| [`internal/ui/printer.go`](../internal/ui/printer.go) | Color-aware and JSON-aware user messages. |
| [`internal/ui/spinner.go`](../internal/ui/spinner.go) | Spinner and download progress bar output. |
| [`internal/ui/printer_test.go`](../internal/ui/printer_test.go) | Printer behavior tests. |

## Supporting utilities, fixtures, and future placeholders

| Path | Responsibility |
| --- | --- |
| [`internal/atomicfile/atomicfile.go`](../internal/atomicfile/atomicfile.go) | Atomic file write helper used by persistence code. |
| [`internal/atomicfile/atomicfile_test.go`](../internal/atomicfile/atomicfile_test.go) | Atomic write helper tests. |
| [`internal/resolver/doc.go`](../internal/resolver/doc.go) | Future-facing resolver package marker for planned MVS work. |
| [`internal/auth/doc.go`](../internal/auth/doc.go) | Future-facing auth package marker for registry credentials work. |
| [`testdata/valid-project/Prolfile.toml`](../testdata/valid-project/Prolfile.toml) | Fixture manifest for a valid Prolog project. |
| [`testdata/valid-project/src/main.pl`](../testdata/valid-project/src/main.pl) | Fixture source entry for valid project tests. |
| [`testdata/valid-project/tests/main_test.pl`](../testdata/valid-project/tests/main_test.pl) | Fixture Prolog test for valid project tests. |
| [`AGENTS.md`](../AGENTS.md) | Short agent entry point and reference index. |
| [`ROADMAP.md`](../ROADMAP.md) | Implementation phase plan and completion tracking. |
| [`TESTING.md`](../TESTING.md) | Testing strategy reference. |
| [`docs/index.md`](index.md) | Documentation ownership and navigation index. |
| [`docs/command-central.md`](command-central.md) | Current and planned command surface reference. |
| [`docs/flow-diagrams.md`](flow-diagrams.md) | Markdown index for visual command flows. |
| [`docs/codebase-map.md`](codebase-map.md) | Codebase navigation map for subsystem docs. |
| [`docs/project-files.md`](project-files.md) | Prolfile and lockfile format reference. |
| [`docs/security.md`](security.md) | Package acquisition and trust rules. |
| [`docs/package-manager-design.md`](package-manager-design.md) | Dependency, reproducibility, and runtime design notes. |
