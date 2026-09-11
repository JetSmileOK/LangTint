# LangTint

### See your keyboard layout before you type.

A small Windows utility that makes the active layout visible through your **taskbar and mouse pointer** — instead of guessing what you meant after you typed it.

[Русский](README.ru.md) · [Downloads](https://github.com/JetSmileOK/colore_of_aungage/releases) · [Builds](https://github.com/JetSmileOK/colore_of_aungage/actions/workflows/build.yml) · [MIT license](LICENSE)

| Active layout | What you see |
| --- | --- |
| English | A light-blue taskbar background and blue Arrow/Hand pointers with a dark outline. |
| Russian | Your normal taskbar and configured Windows pointers. |

Text-selection and resize pointers stay unchanged. There is **no translucent window covering your taskbar icons**.

## The idea

The tiny ENG/RUS indicator is easy to overlook. LangTint turns a much larger part of the desktop into a cue you can notice with peripheral vision.

**No automatic text correction. No language guessing from what you type. Just a visible layout.**

## Download and use

Use the ZIP asset on an explicitly published [release](https://github.com/JetSmileOK/colore_of_aungage/releases), not GitHub's “Source code” ZIP. Until a release is published, successful [Actions runs](https://github.com/JetSmileOK/colore_of_aungage/actions/workflows/build.yml) provide preview artifacts; GitHub may require sign-in for those downloads.

Extract the whole package, then run `RUN_ALL_AND_INSTALL.cmd` or double-click `TaskbarLayoutTint_v1_5_3.exe`. Keep its `.manifest` file beside it. The current package installs for the current Windows user and includes `STATUS.cmd` and `UNINSTALL.cmd`.

**Current repository baseline: v1.5.3, Windows 10 x64, ordinary Explorer taskbar, RU/EN.** Its installer tests two Alt+Shift changes and therefore expects Alt+Shift and exactly that two-layout cycle. Windows 11, ARM64, custom taskbars and other configurations are not advertised as supported. The separately prepared v1.6 archive is not silently substituted by this release-infrastructure change.

## Signing status

**SignPath integration is prepared, not approved or active. Current builds are unsigned.** Windows may show an unknown-publisher or reputation warning. A checksum checks file integrity; it is not a trusted publisher signature. Do not disable Windows protection to install this utility.

[Code signing policy](CODE_SIGNING_POLICY.md) · [Signing setup](docs/SIGNPATH_SETUP.md)

## Small by design

- Reacts to supported layout-switch shortcuts and foreground-window events; no permanent language polling.
- Uses a separate visual worker, not taskbar rendering inside the keyboard callback.
- Uses native taskbar composition, not an overlay.
- Does not read clipboard contents, save typed text or include application telemetry/network code. See [Privacy](PRIVACY.md).

“Event-driven” does not mean zero work: the keyboard callback receives keyboard events, and layout reads/visual changes cost time. Performance is machine-dependent; the installer includes a short CPU check, not a battery-life guarantee.

## Build and verification

The release baseline is pinned to Go 1.23.2 to preserve the previously published build environment. This is a historical toolchain, not a claim that it is current; a supported-toolchain upgrade is a separate pre-production review item.

```sh
python tools/release.py assemble
cd source
go test ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-H=windowsgui -s -w" -o ../TaskbarLayoutTint_v1_5_3.exe .
```

The two larger files remain in the repository's existing `source_fragments/` layout. Assembly verifies every Go file against a reviewed SHA-256 manifest; missing, changed or extra source files stop packaging. See [Release process](docs/RELEASING.md).

CI checks source integrity, packaging tests, Go tests and the Windows build. These checks **do not replace interactive Windows testing** of Explorer, cursors, resume or different display setups.

## Help improve LangTint

[Report a compatibility problem](https://github.com/JetSmileOK/colore_of_aungage/issues/new/choose), share your Windows build and configuration, or contribute a small reviewed fix. Read [CONTRIBUTING.md](CONTRIBUTING.md) first. Useful feedback matters more than inflated compatibility claims.

If LangTint saves you from another `ghbdtn`, a star helps other people find it.
