@echo off
setlocal
set "REPORT=%USERPROFILE%\Downloads\TaskbarLayoutTint_v1_5_3_STATUS.txt"
"%~dp0TaskbarLayoutTint_v1_5_3.exe" --status --report "%REPORT%"
set "EC=%ERRORLEVEL%"
if exist "%REPORT%" type "%REPORT%"
if exist "%REPORT%" start "" notepad.exe "%REPORT%"
exit /b %EC%
