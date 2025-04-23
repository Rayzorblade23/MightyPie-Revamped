@echo off
cd /d "E:\Repos\MightyPie-Revamped\scripts"
START powershell.exe -ExecutionPolicy Bypass -NoExit -File "nats_custom_start.ps1"
START powershell.exe -ExecutionPolicy Bypass -NoExit -File "run_go_workers.ps1"