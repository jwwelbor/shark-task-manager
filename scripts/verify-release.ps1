#
# Shark Release Verification Script (Windows PowerShell)
#
# This script downloads, verifies, and validates a Shark CLI release.
# It performs integrity checks and basic functionality tests.
#
# Usage:
#   .\verify-release.ps1 -Version <version> [-Platform <platform>] [-Arch <arch>]
#
# Examples:
#   .\verify-release.ps1 -Version "v1.0.0"                      # Windows AMD64 (default)
#   .\verify-release.ps1 -Version "v1.0.0" -Arch "amd64"        # Explicit architecture
#
# Parameters:
#   -Version    Release version (e.g., "v1.0.0") [Required]
#   -Platform   Target platform: windows (default: windows)
#   -Arch       Target architecture: amd64 (default: amd64)
#   -Help       Show this help message
#
# Exit Codes:
#   0 - Success (all checks passed)
#   1 - Invalid arguments or usage
#   2 - Download failed
#   3 - Checksum verification failed
#   4 - Binary validation failed
#

param(
    [Parameter(Mandatory=$false)]
    [string]$Version,

    [Parameter(Mandatory=$false)]
    [string]$Platform = "windows",

    [Parameter(Mandatory=$false)]
    [string]$Arch = "amd64",

    [Parameter(Mandatory=$false)]
    [switch]$Help
)

# Configuration
$Repo = "jwwelbor/shark-task-manager"
$GitHubUrl = "https://github.com/$Repo"
$TempDir = ""

# Print colored messages
function Write-Info {
    param([string]$Message)
    Write-Host "ℹ " -ForegroundColor Blue -NoNewline
    Write-Host $Message
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ " -ForegroundColor Green -NoNewline
    Write-Host $Message
}

function Write-Warning-Custom {
    param([string]$Message)
    Write-Host "⚠ " -ForegroundColor Yellow -NoNewline
    Write-Host $Message
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "✗ " -ForegroundColor Red -NoNewline
    Write-Host $Message
}

function Write-Header {
    param([string]$Message)
    Write-Host ""
    Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Blue
    Write-Host "  $Message" -ForegroundColor Blue
    Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Blue
    Write-Host ""
}

# Show usage information
function Show-Usage {
    $UsageText = @"

Shark Release Verification Script (Windows)

Usage:
  .\verify-release.ps1 -Version <version> [-Platform <platform>] [-Arch <arch>]

Parameters:
  -Version    Release version (e.g., "v1.0.0") [Required]
  -Platform   Target platform: windows (default: windows)
  -Arch       Target architecture: amd64 (default: amd64)
  -Help       Show this help message

Examples:
  .\verify-release.ps1 -Version "v1.0.0"
  .\verify-release.ps1 -Version "v1.0.0" -Arch "amd64"

This script will:
  1. Download the release binary and checksums
  2. Verify SHA256 checksum integrity
  3. Extract and test the binary
  4. Validate basic functionality (--version, --help)
  5. Report success or failure

Exit Codes:
  0 - All checks passed
  1 - Invalid arguments
  2 - Download failed
  3 - Checksum verification failed
  4 - Binary validation failed

"@
    Write-Host $UsageText
    exit 0
}

# Cleanup on exit
function Clean-Up {
    if ($TempDir -and (Test-Path $TempDir)) {
        Write-Info "Cleaning up temporary files..."
        Remove-Item -Path $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# Validate version format
function Test-VersionFormat {
    param([string]$Ver)

    if ($Ver -match '^v\d+\.\d+\.\d+(-.*)?$') {
        return $true
    }
    return $false
}

# Download file
function Get-File {
    param(
        [string]$Url,
        [string]$Output
    )

    try {
        Write-Info "Downloading: $Url"
        $ProgressPreference = 'SilentlyContinue'  # Suppress progress bar for speed
        Invoke-WebRequest -Uri $Url -OutFile $Output -ErrorAction Stop
        $ProgressPreference = 'Continue'
        return $true
    }
    catch {
        Write-Error-Custom "Download failed: $_"
        return $false
    }
}

# Calculate SHA256 hash
function Get-SHA256Hash {
    param([string]$FilePath)

    try {
        $hash = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLower()
        return $hash
    }
    catch {
        Write-Error-Custom "Failed to calculate hash: $_"
        return $null
    }
}

# Extract ZIP file
function Expand-Archive-Custom {
    param(
        [string]$ZipPath,
        [string]$Destination
    )

    try {
        Expand-Archive -Path $ZipPath -DestinationPath $Destination -Force -ErrorAction Stop
        return $true
    }
    catch {
        Write-Error-Custom "Failed to extract archive: $_"
        return $false
    }
}

# Main verification function
function Invoke-Verification {
    param(
        [string]$Version,
        [string]$Platform,
        [string]$Arch
    )

    Write-Header "Shark Release Verification: $Version"

    Write-Info "Target: $Platform/$Arch"
    Write-Info "Repository: $Repo"
    Write-Host ""

    # Create temporary directory
    $script:TempDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "shark-verify-$(Get-Random)") -Force
    Write-Info "Working directory: $TempDir"
    Set-Location $TempDir

    # Construct filename
    $VersionNumber = $Version.TrimStart('v')
    $Filename = "shark_${VersionNumber}_${Platform}_${Arch}.zip"
    $DownloadUrl = "$GitHubUrl/releases/download/$Version/$Filename"
    $ChecksumsUrl = "$GitHubUrl/releases/download/$Version/checksums.txt"

    # Step 1: Download binary
    Write-Header "Step 1: Downloading Release Assets"

    if (-not (Get-File -Url $DownloadUrl -Output $Filename)) {
        Write-Error-Custom "Failed to download binary"
        Write-Info "URL: $DownloadUrl"
        Write-Warning-Custom "Check if the release exists: $GitHubUrl/releases/tag/$Version"
        Clean-Up
        exit 2
    }
    Write-Success "Binary downloaded: $Filename"

    if (-not (Get-File -Url $ChecksumsUrl -Output "checksums.txt")) {
        Write-Error-Custom "Failed to download checksums"
        Write-Info "URL: $ChecksumsUrl"
        Clean-Up
        exit 2
    }
    Write-Success "Checksums downloaded"

    # Step 2: Verify checksum
    Write-Header "Step 2: Verifying SHA256 Checksum"

    # Extract expected checksum from checksums.txt
    $ChecksumsContent = Get-Content "checksums.txt"
    $ChecksumLine = $ChecksumsContent | Select-String -Pattern $Filename

    if (-not $ChecksumLine) {
        Write-Error-Custom "Checksum not found in checksums.txt for: $Filename"
        Write-Info "Contents of checksums.txt:"
        Write-Host $ChecksumsContent
        Clean-Up
        exit 3
    }

    $ExpectedChecksum = ($ChecksumLine.ToString().Split()[0]).ToLower()
    Write-Info "Expected checksum: $ExpectedChecksum"

    # Calculate actual checksum
    $ActualChecksum = Get-SHA256Hash -FilePath $Filename

    if (-not $ActualChecksum) {
        Write-Error-Custom "Failed to calculate checksum"
        Clean-Up
        exit 3
    }

    Write-Info "Actual checksum:   $ActualChecksum"

    # Compare checksums
    if ($ExpectedChecksum -eq $ActualChecksum) {
        Write-Success "Checksum verification PASSED"
        Write-Info "The binary is authentic and has not been tampered with."
    }
    else {
        Write-Error-Custom "Checksum verification FAILED"
        Write-Error-Custom "Expected: $ExpectedChecksum"
        Write-Error-Custom "Actual:   $ActualChecksum"
        Write-Warning-Custom "DO NOT use this binary!"
        Clean-Up
        exit 3
    }

    # Step 3: Extract binary
    Write-Header "Step 3: Extracting Binary"

    Write-Info "Extracting: $Filename"
    if (-not (Expand-Archive-Custom -ZipPath $Filename -Destination ".")) {
        Write-Error-Custom "Failed to extract archive"
        Clean-Up
        exit 4
    }
    Write-Success "Binary extracted"

    # Verify binary exists
    $BinaryPath = Join-Path $TempDir "shark.exe"
    if (-not (Test-Path $BinaryPath)) {
        Write-Error-Custom "Binary 'shark.exe' not found in archive"
        Write-Info "Archive contents:"
        Get-ChildItem
        Clean-Up
        exit 4
    }
    Write-Success "Binary found: shark.exe"

    # Step 4: Validate binary
    Write-Header "Step 4: Validating Binary Functionality"

    # Test --version
    Write-Info "Running: .\shark.exe --version"
    try {
        $VersionOutput = & $BinaryPath --version 2>&1 | Out-String
        Write-Success "Version command executed successfully"
        Write-Info "Output: $($VersionOutput.Trim())"

        # Verify version matches expected
        if ($VersionOutput -match $VersionNumber) {
            Write-Success "Version matches release: $VersionNumber"
        }
        else {
            Write-Warning-Custom "Version string doesn't match expected version"
            Write-Info "Expected to find: $VersionNumber"
            Write-Info "Actual output: $VersionOutput"
        }
    }
    catch {
        Write-Error-Custom "Failed to run --version command: $_"
        Clean-Up
        exit 4
    }

    # Test --help
    Write-Info "Running: .\shark.exe --help"
    try {
        $HelpOutput = & $BinaryPath --help 2>&1 | Out-String
        Write-Success "Help command executed successfully"

        # Verify help output contains expected content
        if ($HelpOutput -match "shark") {
            Write-Success "Help output looks valid"
        }
        else {
            Write-Warning-Custom "Help output may be incomplete"
        }
    }
    catch {
        Write-Error-Custom "Failed to run --help command: $_"
        Clean-Up
        exit 4
    }

    # Step 5: Summary
    Write-Header "Verification Summary"

    Write-Success "All checks passed!"
    Write-Host ""
    Write-Info "Release Information:"
    Write-Host "  Version:    $Version"
    Write-Host "  Platform:   $Platform"
    Write-Host "  Arch:       $Arch"
    Write-Host "  Filename:   $Filename"
    Write-Host "  Checksum:   $ExpectedChecksum"
    Write-Host ""
    Write-Info "The binary is verified and ready for use."
    Write-Info "Binary location: $BinaryPath"
    Write-Host ""
    Write-Info "To install system-wide:"
    Write-Host "  1. Copy shark.exe to a directory in your PATH"
    Write-Host "  2. Or add the directory to your PATH environment variable"
    Write-Host ""
    Write-Info "Example installation:"
    Write-Host "  Copy-Item '$BinaryPath' -Destination 'C:\Program Files\Shark\shark.exe'"
    Write-Host ""
}

# Main entry point
try {
    # Show usage if requested
    if ($Help) {
        Show-Usage
    }

    # Validate required parameters
    if (-not $Version) {
        Write-Error-Custom "Missing required parameter: -Version"
        Write-Host ""
        Show-Usage
    }

    # Validate version format
    if (-not (Test-VersionFormat -Ver $Version)) {
        Write-Error-Custom "Invalid version format: $Version"
        Write-Info "Expected format: vX.Y.Z (e.g., v1.0.0)"
        exit 1
    }

    # Validate platform
    if ($Platform -ne "windows") {
        Write-Error-Custom "Invalid platform: $Platform"
        Write-Info "This script only supports Windows. For Linux/macOS, use verify-release.sh"
        exit 1
    }

    # Validate architecture
    if ($Arch -ne "amd64") {
        Write-Error-Custom "Invalid architecture: $Arch"
        Write-Info "Currently only amd64 is supported for Windows"
        exit 1
    }

    # Run verification
    Invoke-Verification -Version $Version -Platform $Platform -Arch $Arch
}
catch {
    Write-Error-Custom "Unexpected error: $_"
    Write-Error-Custom $_.ScriptStackTrace
    Clean-Up
    exit 4
}
finally {
    # Cleanup is handled by try-catch, but ensure it runs
    if ($TempDir -and (Test-Path $TempDir)) {
        Clean-Up
    }
}
