@echo off
setlocal
chcp 65001 >nul
set "EXE=%~dp0TaskbarLayoutTint_v1_5_3.exe"
set "REPORT=%USERPROFILE%\Downloads\TaskbarLayoutTint_v1_5_3_REPORT.txt"
if exist "%REPORT%" del /q "%REPORT%" >nul 2>&1

echo BUILD EXPECTED: TaskbarLayoutTint_COMPOSITION_CURSOR_v1.5.3_20260911
echo Cursor policy: Arrow + Hand tinted; I-Beam and utility cursors remain Windows-normal.
echo.
if not exist "%EXE%" (
  echo ERROR: %EXE% not found.
  exit /b 91
)

"%EXE%" --accept-install --report "%REPORT%"
set "EC=%ERRORLEVEL%"
echo.
if exist "%REPORT%" type "%REPORT%"
echo.
echo Report: %REPORT%
if exist "%REPORT%" start "" notepad.exe "%REPORT%"
echo Exit code: %EC%
exit /b %EC%
