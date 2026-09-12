# Changelog

## 1.7.1 — product polish

- Removes the legacy no-argument one-click self-installer and automatic Notepad report opening from `LangTint.exe`.
- Makes `LangTint-Setup-x64.exe` the only normal installation entry point.
- Shows the standard modern Welcome → Ready → Install → Finish wizard flow.
- Keeps the installed program directory minimal: `LangTint.exe`, its manifest, the product icon and Inno Setup's standard uninstall files.
- Moves runtime diagnostics out of the program directory into `%LOCALAPPDATA%\LangTint\Logs`.
- Fixes the combined acceptance CPU test so stale `WM_QUIT` cannot produce a false 0-second PASS, and rejects wall times shorter than the requested duration.
- Keeps preflight/uninstall diagnostics hidden in temporary files unless installation fails.

## 1.7.0 — installer release candidate

- Carries forward the v1.6 public hardening runtime: localized layout fallback, Explorer ownership validation, bounded startup recovery and zero permanent polling.
- Keeps the 100-scenario runtime failure matrix and non-mutating visual self-test.
- Adds `LangTint-Setup-x64.exe`: per-user, no-admin installer with modern light/dark UI.
- Adds the LangTint product icon and a normal Windows Installed Apps uninstall entry.
- Installer runs the non-mutating self-test before stopping an existing watcher or changing installation state.
- Adds a maintenance `--stop` command that stops the watcher and restores taskbar/cursors without deleting the user's existing autostart configuration.
- Aligns the built-in install location with `%LOCALAPPDATA%\Programs\LangTint`.
- Removes public release dependence on `.cmd` helper scripts and source-fragment reconstruction.
- CI verifies source hashes, installer policy, runtime tests, Windows x64 PE structure and the official Inno Setup 7.1.0 compiler provenance before compiling Setup.
- Release toolchain updated to Go 1.27.1 and must pass the full compatibility/release gates before publication.

## 1.6.0

- Hardened layout detection and localized Windows input-indicator fallback.
- Added Explorer ownership validation and bounded startup recovery without permanent polling.
- Added a 100-scenario runtime failure matrix.

## 1.5.3

- Restricted cursor tint to Arrow and Hand so the I-Beam remains visible on light backgrounds.
- Preserved direct taskbar composition, event-driven layout tracking and zero permanent polling.
