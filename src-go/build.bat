@echo off
SETLOCAL EnableDelayedExpansion

REM Set the root directory for the Go source files
SET "SRC_DIR=%~dp0"

REM Change to the script's directory so go.mod can be found
cd /D "%SRC_DIR%"

REM Create a clean bin directory
SET "BIN_DIR=%SRC_DIR%bin"
IF EXIST "%BIN_DIR%" (
    echo Cleaning bin directory...
    RMDIR /S /Q "%BIN_DIR%"
)
MKDIR "%BIN_DIR%"

REM List of services to build. The last one is the main orchestrator.
SET "services=buttonManager mouseInputHandler pieButtonExecutor settingsManager shortcutDetector shortcutSetter windowManagement main"

echo Building services...

REM Loop through each service and build it
FOR %%s IN (%services%) DO (
    echo Building %%s...
    SET "INPUT_FILE=%SRC_DIR%cmd\%%s\worker.go"
    IF "%%s"=="main" SET "INPUT_FILE=%SRC_DIR%cmd\%%s\main.go"

    go build -v -o "%BIN_DIR%\%%s.exe" "!INPUT_FILE!"
    IF !ERRORLEVEL! NEQ 0 (
        echo Failed to build %%s.
        GOTO :EOF
    )
)

echo.
echo Build complete.
echo All executables are in the '%BIN_DIR%' directory.
echo Run 'main.exe' from the 'bin' directory to start all services.

ENDLOCAL
