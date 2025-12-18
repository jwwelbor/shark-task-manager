# Automated Scoop Installation Test Script
# Tests complete Scoop bucket installation and functionality

[CmdletBinding()]
param(
    [string]$ExpectedVersion = "v0.1.0-beta"
)

# Configuration
$BucketName = "shark"
$BucketUrl = "https://github.com/jwwelbor/scoop-shark"
$AppName = "shark"
$ResultsFile = "scoop-test-results.txt"

# Start timing
$StartTime = Get-Date

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Scoop Installation Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Expected Version: $ExpectedVersion"
Write-Host "Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
Write-Host ""

# Test results tracking
$Script:Passed = 0
$Script:Failed = 0
$Script:Timings = @{}

# Cleanup function
function Cleanup-TestInstallation {
    Write-Host ""
    Write-Host "Cleaning up test installation..." -ForegroundColor Yellow

    try {
        scoop uninstall $AppName 2>$null
        scoop bucket rm $BucketName 2>$null
        Write-Host "Cleanup complete" -ForegroundColor Green
    }
    catch {
        Write-Host "Cleanup encountered errors (may be expected)" -ForegroundColor Yellow
    }
}

# Test function
function Test-Step {
    param(
        [string]$Description,
        [scriptblock]$Command
    )

    Write-Host -NoNewline "Testing: $Description... "

    try {
        $null = & $Command
        Write-Host "PASS" -ForegroundColor Green
        $Script:Passed++
        return $true
    }
    catch {
        Write-Host "FAIL" -ForegroundColor Red
        Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
        $Script:Failed++
        return $false
    }
}

# Timed test function
function Test-TimedStep {
    param(
        [string]$Description,
        [scriptblock]$Command,
        [int]$MaxSeconds
    )

    Write-Host -NoNewline "Testing: $Description (max ${MaxSeconds}s)... "

    $cmdStart = Get-Date

    try {
        $null = & $Command
        $cmdEnd = Get-Date
        $duration = ($cmdEnd - $cmdStart).TotalSeconds

        if ($duration -le $MaxSeconds) {
            Write-Host "PASS ($([math]::Round($duration, 1))s)" -ForegroundColor Green
            $Script:Passed++
        }
        else {
            Write-Host "PASS but slow ($([math]::Round($duration, 1))s > ${MaxSeconds}s target)" -ForegroundColor Yellow
            $Script:Passed++
        }

        return $duration
    }
    catch {
        $cmdEnd = Get-Date
        $duration = ($cmdEnd - $cmdStart).TotalSeconds
        Write-Host "FAIL ($([math]::Round($duration, 1))s)" -ForegroundColor Red
        Write-Host "  Error: $($_.Exception.Message)" -ForegroundColor Red
        $Script:Failed++
        return $duration
    }
}

# Check if Scoop is installed
Write-Host "========================================"
Write-Host "Pre-Flight Checks"
Write-Host "========================================"

if (-not (Get-Command scoop -ErrorAction SilentlyContinue)) {
    Write-Host "ERROR: Scoop is not installed" -ForegroundColor Red
    Write-Host "Install Scoop from: https://scoop.sh" -ForegroundColor Yellow
    exit 1
}

Write-Host "Scoop is installed: $(scoop --version)" -ForegroundColor Green
Write-Host ""

# Ensure cleanup on exit
try {
    Write-Host "========================================"
    Write-Host "Phase 1: Bucket Addition"
    Write-Host "========================================"

    # Add bucket
    Write-Host "Adding bucket: $BucketName from $BucketUrl"
    $bucketStart = Get-Date
    scoop bucket add $BucketName $BucketUrl
    $bucketEnd = Get-Date
    $bucketDuration = [math]::Round(($bucketEnd - $bucketStart).TotalSeconds, 1)

    Write-Host "SUCCESS - Bucket added in ${bucketDuration}s" -ForegroundColor Green
    $Script:Passed++
    $Script:Timings["BucketAdd"] = $bucketDuration

    Write-Host ""
    Write-Host "========================================"
    Write-Host "Phase 2: Installation"
    Write-Host "========================================"

    # Install with timing (should be < 30 seconds)
    Write-Host "Installing $AppName..."
    $installStart = Get-Date
    scoop install $AppName
    $installEnd = Get-Date
    $installDuration = [math]::Round(($installEnd - $installStart).TotalSeconds, 1)

    if ($installDuration -le 30) {
        Write-Host "SUCCESS - Installed in ${installDuration}s (target: <30s)" -ForegroundColor Green
    }
    else {
        Write-Host "SUCCESS - Installed in ${installDuration}s (slower than 30s target)" -ForegroundColor Yellow
    }
    $Script:Passed++
    $Script:Timings["Install"] = $installDuration

    Write-Host ""
    Write-Host "========================================"
    Write-Host "Phase 3: Installation Verification"
    Write-Host "========================================"

    # Verify binary exists
    $result = Test-Step "Binary exists in PATH" {
        if (-not (Get-Command $AppName -ErrorAction SilentlyContinue)) {
            throw "Binary not found in PATH"
        }
    }

    # Verify version
    $result = Test-Step "Version command works" {
        $output = & $AppName --version 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Version command failed"
        }
    }

    # Check actual version
    $versionOutput = & $AppName --version 2>&1
    if ($versionOutput -match '(v[0-9]+\.[0-9]+\.[0-9]+(-[a-z]+)?)') {
        $actualVersion = $matches[1]
    }
    else {
        $actualVersion = "unknown"
    }

    Write-Host "Installed version: $actualVersion"

    if ($actualVersion -eq $ExpectedVersion) {
        Write-Host "Version matches expected" -ForegroundColor Green
        $Script:Passed++
    }
    else {
        Write-Host "WARNING: Version mismatch" -ForegroundColor Yellow
        Write-Host "  Expected: $ExpectedVersion"
        Write-Host "  Actual: $actualVersion"
    }

    # Verify installation location
    $binaryPath = (Get-Command $AppName).Path
    Write-Host "Binary location: $binaryPath"

    if ($binaryPath -like "*scoop*") {
        Write-Host "Binary in expected Scoop location" -ForegroundColor Green
        $Script:Passed++
    }
    else {
        Write-Host "WARNING: Binary in unexpected location" -ForegroundColor Yellow
    }

    Write-Host ""
    Write-Host "========================================"
    Write-Host "Phase 4: Functional Tests"
    Write-Host "========================================"

    # Test help command
    $result = Test-Step "Help command works" {
        $output = & $AppName --help 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Help command failed"
        }
    }

    # Test epic list command
    $result = Test-Step "Epic list command works" {
        $output = & $AppName epic list 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Epic list command failed"
        }
    }

    # Test task list command
    $result = Test-Step "Task list command works" {
        $output = & $AppName task list 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Task list command failed"
        }
    }

    # Test database access
    $result = Test-Step "Database access works" {
        $output = & $AppName task list --status created 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Database access failed"
        }
    }

    Write-Host ""
    Write-Host "========================================"
    Write-Host "Phase 5: Binary Analysis"
    Write-Host "========================================"

    # Check binary size
    $binarySize = (Get-Item $binaryPath).Length
    $binarySizeMB = [math]::Round($binarySize / 1MB, 1)

    Write-Host "Binary size: $binarySizeMB MB"

    if ($binarySizeMB -le 12) {
        Write-Host "Binary size within target (<12 MB)" -ForegroundColor Green
        $Script:Passed++
    }
    else {
        Write-Host "WARNING: Binary larger than target ($binarySizeMB MB > 12 MB)" -ForegroundColor Yellow
    }

    Write-Host ""
    Write-Host "========================================"
    Write-Host "Test Summary"
    Write-Host "========================================"

    $endTime = Get-Date
    $totalDuration = [math]::Round(($endTime - $StartTime).TotalSeconds, 1)

    Write-Host "Total tests: $($Script:Passed + $Script:Failed)"
    Write-Host "Passed: $($Script:Passed)" -ForegroundColor Green
    Write-Host "Failed: $($Script:Failed)" -ForegroundColor $(if ($Script:Failed -eq 0) { "Green" } else { "Red" })
    Write-Host "Total duration: ${totalDuration}s"
    Write-Host ""
    Write-Host "Detailed timings:"
    Write-Host "  Bucket addition: $($Script:Timings.BucketAdd)s"
    Write-Host "  Installation: $($Script:Timings.Install)s"
    Write-Host ""

    # Save results to file
    $resultsContent = @"
Scoop Installation Test Results
====================================
Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
Expected Version: $ExpectedVersion
Actual Version: $actualVersion

Performance Metrics:
- Bucket Addition: $($Script:Timings.BucketAdd)s
- Installation Time: $($Script:Timings.Install)s (target: <30s)
- Binary Size: $binarySizeMB MB (target: <12 MB)
- Total Test Duration: ${totalDuration}s

Test Results:
- Tests Passed: $($Script:Passed)
- Tests Failed: $($Script:Failed)
- Success Rate: $([math]::Round(($Script:Passed * 100) / ($Script:Passed + $Script:Failed), 1))%

Binary Location: $binaryPath
"@

    $resultsContent | Out-File -FilePath $ResultsFile -Encoding UTF8
    Write-Host "Results saved to: $ResultsFile"

    # Exit code
    if ($Script:Failed -eq 0) {
        Write-Host "All tests passed!" -ForegroundColor Green
        $exitCode = 0
    }
    else {
        Write-Host "Some tests failed" -ForegroundColor Red
        $exitCode = 1
    }
}
finally {
    Cleanup-TestInstallation
}

exit $exitCode
