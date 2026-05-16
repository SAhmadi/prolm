# prolm Command Central

Canonical command-surface reference for the current CLI, future website docs, and acceptance-test planning.

## Status Legend

- `Available` — implemented and part of the live CLI today
- `Phase 1.5` — planned for CLI UX and command-contract hardening
- `Phase 2` — planned for production hardening
- `Stubbed` — flag or surface is visible today but intentionally not implemented yet
- `Not implemented` — documented future command that is not yet present in the live CLI

## Root Command

### `prolm`

- Status: `Available`
- Usage: `prolm [command]`
- Global flags:
  - `--config string`
  - `--runtime string`
  - `--verbose`
  - `--no-color`
  - `--json`
  - `-h, --help`
- Current behavior:
  - With no subcommand, prints root help
  - With only global flags and no subcommand (for example `prolm --no-color`), prints a short guidance line then root help
  - Global flags can be combined with help, for example `prolm --no-color --help`

## Project Setup

### `prolm new <name>`

- Status: `Available`
- Purpose: create a new Prolog project in a new directory
- Positional args:
  - `<name>` required
- Local flags:
  - `--template string`
    - `app` — `Available`
    - `library` — `Phase 2`
    - `cli` — `Phase 2`
  - `--runtime string` — `Available`
- Expected outcome:
  - creates project directory
  - writes `Prolfile.toml`
  - writes empty `Prolfile.lock`
  - creates starter source and test files
  - initializes git if available
- Expected output shape:
  - success message plus next-step hint
  - missing name should return a clear argument error and usage text
- Examples:
  - `prolm new my-app`
  - `prolm new my-app --template app`
  - `prolm new my-app --runtime scryer`
- Flag precedence:
  - local `new --runtime` overrides root/global `--runtime` when both are provided
  - if neither is provided, `new` defaults the project runtime to `swi`
- Phase 1.5 follow-up:
  - verify shipped-binary missing-name behavior
  - improve `--template` help text

### `prolm init`

- Status: `Available`
- Purpose: initialize prolm in an existing directory
- Local flags:
  - `--yes` — `Available`
  - `--scan` — `Available`
- Expected output shape:
  - success message, optional dependency-scan summary, next-step hint
- Examples:
  - `prolm init`
  - `prolm init --yes`

## Dependency Management

### `prolm install`

- Status: `Available`
- Purpose: install dependencies declared in `Prolfile.toml`
- Current contract:
  - manifest-driven
  - does not accept package names or URLs as positional inputs
- Local flags:
  - `--frozen` — `Stubbed`
  - `--offline` — `Stubbed`
  - `--no-verify` — `Stubbed`
- Expected output shape:
  - success summary like `Installed N packages in Xs`
  - clear error for unsupported stub flags
- Examples:
  - `prolm install`
- Notes:
  - `install` remains manifest-driven for Phase 1 and Phase 1.5
  - generated lockfiles now include non-empty `[meta].prolfile_hash` values based on manifest dependencies
  - empty lockfiles keep table-array shape by omitting package entries (no `package = []`)
  - `prolm` follows a cargo-style project workflow: `add`/`remove` mutate dependencies, `install` syncs from manifest+lock
  - no separate `prolm uninstall` command for project dependencies in this phase

### `prolm add <name|url>`

- Status: `Available`
- Purpose: first pre-registry workflow for adding a dependency from a package name or remote source
- Inputs:
  - SWI package name
  - direct SWI URL
  - GitHub URL
- Current behavior:
  - update `Prolfile.toml`
  - install the dependency
  - write/update `Prolfile.lock`
  - persist dependency constraints as `^<resolved-version>` when a stable version can be resolved; otherwise keep `*` with a warning
- Examples:
  - `prolm add aop`
  - `prolm add 'https://www.swi-prolog.org/pack/list?p=aop'`
  - `prolm add https://github.com/hargettp/aop`
- Notes:
  - SWI listing URLs are normalized to package-name installs
  - GitHub repository URLs are resolved to the latest stable semver tag/release (no HEAD fallback)
  - direct archive URLs are installed via URL override
- Future flags:
  - `--dev` — `Phase 2`
  - `--exact` — `Phase 2`

### `prolm remove <pack>`

- Status: `Available`
- Purpose: remove a dependency from the project manifest and sync lock/install state
- Current behavior:
  - remove from `[dependencies]` by default
  - return a clear error/hint if the package is not declared
- Examples:
  - `prolm remove aop`
- Future behavior:
  - once `--dev` exists, support removal from `[dev-dependencies]`
- Notes:
  - `remove` is the canonical dependency-removal command
  - no `uninstall` alias in Phase 1.5

### `prolm update [pack]`

- Status: `Phase 2`
- Examples:
  - `prolm update`
  - `prolm update aop`

## Run and Test

### `prolm run [entry] [-- <prolog-args>]`

- Status: `Available`
- Local flags:
  - `--goal string` — `Available`
  - `--runtime string` — `Available`
- Expected output shape:
  - executes the configured entry point through the selected runtime

### `prolm test [file]`

- Status: `Available`
- Local flags:
  - `--filter string` — `Available`
  - `--verbose` — `Available`
  - `--timeout duration` — `Available`
  - `--watch` — `Phase 2`
  - `--coverage` — `Not implemented`
- Expected output shape:
  - formatted test results
  - non-zero exit on any failing test

### `prolm check`

- Status: `Available`
- Local flags:
  - `--strict` — `Available`
  - `--no-deps` — `Available`
  - `--timeout duration` — `Available`
- Expected output shape:
  - formatted diagnostics and summary
  - no internal authoring-tool references in help or errors

### `prolm repl`

- Status: `Phase 2`

### `prolm env`

- Status: `Available`
- Expected output shape:
  - version
  - runtime status
  - store location
  - project status
  - proxy status

## Completion

### `prolm completion [bash|zsh|fish|powershell]`

- Status: `Available`
- Purpose: generate shell completion scripts
- Examples:
  - `prolm completion zsh`
  - `prolm completion bash`
  - `prolm completion fish`
- Setup:
  - macOS (zsh):
    - `mkdir -p ~/.zsh/completions`
    - `prolm completion zsh > ~/.zsh/completions/_prolm`
    - ensure `fpath=(~/.zsh/completions $fpath)` and `autoload -Uz compinit && compinit` are in `~/.zshrc`
  - Linux (bash):
    - `mkdir -p ~/.local/share/bash-completion/completions`
    - `prolm completion bash > ~/.local/share/bash-completion/completions/prolm`
    - reload shell (or `source ~/.bashrc`)
  - Linux (zsh):
    - `mkdir -p ~/.local/share/zsh/site-functions`
    - `prolm completion zsh > ~/.local/share/zsh/site-functions/_prolm`
    - ensure that directory is in `fpath`, then run `autoload -Uz compinit && compinit`
- Validation coverage:
  - generated completion verified for macOS `zsh`, Linux `bash`, and Linux `zsh`
  - Windows completion setup is deferred to the Windows support phase

## Version and Help

### `prolm version`

- Status: `Available`

### `prolm help [command]`

- Status: `Available`
- Phase 1.5 follow-up:
  - keep help text aligned with real implementation status
  - keep global-flag behavior and help messaging consistent

## Registry-Era Commands

### `prolm search <term>`

- Status: `Phase 3`

### `prolm info <pack>[@version]`

- Status: `Phase 3`
- Examples:
  - `prolm info aop`
  - `prolm info aop@0.0.9`

### `prolm publish`

- Status: `Phase 3`

### `prolm yank <version>`

- Status: `Phase 3`

### `prolm login`

- Status: `Phase 3`

### `prolm logout`

- Status: `Phase 3`

### `prolm owner`

- Status: `Phase 3`
