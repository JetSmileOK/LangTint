@echo off
setlocal
cd /d "%~dp0\.."
python tools\assemble_source.py || exit /b 1
pushd source
go test ./... || exit /b 1
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go vet ./... || exit /b 1
go build -trimpath -ldflags="-H=windowsgui -s -w" -o ..\TaskbarLayoutTint_v1_5_3.exe . || exit /b 1
popd
echo Build complete: TaskbarLayoutTint_v1_5_3.exe
endlocal
