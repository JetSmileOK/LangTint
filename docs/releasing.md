# Releasing LangTint

## Current release candidate

The current public candidate is **v1.7.1**. It carries forward the reviewed v1.6 hardening runtime and adds only installer/migration behavior required for a normal Windows setup flow.

Normal users should receive **`LangTint-Setup-x64.exe`**. The portable ZIP remains an advanced/testing option.

## CI gates

The `Build and test Windows x64` workflow runs on pull requests and pushes. It:

1. verifies the reviewed source SHA-256 manifest;
2. verifies installer policy and repository release tests;
3. runs the Go runtime suite 100 times;
4. runs the race detector on Linux;
5. vets/builds the Windows x64 GUI target;
6. stages a deterministic portable ZIP;
7. downloads the pinned official Inno Setup 7.1.0 asset;
8. checks its pinned SHA-256 and Authenticode publisher before installation on the hosted Windows runner;
9. compiles `LangTint-Setup-x64.exe`;
10. uploads only the tested installer, portable ZIP and hashes.

A hosted runner does **not** replace an interactive Windows 10 Explorer acceptance test. The final candidate still needs a real desktop check for taskbar color, Arrow/Hand cursor color, upgrade, uninstall, reboot, sleep/resume and multiple monitors.

## Draft release

Run **Prepare LangTint release draft** from `main` and enter `v1.7.1`.

The workflow repeats the release gates, compiles the installer, recalculates SHA-256 and creates a **draft prerelease** only. It refuses to overwrite an existing release. Stable publication remains a deliberate manual action after real-machine acceptance.

Recommended release assets:

- `LangTint-Setup-x64.exe` — normal users;
- `LangTint-v1.7.1-Windows10-x64-unsigned.zip` — advanced/portable testing;
- `SHA256SUMS.txt` — integrity verification.

Do not publish local binaries in place of CI outputs.

## Signing

Current release candidates are explicitly **unsigned**. See `docs/signing/` and `packaging/signpath-workflow.yml.example`.

The signing template remains inactive until SignPath Foundation approval and protected repository configuration are complete. Production signing must eventually cover both the runtime and the final installer: sign the runtime first, compile Setup with the signed runtime, then sign and verify the final Setup executable. Never claim a signed release from a partially signed chain.

## Versioning and corrections

Published assets are immutable evidence. Do not replace an already published binary under the same version. Fixes require a new version/tag.

The runtime/release toolchain is pinned to Go 1.27.1, the current stable release reviewed for this candidate. Future toolchain updates require the same compatibility and release-gate retesting before signing.
