# LangTint

**See your keyboard layout before you type.**

[![Build](https://github.com/JetSmileOK/colore_of_aungage/actions/workflows/build.yml/badge.svg)](https://github.com/JetSmileOK/colore_of_aungage/actions/workflows/build.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Windows 10 x64](https://img.shields.io/badge/Windows-10%20x64-0078D4.svg)](#compatibility)

LangTint is a tiny Windows utility that turns the active keyboard layout into an **ambient visual cue**. Instead of noticing the wrong layout after typing `ghbdtn`, you can see it before the first keystroke.

[Русский](README.ru.md) · [Releases](https://github.com/JetSmileOK/colore_of_aungage/releases) · [Privacy](docs/privacy.md) · [Security](SECURITY.md)

## What you see

| Active layout | Taskbar | Pointer |
| --- | --- | --- |
| **English** | light blue `#B7E9FF` | blue **Arrow** and **Hand**, dark outline |
| **Russian** | normal Windows taskbar | normal Windows cursor scheme |

Text-selection, resize, busy and other utility pointers remain unchanged, so they keep their normal contrast. There is **no translucent overlay over taskbar icons**.

## Why this exists

The tiny `ENG/RUS` indicator is easy to miss. LangTint makes layout state visible through peripheral vision without trying to analyze or rewrite what you type.

**No automatic text correction. No language guessing. No permanent polling. Just a visible layout.**

## Download

Use an explicitly published asset from [Releases](https://github.com/JetSmileOK/colore_of_aungage/releases). Successful [CI runs](https://github.com/JetSmileOK/colore_of_aungage/actions/workflows/build.yml) also produce unsigned preview artifacts for testing.

The current repository baseline is **v1.5.3 for Windows 10 x64**. Public-release hardening and a friendlier installer are being prepared separately; unsupported platforms are not silently advertised as working.

Current binaries are **unsigned**, so Windows can show an unknown-publisher / reputation warning. SignPath Foundation integration is prepared but is not claimed as active until the project is actually approved and the resulting signature is verified.

## Small by design

- Event-driven layout-switch and foreground-window handling.
- Direct taskbar composition — no overlay window.
- Visual work runs outside the keyboard callback.
- No permanent layout polling.
- No mouse hook.
- No application telemetry or network client code.
- Does not save typed text or read clipboard contents.

The keyboard callback still receives Windows keyboard events; “event-driven” does **not** mean literally zero CPU work. See [Privacy](docs/privacy.md) for the exact scope.

## Compatibility

Currently advertised and tested as a baseline for:

- Windows 10 x64;
- standard Explorer taskbar;
- RU/EN configuration described by the current release.

Windows 11, ARM64 and custom taskbars are not advertised as supported until they receive explicit real-machine validation.

## Build from source

The project source lives under [`source/`](source/). Two large Win32 files are reconstructed deterministically from byte-exact fragments kept under `tools/internal/` before testing and building.

```sh
cd source
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H=windowsgui -s -w" -o ../TaskbarLayoutTint_v1_5_3.exe .
```

Release packaging additionally verifies the exact reviewed source manifest and package contents. See [Release engineering](docs/releasing.md).

## Repository layout

```text
source/       application source and tests
packaging/    release metadata, manifests and portable-package scripts
tools/        build/release tools; internal fragments are tucked away here
docs/         privacy, signing and engineering notes
.github/      CI, issue templates and automation
```

Historical validation reports are kept under `docs/engineering/`, not in the product-facing root.

## Contributing

Bug reports and focused fixes are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) first and remove personal data from logs before posting them.

If LangTint saves you from another `ghbdtn`, a ⭐ helps other people discover it.

---

Created by **[JetSmileOK](https://github.com/JetSmileOK)** · [MIT License](LICENSE)
