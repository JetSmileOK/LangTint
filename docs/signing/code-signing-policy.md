# Code signing policy

## Status

**Prepared only. No SignPath approval, certificate or production signing has been obtained through this repository change. Current builds are unsigned.** Never label a release signed unless its delivered executable passes Authenticode verification against the expected certificate.

The project intends to apply for free open-source signing. After acceptance, the required attribution will be: **Free code signing provided by SignPath.io, certificate by SignPath Foundation.** This sentence describes the intended provider, not current sponsorship.

## Responsibilities

Repository owner and proposed signing approver: [@JetSmileOK](https://github.com/JetSmileOK). External contributions require maintainer review. Signing approval must remain manual. All people with repository/signing access must enable multi-factor authentication before production signing. Their configuration has not been verified here.

## Release rules

1. Build from reviewed public source in GitHub-hosted runners. Do not sign a local replacement executable.
2. Pin workflow actions to full commit SHAs and keep tokens in GitHub environment secrets.
3. Restrict signing to the default release branch and require environment approval plus SignPath production approval.
4. Enforce product/version metadata. Verify Authenticode status, certificate identity and resulting executable hash before packaging.
5. Recalculate checksums after signing. A signing error must stop the signed release, never silently substitute an unsigned file.
6. Do not overwrite a published release's assets. Create a new version for a correction.

A signature authenticates publisher/integrity. It does not prove absence of bugs and does not guarantee SmartScreen reputation. See [Privacy](../privacy.md), [SignPath setup](signpath-setup.md) and the official [Foundation terms](https://signpath.org/terms.html).
