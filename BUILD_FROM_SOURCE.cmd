@echo off
setlocal
where python >nul 2>nul || (
  echo Python is required to assemble the two large Win32 source files.
  exit /b 1
)
where go >nul 2>nul || (
  echo Go is not installed or not in PATH.
  echo Install Go 1.23.x from https://go.dev/dl/
  exit /b 1
)
python "%~dp0ASSEMBLE_SOURCE.py" || exit /b 1
pushd "%~dp0source"
go test ./... || exit /b 1
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -trimpath -ldflags="-H=windowsgui -s -w" -o "..\TaskbarLayoutTint_v1_5_3.exe" . || exit /b 1
popd
echo.
echo Build complete: TaskbarLayoutTint_v1_5_3.exe
endlocal
