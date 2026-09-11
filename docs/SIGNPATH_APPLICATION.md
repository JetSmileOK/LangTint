# SignPath application draft — maintainer review required

This is a draft for the owner to review and submit. It has not been sent.

**Project:** LangTint (formerly TaskbarLayoutTint)

**Repository:** https://github.com/JetSmileOK/colore_of_aungage

**License:** MIT. Runtime is Go; no third-party Go modules are declared. Go's runtime/standard-library license is included in `THIRD_PARTY_NOTICES.md`.

**Maintainer / proposed signing approver:** GitHub @JetSmileOK. MFA and role assignment require owner confirmation.

**Purpose:** A Windows 10 x64 visual keyboard-layout indicator. English changes taskbar appearance and the system Arrow/Hand cursor roles; Russian restores normal appearance. It does not correct text. Text and other utility cursor roles remain unchanged.

**Sensitive functionality disclosure:** The utility uses a global low-level keyboard hook to detect shortcut modifiers and foreground-window notifications to trigger a layout read. It changes taskbar composition and system cursor roles and installs a current-user autostart entry. It contains no application telemetry/network or clipboard-reading feature and does not store typed text. The hook is not a text capture feature.

**Release/download URL:** Owner must insert the actual published release, not a draft link.

**Signing scope:** Executable built by the public GitHub Actions workflow. No third-party binary re-signing, kernel driver or EV certificate purchase is requested.

**Status:** The code is published; release/signing preparation is under review. Foundation acceptance, Windows product/version resources, supported-toolchain review and signing configuration are not claimed complete. Please advise whether the project's current release history and reputation meet your criteria.
