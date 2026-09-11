# TaskbarLayoutTint v1.5.3

Windows 10 x64 utility that makes the active keyboard language visible without permanent polling.

## Behavior

| Language | Taskbar | Cursor |
|---|---|---|
| **RU** | Normal Windows taskbar | Normal Windows cursor scheme |
| **EN** | Light-blue `#B7E9FF` taskbar background | Light-blue **Arrow** and **Hand** with dark outline |

`I-Beam`, resize, busy, move, cross and other utility cursors remain the normal Windows cursors, so text/editing cursors keep their contrast.

## Architecture

- Event-driven `WH_KEYBOARD_LL` shortcut detection.
- Foreground-window change notifications.
- Direct Windows taskbar composition — **no overlay window**.
- Dedicated visual worker, separate from the keyboard hook.
- MSAA is used only as an event-time fallback when the foreground-thread HKL cannot be read.
- **No permanent language polling.**
- **No mouse hook.**
- Windows theme/accent registry values are not modified.

Current build ID:

```text
TaskbarLayoutTint_COMPOSITION_CURSOR_v1.5.3_20260911
```

Target: **Windows 10 x64**.

## Download the tested Windows build

Open **Actions** → **Build Windows x64** → the latest successful run, then download the artifact:

```text
TaskbarLayoutTint-v1.5.3-Win10-x64
```

The artifact contains the Windows EXE and a ZIP with:

- `TaskbarLayoutTint_v1_5_3.exe`
- `TaskbarLayoutTint_v1_5_3.exe.manifest`
- `RUN_ALL_AND_INSTALL.cmd`
- `STATUS.cmd`
- `UNINSTALL.cmd`
- `README_RU.txt`
- `BUILD_ID.txt`

## Install

After downloading and extracting the artifact package, either run:

```text
RUN_ALL_AND_INSTALL.cmd
```

or double-click:

```text
TaskbarLayoutTint_v1_5_3.exe
```

Before installation the program runs its Windows-side acceptance test and a real CPU test. If a gate fails, installation is not completed.

Report path:

```text
%USERPROFILE%\Downloads\TaskbarLayoutTint_v1_5_3_REPORT.txt
```

## Build from source

Requirements:

- Go 1.23.x
- Python 3

On Windows run:

```text
BUILD_FROM_SOURCE.cmd
```

Two large Win32 source files are stored as checked-in byte fragments in `source_fragments/`. `ASSEMBLE_SOURCE.py` reconstructs them before testing/building. This assembly is deterministic; the checked-in SHA-256 values and validation report are included in the repository.

The GitHub Actions workflow performs the same assembly, runs tests and Windows `go vet`, then cross-compiles the Windows x64 executable.

## Validation

See:

- `TEST_REPORT_v1_5_3_FINAL.txt`
- `STATIC_AUDIT_v1_5_3.json`
- `SHA256SUMS.txt`

The delivered v1.5.3 was validated with repeated unit/state tests, race/checkptr tests, Linux/Windows vet, Windows x64 builds and static/binary audits.

## Source layout

- `source/` — normal Go source files.
- `source_fragments/` — exact fragments for `app_windows.go` and `winapi_windows.go`.
- `ASSEMBLE_SOURCE.py` — reconstructs those two files.
- `.github/workflows/build.yml` — CI test/build/package workflow.

## Notes

- The prebuilt EXE is unsigned, so Windows SmartScreen may show an **Unknown publisher** warning.
- This version is intended for Windows 10 x64.
- v1.5.3 tints only Arrow and Hand; text and utility cursor types stay in the user's normal Windows cursor scheme.
