# TaskbarLayoutTint v1.5.3

Windows 10 x64 utility that makes the current keyboard language visible at a glance.

## What it does

- **RU** — normal Windows taskbar and normal cursor scheme.
- **EN** — taskbar background becomes light blue `#B7E9FF`.
- On EN, only the standard **Arrow** and **Hand** cursors are tinted blue with a dark outline.
- I-Beam, resize, busy, move and other service cursors remain standard Windows cursors.
- No overlay window over the taskbar.
- No permanent polling.
- No mouse hook.

The app is event-driven: it reacts to keyboard-layout shortcut events and foreground-window changes, then updates the taskbar/cursor state.

## Current build

`TaskbarLayoutTint_COMPOSITION_CURSOR_v1.5.3_20260911`

Target: **Windows 10 x64**.

## Install

Use either:

```text
RUN_ALL_AND_INSTALL.cmd
```

or double-click:

```text
TaskbarLayoutTint_v1_5_3.exe
```

Before installation, the program runs an acceptance test and CPU test. If a gate fails, installation is not completed. Previous test versions are cleaned up and the normal taskbar/cursor scheme is restored first.

Report path:

```text
%USERPROFILE%\Downloads\TaskbarLayoutTint_v1_5_3_REPORT.txt
```

## Uninstall

Run:

```text
UNINSTALL.cmd
```

## Source

The complete Go source is in [`source/`](source/).

No third-party Go modules are used.

## Notes

- This build was tested for Windows 10 x64.
- The prebuilt EXE is unsigned, so Windows SmartScreen may show an “Unknown publisher” warning.
- System theme/accent registry values are not modified.

## Version 1.5.3 change

Earlier builds tinted more system cursor types. On light/gray backgrounds the thin I-Beam could become hard to see. v1.5.3 now tints only **Arrow** and **Hand**; all text/service cursors stay standard Windows cursors.
