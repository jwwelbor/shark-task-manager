# End-to-End Release Testing Procedures

## Overview

This document provides step-by-step procedures for conducting complete end-to-end release testing for the Shark Task Manager. These procedures ensure that all distribution channels work correctly before a production release.

**Target Release:** v0.1.0-beta (and subsequent releases)
**Last Updated:** 2025-12-18
**Related Task:** T-E04-F08-004

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Pre-Testing Setup](#2-pre-testing-setup)
3. [Creating Beta Release](#3-creating-beta-release)
4. [GitHub Release Validation](#4-github-release-validation)
5. [Homebrew Testing](#5-homebrew-testing)
6. [Scoop Testing](#6-scoop-testing)
7. [Manual Download Testing](#7-manual-download-testing)
8. [Performance Testing](#8-performance-testing)
9. [Functional Testing](#9-functional-testing)
10. [Results Documentation](#10-results-documentation)
11. [Beta Cleanup](#11-beta-cleanup)
12. [Troubleshooting](#12-troubleshooting)

---

## 1. Prerequisites

### 1.1 Required Access

- [ ] GitHub repository write access (for creating tags)
- [ ] GitHub Actions access (to view workflow runs)
- [ ] Access to homebrew-shark repository (to verify formula)
- [ ] Access to scoop-shark repository (to verify manifest)

### 1.2 Required Secrets

Verify these secrets are configured in repository settings:

- [ ] `GITHUB_TOKEN` (automatically provided by GitHub)
- [ ] `HOMEBREW_TAP_TOKEN` (personal access token with repo scope)
- [ ] `SCOOP_BUCKET_TOKEN` (personal access token with repo scope)

To verify:
```bash
# Navigate to: https://github.com/jwwelbor/shark-task-manager/settings/secrets/actions
```

### 1.3 Required Platforms

For complete testing, you need access to:

- [ ] macOS (Intel) machine or VM
- [ ] macOS (Apple Silicon) machine or VM
- [ ] Windows machine or VM (with Scoop installed)
- [ ] Linux machine or VM

**Note:** If you don't have access to all platforms, you can:
1. Use GitHub Actions for automated testing (see `.github/workflows/release-test.yml`)
2. Use cloud VMs (AWS, Azure, GCP)
3. Use GitHub Codespaces
4. Ask team members on different platforms to assist

### 1.4 Required Tools

**On macOS:**
```bash
# Install Homebrew if not present
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Verify installation
brew --version
```

**On Windows:**
```powershell
# Install Scoop if not present
Set-ExecutionPolicy RemoteSigned -Scope CurrentUser
irm get.scoop.sh | iex

# Verify installation
scoop --version
```

**On Linux:**
```bash
# Install required tools
sudo apt-get update
sudo apt-get install -y wget curl tar gzip

# Verify tools
wget --version
sha256sum --version
```

### 1.5 Pre-Flight Checks

Before starting release testing:

```bash
# 1. Ensure you're on main branch with latest code
git checkout main
git pull origin main

# 2. Verify all tests pass locally (if Go is installed)
go test ./...

# 3. Verify GoReleaser configuration is valid
goreleaser check

# 4. Check for any uncommitted changes
git status
```

---

## 2. Pre-Testing Setup

### 2.1 Create Testing Checklist

Copy the release testing report template:

```bash
cp docs/release-testing/release-testing-report.md docs/release-testing/v0.1.0-beta-report.md
```

### 2.2 Record Test Environment

Document your test environment in the report:

- Operating system versions
- Processor architectures
- Network conditions
- Homebrew/Scoop versions

### 2.3 Clean Previous Test Artifacts

If this is a re-test, clean up previous installations:

**macOS:**
```bash
# Remove previous Homebrew installation
brew uninstall shark 2>/dev/null || true
brew untap jwwelbor/shark 2>/dev/null || true
```

**Windows:**
```powershell
# Remove previous Scoop installation
scoop uninstall shark 2>$null
scoop bucket rm shark 2>$null
```

**Linux:**
```bash
# Remove any previous manual installations
rm -rf /tmp/shark* ~/shark
```

---

## 3. Creating Beta Release

### 3.1 Create and Push Tag

**IMPORTANT:** Ensure all changes are committed and pushed to `main` branch.

```bash
# 1. Verify you're on main branch with latest code
git checkout main
git pull origin main

# 2. Create annotated tag
git tag -a v0.1.0-beta -m "Beta release for end-to-end testing"

# 3. Push tag to trigger release workflow
git push origin v0.1.0-beta
```

**Record timestamp:** Note the exact time you pushed the tag.

### 3.2 Monitor Workflow Execution

1. Navigate to GitHub Actions:
   ```
   https://github.com/jwwelbor/shark-task-manager/actions
   ```

2. Find the "Release" workflow run triggered by your tag

3. Monitor the workflow progress:
   - [ ] Test phase completes
   - [ ] Build phase completes
   - [ ] GoReleaser creates draft release
   - [ ] Assets are uploaded
   - [ ] Homebrew formula is committed
   - [ ] Scoop manifest is committed

4. **Record metrics:**
   - Workflow start time
   - Test phase duration
   - Build phase duration
   - Total workflow duration
   - Any warnings or errors

### 3.3 Verify Workflow Success

Once workflow completes:

- [ ] Workflow status is green (success)
- [ ] No failed jobs
- [ ] All steps completed successfully
- [ ] Artifacts uploaded

**If workflow fails:** See [Troubleshooting](#12-troubleshooting) section.

---

## 4. GitHub Release Validation

### 4.1 Access Draft Release

1. Navigate to releases:
   ```
   https://github.com/jwwelbor/shark-task-manager/releases
   ```

2. Find the draft release for `v0.1.0-beta`

### 4.2 Verify Release Assets

Check that all 6 assets are present:

- [ ] `shark_0.1.0-beta_linux_amd64.tar.gz`
- [ ] `shark_0.1.0-beta_linux_arm64.tar.gz`
- [ ] `shark_0.1.0-beta_darwin_amd64.tar.gz`
- [ ] `shark_0.1.0-beta_darwin_arm64.tar.gz`
- [ ] `shark_0.1.0-beta_windows_amd64.zip`
- [ ] `checksums.txt`

### 4.3 Verify Asset Sizes

Record the size of each asset:

```bash
# Asset sizes should be < 10 MB each
```

| Asset | Size | Status |
|-------|------|--------|
| linux_amd64 | [record] | ✅/❌ |
| linux_arm64 | [record] | ✅/❌ |
| darwin_amd64 | [record] | ✅/❌ |
| darwin_arm64 | [record] | ✅/❌ |
| windows_amd64 | [record] | ✅/❌ |

### 4.4 Review Release Notes

- [ ] Release notes are auto-generated
- [ ] Changelog entries are present
- [ ] Installation instructions are correct
- [ ] Links are valid

### 4.5 Verify Distribution Repositories

**Homebrew Formula:**
1. Navigate to: `https://github.com/jwwelbor/homebrew-shark`
2. Check latest commit
3. Verify `Formula/shark.rb` was updated
4. Review formula contents for correctness

**Scoop Manifest:**
1. Navigate to: `https://github.com/jwwelbor/scoop-shark`
2. Check latest commit
3. Verify `bucket/shark.json` was updated
4. Review manifest contents for correctness

---

## 5. Homebrew Testing

### 5.1 Automated Testing (Recommended)

Use the automated test script:

```bash
# From project root
./scripts/test-homebrew.sh v0.1.0-beta
```

This script will:
- Add the tap
- Install shark
- Verify version
- Run functional tests
- Generate test report

**Save the output:** The script generates `homebrew-test-results.txt`

### 5.2 Manual Testing (If Automation Fails)

If the automated script fails or you want to test manually:

#### 5.2.1 Add Tap

```bash
brew tap jwwelbor/shark
```

**Verify:**
```bash
brew tap | grep shark
# Should show: jwwelbor/shark
```

#### 5.2.2 Install Shark

```bash
# Start timer
time brew install shark
```

**Record installation time:** Should be < 30 seconds

#### 5.2.3 Verify Installation

```bash
# Check version
shark --version
# Expected output: shark version v0.1.0-beta

# Check binary location
which shark
# Expected: /opt/homebrew/bin/shark (ARM) or /usr/local/bin/shark (Intel)

# Check binary size
ls -lh $(which shark)
```

#### 5.2.4 Run Functional Tests

```bash
# Test help command
shark --help

# Test epic commands
shark epic list

# Test task commands
shark task list

# Test database access
shark task list --status created
```

### 5.3 Test on Both Architectures

If possible, repeat testing on both:
- macOS Intel (x86_64)
- macOS Apple Silicon (ARM64)

### 5.4 Clean Up

```bash
brew uninstall shark
brew untap jwwelbor/shark
```

---

## 6. Scoop Testing

### 6.1 Automated Testing (Recommended)

Use the automated test script:

```powershell
# From project root
.\scripts\test-scoop.ps1 -ExpectedVersion v0.1.0-beta
```

This script will:
- Add the bucket
- Install shark
- Verify version
- Run functional tests
- Generate test report

**Save the output:** The script generates `scoop-test-results.txt`

### 6.2 Manual Testing (If Automation Fails)

If the automated script fails or you want to test manually:

#### 6.2.1 Add Bucket

```powershell
scoop bucket add shark https://github.com/jwwelbor/scoop-shark
```

**Verify:**
```powershell
scoop bucket list
# Should show shark bucket
```

#### 6.2.2 Install Shark

```powershell
# Start timer
Measure-Command { scoop install shark }
```

**Record installation time:** Should be < 30 seconds

#### 6.2.3 Verify Installation

```powershell
# Check version
shark --version
# Expected output: shark version v0.1.0-beta

# Check binary location
(Get-Command shark).Path
# Expected: In scoop apps directory

# Check binary size
(Get-Item (Get-Command shark).Path).Length / 1MB
```

#### 6.2.4 Run Functional Tests

```powershell
# Test help command
shark --help

# Test epic commands
shark epic list

# Test task commands
shark task list

# Test database access
shark task list --status created
```

### 6.3 Clean Up

```powershell
scoop uninstall shark
scoop bucket rm shark
```

---

## 7. Manual Download Testing

### 7.1 Automated Testing (Recommended)

Use the automated test script:

```bash
# From project root (works on macOS, Linux, Windows Git Bash)
./scripts/test-manual.sh v0.1.0-beta
```

This script will:
- Auto-detect platform
- Download appropriate archive
- Download and verify checksums
- Extract and test binary
- Generate test report

**Save the output:** The script generates `/tmp/manual-test-results.txt`

### 7.2 Manual Testing (If Automation Fails)

#### 7.2.1 Determine Platform

Identify your platform and architecture:

| OS | Architecture | Archive Name |
|----|-------------|--------------|
| Linux | x86_64 | shark_0.1.0-beta_linux_amd64.tar.gz |
| Linux | ARM64 | shark_0.1.0-beta_linux_arm64.tar.gz |
| macOS | Intel | shark_0.1.0-beta_darwin_amd64.tar.gz |
| macOS | Apple Silicon | shark_0.1.0-beta_darwin_arm64.tar.gz |
| Windows | x86_64 | shark_0.1.0-beta_windows_amd64.zip |

#### 7.2.2 Download Assets

```bash
# Set variables
VERSION="v0.1.0-beta"
ARCHIVE_NAME="shark_0.1.0-beta_linux_amd64.tar.gz"  # Adjust for your platform
BASE_URL="https://github.com/jwwelbor/shark-task-manager/releases/download/${VERSION}"

# Download archive
wget "${BASE_URL}/${ARCHIVE_NAME}"

# Download checksums
wget "${BASE_URL}/checksums.txt"
```

**Record download time and size**

#### 7.2.3 Verify Checksum

```bash
# Verify checksum
sha256sum -c checksums.txt --ignore-missing

# Expected output:
# shark_0.1.0-beta_linux_amd64.tar.gz: OK
```

**If checksum fails:** DO NOT USE THE BINARY. This indicates a security issue.

#### 7.2.4 Extract Archive

**For .tar.gz (Linux/macOS):**
```bash
tar -xzf shark_*.tar.gz
```

**For .zip (Windows):**
```bash
unzip shark_*.zip
# Or use Windows Explorer
```

#### 7.2.5 Test Binary

```bash
# Make executable (Unix-like systems)
chmod +x shark

# Test version
./shark --version
# Expected: shark version v0.1.0-beta

# Test help
./shark --help

# Test command
./shark epic list
```

### 7.3 Test Multiple Platforms

If possible, repeat manual download testing on:
- Linux (amd64)
- macOS (Intel or ARM)
- Windows (amd64)

---

## 8. Performance Testing

### 8.1 Build Performance

From the GitHub Actions workflow, record:

| Metric | Duration | Target | Status |
|--------|----------|--------|--------|
| Total Workflow | [record] | < 10 min | ✅/❌ |
| Test Phase | [record] | < 3 min | ✅/❌ |
| Build Phase | [record] | < 5 min | ✅/❌ |
| GoReleaser | [record] | < 5 min | ✅/❌ |

### 8.2 Binary Sizes

From GitHub Release assets, record:

| Asset | Size (MB) | Target | Status |
|-------|-----------|--------|--------|
| linux_amd64 | [record] | < 10 MB | ✅/❌ |
| linux_arm64 | [record] | < 10 MB | ✅/❌ |
| darwin_amd64 | [record] | < 10 MB | ✅/❌ |
| darwin_arm64 | [record] | < 10 MB | ✅/❌ |
| windows_amd64 | [record] | < 10 MB | ✅/❌ |

### 8.3 Installation Performance

From test scripts, record:

| Platform | Installation Time | Target | Status |
|----------|------------------|--------|--------|
| Homebrew (Intel) | [record] | < 30s | ✅/❌ |
| Homebrew (ARM) | [record] | < 30s | ✅/❌ |
| Scoop (Windows) | [record] | < 30s | ✅/❌ |

### 8.4 Runtime Performance

Test command execution times:

```bash
# Test version command (should be < 50ms)
time shark --version

# Test help command (should be < 100ms)
time shark --help

# Test list command (should be < 200ms)
time shark epic list

# Test database query (should be < 500ms)
time shark task list
```

Record results:

| Command | Time | Notes |
|---------|------|-------|
| --version | [record] | |
| --help | [record] | |
| epic list | [record] | |
| task list | [record] | |

---

## 9. Functional Testing

### 9.1 Core Commands

Test these commands on each platform:

```bash
# Version and help
shark --version
shark --help

# Epic commands
shark epic list
shark epic create

# Task commands
shark task list
shark task get T-E01-F01-001
shark task list --status created
```

### 9.2 Database Operations

Test database functionality:

```bash
# Create database (if not exists)
shark task list

# Verify database created
# Location: ./shark-tasks.db

# Test write operation
shark task start T-E01-F01-001

# Test read operation
shark task get T-E01-F01-001

# Test query operation
shark task list --status in_progress
```

### 9.3 Edge Cases

Test error handling:

```bash
# Invalid task ID
shark task get INVALID-TASK-ID
# Expected: Error message

# Empty result
shark task list --status nonexistent-status
# Expected: Empty list or error

# Missing database (delete and recreate)
rm shark-tasks.db
shark task list
# Expected: Creates new database
```

---

## 10. Results Documentation

### 10.1 Update Testing Report

Fill in the template at `docs/release-testing/v0.1.0-beta-report.md`:

1. Update executive summary
2. Fill in all test results
3. Record performance metrics
4. Document any issues found
5. Add screenshots if helpful

### 10.2 Update Performance Comparison

Fill in `docs/release-testing/performance-comparison.md`:

1. Compare against baseline metrics
2. Calculate percentage changes
3. Identify regressions and improvements
4. Add recommendations

### 10.3 Save Test Artifacts

Collect and save:

- Test script output files
- Screenshots of workflow runs
- Screenshots of successful installations
- Performance measurement data
- Any error logs

Organize in:
```
docs/release-testing/v0.1.0-beta/
├── homebrew-test-results.txt
├── scoop-test-results.txt
├── manual-test-results.txt
├── screenshots/
└── logs/
```

---

## 11. Beta Cleanup

### 11.1 Decide on Beta Artifact Retention

**Option A: Keep Beta Release**
- Publish the draft release
- Leave tag in place
- Useful for historical reference

**Option B: Delete Beta Release (Recommended for clean production)**
```bash
# Delete draft release from GitHub UI
# Then delete tag:
git tag -d v0.1.0-beta
git push origin :refs/tags/v0.1.0-beta
```

### 11.2 Clean Up Local Environment

```bash
# Remove any test installations
brew uninstall shark 2>/dev/null || true
brew untap jwwelbor/shark 2>/dev/null || true

# Remove test files
rm -rf /tmp/shark* ~/shark

# Clean up git state
git fetch --prune --prune-tags
```

### 11.3 Document Lessons Learned

Update release process documentation with:
- Issues encountered
- Solutions applied
- Process improvements
- Notes for next release

---

## 12. Troubleshooting

### 12.1 Workflow Failures

**Test phase fails:**
```bash
# Run tests locally to diagnose
go test -v ./...

# Check for race conditions
go test -race ./...

# Check specific package
go test -v ./internal/database
```

**Build phase fails:**
```bash
# Test GoReleaser locally
goreleaser build --snapshot --clean

# Check for configuration errors
goreleaser check
```

**Distribution publishing fails:**
- Verify HOMEBREW_TAP_TOKEN has correct permissions
- Verify SCOOP_BUCKET_TOKEN has correct permissions
- Check target repositories exist and are accessible

### 12.2 Homebrew Issues

**Tap not found:**
```bash
# Verify repository exists
curl -s https://github.com/jwwelbor/homebrew-shark | grep "404"

# Manually add tap with full URL
brew tap jwwelbor/shark https://github.com/jwwelbor/homebrew-shark
```

**Installation fails:**
```bash
# Check formula
brew info jwwelbor/shark/shark

# View formula file
brew cat jwwelbor/shark/shark

# Check formula for errors
brew audit jwwelbor/shark/shark

# Try verbose installation
brew install -v jwwelbor/shark/shark
```

**Formula not found after tap:**
```bash
# Update Homebrew
brew update

# Re-add tap
brew untap jwwelbor/shark
brew tap jwwelbor/shark
```

### 12.3 Scoop Issues

**Bucket not found:**
```powershell
# Verify repository exists
Invoke-WebRequest https://github.com/jwwelbor/scoop-shark

# Manually add bucket
scoop bucket add shark https://github.com/jwwelbor/scoop-shark.git
```

**Installation fails:**
```powershell
# Check manifest
scoop info shark

# View manifest file
scoop cat shark

# Try verbose installation
scoop install shark -v
```

**Manifest not found after adding bucket:**
```powershell
# Update Scoop
scoop update

# Re-add bucket
scoop bucket rm shark
scoop bucket add shark https://github.com/jwwelbor/scoop-shark
```

### 12.4 Checksum Verification Fails

**If checksum doesn't match:**

1. **DO NOT USE THE BINARY** - Security concern
2. Re-download the archive
3. Re-download checksums.txt
4. Verify again
5. If still fails, report issue and investigate:
   - Check GoReleaser generated checksums correctly
   - Check GitHub release assets weren't corrupted
   - Check for man-in-the-middle attack (unlikely on HTTPS)

### 12.5 Binary Doesn't Execute

**"Permission denied" (Unix-like):**
```bash
chmod +x shark
./shark --version
```

**"Cannot execute binary file" (Wrong architecture):**
```bash
# Check architecture
file shark

# Download correct binary for your architecture
uname -m  # Shows your architecture
```

**"DLL not found" (Windows):**
- Ensure binary is built statically (CGO_ENABLED=0)
- Check for missing Visual C++ Redistributables

---

## Appendix A: Quick Reference

### Test Script Locations

- Homebrew: `./scripts/test-homebrew.sh`
- Scoop: `./scripts/test-scoop.ps1`
- Manual: `./scripts/test-manual.sh`

### Documentation Locations

- Testing Report Template: `docs/release-testing/release-testing-report.md`
- Performance Comparison: `docs/release-testing/performance-comparison.md`
- This Document: `docs/release-testing/testing-procedures.md`

### Important URLs

- GitHub Actions: `https://github.com/jwwelbor/shark-task-manager/actions`
- Releases: `https://github.com/jwwelbor/shark-task-manager/releases`
- Homebrew Tap: `https://github.com/jwwelbor/homebrew-shark`
- Scoop Bucket: `https://github.com/jwwelbor/scoop-shark`

---

## Appendix B: Checklist Summary

Use this checklist for quick validation:

### Pre-Release
- [ ] All tests passing locally
- [ ] Secrets configured
- [ ] Test platforms available
- [ ] Test environment documented

### Release Creation
- [ ] Tag created and pushed
- [ ] Workflow executed successfully
- [ ] Draft release created
- [ ] All 6 assets present
- [ ] Formula/manifest committed

### Distribution Testing
- [ ] Homebrew Intel tested
- [ ] Homebrew ARM tested
- [ ] Scoop Windows tested
- [ ] Manual Linux tested
- [ ] Manual macOS tested
- [ ] Manual Windows tested

### Performance Validation
- [ ] Build time < 10 min
- [ ] Assets < 10 MB each
- [ ] Installation < 30s each
- [ ] Commands execute quickly

### Functional Validation
- [ ] Version command works
- [ ] Help command works
- [ ] Core commands work
- [ ] Database operations work
- [ ] Edge cases handled

### Documentation
- [ ] Testing report completed
- [ ] Performance comparison updated
- [ ] Issues documented
- [ ] Artifacts saved

### Cleanup
- [ ] Beta release decision made
- [ ] Local cleanup completed
- [ ] Lessons documented
- [ ] Ready for production release

---

**Document Version:** 1.0
**Last Updated:** 2025-12-18
**Maintained By:** DevOps Team
**Related Task:** T-E04-F08-004
