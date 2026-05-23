# Changelog

All notable user-facing changes to `prolm` are documented here.

## v0.1.0

Initial MVP release of `prolm`, a project manager and build tool for Prolog
projects.

### Highlights

- Project scaffolding with `prolm new` and `prolm init`
- `Prolfile.toml` and reproducible `Prolfile.lock` metadata
- SWI-Prolog pack index integration
- Dependency install, add, and remove workflows
- Safe package fetching, checksum verification, archive extraction, and local
  store management
- Runtime-backed `prolm run`, `prolm test`, and `prolm check`
- Environment diagnostics through `prolm env`
- Shell completion and hardened command/help contracts
- Linux and macOS release binaries with checksums and GitHub artifact
  attestations

### Notes

- SWI-Prolog is the supported runtime for this MVP.
- GNU Prolog, Scryer Prolog, Windows binaries, registry publishing, and
  package-manager distribution remain future roadmap work.
