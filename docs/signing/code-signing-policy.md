# Code signing policy

## Status

**Prepared only. SignPath Foundation approval and production signing are not active yet. Current v1.7.0 candidates are unsigned.**

Never label a release signed unless the actual downloadable artifacts pass Authenticode verification against the expected certificate and their post-signing hashes are published.

The project intends to apply for free open-source signing. After acceptance, the required attribution will be: **Free code signing provided by SignPath.io, certificate by SignPath Foundation.** This describes the intended provider, not current sponsorship.

## Responsibilities

Repository owner and proposed signing approver: [@JetSmileOK](https://github.com/JetSmileOK). External changes require maintainer review. Signing approval remains manual. Repository/signing accounts must use multi-factor authentication before production signing.

## Release rules

1. Build from reviewed public source on GitHub-hosted runners. Never substitute a locally built executable.
2. Pin workflow actions and external build-tool downloads to reviewed identities/digests.
3. Keep signing credentials only in protected GitHub environments/secrets.
4. Verify product/version metadata, certificate identity, timestamp and SHA-256 after signing.
5. Sign the **runtime first**, compile the installer with that signed runtime, then sign and verify **`LangTint-Setup-x64.exe`**. A signed installer containing an unsigned runtime is not the intended production chain.
6. Recalculate all release hashes after signing.
7. A signing failure stops the release; it must never fall back silently to an unsigned asset.
8. Never overwrite published release assets. Corrections get a new version.

A valid signature authenticates publisher/integrity; it does not prove absence of bugs and does not guarantee immediate SmartScreen reputation.
