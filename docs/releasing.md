# Releasing LangTint

## Current release candidate

The current public candidate is **v1.7.1**. It carries forward the reviewed v1.6 hardening runtime and the polished graphical installer.

Normal users should receive exactly one public binary:

**`LangTint-Setup.exe`**

The portable ZIP is still built and verified internally by CI for reproducibility/testing, but it is not a normal-user release asset.

## CI gates

The `Build and test Windows x64` workflow runs on pull requests and pushes. It:

1. verifies the reviewed source SHA-256 manifest;
2. verifies installer policy and repository release tests;
3. runs the Go runtime suite 100 times;
4. runs the race detector on Linux;
5. vets/builds the Windows x64 GUI target;
6. stages a deterministic portable ZIP for internal validation;
7. downloads the pinned official Inno Setup 7.1.0 asset;
8. checks its pinned SHA-256 and Authenticode publisher before installation on the hosted Windows runner;
9. compiles the installer;
10. deep-checks embedded image resources;
11. actually starts the compiled Setup on Windows and rejects known startup/decompression failures;
12. uploads tested CI artifacts and hashes.

A hosted runner does **not** replace an interactive Windows 10 Explorer acceptance test. The final candidate still needs a real desktop check for taskbar color, Arrow/Hand cursor color, upgrade, uninstall, reboot, sleep/resume and multiple monitors.

## Draft release

Run **Prepare LangTint release draft** from `main` and enter `v1.7.1`.

The workflow repeats the release gates, compiles Setup, recalculates SHA-256 and creates a **draft prerelease** only. It refuses to overwrite an existing release.

The release workflow regression suite requires Git and PowerShell 7 (`pwsh`)
on PATH. Run `python -m unittest discover -s tools -p "test_release_workflow.py" -v`
locally before pushing workflow edits. It executes the real workflow blocks with
the Actions PowerShell exit-code wrapper, disposable Git repositories and a
stubbed GitHub CLI (no network or release writes). The regular Linux/Windows CI
test discovery also runs these checks. Coverage includes absent/lightweight/
annotated/conflicting tags, API failures, existing drafts, pagination and
fail-fast release gates. This is not Windows desktop acceptance.

Public release assets:

- `LangTint-Setup.exe` — the only normal-user binary;
- `SHA256SUMS.txt` — integrity verification.

Do not publish local binaries in place of CI outputs.

After real-machine acceptance, promote the draft deliberately. Once the first stable release exists, README/download links may use GitHub's stable latest-asset path:

`releases/latest/download/LangTint-Setup.exe`

## Signing

Current release candidates are explicitly **unsigned**. See `docs/signing/` and `packaging/signpath-workflow.yml.example`.

The signing template remains inactive until SignPath Foundation approval and protected repository configuration are complete. Production signing must eventually cover both the runtime and final installer. Never claim a signed release from a partially signed chain.

## Versioning and corrections

Published assets are immutable evidence. Do not silently replace an already published binary under the same version. Fixes require a new version/tag.

The runtime/release toolchain is pinned to Go 1.27.1 for the current candidate. Future toolchain updates require the same compatibility and release-gate retesting before signing.
