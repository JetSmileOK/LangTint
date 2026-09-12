# Launch checklist

Repository: **JetSmileOK/LangTint**.

Positioning: **See your keyboard layout before you type.** LangTint turns the Windows taskbar and pointer into an ambient keyboard-layout cue; it does not rewrite text or guess what language the user intended.

## Launch blockers

Before broad promotion:

- publish a real GitHub Release from `main`;
- expose one obvious public download: `LangTint-Setup.exe`;
- add a genuine 5–10 second RU → EN → RU recording near the top of the README;
- add a repository social preview based on the real product UI;
- keep the unsigned status explicit until SignPath is actually active;
- never claim Windows 11 or universal compatibility before real validation.

## Two-stage launch

### Stage 1 — current Windows 10 release

Use this to collect first users, bug reports and stars without pretending the compatibility matrix is broader than it is.

Suggested post:

> I kept typing in the wrong keyboard layout, so I made the layout visible in the Windows taskbar and mouse pointer. No text correction, no language guessing, no permanent polling — just a cue you notice before typing. LangTint is open source and currently targets Windows 10 x64.

Good targets are communities where multilingual Windows users and open-source utility users already gather. Do not mass-post identical copy or buy engagement.

### Stage 2 — broader launch

Do this after Windows 11 validation and a signed stable installer. That removes the two biggest adoption objections: “does it work on my current Windows?” and “why does SmartScreen warn me?”

At that point consider:

- Show HN;
- relevant Windows / productivity / open-source communities;
- WinGet / Scoop distribution;
- curated Windows-utility lists;
- a short product page or GitHub Pages landing page if the README becomes too dense.

## GitHub presentation

The first viewport should answer four questions immediately:

1. What problem does this solve?
2. What changes on screen?
3. How do I install it?
4. Can I trust it?

Required assets:

- product icon;
- real animated demo;
- CI / MIT / platform badges;
- one-file install callout;
- privacy and security links;
- roadmap;
- clear star CTA without begging or spam.

Suggested repository description:

> See your keyboard layout before you type. A lightweight taskbar and cursor indicator for Windows.

Recommended topics:

`windows`, `windows-utility`, `keyboard-layout`, `keyboard-language`, `language-indicator`, `input-method`, `multilingual`, `taskbar`, `cursor`, `productivity`, `win32`, `golang`.

## Social preview

Use a 1280×640 image with a solid or carefully tested background. It should show the LangTint icon, the tagline, and a real before/after visual from the product. Do not fabricate a fake Windows screenshot.

## What not to do

- do not buy stars;
- do not spam unrelated repositories/issues;
- do not fabricate testimonials or benchmarks;
- do not hide SmartScreen/signing limitations;
- do not add broad compatibility badges that were not actually tested.

The strongest story is simple: a familiar multilingual typing problem, an immediately visible solution, a trustworthy installer, and a real demo.
