# Test Criteria: Distribution & Release Automation

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F08-distribution-release
**Date**: 2025-12-17
**Author**: tdd-agent

## Executive Summary

This document defines comprehensive test criteria for the automated release and distribution system of the Shark CLI tool. Tests cover GoReleaser configuration, GitHub Actions workflows, multi-platform builds, package manager distribution, security controls, and performance requirements.

**Testing Philosophy**: Validate every distribution channel, platform, and security control before production release.

---

## 1. Test Organization

### 1.1 Test Levels

```
┌─────────────────────────────────────────────────────────┐
│ Level 1: Configuration Validation                       │
│ - GoReleaser config syntax                              │
│ - GitHub Actions workflow syntax                        │
│ - YAML linting                                          │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ Level 2: Local Build Testing                            │
│ - Snapshot builds                                       │
│ - Binary verification                                   │
│ - Checksum generation                                   │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ Level 3: CI/CD Integration Testing                      │
│ - GitHub Actions workflow execution                     │
│ - Draft release creation                                │
│ - Asset upload verification                             │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ Level 4: Distribution Testing                           │
│ - Homebrew tap installation                             │
│ - Scoop bucket installation                             │
│ - Manual download verification                          │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│ Level 5: Security & Performance Testing                 │
│ - Checksum verification                                 │
│ - Token security                                        │
│ - Build time validation                                 │
└─────────────────────────────────────────────────────────┘
```

### 1.2 Test Categories

| Category | Test Count | Automation | Priority |
|----------|-----------|------------|----------|
| Configuration Validation | 12 | Automated | Critical |
| Build Testing | 18 | Automated + Manual | Critical |
| Distribution Testing | 15 | Manual | High |
| Security Testing | 10 | Automated + Manual | Critical |
| Performance Testing | 8 | Automated | High |
| Documentation Testing | 6 | Manual | Medium |
| **Total** | **69 tests** | **Mixed** | **-** |

---

## 2. Configuration Validation Tests

### TC-001: GoReleaser Configuration Syntax

**Given** I have created `.goreleaser.yml` in repository root
**When** I run `goreleaser check`
**Then** the configuration is valid with no errors
**And** no warnings are displayed

**Automation**: CLI command
```bash
goreleaser check
echo $?  # Should be 0 (success)
```

**Expected Output**:
```
✓ config is valid
```

---

### TC-002: GoReleaser Build Targets

**Given** `.goreleaser.yml` is configured
**When** I inspect the `builds` section
**Then** it includes all required targets:
- Linux amd64
- Linux arm64
- macOS amd64 (darwin/amd64)
- macOS arm64 (darwin/arm64)
- Windows amd64

**Automation**: YAML parsing
```bash
yq '.builds[0].goos' .goreleaser.yml | grep -E '(linux|darwin|windows)'
yq '.builds[0].goarch' .goreleaser.yml | grep -E '(amd64|arm64)'
```

---

### TC-003: Binary Naming Configuration

**Given** `.goreleaser.yml` is configured
**When** I inspect the `builds` section
**Then** binary name is `shark` (or `shark.exe` for Windows)
**And** no platform-specific suffixes in binary name

**Automation**: YAML parsing
```bash
yq '.builds[0].binary' .goreleaser.yml | grep '^shark$'
```

---

### TC-004: Archive Format Configuration

**Given** `.goreleaser.yml` is configured
**When** I inspect the `archives` section
**Then** Linux/macOS archives are `.tar.gz` format
**And** Windows archives are `.zip` format
**And** archives include README.md and LICENSE

**Automation**: YAML parsing
```bash
yq '.archives[0].format_overrides[] | select(.goos == "windows") | .format' .goreleaser.yml | grep '^zip$'
yq '.archives[0].files[]' .goreleaser.yml | grep -E '(README|LICENSE)'
```

---

### TC-005: Checksum Configuration

**Given** `.goreleaser.yml` is configured
**When** I inspect the `checksum` section
**Then** checksum file is named `checksums.txt`
**And** algorithm is `sha256`

**Automation**: YAML parsing
```bash
yq '.checksum.name_template' .goreleaser.yml | grep 'checksums.txt'
yq '.checksum.algorithm' .goreleaser.yml | grep 'sha256'
```

---

### TC-006: Version Embedding Configuration

**Given** `.goreleaser.yml` is configured
**When** I inspect the `ldflags` section
**Then** it includes `-X main.Version={{.Version}}`
**And** it includes `-s -w` for binary stripping

**Automation**: YAML parsing
```bash
yq '.builds[0].ldflags[]' .goreleaser.yml | grep -- '-X main.Version='
yq '.builds[0].ldflags[]' .goreleaser.yml | grep -- '-s -w'
```

---

### TC-007: GitHub Actions Workflow Syntax

**Given** I have created `.github/workflows/release.yml`
**When** I validate the workflow syntax
**Then** the YAML is valid with no syntax errors

**Automation**: GitHub CLI
```bash
gh workflow view release --yaml > /dev/null
echo $?  # Should be 0 (success)
```

---

### TC-008: Workflow Trigger Configuration

**Given** `.github/workflows/release.yml` exists
**When** I inspect the trigger configuration
**Then** it triggers on tag pushes matching `v*` pattern

**Automation**: YAML parsing
```bash
yq '.on.push.tags[]' .github/workflows/release.yml | grep '^v\*$'
```

---

### TC-009: Workflow Permissions Configuration

**Given** `.github/workflows/release.yml` exists
**When** I inspect the permissions
**Then** it includes `contents: write` permission

**Automation**: YAML parsing
```bash
yq '.jobs.release.permissions.contents' .github/workflows/release.yml | grep '^write$'
```

---

### TC-010: Homebrew Tap Configuration

**Given** `.goreleaser.yml` is configured
**When** I inspect the `brews` section
**Then** it references `homebrew-shark` repository
**And** it includes formula name, description, and test block

**Automation**: YAML parsing
```bash
yq '.brews[0].repository.name' .goreleaser.yml | grep 'homebrew-shark'
yq '.brews[0].test' .goreleaser.yml | grep 'shark'
```

---

### TC-011: Scoop Bucket Configuration

**Given** `.goreleaser.yml` is configured
**When** I inspect the `scoops` section
**Then** it references `scoop-shark` repository
**And** it includes manifest name and description

**Automation**: YAML parsing
```bash
yq '.scoops[0].repository.name' .goreleaser.yml | grep 'scoop-shark'
yq '.scoops[0].description' .goreleaser.yml
```

---

### TC-012: Draft Release Configuration

**Given** `.goreleaser.yml` is configured
**When** I inspect the `release` section
**Then** `draft: true` is set
**And** `prerelease: auto` is configured

**Automation**: YAML parsing
```bash
yq '.release.draft' .goreleaser.yml | grep '^true$'
yq '.release.prerelease' .goreleaser.yml | grep '^auto$'
```

---

## 3. Build Testing

### TC-013: Local Snapshot Build

**Given** GoReleaser is installed and configured
**When** I run `goreleaser build --snapshot --clean`
**Then** the build completes successfully
**And** builds all 5 platform binaries
**And** no errors are displayed

**Automation**: CLI command
```bash
rm -rf dist/
time goreleaser build --snapshot --clean
echo $?  # Should be 0

# Verify outputs
ls -1 dist/ | grep -E '(linux|darwin|windows)_(amd64|arm64)' | wc -l
# Should be 5
```

**Expected Duration**: <6 minutes

---

### TC-014: Binary Size Validation

**Given** Snapshot build is complete
**When** I check the size of compressed archives
**Then** each archive is <4 MB (.tar.gz) or <5 MB (.zip)

**Automation**: CLI command
```bash
for file in dist/*.tar.gz dist/*.zip; do
  size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file")
  size_mb=$((size / 1024 / 1024))
  echo "$(basename $file): ${size_mb} MB"

  if [ $size_mb -gt 5 ]; then
    echo "ERROR: Archive size exceeds limit"
    exit 1
  fi
done
```

---

### TC-015: Binary Execution Test (Linux)

**Given** Snapshot build is complete
**When** I execute `./dist/shark_linux_amd64/shark --version`
**Then** the binary runs without errors
**And** outputs a version string

**Automation**: CLI command
```bash
./dist/shark_linux_amd64/shark --version
echo $?  # Should be 0
```

---

### TC-016: Binary Execution Test (macOS)

**Given** Snapshot build is complete
**When** I execute `./dist/shark_darwin_arm64/shark --version`
**Then** the binary runs without errors (or Rosetta 2 on Intel Mac)
**And** outputs a version string

**Automation**: CLI command (on macOS)
```bash
./dist/shark_darwin_arm64/shark --version
echo $?  # Should be 0
```

---

### TC-017: Binary Execution Test (Windows)

**Given** Snapshot build is complete
**When** I execute `dist/shark_windows_amd64/shark.exe --version`
**Then** the binary runs without errors
**And** outputs a version string

**Automation**: CLI command (on Windows)
```powershell
.\dist\shark_windows_amd64\shark.exe --version
$LASTEXITCODE  # Should be 0
```

---

### TC-018: Version Embedding Verification

**Given** Snapshot build is complete
**When** I run `shark --version` on any platform
**Then** the output includes a version string (not "dev" or "unknown")

**Automation**: CLI command
```bash
version=$(./dist/shark_linux_amd64/shark --version)
if echo "$version" | grep -E '(dev|unknown)'; then
  echo "ERROR: Version not embedded correctly"
  exit 1
fi
```

---

### TC-019: Checksum File Generation

**Given** Snapshot build is complete
**When** I inspect `dist/checksums.txt`
**Then** the file exists
**And** contains SHA256 checksums for all 5 archives

**Automation**: CLI command
```bash
test -f dist/checksums.txt
grep -c 'shark_.*\(tar.gz\|zip\)' dist/checksums.txt
# Should be 5
```

---

### TC-020: Checksum Verification

**Given** Snapshot build is complete
**When** I run `sha256sum -c dist/checksums.txt`
**Then** all checksums verify successfully
**And** no mismatches are reported

**Automation**: CLI command
```bash
cd dist
sha256sum -c checksums.txt
echo $?  # Should be 0
```

---

### TC-021: Archive Contents Verification

**Given** Snapshot build is complete
**When** I extract `dist/shark_<platform>.tar.gz`
**Then** the archive contains:
- `shark` binary (or `shark.exe`)
- `README.md`
- `LICENSE`

**Automation**: CLI command
```bash
tar -tzf dist/shark_*_linux_amd64.tar.gz | grep -E '(shark|README|LICENSE)' | wc -l
# Should be 3
```

---

### TC-022: Cross-Platform Binary Test (All Platforms)

**Given** Snapshot build is complete
**When** I test all 5 binaries
**Then** each binary:
- Executes without errors
- Displays correct version
- Has expected file permissions (executable)

**Automation**: Test script
```bash
#!/bin/bash
platforms=(
  "linux_amd64"
  "linux_arm64"
  "darwin_amd64"
  "darwin_arm64"
  "windows_amd64"
)

for platform in "${platforms[@]}"; do
  echo "Testing $platform..."

  if [[ "$platform" == *"windows"* ]]; then
    binary="dist/shark_${platform}/shark.exe"
  else
    binary="dist/shark_${platform}/shark"
  fi

  if [ ! -f "$binary" ]; then
    echo "ERROR: Binary not found: $binary"
    exit 1
  fi

  # Skip execution test for non-native platforms (would require emulation)
  echo "✓ $platform binary exists"
done
```

---

### TC-023: Build Performance Test

**Given** GoReleaser is configured
**When** I run a snapshot build
**Then** the build completes in <6 minutes

**Automation**: CLI command with timing
```bash
START=$(date +%s)
goreleaser build --snapshot --clean
END=$(date +%s)
DURATION=$((END - START))

echo "Build duration: ${DURATION}s"

if [ $DURATION -gt 360 ]; then
  echo "ERROR: Build time exceeded 6 minutes"
  exit 1
fi
```

---

### TC-024: CGO Compilation Test

**Given** Project uses `mattn/go-sqlite3` (CGO dependency)
**When** I build with `CGO_ENABLED=1`
**Then** the build succeeds for all platforms
**And** binaries link correctly to SQLite

**Automation**: CLI command
```bash
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o test-binary ./cmd/shark
ldd test-binary | grep -E '(sqlite|libc)'
rm test-binary
```

---

### TC-025: Binary Stripping Verification

**Given** Binaries are built with `-s -w` ldflags
**When** I inspect binary size
**Then** binary is smaller than default build (no debug symbols)

**Automation**: Comparison build
```bash
# Build with default flags
go build -o shark-default ./cmd/shark
SIZE_DEFAULT=$(stat -f%z shark-default 2>/dev/null || stat -c%s shark-default)

# Build with -s -w
go build -ldflags="-s -w" -o shark-stripped ./cmd/shark
SIZE_STRIPPED=$(stat -f%z shark-stripped 2>/dev/null || stat -c%s shark-stripped)

echo "Default: $((SIZE_DEFAULT / 1024 / 1024)) MB"
echo "Stripped: $((SIZE_STRIPPED / 1024 / 1024)) MB"

if [ $SIZE_STRIPPED -ge $SIZE_DEFAULT ]; then
  echo "ERROR: Stripping did not reduce binary size"
  exit 1
fi

rm shark-default shark-stripped
```

---

### TC-026: Archive Format Verification

**Given** Snapshot build is complete
**When** I inspect archive formats
**Then** Linux archives are `.tar.gz`
**And** macOS archives are `.tar.gz`
**And** Windows archives are `.zip`

**Automation**: CLI command
```bash
ls dist/*.tar.gz | grep -E '(linux|darwin)' | wc -l  # Should be 4
ls dist/*.zip | grep 'windows' | wc -l  # Should be 1
```

---

### TC-027: Build Artifact Cleanup

**Given** I run `goreleaser build --clean`
**When** the build completes
**Then** previous `dist/` contents are removed
**And** only new artifacts exist

**Automation**: CLI command
```bash
# Create dummy file
mkdir -p dist
touch dist/old-file.txt

# Run build with --clean
goreleaser build --snapshot --clean

# Verify old file removed
if [ -f dist/old-file.txt ]; then
  echo "ERROR: Old artifacts not cleaned"
  exit 1
fi
```

---

### TC-028: Release Notes Template

**Given** `.goreleaser.yml` includes release notes configuration
**When** GoReleaser generates release notes
**Then** notes include version number
**And** notes include commit history
**And** notes include installation instructions

**Automation**: Manual inspection (GoReleaser preview)
```bash
goreleaser release --snapshot --skip-publish --skip-sign
cat dist/RELEASE_NOTES.md
```

---

### TC-029: Parallel Build Verification

**Given** GoReleaser builds multiple platforms
**When** I monitor build logs
**Then** builds run in parallel (not sequential)
**And** total time is less than sum of individual builds

**Automation**: Log analysis (manual)
- Review GitHub Actions logs
- Look for parallel execution indicators

---

### TC-030: Module Cache Utilization

**Given** GitHub Actions caches Go modules
**When** I run workflow twice
**Then** second run reuses cached modules
**And** second run is faster than first run

**Automation**: GitHub Actions cache check
```yaml
- uses: actions/setup-go@v5
  with:
    cache: true
# Check logs for "Cache hit: true"
```

---

## 4. CI/CD Integration Tests

### TC-031: GitHub Actions Workflow Trigger

**Given** I create a Git tag matching `v*` pattern
**When** I push the tag to GitHub
**Then** the release workflow triggers automatically
**And** workflow appears in Actions tab

**Automation**: GitHub CLI
```bash
git tag v0.0.1-test
git push origin v0.0.1-test

# Wait 10 seconds
sleep 10

# Check workflow status
gh run list --workflow=release.yml --limit=1
```

---

### TC-032: Test Execution Gate

**Given** Workflow is running
**When** Tests execute (`go test ./...`)
**Then** tests must pass before GoReleaser runs
**And** workflow fails if tests fail

**Automation**: Inject test failure and verify
```bash
# Temporarily break a test
# Push tag
# Verify workflow fails at test step
```

---

### TC-033: GoReleaser Execution in CI

**Given** Tests pass in workflow
**When** GoReleaser executes
**Then** GoReleaser builds all platforms successfully
**And** no errors in workflow logs

**Automation**: GitHub CLI
```bash
gh run view <run-id> --log | grep "GoReleaser"
gh run view <run-id> --log | grep -i error | wc -l  # Should be 0
```

---

### TC-034: Draft Release Creation

**Given** GoReleaser completes successfully
**When** Workflow finishes
**Then** a draft release is created on GitHub
**And** release is not published (still draft)

**Automation**: GitHub CLI
```bash
gh release list | grep "v0.0.1-test" | grep "Draft"
```

---

### TC-035: GitHub Release Assets Upload

**Given** Draft release is created
**When** I inspect release assets
**Then** all 6 files are uploaded:
- 5 platform archives
- 1 checksums.txt

**Automation**: GitHub CLI
```bash
gh release view v0.0.1-test --json assets --jq '.assets[].name' | wc -l
# Should be 6
```

---

### TC-036: Release Notes Generation

**Given** Draft release is created
**When** I view release notes
**Then** notes include version number
**And** notes include commit messages since last tag
**And** notes include installation instructions

**Automation**: GitHub CLI
```bash
gh release view v0.0.1-test --json body --jq '.body' | grep "v0.0.1-test"
gh release view v0.0.1-test --json body --jq '.body' | grep -i "installation"
```

---

### TC-037: Workflow Performance

**Given** Workflow is triggered
**When** Workflow completes
**Then** total duration is <12 minutes

**Automation**: GitHub CLI
```bash
gh run view <run-id> --json conclusion,updatedAt,createdAt
# Calculate duration from timestamps
```

---

### TC-038: Homebrew Tap Update

**Given** GoReleaser completes
**When** I check `homebrew-shark` repository
**Then** `Formula/shark.rb` is created/updated
**And** commit is made by GoReleaser

**Automation**: Git command
```bash
git clone https://github.com/jwwelbor/homebrew-shark
cd homebrew-shark
git log -1 --oneline | grep "Brew formula update for shark"
test -f Formula/shark.rb
```

---

### TC-039: Scoop Bucket Update

**Given** GoReleaser completes
**When** I check `scoop-shark` repository
**Then** `bucket/shark.json` is created/updated
**And** commit is made by GoReleaser

**Automation**: Git command
```bash
git clone https://github.com/jwwelbor/scoop-shark
cd scoop-shark
git log -1 --oneline | grep "Scoop manifest update for shark"
test -f bucket/shark.json
```

---

### TC-040: GitHub Token Authentication

**Given** Workflow uses `GITHUB_TOKEN`
**When** GoReleaser creates release
**Then** authentication succeeds
**And** no permission errors in logs

**Automation**: GitHub CLI (check logs)
```bash
gh run view <run-id> --log | grep -i "permission denied"
echo $?  # Should be 1 (no matches found)
```

---

### TC-041: Workflow Idempotency

**Given** A release workflow completed successfully
**When** I re-run the same workflow
**Then** it produces identical artifacts
**And** checksums match

**Automation**: Manual re-run and comparison
```bash
# Download artifacts from run 1
wget https://github.com/.../shark_v0.0.1-test_linux_amd64.tar.gz -O run1.tar.gz

# Re-run workflow (same tag)
# Download artifacts from run 2
wget https://github.com/.../shark_v0.0.1-test_linux_amd64.tar.gz -O run2.tar.gz

# Compare checksums
sha256sum run1.tar.gz run2.tar.gz
# Checksums should match
```

---

### TC-042: Workflow Failure Handling

**Given** GoReleaser encounters an error
**When** Workflow fails
**Then** no partial release is created
**And** workflow status is "failed"

**Automation**: Inject build error and verify
```bash
# Modify .goreleaser.yml with invalid config
# Push tag
# Verify workflow fails cleanly
gh run view <run-id> --json conclusion --jq '.conclusion'
# Should be "failure"
```

---

### TC-043: Parallel Package Manager Updates

**Given** GoReleaser completes builds
**When** Package manager updates execute
**Then** Homebrew and Scoop updates run concurrently
**And** total update time is <2 minutes

**Automation**: Log analysis (manual)
- Review GitHub Actions logs
- Look for parallel execution timestamps

---

### TC-044: Secret Masking in Logs

**Given** Workflow uses `HOMEBREW_TAP_TOKEN` and `SCOOP_BUCKET_TOKEN`
**When** I inspect workflow logs
**Then** token values are masked (shown as ***)
**And** no plaintext secrets are visible

**Automation**: GitHub CLI
```bash
gh run view <run-id> --log | grep -E '(HOMEBREW_TAP_TOKEN|SCOOP_BUCKET_TOKEN)'
# Should show *** instead of actual token
```

---

### TC-045: Workflow Artifact Retention

**Given** Workflow completes
**When** I check GitHub Actions artifacts
**Then** build artifacts are retained for 90 days

**Automation**: GitHub UI (manual check)
- Visit Actions → Workflow run
- Check artifact retention policy

---

## 5. Distribution Testing

### TC-046: Homebrew Tap Addition

**Given** Homebrew is installed
**When** I run `brew tap jwwelbor/shark`
**Then** the tap is added successfully
**And** no errors are displayed

**Automation**: CLI command (macOS/Linux)
```bash
brew tap jwwelbor/shark
brew tap | grep 'jwwelbor/shark'
```

---

### TC-047: Homebrew Installation (macOS Intel)

**Given** Tap is added
**When** I run `brew install shark` on macOS Intel
**Then** Shark CLI installs successfully
**And** binary is installed to `/usr/local/bin/shark`

**Automation**: CLI command (macOS Intel)
```bash
brew install shark
which shark | grep '/usr/local/bin/shark'
shark --version
```

---

### TC-048: Homebrew Installation (macOS Apple Silicon)

**Given** Tap is added
**When** I run `brew install shark` on macOS Apple Silicon
**Then** Shark CLI installs successfully
**And** binary is installed to `/opt/homebrew/bin/shark`

**Automation**: CLI command (macOS ARM64)
```bash
brew install shark
which shark | grep '/opt/homebrew/bin/shark'
shark --version
```

---

### TC-049: Homebrew Installation (Linux)

**Given** Homebrew for Linux is installed
**When** I run `brew install shark`
**Then** Shark CLI installs successfully
**And** binary is accessible in PATH

**Automation**: CLI command (Linux with Homebrew)
```bash
brew install shark
which shark
shark --version
```

---

### TC-050: Homebrew Version Verification

**Given** Shark is installed via Homebrew
**When** I run `shark --version`
**Then** the version matches the release tag (e.g., v1.0.0)

**Automation**: CLI command
```bash
version=$(shark --version | awk '{print $NF}')
if [ "$version" != "v1.0.0" ]; then
  echo "ERROR: Version mismatch"
  exit 1
fi
```

---

### TC-051: Scoop Bucket Addition

**Given** Scoop is installed (Windows)
**When** I run `scoop bucket add shark https://github.com/jwwelbor/scoop-shark`
**Then** the bucket is added successfully
**And** no errors are displayed

**Automation**: PowerShell (Windows)
```powershell
scoop bucket add shark https://github.com/jwwelbor/scoop-shark
scoop bucket list | Select-String 'shark'
```

---

### TC-052: Scoop Installation (Windows)

**Given** Bucket is added
**When** I run `scoop install shark`
**Then** Shark CLI installs successfully
**And** binary is installed to `~/scoop/apps/shark/current/shark.exe`

**Automation**: PowerShell (Windows)
```powershell
scoop install shark
Test-Path ~\scoop\apps\shark\current\shark.exe
shark --version
```

---

### TC-053: Scoop Version Verification

**Given** Shark is installed via Scoop
**When** I run `shark --version`
**Then** the version matches the release tag

**Automation**: PowerShell (Windows)
```powershell
$version = (shark --version) -replace '.*version ', ''
if ($version -ne 'v1.0.0') {
  Write-Error "Version mismatch"
  exit 1
}
```

---

### TC-054: Manual Download (Linux)

**Given** Release is published on GitHub
**When** I download `shark_1.0.0_linux_amd64.tar.gz`
**Then** the download completes successfully
**And** file size is 3-4 MB

**Automation**: CLI command
```bash
wget https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_linux_amd64.tar.gz

size=$(stat -c%s shark_1.0.0_linux_amd64.tar.gz)
size_mb=$((size / 1024 / 1024))
if [ $size_mb -lt 2 ] || [ $size_mb -gt 5 ]; then
  echo "ERROR: Unexpected file size: ${size_mb} MB"
  exit 1
fi
```

---

### TC-055: Manual Checksum Verification

**Given** I downloaded a binary archive
**When** I download `checksums.txt` and verify
**Then** the checksum matches

**Automation**: CLI command
```bash
wget https://github.com/.../checksums.txt
sha256sum -c checksums.txt --ignore-missing
# Should output: shark_1.0.0_linux_amd64.tar.gz: OK
```

---

### TC-056: Manual Installation and Execution

**Given** I verified the checksum
**When** I extract and execute the binary
**Then** `shark --version` runs successfully

**Automation**: CLI command
```bash
tar -xzf shark_1.0.0_linux_amd64.tar.gz
./shark --version
echo $?  # Should be 0
```

---

### TC-057: Package Manager Upgrade (Homebrew)

**Given** An older version is installed
**When** A new release is published and I run `brew upgrade shark`
**Then** Shark upgrades to the new version

**Automation**: CLI command (macOS/Linux)
```bash
brew upgrade shark
new_version=$(shark --version | awk '{print $NF}')
# Verify new version is greater than old version
```

---

### TC-058: Package Manager Upgrade (Scoop)

**Given** An older version is installed
**When** A new release is published and I run `scoop update shark`
**Then** Shark upgrades to the new version

**Automation**: PowerShell (Windows)
```powershell
scoop update shark
$newVersion = (shark --version) -replace '.*version ', ''
# Verify version updated
```

---

### TC-059: Installation on Fresh System

**Given** A system with no Go toolchain installed
**When** I install Shark via package manager
**Then** Shark installs and runs without requiring Go

**Automation**: Docker container test
```bash
docker run --rm -it ubuntu:latest bash -c "
  apt-get update && apt-get install -y curl
  curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh | bash
  brew tap jwwelbor/shark
  brew install shark
  shark --version
"
```

---

### TC-060: Multi-Platform Installation Verification

**Given** Releases are published for all platforms
**When** I test installation on macOS, Linux, and Windows
**Then** all installations succeed
**And** `shark --version` works on all platforms

**Automation**: Multi-platform CI test
- GitHub Actions matrix strategy
- Test on ubuntu-latest, macos-latest, windows-latest

---

## 6. Security Testing

### TC-061: Checksum File Integrity

**Given** Release includes `checksums.txt`
**When** I download and inspect the file
**Then** it contains exactly 5 checksums (one per platform)
**And** each line has format: `<hash> <filename>`

**Automation**: CLI command
```bash
wget https://github.com/.../checksums.txt
wc -l checksums.txt  # Should be 5
grep -E '^[0-9a-f]{64} shark_' checksums.txt | wc -l  # Should be 5
```

---

### TC-062: Checksum Mismatch Detection

**Given** I intentionally corrupt a downloaded archive
**When** I run checksum verification
**Then** verification fails with clear error message

**Automation**: CLI command
```bash
wget https://github.com/.../shark_1.0.0_linux_amd64.tar.gz
wget https://github.com/.../checksums.txt

# Corrupt file
echo "corruption" >> shark_1.0.0_linux_amd64.tar.gz

# Verify (should fail)
sha256sum -c checksums.txt --ignore-missing
echo $?  # Should be non-zero (failure)
```

---

### TC-063: HTTPS Download Enforcement

**Given** All download URLs use HTTPS
**When** I attempt HTTP download (if not auto-redirected)
**Then** connection fails or redirects to HTTPS

**Automation**: CLI command
```bash
curl -I http://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_linux_amd64.tar.gz
# Should redirect to HTTPS or fail
```

---

### TC-064: GitHub Token Scope Verification

**Given** Workflow uses PATs for package managers
**When** I inspect token permissions
**Then** tokens have minimal scope (contents:write only)
**And** tokens are repository-scoped (not account-wide)

**Automation**: Manual verification
- Review PAT settings in GitHub UI
- Confirm scope limitations

---

### TC-065: Secret Exposure Prevention

**Given** Workflow logs are public
**When** I search logs for token values
**Then** no plaintext tokens are visible
**And** all secrets are masked as `***`

**Automation**: GitHub CLI
```bash
gh run view <run-id> --log > workflow.log
grep -E '[a-z]{3}_[A-Za-z0-9]{40}' workflow.log
# Should find no GitHub tokens
```

---

### TC-066: Binary Signing (Future)

**Given** Binaries are built
**When** I inspect binary metadata
**Then** binaries are unsigned (expected for F08)
**And** documentation notes future code signing

**Automation**: Manual check
```bash
codesign -dv ./shark  # macOS (should fail - not signed)
```

**Note**: Code signing is out of scope for F08, but test documents the gap.

---

### TC-067: Supply Chain Attack Detection

**Given** GoReleaser action is used in workflow
**When** I inspect action version
**Then** action is pinned to specific version (not `@latest`)

**Automation**: YAML parsing
```bash
yq '.jobs.release.steps[] | select(.uses | contains("goreleaser")) | .uses' .github/workflows/release.yml
# Should not end with @latest or @v5 (should be @v5.0.0 or SHA)
```

---

### TC-068: Dependency Vulnerability Scanning

**Given** Project has Go dependencies
**When** I run vulnerability scanner
**Then** no high/critical vulnerabilities are found

**Automation**: CLI command
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
echo $?  # Should be 0 (no vulnerabilities)
```

---

### TC-069: Archive Extraction Safety

**Given** I download a release archive
**When** I extract the archive
**Then** extraction does not write files outside of target directory (no path traversal)

**Automation**: CLI command
```bash
mkdir test-extract
tar -xzf shark_1.0.0_linux_amd64.tar.gz -C test-extract
find test-extract -type f | grep -E '^\.\./|^/' && echo "ERROR: Path traversal detected" || echo "OK"
```

---

### TC-070: Package Manager Checksum Verification

**Given** Homebrew formula includes checksums
**When** Homebrew installs Shark
**Then** Homebrew automatically verifies checksums before installation

**Automation**: Homebrew install (verbose)
```bash
brew install shark --verbose 2>&1 | grep -i 'checksum\|sha256'
# Should show checksum verification step
```

---

## 7. Performance Testing

### TC-071: Build Time Performance

**Given** GoReleaser builds all platforms
**When** I measure build duration
**Then** total build time is <5 minutes

**Automation**: Timed build
```bash
START=$(date +%s)
goreleaser build --snapshot --clean
END=$(date +%s)
DURATION=$((END - START))

if [ $DURATION -gt 300 ]; then
  echo "ERROR: Build time exceeded 5 minutes (${DURATION}s)"
  exit 1
fi
```

---

### TC-072: Workflow Time Performance

**Given** Workflow is triggered by tag
**When** I measure total workflow duration
**Then** workflow completes in <10 minutes

**Automation**: GitHub CLI
```bash
gh run view <run-id> --json durationMs --jq '.durationMs / 1000 / 60'
# Should be <10
```

---

### TC-073: Binary Size Performance

**Given** Binaries are built
**When** I measure binary sizes
**Then** uncompressed binaries are <12 MB
**And** compressed archives are <5 MB

**Automation**: CLI command
```bash
for binary in dist/shark_*/shark*; do
  size=$(stat -f%z "$binary" 2>/dev/null || stat -c%s "$binary")
  size_mb=$((size / 1024 / 1024))
  echo "$(basename $(dirname $binary)): ${size_mb} MB"

  if [ $size_mb -gt 12 ]; then
    echo "ERROR: Binary size exceeds 12 MB"
    exit 1
  fi
done
```

---

### TC-074: CLI Startup Time Performance

**Given** Shark CLI is installed
**When** I measure `shark --version` execution time
**Then** startup time is <50 ms

**Automation**: CLI command with timing
```bash
for i in {1..10}; do
  time shark --version 2>&1 | grep '^real'
done | awk '{sum+=$2} END {print "Average:", sum/10, "ms"}'
# Should be <50ms
```

---

### TC-075: Homebrew Installation Performance

**Given** Homebrew tap is configured
**When** I measure `brew install shark` duration
**Then** installation completes in <30 seconds

**Automation**: CLI command with timing
```bash
time brew install shark
# Check "real" time in output
```

---

### TC-076: Scoop Installation Performance

**Given** Scoop bucket is configured
**When** I measure `scoop install shark` duration
**Then** installation completes in <30 seconds

**Automation**: PowerShell with timing
```powershell
Measure-Command { scoop install shark }
# Check TotalSeconds property
```

---

### TC-077: Download Speed Test

**Given** Binary archives are hosted on GitHub Releases
**When** I download an archive
**Then** download completes in <10 seconds on 10 Mbps connection

**Automation**: CLI command with timing
```bash
time wget https://github.com/.../shark_1.0.0_linux_amd64.tar.gz
# Check "real" time (should be <10s on decent connection)
```

---

### TC-078: Parallel Build Performance

**Given** GoReleaser builds 5 platforms
**When** I compare parallel vs. sequential build time
**Then** parallel build is >4x faster (5 platforms / ~1.2 overhead)

**Automation**: Log analysis
- Review GitHub Actions logs
- Compare "Build time" for parallel execution
- Note: Sequential build would be ~15 minutes (5 × 3 min)
- Parallel build should be ~4.5 minutes

---

## 8. Documentation Testing

### TC-079: README.md Installation Instructions

**Given** README.md is updated
**When** I read the installation section
**Then** it includes instructions for:
- Homebrew (macOS/Linux)
- Scoop (Windows)
- Manual download

**Automation**: Grep check
```bash
grep -i 'homebrew\|brew install' README.md
grep -i 'scoop\|scoop install' README.md
grep -i 'manual\|download' README.md
```

---

### TC-080: CONTRIBUTING.md Release Process

**Given** CONTRIBUTING.md is updated
**When** I read the release section
**Then** it documents:
- How to create a release (tag creation)
- Version numbering (SemVer)
- Release workflow steps

**Automation**: Grep check
```bash
grep -i 'release\|tag\|version' CONTRIBUTING.md
```

---

### TC-081: SECURITY.md Vulnerability Reporting

**Given** SECURITY.md exists
**When** I read the security policy
**Then** it documents:
- How to report vulnerabilities
- Expected response time
- Checksum verification instructions

**Automation**: File existence check
```bash
test -f SECURITY.md
grep -i 'vulnerability\|report\|checksum' SECURITY.md
```

---

### TC-082: Release Verification Scripts

**Given** Release verification scripts exist
**When** I execute `scripts/verify-release.sh`
**Then** script successfully verifies checksums

**Automation**: Script execution
```bash
chmod +x scripts/verify-release.sh
./scripts/verify-release.sh v1.0.0 linux_amd64
echo $?  # Should be 0
```

---

### TC-083: Homebrew Tap README

**Given** `homebrew-shark` repository has README.md
**When** I read the README
**Then** it includes:
- Tap installation instructions
- Shark CLI installation command
- Link to main repository

**Automation**: Grep check
```bash
cd homebrew-shark
grep -i 'brew tap\|brew install' README.md
grep 'github.com/jwwelbor/shark-task-manager' README.md
```

---

### TC-084: Scoop Bucket README

**Given** `scoop-shark` repository has README.md
**When** I read the README
**Then** it includes:
- Bucket addition instructions
- Shark CLI installation command
- Link to main repository

**Automation**: Grep check
```bash
cd scoop-shark
grep -i 'scoop bucket add\|scoop install' README.md
grep 'github.com/jwwelbor/shark-task-manager' README.md
```

---

## 9. Acceptance Test Scenarios

### Scenario 1: Complete Release Workflow (v1.0.0)

**Test Steps**:
1. Ensure all code changes are committed and pushed
2. Run full test suite locally: `make test` (all tests pass)
3. Validate GoReleaser config: `goreleaser check` (no errors)
4. Test local snapshot build: `goreleaser build --snapshot --clean` (<5 min, all platforms)
5. Create production tag: `git tag v1.0.0`
6. Push tag: `git push origin v1.0.0`
7. Monitor GitHub Actions workflow (completes in <10 min)
8. Verify draft release created with 6 assets
9. Review release notes
10. Publish release
11. Verify Homebrew tap updated (Formula/shark.rb committed)
12. Verify Scoop bucket updated (bucket/shark.json committed)
13. Test Homebrew installation: `brew install shark`
14. Test Scoop installation: `scoop install shark`
15. Test manual download and checksum verification
16. Run `shark --version` on all platforms (outputs v1.0.0)

**Expected Result**: All steps complete successfully, v1.0.0 available via all distribution channels

---

### Scenario 2: Hotfix Release (v1.0.1)

**Test Steps**:
1. Fix critical bug on main branch
2. Run tests: `make test` (all pass)
3. Create patch tag: `git tag v1.0.1`
4. Push tag: `git push origin v1.0.1`
5. Monitor workflow (completes successfully)
6. Publish release
7. Verify package managers update automatically
8. Test upgrade: `brew upgrade shark`, `scoop update shark`
9. Verify version: `shark --version` (outputs v1.0.1)

**Expected Result**: Hotfix deployed rapidly (<15 min from tag to user installation)

---

### Scenario 3: Beta Release (v1.1.0-beta)

**Test Steps**:
1. Develop new feature on feature branch
2. Merge to main
3. Create beta tag: `git tag v1.1.0-beta`
4. Push tag: `git push origin v1.1.0-beta`
5. Monitor workflow
6. Verify release marked as "pre-release" (auto-detection)
7. Verify beta not added to Homebrew/Scoop (production only)
8. Test manual download and installation
9. Get user feedback
10. Create production tag: `git tag v1.1.0` (after beta testing)

**Expected Result**: Beta releases work but don't pollute package managers

---

### Scenario 4: Rollback Failed Release

**Test Steps**:
1. Detect critical bug in v1.2.0 after publishing
2. Delete release on GitHub
3. Delete tag: `git tag -d v1.2.0`, `git push origin :refs/tags/v1.2.0`
4. Revert Homebrew formula commit in `homebrew-shark`
5. Revert Scoop manifest commit in `scoop-shark`
6. Fix bug
7. Create hotfix tag: `git tag v1.2.1`
8. Complete release workflow for v1.2.1

**Expected Result**: Failed release cleaned up, hotfix deployed successfully

---

## 10. Test Execution Plan

### Phase 1: Configuration Validation (Day 1)
- Run TC-001 through TC-012 (configuration tests)
- Fix any configuration errors
- Document baseline metrics

### Phase 2: Local Build Testing (Day 2-3)
- Run TC-013 through TC-030 (build tests)
- Verify all platforms build successfully
- Measure performance baselines

### Phase 3: CI/CD Integration Testing (Day 4-5)
- Run TC-031 through TC-045 (workflow tests)
- Test with v0.0.1-test tag
- Debug any workflow issues

### Phase 4: Distribution Testing (Day 6-7)
- Run TC-046 through TC-060 (installation tests)
- Test on real macOS, Linux, Windows machines
- Verify package manager installations

### Phase 5: Security & Performance Testing (Day 8-9)
- Run TC-061 through TC-078 (security and performance tests)
- Verify checksum mechanisms
- Measure and document performance

### Phase 6: Documentation Testing (Day 10)
- Run TC-079 through TC-084 (documentation tests)
- Verify all docs updated
- Test scripts on all platforms

### Phase 7: Acceptance Testing (Day 11-12)
- Run complete release scenarios
- Test v0.1.0-beta release
- Validate end-to-end workflow

### Phase 8: Production Release (Day 13)
- Cut v1.0.0 release
- Verify all acceptance criteria
- Monitor for issues

---

## 11. Test Metrics

### Success Criteria

**Configuration**:
- ✅ 100% configuration tests pass (12/12)

**Build**:
- ✅ 100% build tests pass (18/18)
- ✅ Build time <5 minutes
- ✅ Binary size <12 MB

**CI/CD**:
- ✅ 100% workflow tests pass (15/15)
- ✅ Workflow time <10 minutes

**Distribution**:
- ✅ 100% installation tests pass (15/15)
- ✅ All 3 distribution channels working

**Security**:
- ✅ 100% security tests pass (10/10)
- ✅ All checksums verify
- ✅ No secrets exposed in logs

**Performance**:
- ✅ 100% performance tests pass (8/8)
- ✅ All targets met

**Documentation**:
- ✅ 100% documentation tests pass (6/6)

**Overall**:
- ✅ 84/84 tests pass (100%)
- ✅ All acceptance scenarios complete successfully

---

## 12. Test Automation Strategy

### Automated Tests (54/84)

**Configuration Tests** (12):
- GoReleaser check
- YAML parsing
- GitHub CLI validation

**Build Tests** (15):
- Snapshot builds
- Binary verification
- Performance benchmarks

**CI/CD Tests** (12):
- GitHub Actions checks
- Release verification
- Log analysis

**Security Tests** (8):
- Checksum verification
- Token scope checks
- Vulnerability scanning

**Performance Tests** (7):
- Build timing
- Binary size checks
- Installation timing

### Manual Tests (30/84)

**Distribution Tests** (15):
- Package manager installations (macOS, Linux, Windows)
- Multi-platform verification
- Upgrade testing

**Documentation Tests** (6):
- README review
- Script testing
- User guide validation

**Acceptance Scenarios** (4):
- End-to-end release workflows
- Rollback procedures

**Exploratory Testing** (5):
- Edge cases
- Error handling
- User experience

### Continuous Integration

**Pre-Commit** (local):
```bash
# Run before every commit
go test ./...
golangci-lint run
goreleaser check
```

**PR Checks** (GitHub Actions):
```yaml
- Run unit tests
- Run integration tests
- Lint code
- Validate GoReleaser config
```

**Release Workflow** (GitHub Actions):
```yaml
- Run all tests
- Build all platforms
- Validate checksums
- Performance checks
```

---

## 13. Conclusion

This comprehensive test suite ensures the Distribution & Release Automation feature meets all functional, performance, security, and documentation requirements. With 84 tests covering configuration, build, distribution, security, and performance aspects, the test criteria provide confidence that the automated release system works reliably across all supported platforms and distribution channels.

**Key Testing Priorities**:
1. **Configuration correctness** - Ensure GoReleaser and GitHub Actions are properly configured
2. **Multi-platform builds** - Verify all 5 platforms build successfully
3. **Distribution channels** - Test Homebrew, Scoop, and manual downloads
4. **Security controls** - Validate checksum verification and secret management
5. **Performance targets** - Confirm build time, binary size, and installation speed

**Test Coverage**: 84 tests across 9 categories
**Automation Level**: 64% automated, 36% manual
**Estimated Test Execution Time**: 12-13 days (including setup and debugging)

---

**Test Criteria Status**: ✅ Complete and Ready for Execution
**Next Step**: Begin Phase 1 implementation with test-driven approach
