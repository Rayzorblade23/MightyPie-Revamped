$host.UI.RawUI.WindowTitle = "Go Workers"

# Hardcoded folder path
$FolderPath = "E:\Repos\MightyPie-Revamped\src-go\cmd"

# Store worker processes for cleanup
$script:workerProcesses = @()

# Cleanup function to terminate all processes
function Cleanup {
    Write-Host "`nTerminating all workers..." -ForegroundColor Yellow
    
    foreach ($worker in $script:workerProcesses) {
        if ($worker.Process -and -not $worker.Process.HasExited) {
            try {
                $worker.Process.Kill()
                $worker.Process.WaitForExit()
                $worker.Process.Close()
                $worker.Process.Dispose()
            }
            catch { }
        }
    }
    
    Write-Host "All workers terminated." -ForegroundColor Yellow
}

# Register cleanup for Ctrl+C
$null = Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { Cleanup }

# Ensure Go is installed and available
if (-not (Get-Command "go" -ErrorAction SilentlyContinue)) {
    Write-Error "Go is not installed or not in the system PATH."
    exit 1
}

# Find all worker.go files
$workerFiles = Get-ChildItem -Path $FolderPath -Recurse -Filter "worker.go"

if ($workerFiles.Count -eq 0) {
    Write-Host "No worker.go files found in $FolderPath"
    exit 0
}

# Assuming your go.mod is at E:\Repos\MightyPie-Revamped\src-go
$modRoot = "E:\Repos\MightyPie-Revamped\src-go"

foreach ($file in $workerFiles) {
    Write-Host "`nBuilding and starting: $($file.FullName)" -ForegroundColor Cyan
    
    $exePath = Join-Path $file.DirectoryName "worker.exe"
    
    # Build from module root, specifying relative path to worker.go
    $relativePath = Resolve-Path -Relative $file.FullName
    
    Push-Location $modRoot
    & go build -o $exePath $relativePath
    Pop-Location

    if (-not (Test-Path $exePath)) {
        Write-Host "Build failed: $exePath does not exist. Skipping..." -ForegroundColor Red
        continue
    }

    # Run the built executable
    $process = Start-Process -FilePath $exePath -WorkingDirectory $file.DirectoryName -NoNewWindow -PassThru

    $script:workerProcesses += @{
        Process = $process
        File = $file.FullName
    }
}



try {
    Write-Host "`nPress Ctrl+C to terminate all workers..." -ForegroundColor Yellow
    while ($script:workerProcesses | Where-Object { -not $_.Process.HasExited }) {
        Start-Sleep -Milliseconds 100
    }
}
finally {
    Cleanup
}