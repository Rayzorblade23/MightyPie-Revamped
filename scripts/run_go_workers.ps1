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

foreach ($file in $workerFiles) {
    Write-Host "`nStarting: $($file.FullName)" -ForegroundColor Cyan
    
    $process = Start-Process -FilePath "go" -ArgumentList "run worker.go" `
        -WorkingDirectory $file.DirectoryName -NoNewWindow -PassThru

    $workerName = Split-Path $file.DirectoryName -Leaf

    # Store process info
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