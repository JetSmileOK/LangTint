# LangTint roadmap

LangTint is intentionally small: make the current keyboard layout visible before typing, without reading or rewriting text.

This roadmap focuses on reach, trust and compatibility rather than feature bloat.

## P0 — public launch

- [ ] Publish the first GitHub Release with one obvious asset: `LangTint-Setup.exe`.
- [ ] Add a genuine 5–10 second RU → EN → RU screen recording to the top of the README.
- [ ] Add a GitHub social preview image based on the real product UI.
- [ ] Apply for free open-source code signing through SignPath Foundation after the first release is public.
- [ ] Keep release notes explicit about the current Windows 10 x64 support boundary.

## P1 — reach the current Windows audience

### Windows 11

Windows 11 is the highest-priority compatibility target. Support is not claimed until the real Explorer taskbar, cursor restore path, install/upgrade/uninstall and sleep/restart flows pass on real Windows 11 machines.

Tracking goals:

- Explorer taskbar discovery and ownership checks;
- taskbar composition behavior;
- Arrow/Hand cursor tint and full restore;
- multi-monitor taskbars;
- Explorer restart;
- sleep/resume;
- upgrade and uninstall.

### Per-language visual mapping

The first public behavior is intentionally simple: English is blue, Russian is normal. A broader release should allow users to choose which layout is neutral and which layouts receive a tint, with safe defaults and no text inspection.

### Distribution

After a stable signed release:

- WinGet manifest;
- optional Scoop manifest;
- direct `releases/latest/download/LangTint-Setup.exe` link;
- reproducible release notes and SHA-256.

## P2 — usability without clutter

Potential additions only if they preserve the core “ambient, zero-distraction” idea:

- small settings UI for language/color mapping;
- pause/disable control;
- optional tray entry without a permanent status popup;
- additional translations;
- import/export of visual preferences.

## Non-goals

LangTint should not become another automatic language guesser or text-rewriting utility.

- no typed-text collection;
- no clipboard monitoring for language detection;
- no automatic correction of user text;
- no advertising/telemetry SDK;
- no permanent layout polling when event-driven detection is available.

## Voting and contributions

Open a feature request or add 👍 to an existing issue. Compatibility reports with the exact Windows build, monitor configuration and input layouts are especially useful.
