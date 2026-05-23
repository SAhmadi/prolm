# prolm Release Runbook

This is the release checklist for publishing `prolm` binaries to GitHub
Releases. Release tags are the only publishing trigger.

## Release Policy

- Publish releases from `main` only.
- Use semver tags with a leading `v`, for example `v0.1.0`.
- The first supported binary targets are Linux and macOS, both `amd64` and
  `arm64`.
- Windows binaries, package-manager taps, and signing keys are deferred until
  the release cadence and runtime support are steadier.
- GitHub artifact attestations are skipped while this repository is user-owned
  and private; enable them by making the repository public or moving to a
  supported repository type.

## Preflight

1. Merge the intended release changes into `main`.
2. Confirm the `main` CI workflow is green.
3. Run the local checks when preparing the release:

```bash
make ci
go test ./cmd -run 'Version|Env' -count=1
goreleaser check
goreleaser release --snapshot --clean
```

4. Inspect `dist/` from the snapshot run and confirm archives exist for:
   - `linux_amd64`
   - `linux_arm64`
   - `darwin_amd64`
   - `darwin_arm64`
   - `checksums.txt`

## Publish

Create and push the release tag from the commit on `main`:

```bash
git checkout main
git pull --ff-only
git tag vX.Y.Z
git push origin vX.Y.Z
```

The `Release` workflow will run the full test matrix again before publishing.
If tests fail, no release is created.

## Verify

After the workflow completes:

1. Open the GitHub Releases page and confirm the new release is visible.
2. Confirm the four Linux/macOS archives and `checksums.txt` are attached.
3. Confirm the release notes match the relevant `CHANGELOG.md` summary.
4. If the repository is public or otherwise supports attestations, confirm
   GitHub artifact attestations were created for the checksum file.
5. Download one archive, unpack it, and verify the binary reports the tag:

```bash
./prolm version
```

For tag `v0.1.0`, the command should print:

```text
prolm 0.1.0
```

6. Verify the downloaded archive checksum against `checksums.txt`.

## Failure Handling

- If the release workflow fails before publishing, fix the issue and push a new
  commit plus a new tag.
- If a release was published with bad assets, delete the GitHub Release and tag,
  then publish a corrected semver tag.
- Do not replace package-manager-visible artifacts under the same version once
  external distribution channels exist; publish a new version instead.
