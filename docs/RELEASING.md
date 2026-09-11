# Releasing LangTint

## Scope of this change

This is release infrastructure for the exact v1.5.3 source already published in this repository. Application `.go` files are unchanged. The separately supplied v1.6 archive must have its own reviewed import/compatibility evidence; do not replace the source underneath a v1.5.3 tag.

## Preview build

The `Build Windows x64` workflow runs on push and pull request with read-only repository permissions. It assembles/verifies the source snapshot, runs Go and packaging tests, builds a Windows GUI executable, and uploads an explicitly **unsigned** ZIP plus SHA256SUMS. Do not run `--self-test`, `--install` or `--accept-install` on a hosted runner: these affect the desktop and do not establish Windows 10 user-session compatibility.

## Draft release

Run `Prepare release draft` from **main** in Actions. Enter the exact tag `v1.5.3` for the current `packaging/release.json`. The workflow checks that the tag/version match, refuses an existing release, builds from that run's immutable commit and creates a **draft prerelease** with unsigned assets and hashes. It does not publish a stable release automatically.

Before publishing the draft, verify source provenance, package checksums, build ID, current signing status, and real Windows acceptance/visual results. Review compatibility, clean installation, upgrade, uninstall, reboot, sleep/resume and multiple displays. The release notes must distinguish tested cases from unsupported ones. Use a new tag rather than replacing any already published asset.

## Signed release preparation

See `packaging/signpath-workflow.yml.example` and `docs/SIGNPATH_SETUP.md`. This template is intentionally **not an active workflow**. It requires Foundation acceptance, an approved project/policy, trusted-build integration, product/version resources and protected secrets. Do not rename it to `.github/workflows/...` until those requirements and the signature-verification step have been tested. No paid service is provisioned by these files.

## Output and integrity

`tools/release.py stage` uses an allowlist; personal reports, logs, old EXEs, source fragments and secrets are not included in the end-user ZIP. `package` uses deterministic ZIP timestamps and refuses extra files, stale hashes or an existing archive. The inside SHA256SUMS checks members; the outside SHA256SUMS checks the final ZIP. SHA-256 is integrity evidence, not code signing.

The Go toolchain remains pinned to the old build baseline for this infrastructure change. Updating it is a required separate security/compatibility review before production signing. Nothing here promises a new graphical installer, portable mode, Store acceptance, free-service approval or universal Windows support.
