# Hardcoded folder path
$FolderPath = "E:\Repos\MightyPie-Revamped\src-go\cmd"

# Store worker processes for cleanup
$script:workerProcesses = @()

# Cleanup function to terminate all processes
function Cleanup {
    Write-Host "`nTerminating all workers..." -ForegroundColor Yellow
    foreach ($worker in $script:workerProcesses) {
        if (-not $worker.Process.HasExited) {
            $worker.Process.Kill()
            $worker.Process.WaitForExit()
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
    
    $processInfo = New-Object System.Diagnostics.ProcessStartInfo
    $processInfo.FileName = "go"
    $processInfo.Arguments = "run worker.go"
    $processInfo.RedirectStandardOutput = $true
    $processInfo.RedirectStandardError = $true
    $processInfo.UseShellExecute = $false
    $processInfo.CreateNoWindow = $true
    $processInfo.WorkingDirectory = $file.DirectoryName

    $process = Start-Process -FilePath $processInfo.FileName -ArgumentList $processInfo.Arguments -WorkingDirectory $processInfo.WorkingDirectory -NoNewWindow -PassThru

    # Store process info for later
    $script:workerProcesses += @{
        Process = $process
        File = $file.FullName
    }

    # Create job to handle output asynchronously
    $null = Start-Job -ScriptBlock {
        param($processId, $workerName)
        $process = Get-Process -Id $processId
        while (!$process.HasExited) {
            $line = $process.StandardOutput.ReadLine()
            if ($line) {
                Write-Host "[$workerName] $line" -ForegroundColor Green
            }
        }
    } -ArgumentList $process.Id, (Split-Path $file.DirectoryName -Leaf)

    # Create job to handle errors asynchronously
    $null = Start-Job -ScriptBlock {
        param($processId, $workerName)
        $process = Get-Process -Id $processId
        while (!$process.HasExited) {
            $line = $process.StandardError.ReadLine()
            if ($line) {
                Write-Host "[$workerName] $line" -ForegroundColor Red
            }
        }
    } -ArgumentList $process.Id, (Split-Path $file.DirectoryName -Leaf)
}

try {
    # Wait for all processes to complete or user interrupt
    Write-Host "`nPress Ctrl+C to terminate all workers..." -ForegroundColor Yellow
    while ($script:workerProcesses | Where-Object { -not $_.Process.HasExited }) {
        Start-Sleep -Seconds 1
    }
}
finally {
    # Ensure cleanup runs
    Cleanup
    Get-Job | Remove-Job -Force
}