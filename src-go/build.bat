@echo off
SETLOCAL EnableDelayedExpansion

REM Set the root directory for the Go source files
SET "SRC_DIR=%~dp0"

REM Change to the script's directory so go.mod can be found
cd /D "%SRC_DIR%"

REM Create assets directory in Tauri if it doesn't exist
SET "ASSETS_BIN_DIR=%SRC_DIR%..\src-tauri\assets\src-go\bin"
IF NOT EXIST "%ASSETS_BIN_DIR%" (
    echo Creating Tauri assets bin directory...
    MKDIR "%ASSETS_BIN_DIR%"
)

REM Parse arguments
SET "MODE=incremental"
IF "%1"=="--clean" SET "MODE=clean"

echo Building Go services in %MODE% mode...

REM List of services to build
SET "services=buttonManager mouseInputHandler pieButtonExecutor settingsManager shortcutDetector shortcutSetter windowManagement"

IF "%MODE%"=="clean" (
    echo Cleaning output directory...
    IF EXIST "%ASSETS_BIN_DIR%" (
        DEL /Q "%ASSETS_BIN_DIR%\*.exe" 2>nul
    )
)

FOR %%s IN (%services%) DO (
    echo Building %%s...
    go build -v -o "%ASSETS_BIN_DIR%\%%s.exe" "./cmd/%%s"
    
    IF !ERRORLEVEL! NEQ 0 (
        echo Failed to build %%s.
        EXIT /B !ERRORLEVEL!
    )
)

REM Build Go backend as mightypie-backend.exe
go build -v -o "%ASSETS_BIN_DIR%\mightypie-backend.exe" "./cmd/main"

echo.
echo Build complete.
echo All executables are in the '%ASSETS_BIN_DIR%' directory.
echo.

ENDLOCAL
