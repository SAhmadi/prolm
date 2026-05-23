# prolm Testing Guide

This is the canonical testing reference for `prolm`: current local commands,
CI behavior, test ownership, security coverage, and future compatibility work.

Always run `make ci` before opening a pull request.

## Local Checks

`make ci` matches the CI verification path:

```bash
make ci
```

That target runs:

```bash
go vet ./...
staticcheck ./...
go test -race -count=1 ./...
```

Install `staticcheck` once if it is not already available:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
```

Useful narrower commands:

```bash
# Fast package test pass
make test

# Race-enabled test pass
make test-race

# Static checks only
make vet
make lint

# One package
go test ./internal/installer/...

# One command test
go test ./cmd/... -run TestRunCommand_MissingLockfile -v

# Phase 1 end-to-end acceptance smoke test
go test ./cmd -run TestSmoke_EndToEndPhase1Acceptance -count=1
```

Use `-count=1` when a result must bypass Go's test cache.

## CI And Releases

The workflow lives in [`.github/workflows/ci.yml`](.github/workflows/ci.yml)
and runs for pull requests and pushes targeting `main`. It is the non-release
quality gate: every run installs SWI-Prolog, installs `staticcheck`, and runs
`make ci` on Linux and macOS.

| Job | Runner | Current behavior |
| --- | --- | --- |
| `test` | Ubuntu and macOS matrix | Installs SWI-Prolog and `staticcheck`, then runs `make ci`. |
| `snapshot` | Ubuntu | Runs a GoReleaser snapshot build after tests pass. It validates release packaging but does not publish artifacts. |

The CI test job currently covers SWI-Prolog on Linux and macOS. GNU Prolog,
Scryer Prolog, and Windows compatibility remain future work.

Release publishing lives in
[`.github/workflows/release.yml`](.github/workflows/release.yml). It runs only
for pushed tags matching `v*`, repeats the full Linux/macOS CI matrix, then uses
GoReleaser to publish Linux and macOS archives, `checksums.txt`, release notes
from `CHANGELOG.md`, and GitHub artifact attestations to the repository's
Releases page. Artifact attestations are skipped while the repository is a
user-owned private repository because GitHub does not support that feature for
this repository type.

See [Release Runbook](docs/releases.md) before tagging a public release.

The full GitHub Actions workflow can also be exercised locally with `act` when
Docker is available:

```bash
# macOS
brew install act

# Run workflow variants
act pull_request
act pull_request -j test
act pull_request --dryrun
```

For Linux installation and runner-image tuning, follow the `act` project
documentation rather than copying CI configuration into this repo.

## Current Test Layout

The current test tree is organized around package ownership:

| Area | Tests | Coverage focus |
| --- | --- | --- |
| `cmd/` | command, smoke, help-contract, dependency, runtime, env tests | CLI wiring, output contracts, subprocess smoke behavior |
| `internal/manifest/` | manifest and validation tests | discovery, parsing, deterministic save, semantic validation |
| `internal/lockfile/` | lockfile tests | round trips, atomic writes, conflict markers, metadata shape |
| `internal/registry/` and `internal/httputil/` | registry, cache, retry, localhost tests | SWI HTML parsing, caching, bounded HTTP behavior |
| `internal/installer/` | fetch, verify, unpack, store, install tests | download limits, checksum integrity, archive safety, store sync |
| `internal/runtime/` | runtime tests | SWI detection and invocation arguments |
| `internal/scaffold/` | scaffold, init, scanner tests | generated projects, prompts/defaults, dependency scan behavior |
| `internal/testrunner/` and `internal/checker/` | discovery, runner/checker, parser, reporter tests | runtime-backed analysis and result formatting |
| `internal/ui/` and `internal/atomicfile/` | printer and atomic file tests | user output and persistence helpers |
| `pkg/prolfile/` | public type tests | metadata type serialization and namespace support |

Use the [Project Component Overview](docs/codebase-knowledge-graph/Project%20Component%20Overview.md)
for the detailed file map when locating a specific test file.

## Test Strategy

| Layer | Expected test style |
| --- | --- |
| Manifest and lockfile persistence | Unit tests for parse, validation, deterministic write, and malformed input |
| Registry and HTTP callers | Unit tests with `httptest` or local helpers; do not depend on real network |
| Installer and store | Unit and integration-style tests with adversarial archive/checksum coverage |
| Runtime invocation | Argument and detection tests around runtime adapters |
| CLI commands | Command tests around flags, output contracts, and subprocess paths where process behavior matters |
| Fixtures | Integration scenarios under `testdata/` when project shape matters |

Keep unit seams in structs or interfaces rather than mutable package globals.
Mock registry behavior at the interface boundary and use `httptest` for HTTP
response behavior.

## Security And Adversarial Coverage

Security-critical changes need hostile-input tests. Touches to
`internal/installer/` or registry input handling should keep these cases
covered.

| Attack or failure | Primary test area | Examples |
| --- | --- | --- |
| Path traversal archives | `internal/installer/unpack_test.go` | `../../.bashrc`, `/etc/passwd`, paths that escape after cleaning |
| Unsafe links and special files | `internal/installer/unpack_test.go` | symlink escape, chained symlinks, hardlink escape, device files, FIFOs |
| Resource exhaustion | `internal/installer/unpack_test.go` | oversized downloads, extracted-size limits, excessive file counts |
| Checksum corruption | `internal/installer/verify_test.go` | truncated downloads and bit-flipped bytes |
| Malformed project metadata | `internal/manifest/*_test.go`, `internal/lockfile/*_test.go` | invalid TOML, invalid fields, conflict markers |
| Malformed registry/cache input | `internal/registry/*_test.go` | missing fields, invalid links, HTTP failure and retry paths |

Pair these tests with the invariants in [Security Rules](docs/security.md).

## Writing Tests

- Use `t.TempDir()` for filesystem tests.
- Use table-driven tests for input families and edge cases.
- Keep test filenames about behavior, not phase numbers.
- Internal packages return errors; command layers own CLI exit behavior.
- Error strings stay lowercase so `staticcheck` ST1005 remains clean.
- Wrap errors with `%w` and context in production code so tests can inspect
  useful failures.

Example seam:

```go
// Prefer dependencies injected through the tested object.
type myCmd struct {
    runner runnerFunc // nil means use the real implementation
}
```

## Future Coverage

Planned fuzz targets for major dependency and persistence surfaces:

```bash
go test ./internal/resolver  -fuzz FuzzConstraint     -fuzztime 60s
go test ./internal/installer -fuzz FuzzSafeExtract    -fuzztime 60s
go test ./internal/manifest  -fuzz FuzzProlfileToml   -fuzztime 60s
go test ./internal/lockfile  -fuzz FuzzLockfileContent -fuzztime 60s
```

Only run targets that exist on the current branch. Resolver coverage and some
fuzz targets arrive with later roadmap phases.

Future runtime compatibility coverage should add matrix scenarios for SWI,
GNU Prolog, and Scryer across supported operating systems. Runtime scenarios
should exercise `run`, `test`, `check`, REPL startup when available, module
loading differences, and runtime-specific flag handling.
