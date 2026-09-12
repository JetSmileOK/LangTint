# Security

Do not post credentials, private diagnostics or vulnerability details in a public issue. Use GitHub's private vulnerability reporting button when it is available; otherwise ask the maintainer for a private contact channel before sharing sensitive details. Enabling private reporting is a repository-owner setup task.

Current scope is the documented Windows 10 x64 build, not Windows 11, a general-purpose keyboard logger or a security product. The low-level hook must never record typed text. Windows cursor and taskbar APIs have session-wide effects; rollback and normal shutdown behavior need Windows testing.

Release credentials belong in protected GitHub environments, never in code or logs. CI on pull requests must not receive production signing credentials. Any invalid/missing signing configuration or certificate must block signed output. Do not advise users to disable Defender or install a self-signed root certificate.

Release candidates are built with Go 1.27.1, the current stable toolchain reviewed for this candidate. Production releases must stay on a supported Go toolchain with current security fixes; signing and green CI do not replace security maintenance.
