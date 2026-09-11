# Contributing

Keep LangTint small: visible keyboard-layout state, no text rewriting, no telemetry and no permanent polling. Explain a change before broad refactoring.

Open a focused issue or pull request. Include Windows version/build, processor architecture, layout list, shortcut, cursor scheme and any taskbar customization. Redact account names and paths in logs. Never upload credentials or private text.

Run `python tools/release.py assemble`, `python -m unittest discover -s tools -p "test_*.py"`, and `go test ./...` in `source/`. Run Windows vet/build as described in the README. Packaging changes must preserve the reviewed source manifest unless a deliberate runtime update is being reviewed.

Distinguish Linux unit tests, Windows build checks and actual interactive Windows testing. Do not call a cross-build an Explorer compatibility test. Do not turn off a failing test just to publish.

Contributions are accepted under the repository's MIT license. Preserve upstream notices; identify any reused code and its license. Do not copy GPL implementation code into this MIT project without a compatible licensing decision.
