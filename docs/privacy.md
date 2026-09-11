# Privacy

LangTint's application code has no telemetry, analytics, HTTP client or clipboard-reading feature. It does not buffer or write the text you type.

It installs a global low-level keyboard hook to notice modifier/layout-switch shortcuts. The hook receives keyboard events, examines virtual-key codes and passes input onward. Foreground-window notifications trigger a layout read. A fallback can read the Windows input indicator's accessible name.

Diagnostic files may contain Windows paths, user-directory names, process/window identifiers, language names, version/build information and errors. Review and redact them before posting to public issues. Reports are not uploaded automatically. The program can launch Notepad to show a local report.

Installation writes program files and a current-user autostart entry. English mode changes the taskbar appearance and standard Arrow/Hand cursor roles in the Windows session. Normal exit/uninstall attempts restoration; abrupt termination and third-party customization can affect restoration.

GitHub handles repository visits, downloads, issues and CI under its own privacy policy. If SignPath is activated, release binaries and build/provenance metadata are sent to SignPath by CI, not from users' desktops. No user logs should be included in signing requests.
