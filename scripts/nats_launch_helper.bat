@echo off
cd /d "E:\Repos\MightyPie-Revamped\scripts"

REM Check if NATS is already running by looking for the default port 4222
netstat -an | find "4222" | find "LISTENING" > nul
if errorlevel 1 (
    echo NATS is not running. Starting NATS server...
    START powershell.exe -ExecutionPolicy Bypass -NoExit -File "nats_custom_start.ps1"
) else (
    echo NATS is already running
)

REM Always start the workers
START powershell.exe -ExecutionPolicy Bypass -NoExit -File "run_go_workers.ps1"