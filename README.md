<p align="center">
  <img src="assets/LangTint-256.png" width="112" alt="LangTint icon">
</p>

<h1 align="center">LangTint</h1>
<p align="center"><strong>See your keyboard layout before you type.</strong></p>
<p align="center">Stop typing <code>ghbdtn</code>.</p>

<p align="center">
  <a href="https://github.com/JetSmileOK/LangTint/actions/workflows/build.yml"><img src="https://github.com/JetSmileOK/LangTint/actions/workflows/build.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT"></a>
  <img src="https://img.shields.io/badge/Windows-10%20x64-0078D4.svg" alt="Windows 10 x64">
</p>

LangTint turns your **Windows taskbar and mouse pointer into an ambient keyboard-layout indicator**. The active layout is visible in peripheral vision before you type the first character.

[Русский](README.ru.md) · [Releases](https://github.com/JetSmileOK/LangTint/releases) · [Roadmap](ROADMAP.md) · [Privacy](docs/privacy.md) · [Security](SECURITY.md)

## The idea in 3 seconds

| Active layout | Taskbar | Pointer |
| --- | --- | --- |
| **English** | light blue `#B7E9FF` | blue **Arrow** and **Hand** with a dark outline |
| **Russian** | normal Windows taskbar | normal Windows cursor scheme |

Text selection, resize, busy and other system cursors stay untouched. LangTint changes the real Explorer taskbar directly — **no translucent overlay over icons or text**.

> **No text correction. No language guessing. No permanent polling. Just a visible layout.**

## Why this approach is different

Most layout helpers make you look at a tiny language code, draw another badge near the caret, or fix text after the mistake already happened. LangTint takes a different approach: make the state visible in UI you already see.

| Approach | What you notice | When |
| --- | --- | --- |
| Windows language code | small `ENG/RUS/...` indicator | only when you look for it |
| caret / popup badge | extra UI near text or pointer | while focusing on the badge |
| auto-correction | rewritten text | after typing starts |
| **LangTint** | taskbar + standard pointer change | **before typing, through peripheral vision** |

LangTint does not inspect typed text, read the clipboard, or try to infer what language you intended.

## Install

Normal users need exactly **one file**:

**`LangTint-Setup.exe`**

Download it from [Releases](https://github.com/JetSmileOK/LangTint/releases) and double-click it. The installer:

- installs only for the current Windows user;
- requires **no administrator rights**;
- offers autostart with Windows and launch-after-install options;
- runs a non-mutating compatibility self-test before enabling autostart;
- rolls back safely if the self-test fails;
- registers a normal uninstall entry in **Settings → Apps → Installed apps**;
- keeps the installed folder minimal;
- restores the Windows cursor scheme and removes LangTint autostart during uninstall.

Until free code signing is activated, preview builds are **unsigned**, so Windows may show an unknown-publisher or reputation warning.

## Compatibility

The current v1.7.1 candidate deliberately supports:

- Windows 10 x64, builds **14393 through 19045**;
- the standard Explorer taskbar;
- one or more installed keyboard layouts, including RU/EN;
- Alt+Shift, Ctrl+Shift and Win+Space switching.

Windows 11, ARM64 and replacement/custom taskbars are not claimed as supported until they receive explicit real-machine validation. See the [roadmap](ROADMAP.md).

## Lightweight by design

- event-driven keyboard-layout and foreground-window handling;
- direct taskbar composition, no overlay window;
- dedicated visual worker outside the keyboard callback;
- no permanent layout polling;
- no mouse hook;
- no application telemetry or network client code;
- no clipboard reading and no storage of typed text.

The low-level keyboard hook receives Windows keyboard events only so LangTint can notice layout-switch shortcuts. See [Privacy](docs/privacy.md) for the exact scope.

## Quality gates

The runtime includes a **100-scenario failure matrix** covering switching chords, localized Windows input-indicator names, taskbar ownership, startup recovery, event races and fail-safe behavior.

The installer pipeline additionally validates image assets, source hashes, Windows x64 PE structure, autostart/uninstall rules, supply-chain pins and the compiled Setup itself. The compiled installer is actually started on a Windows CI runner so startup failures are caught before a release candidate is produced.

A hosted runner does not replace real Explorer acceptance; stable promotion still requires a real Windows desktop pass.

## Roadmap

The highest-impact next steps are:

1. signed stable releases via SignPath Foundation;
2. genuine Windows 11 validation/support;
3. per-language color mapping instead of a fixed EN/RU presentation;
4. WinGet/Scoop-style distribution after the stable signed release.

See [ROADMAP.md](ROADMAP.md) and vote on feature issues with 👍.

## Build from source

Reviewed release toolchain: **Go 1.27.1**.

```sh
cd source
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-H=windowsgui -s -w" -o ../build/LangTint.exe .
```

Installer source: [`packaging/installer/LangTint.iss`](packaging/installer/LangTint.iss).

## Contributing

Bug reports, compatibility reports and focused fixes are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) first and remove personal data from logs before posting them.

If LangTint saves you from another `ghbdtn`, **star the repository** — it directly helps other multilingual Windows users discover it.

---

Created by **[JetSmileOK](https://github.com/JetSmileOK)** · [MIT License](LICENSE)
