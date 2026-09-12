<p align="center">
  <img src="assets/LangTint-256.png" width="112" alt="LangTint icon">
</p>

<h1 align="center">LangTint</h1>
<p align="center"><strong>See your keyboard layout before you type.</strong></p>

<p align="center">
  <a href="https://github.com/JetSmileOK/LangTint/actions/workflows/build.yml"><img src="https://github.com/JetSmileOK/LangTint/actions/workflows/build.yml/badge.svg" alt="CI"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT"></a>
  <img src="https://img.shields.io/badge/Windows-10%20x64-0078D4.svg" alt="Windows 10 x64">
</p>

LangTint makes the active keyboard layout visible through **peripheral vision**. Instead of noticing the wrong layout after typing `ghbdtn`, you can see it before the first keystroke.

[Русский](README.ru.md) · [Releases](https://github.com/JetSmileOK/LangTint/releases) · [Privacy](docs/privacy.md) · [Security](SECURITY.md)

## What it does

| Active layout | Taskbar | Pointer |
| --- | --- | --- |
| **English** | light blue `#B7E9FF` | blue **Arrow** and **Hand** with a dark outline |
| **Russian** | normal Windows taskbar | normal Windows cursor scheme |

Text selection, resize, busy and other system cursors stay untouched. LangTint uses direct taskbar composition — **no translucent overlay over icons or text**.

> **No text correction. No language guessing. No permanent polling. Just a visible layout.**

## Install

For normal users there is one file:

**`LangTint-Setup-x64.exe`**

Download it from [Releases](https://github.com/JetSmileOK/LangTint/releases) and double-click it. The installer:

- installs only for the current Windows user;
- requires **no administrator rights**;
- offers **Start LangTint automatically with Windows**;
- offers **Launch LangTint now** after installation;
- runs LangTint's non-mutating visual self-test before enabling autostart;
- rolls back safely if the self-test fails;
- registers a normal uninstall entry in **Settings → Apps → Installed apps**;
- restores the Windows cursor scheme and removes LangTint autostart during uninstall.

The installer is built with a modern light/dark UI and the LangTint icon. Until free code signing is activated, preview builds are **unsigned**, so Windows may show an unknown-publisher or reputation warning.

## Compatibility

The v1.7.0 candidate deliberately supports:

- Windows 10 x64, builds **14393 through 19045** (the runtime gate fails safe outside the validated Windows 10 range);
- the standard Explorer taskbar;
- one or more installed keyboard layouts, including RU/EN;
- Alt+Shift, Ctrl+Shift and Win+Space switching.

Windows 11, ARM64 and replacement/custom taskbars are rejected or left unsupported until they receive explicit real-machine validation. This is intentional fail-safe behavior, not a silent compatibility claim.

## Why it is lightweight

- Event-driven keyboard-layout and foreground-window handling.
- Direct taskbar composition; no overlay window.
- Dedicated visual worker outside the keyboard callback.
- No permanent layout polling.
- No mouse hook.
- No application telemetry or network client code.
- No clipboard reading and no storage of typed text.

The low-level keyboard hook still receives Windows keyboard events so it can notice layout-switch shortcuts; see [Privacy](docs/privacy.md) for the exact scope.

## Quality gates

The v1.7 runtime includes a **100-scenario failure matrix** covering layout switching, localized Windows input-indicator names, taskbar ownership, startup recovery, event races and fail-safe behavior.

The release pipeline separately checks installer policy, source hashes, Windows x64 PE structure, uninstall/autostart rules, release assets, and supply-chain pinning. CI pins the reviewed release toolchain to **Go 1.27.1** for reproducibility; toolchain upgrades are reviewed separately.

A CI build is not described as an interactive Explorer compatibility test. Stable promotion still requires a real Windows 10 desktop acceptance pass.

## Build from source

Reviewed release toolchain: **Go 1.27.1**.

```sh
cd source
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-H=windowsgui -s -w" -o ../build/LangTint.exe .
```

The installer source is [`packaging/installer/LangTint.iss`](packaging/installer/LangTint.iss). CI bootstraps the official Inno Setup 7.1.0 release only after checking its pinned SHA-256 and Authenticode publisher.

## Repository layout

```text
assets/       LangTint product icon
source/       application source and tests
packaging/    installer, release metadata and signing preparation
tools/        release, supply-chain and installer tests
docs/         privacy, release and engineering notes
.github/      CI and release automation
```

## Contributing

Bug reports and focused fixes are welcome. Read [CONTRIBUTING.md](CONTRIBUTING.md) first and remove personal data from logs before posting them.

If LangTint saves you from another `ghbdtn`, a ⭐ helps other people discover it.

---

Created by **[JetSmileOK](https://github.com/JetSmileOK)** · [MIT License](LICENSE)
