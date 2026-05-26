# Security Policy

`prolm` is a package manager, so security reports are treated as high priority
even while the project is in early development.

## Reporting A Vulnerability

Please do not open a public issue for a vulnerability that could put users at
risk before a fix is available.

Report suspected vulnerabilities through GitHub's private vulnerability
reporting flow if it is enabled for the repository. If that is not available,
contact the maintainer directly and include:

- affected command or package
- reproduction steps
- expected and actual behavior
- security impact
- any relevant logs with secrets removed

## Scope

Security-sensitive areas include:

- package download and registry handling
- archive verification and extraction
- lockfile and manifest parsing
- credential handling
- runtime and subprocess invocation
- release packaging

Project security invariants and review rules live in
[docs/security.md](docs/security.md). Those rules are part of the review
contract for changes in this repository.

## Supported Versions

The project has not published a stable release yet. Until then, fixes target
the `main` branch and the latest tagged prerelease, if one exists.
