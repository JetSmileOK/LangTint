# Free SignPath preparation

**No payment, subscription purchase, application submission or signing activation was performed.** The repository is being prepared for a possible free Foundation application, not promised acceptance.

## Eligibility and remaining work

The [official terms](https://signpath.org/terms.html) require an OSI-approved license, maintained/documented/released software, trusted builds, MFA, named responsible people and manual signing approval. The Foundation also requires verifiable project reputation: a new repository is not automatically eligible.

Owner checklist:
- Confirm rights to the project's code and the MIT license, MFA for GitHub/SignPath, and signing roles.
- Publish a clearly marked unsigned release after Windows acceptance; record the release/download URL.
- Apply at https://signpath.org/apply.html using `docs/signing/signpath-application.md` as a factual draft. Do not claim approval or sponsorship in the application.
- After approval, install the SignPath GitHub App with access to this repository and configure the GitHub trusted build system.
- Add Windows VERSIONINFO to the release build: ProductName `LangTint`, ProductVersion/FileVersion matching the release, OriginalFilename matching the executable. **The historical v1.5.3 binary does not establish this prerequisite.** The template fails when metadata is missing; do not weaken that guard.
- Review a supported Go toolchain before production signing, without changing runtime behavior unnoticed.
- Create protected GitHub environment `signing`, restrict deployment branches to main, and require owner review.
- Store `SIGNPATH_API_TOKEN` as an environment secret, never send it in a chat or commit it.
- Set environment variables `SIGNPATH_ORGANIZATION_ID`, `SIGNPATH_PROJECT_SLUG`, `SIGNPATH_SIGNING_POLICY_SLUG`, `SIGNPATH_ARTIFACT_CONFIGURATION_SLUG`, and `SIGNPATH_CERT_THUMBPRINT` from the approved configuration. `SIGNPATH_READY=true` is set only after end-to-end validation.
- Confirm manual approval is required by the SignPath production policy. A GitHub review is not a substitute for that requirement.

## Template and artifact configuration

`packaging/signpath-workflow.yml.example` builds on a GitHub-hosted Windows runner, uploads the unsigned executable as an Actions artifact and passes its `artifact-id` to the pinned official SignPath action. No arbitrary local binary URL is accepted. `packaging/signpath-artifact.xml` signs only the named project EXE and restricts product/version metadata; review it in SignPath against an actual sample artifact.

After signing, `tools/verify_signature.ps1` checks Windows Authenticode status, expected certificate thumbprint, timestamp, product/version metadata and the exact resulting SHA-256. Packaging occurs **after** that check. Failed signing never falls back silently to an unsigned signed-labelled release.

Until activation, keep the README and release status **unsigned**. Even a valid signature is not a promise that SmartScreen will never warn.

Sources: https://docs.signpath.io/trusted-build-systems/github · https://docs.signpath.io/artifact-configuration/examples · https://signpath.org/terms.html
