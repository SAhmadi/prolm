# CLI Surface

Back to [Codebase Map](codebase-map.md) and [Project Component Overview](component-overview.md).

The CLI layer lives in [`cmd/`](../cmd). It owns Cobra command definitions,
argument parsing, command-specific orchestration, and user-facing command
contracts. Business logic should stay in the internal packages it calls.

## Entry points

- [`main.go`](../main.go) calls `cmd.Execute()`.
- [`cmd/root.go`](../cmd/root.go) defines root behavior, global flags, and
  Cobra setup.
- [`cmd/command_contract_test.go`](../cmd/command_contract_test.go) protects
  the help and command-surface contract.
- [Command Central](command-central.md) is the current and planned command
  reference, with status on each command.

## Command clusters

- Project setup commands in [`cmd/new.go`](../cmd/new.go) and
  [`cmd/init.go`](../cmd/init.go) call [Scaffolding And UX](scaffolding-ux.md).
- Dependency commands in [`cmd/install.go`](../cmd/install.go),
  [`cmd/add.go`](../cmd/add.go), and [`cmd/remove.go`](../cmd/remove.go)
  coordinate [Dependency Install Flow](dependency-install-flow.md) and [Project Metadata Flow](project-metadata-flow.md).
- Runtime commands in [`cmd/run.go`](../cmd/run.go),
  [`cmd/test.go`](../cmd/test.go), and [`cmd/check.go`](../cmd/check.go)
  coordinate [Runtime Test And Check Flow](runtime-test-check-flow.md).
- [`cmd/env.go`](../cmd/env.go) reports detected runtime, store, project, and
  environment information.

## Command responsibility table

| Command | CLI file | Main package responsibilities |
| --- | --- | --- |
| `prolm` | [`cmd/root.go`](../cmd/root.go) | Cobra setup, global flags, Viper binding, usage/error presentation. |
| `new` | [`cmd/new.go`](../cmd/new.go) | Validate CLI input, then call scaffold project creation. |
| `init` | [`cmd/init.go`](../cmd/init.go) | Apply `--yes`/`--scan`, then call existing-directory scaffold init. |
| `install` | [`cmd/install.go`](../cmd/install.go) | Load manifest and lock, reject stub flags, coordinate registry/install/lock save. |
| `add` | [`cmd/add.go`](../cmd/add.go) | Parse package or URL input, mutate manifest intent, run dependency sync. |
| `remove` | [`cmd/remove.go`](../cmd/remove.go) | Remove manifest dependency intent, run dependency sync. |
| `run` | [`cmd/run.go`](../cmd/run.go) | Resolve entry or script, load deps, choose runtime, execute. |
| `test` | [`cmd/test.go`](../cmd/test.go) | Discover/select tests, load deps, choose runtime, report results. |
| `check` | [`cmd/check.go`](../cmd/check.go) | Discover source, load deps, choose runtime, report diagnostics. |
| `env` | [`cmd/env.go`](../cmd/env.go) | Report runtime, manifest, store, and environment state. |
| `version` | [`cmd/version.go`](../cmd/version.go) | Print binary version in text or JSON mode. |

## Shared command helpers

- [`cmd/manifest_helpers.go`](../cmd/manifest_helpers.go) centralizes command
  manifest loading.
- [`cmd/store_helpers.go`](../cmd/store_helpers.go) checks local store state
  before run, test, and check workflows use installed dependencies.
- [`cmd/deps_sync.go`](../cmd/deps_sync.go) connects dependency command
  mutations to registry and installer sync.
- [`cmd/manifest_clone.go`](../cmd/manifest_clone.go) protects manifest
  rollback paths while dependency commands mutate metadata.
- [`cmd/add_github.go`](../cmd/add_github.go) resolves stable GitHub tag and
  release inputs for `prolm add`.

## Related notes

- [Project Metadata Flow](project-metadata-flow.md)
- [Dependency Install Flow](dependency-install-flow.md)
- [Runtime Test And Check Flow](runtime-test-check-flow.md)
- [Scaffolding And UX](scaffolding-ux.md)
