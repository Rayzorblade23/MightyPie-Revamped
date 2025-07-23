@echo off
REM Development build script for Go services.
REM This script performs an incremental build without cleaning the output directory
REM to leverage Go's caching for faster startup times during development.

REM Ensure we are in the script's directory so go.mod can be found.
cd /D "%~dp0"

REM Delete the previously renamed main executable to prevent 'duplicate file' errors.
if exist "bin\main-x86_64-pc-windows-msvc.exe" del "bin\main-x86_64-pc-windows-msvc.exe"

REM Build all services. Go will only recompile what has changed.
echo Building Go services (incremental)...
go build -v -o "bin\buttonManager.exe" "./cmd/buttonManager"
go build -v -o "bin\mouseInputHandler.exe" "./cmd/mouseInputHandler"
go build -v -o "bin\pieButtonExecutor.exe" "./cmd/pieButtonExecutor"
go build -v -o "bin\settingsManager.exe" "./cmd/settingsManager"
go build -v -o "bin\shortcutDetector.exe" "./cmd/shortcutDetector"
go build -v -o "bin\shortcutSetter.exe" "./cmd/shortcutSetter"
go build -v -o "bin\windowManagement.exe" "./cmd/windowManagement"
go build -v -o "bin\main.exe" "./cmd/main"

REM Rename the main executable for Tauri sidecar compatibility.
ren "bin\main.exe" "main-x86_64-pc-windows-msvc.exe"

echo Development build complete.
