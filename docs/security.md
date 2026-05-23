# Security Rules

`prolm` is a package manager. Registry metadata, manifests, lockfiles, remote
archives, and package source are inputs at a trust boundary. These rules are
non-negotiable for code review.

## Required Rules

| ID | Rule |
| --- | --- |
| SEC-1 | Verify SHA-256 on every package archive before unpacking, including archives read from cache. Delete mismatches and fail hard. |
| SEC-2 | Extract archives only through safe path handling. Reject traversal, absolute paths, unsafe hardlinks, unsafe symlinks, device files, pipes, and sockets. |
| SEC-3 | Never execute dependency install scripts. Downloaded code is not run during install. |
| SEC-4 | Require HTTPS for registry and download URLs. Reject `http://` inputs from registries and lockfiles. |
| SEC-5 | Never log credentials or place them in `Prolfile.toml`, `Prolfile.lock`, or error messages. |
| SEC-6 | Credentials use `0600` permissions. Store directories use `0755` permissions and should be checked on load. |
| SEC-7 | Guard store write operations with the store lock so concurrent installs cannot corrupt state. |
| SEC-8 | Write lockfiles atomically: write a temporary file, then rename it into place. |
| SEC-9 | Validate every external field before it becomes a path, URL, package identity, version, or execution argument. |
| SEC-10 | Respect rate limits and retry guidance. Do not hammer registries or GitHub. |
| SEC-11 | Do not silently choose between registries that claim the same package name; require explicit source selection. |
| SEC-12 | Enforce archive and extraction limits. Abort and remove partial output on resource-limit violations. |
| SEC-13 | Resolve symlinks within the extraction destination and reject chains or targets that escape it. |
| SEC-14 | Validate lockfile source URLs against expected registry origins or a configured allowlist. |

Current limits:

- Maximum archive download: 50 MB, configurable through
  `PROLM_MAX_TARBALL_SIZE`.
- Maximum extracted size: 200 MB.
- Maximum extracted file count: 10,000 files.
- Maximum single extracted file size: 20 MB.

## Extraction Checklist

Every archive entry must be joined under the extraction destination, cleaned,
and checked to remain inside that destination before creation. Extraction code
must also guard symlink resolution, cleanup on failure, archive size, extracted
size, file count, and file type.

Do not reduce the checks to a string prefix test that can confuse sibling paths
such as `/store/pkg` and `/store/pkg-evil`; include a path separator boundary
or use a path-relative containment check.

## Integrity And Trust

- Lockfile checksums protect package bytes from unexpected change. They do not
  prove who authored or published those bytes.
- New resolutions skip yanked versions. Existing locked installs may keep a
  yanked version, but should warn rather than silently upgrade.
- Package versions are immutable once published. Replacements use a new
  version or a yank notice.
- Package source is still arbitrary Prolog code at runtime. `prolm run`,
  `prolm test`, and dependency loading are not a sandbox.
- Do not add telemetry by default. Any future telemetry must be explicit,
  documented, and disableable.

Package-manager design tradeoffs and future trust work are summarized in
[Package Manager Design](package-manager-design.md).
