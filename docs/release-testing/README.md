# Release Testing Infrastructure

## Overview

This directory contains comprehensive end-to-end release testing infrastructure for Shark Task Manager. It includes automated test scripts, documentation templates, and procedures for validating complete release cycles across all distribution channels.

**Purpose:** Ensure every release works correctly on all platforms before production deployment.

**Last Updated:** 2025-12-18
**Task:** T-E04-F08-004

---

## Directory Structure

```
docs/release-testing/
├── README.md                      # This file
├── testing-procedures.md          # Step-by-step testing guide
├── release-testing-report.md      # Report template for test results
├── performance-comparison.md      # Performance metrics comparison template
└── [version]/                     # Results for specific version (e.g., v0.1.0-beta/)
    ├── report.md                  # Completed test report
    ├── performance.md             # Completed performance comparison
    ├── homebrew-test-results.txt  # Homebrew test output
    ├── scoop-test-results.txt     # Scoop test output
    ├── manual-test-results.txt    # Manual download test output
    └── screenshots/               # Screenshots and evidence
```

## Quick Start

### For Testers

1. **Read the procedures:**
   ```bash
   cat docs/release-testing/testing-procedures.md
   ```

2. **Run automated tests:**
   ```bash
   # Homebrew (macOS)
   ./scripts/test-homebrew.sh v0.1.0-beta

   # Scoop (Windows)
   .\scripts\test-scoop.ps1 -ExpectedVersion v0.1.0-beta

   # Manual Download (Any platform)
   ./scripts/test-manual.sh v0.1.0-beta
   ```

3. **Document results:**
   - Copy `release-testing-report.md` template
   - Fill in test results
   - Save in version-specific directory

### For Release Managers

1. **Create beta release:**
   ```bash
   git tag -a v0.1.0-beta -m "Beta release for testing"
   git push origin v0.1.0-beta
   ```

2. **Monitor workflow:**
   - Watch GitHub Actions workflow execute
   - Verify all jobs complete successfully

3. **Coordinate testing:**
   - Distribute testing procedures to team
   - Collect results from all platforms
   - Review and approve release

4. **Make go/no-go decision:**
   - Review completed test reports
   - Check all validation gates passed
   - Approve production release or request fixes

---

## Test Scripts

### Homebrew Installation Test

**Location:** `scripts/test-homebrew.sh`

**Purpose:** Automated testing of Homebrew installation on macOS (Intel and Apple Silicon).

**Usage:**
```bash
./scripts/test-homebrew.sh v0.1.0-beta
```

**What it tests:**
- Tap addition (jwwelbor/shark)
- Installation via `brew install shark`
- Version verification
- Binary location and size
- Functional tests (help, epic list, task list, database access)
- Installation time (target: < 30s)

**Output:** `homebrew-test-results.txt`

**Requirements:**
- macOS with Homebrew installed
- Internet connection
- Bash shell

---

### Scoop Installation Test

**Location:** `scripts/test-scoop.ps1`

**Purpose:** Automated testing of Scoop installation on Windows.

**Usage:**
```powershell
.\scripts\test-scoop.ps1 -ExpectedVersion v0.1.0-beta
```

**What it tests:**
- Bucket addition (scoop-shark)
- Installation via `scoop install shark`
- Version verification
- Binary location and size
- Functional tests (help, epic list, task list, database access)
- Installation time (target: < 30s)

**Output:** `scoop-test-results.txt`

**Requirements:**
- Windows with Scoop installed
- Internet connection
- PowerShell 5.1 or later

---

### Manual Download Test

**Location:** `scripts/test-manual.sh`

**Purpose:** Automated testing of manual binary download, checksum verification, and extraction.

**Usage:**
```bash
./scripts/test-manual.sh v0.1.0-beta
```

**What it tests:**
- Platform/architecture auto-detection
- Binary download from GitHub Releases
- Checksum download
- SHA256 checksum verification
- Archive extraction (.tar.gz or .zip)
- Binary execution and functionality
- Download/install time

**Output:** `/tmp/manual-test-results.txt`

**Requirements:**
- Linux, macOS, or Windows (with Git Bash/WSL)
- `curl`, `wget`, `sha256sum`, `tar`/`unzip`
- Internet connection

**Platforms tested:**
- Linux amd64
- Linux arm64
- macOS amd64 (Intel)
- macOS arm64 (Apple Silicon)
- Windows amd64

---

## GitHub Actions Workflow

### Release Testing Workflow

**Location:** `.github/workflows/release-test.yml`

**Purpose:** Automated testing of releases via GitHub Actions.

**Triggers:**
- Manual dispatch (workflow_dispatch)
- Release published

**Jobs:**
1. **test-manual-download** - Tests manual download on all platforms
2. **test-homebrew** - Tests Homebrew on macOS (Intel + ARM)
3. **test-scoop** - Tests Scoop on Windows
4. **performance-benchmark** - Measures binary size and startup time
5. **report-results** - Collects and summarizes all test results
6. **notify** - Sends notification on completion

**Usage:**

Manual trigger:
```bash
# Via GitHub UI:
# Actions → Release Testing → Run workflow
# Enter version: v0.1.0-beta

# Via GitHub CLI:
gh workflow run release-test.yml -f version=v0.1.0-beta
```

Automatic trigger:
```bash
# Automatically runs when release is published
```

**Artifacts:**
- Individual test result files
- Combined test results archive
- Test summary in workflow summary

---

## Documentation

### Testing Procedures

**File:** `testing-procedures.md`

**Contents:**
- Complete step-by-step guide for end-to-end release testing
- Prerequisites and setup instructions
- Detailed procedures for each distribution channel
- Performance testing guidelines
- Troubleshooting guide
- Quick reference checklist

**Use this when:** Conducting manual release testing.

---

### Release Testing Report

**File:** `release-testing-report.md`

**Contents:**
- Comprehensive test report template
- All validation checkpoints
- Performance metrics tables
- Issue tracking section
- Sign-off section

**Use this when:** Documenting test results for a specific release.

**How to use:**
1. Copy template to version-specific directory
2. Fill in test results as you execute tests
3. Document any issues encountered
4. Complete sign-off section
5. Use for release approval decision

---

### Performance Comparison

**File:** `performance-comparison.md`

**Contents:**
- Performance metrics comparison template
- Baseline vs current release comparison
- Regression and improvement tracking
- Recommendations section

**Use this when:** Analyzing performance changes between releases.

**How to use:**
1. Copy template to version-specific directory
2. Fill in actual metrics from test runs
3. Compare against baseline or previous release
4. Identify regressions and improvements
5. Add recommendations for future releases

---

## Testing Coverage

### Distribution Channels

- ✅ **GitHub Releases** - Manual binary download
- ✅ **Homebrew** - macOS package manager
- ✅ **Scoop** - Windows package manager
- ⏳ **Snap** - Linux package manager (future)
- ⏳ **Docker** - Container distribution (future)

### Platforms

- ✅ **Linux amd64** - Manual download tested
- ✅ **Linux arm64** - Manual download tested
- ✅ **macOS amd64 (Intel)** - Homebrew + Manual tested
- ✅ **macOS arm64 (Apple Silicon)** - Homebrew + Manual tested
- ✅ **Windows amd64** - Scoop + Manual tested

### Test Types

- ✅ **Installation** - All distribution channels
- ✅ **Version Verification** - Correct version installed
- ✅ **Checksum Verification** - SHA256 validation
- ✅ **Functional Tests** - Core commands work
- ✅ **Performance Tests** - Installation time, binary size
- ✅ **Database Tests** - Persistence layer works
- ⏳ **Integration Tests** - Multi-user scenarios (future)
- ⏳ **Load Tests** - Large datasets (future)

---

## Performance Targets

### Build Performance

| Metric | Target | Notes |
|--------|--------|-------|
| Total Workflow | < 10 min | GitHub Actions workflow |
| Test Phase | < 3 min | Running all tests |
| Build Phase | < 5 min | GoReleaser build |

### Binary Sizes

| Metric | Target | Notes |
|--------|--------|-------|
| Archive Size | < 10 MB | Compressed .tar.gz or .zip |
| Binary Size | < 12 MB | Uncompressed executable |

### Installation Performance

| Metric | Target | Notes |
|--------|--------|-------|
| Homebrew | < 30s | Including tap addition |
| Scoop | < 30s | Including bucket addition |
| Manual | < 30s | Download + verify + extract |

### Runtime Performance

| Metric | Target | Notes |
|--------|--------|-------|
| Startup | < 100ms | Cold start time |
| --version | < 50ms | Version command |
| --help | < 100ms | Help command |
| epic list | < 200ms | List command |
| task list | < 500ms | Database query |

---

## Validation Gates

### Mandatory Gates (Must Pass)

- [ ] All GitHub Actions tests pass
- [ ] All 6 platform binaries build successfully
- [ ] All checksums generated correctly
- [ ] Homebrew formula committed and valid
- [ ] Scoop manifest committed and valid
- [ ] At least one installation test passes per distribution channel
- [ ] Version command returns correct version
- [ ] No critical functional bugs

### Recommended Gates (Should Pass)

- [ ] All performance targets met
- [ ] All platforms tested (not just automated)
- [ ] Manual testing completed on real hardware
- [ ] No regressions from previous release
- [ ] User documentation updated
- [ ] Changelog accurate and complete

### Nice-to-Have (Optional)

- [ ] Multiple testers on each platform
- [ ] Testing in different geographic regions
- [ ] Testing on different network speeds
- [ ] Testing with large databases
- [ ] Testing edge cases and error scenarios

---

## Common Issues and Solutions

### Test Script Fails

**Issue:** Test script exits with error

**Solutions:**
1. Check network connectivity
2. Verify release is published (not draft)
3. Check version tag format (must start with 'v')
4. Run script with verbose output for debugging
5. Check script has execute permissions (`chmod +x`)

### Homebrew Installation Fails

**Issue:** `brew install shark` fails

**Solutions:**
1. Run `brew update` to refresh formula database
2. Verify formula committed to homebrew-shark repo
3. Check formula syntax: `brew audit jwwelbor/shark/shark`
4. Remove and re-add tap: `brew untap jwwelbor/shark && brew tap jwwelbor/shark`
5. Check HOMEBREW_TAP_TOKEN permissions

### Scoop Installation Fails

**Issue:** `scoop install shark` fails

**Solutions:**
1. Run `scoop update` to refresh manifest database
2. Verify manifest committed to scoop-shark repo
3. Check manifest syntax: `scoop cat shark`
4. Remove and re-add bucket
5. Check SCOOP_BUCKET_TOKEN permissions

### Checksum Verification Fails

**Issue:** SHA256 checksum doesn't match

**Solutions:**
1. **STOP** - This is a security concern
2. Re-download archive and checksums
3. Verify again
4. If still fails, investigate:
   - Check GoReleaser logs
   - Verify GitHub Actions artifacts
   - Report security incident

### Binary Won't Execute

**Issue:** "Permission denied" or "Cannot execute"

**Solutions:**
1. Unix: `chmod +x shark`
2. Check architecture matches: `file shark` vs `uname -m`
3. Windows: Disable antivirus temporarily (may block unknown executables)
4. Verify download completed (check file size)

---

## Best Practices

### For Test Execution

1. **Test on clean machines** - No prior installations or dev tools
2. **Test real-world scenarios** - Don't just test happy paths
3. **Document everything** - Timestamps, durations, errors
4. **Test all platforms** - Don't rely on automated tests alone
5. **Test different network conditions** - Slow connections reveal issues

### For Documentation

1. **Fill in templates immediately** - Don't wait until end
2. **Include screenshots** - Visual evidence is valuable
3. **Note exact versions** - OS version, tool versions, etc.
4. **Document workarounds** - If you find them, others will need them
5. **Update procedures** - If process changes, update docs

### For Issue Reporting

1. **Be specific** - Exact error messages, not paraphrases
2. **Include context** - Platform, version, steps to reproduce
3. **Categorize severity** - Critical vs nice-to-fix
4. **Suggest solutions** - If you know a fix, share it
5. **Link to evidence** - Screenshots, logs, test results

---

## Release Testing Workflow

```
1. Pre-Testing
   ↓
   - Review procedures
   - Set up test platforms
   - Clean previous installs
   - Prepare documentation

2. Create Release
   ↓
   - Create and push tag
   - Monitor GitHub Actions
   - Verify workflow success
   - Check draft release

3. Automated Testing
   ↓
   - Run test scripts
   - Collect results
   - Check automated workflow
   - Review artifacts

4. Manual Testing
   ↓
   - Test Homebrew (Intel + ARM)
   - Test Scoop (Windows)
   - Test manual downloads (All platforms)
   - Verify functionality

5. Performance Testing
   ↓
   - Measure build times
   - Check binary sizes
   - Test installation times
   - Benchmark runtime

6. Documentation
   ↓
   - Complete test report
   - Update performance comparison
   - Document issues
   - Save artifacts

7. Review & Decision
   ↓
   - Review all results
   - Check validation gates
   - Make go/no-go decision
   - Sign off

8. Cleanup
   ↓
   - Publish or delete beta release
   - Clean up test environments
   - Archive documentation
   - Update process docs
```

---

## Future Improvements

### Short Term (Next Release)

- [ ] Add Linux package manager testing (Snap, APT, YUM)
- [ ] Add Windows installer testing (MSI, EXE)
- [ ] Automate more platforms in GitHub Actions
- [ ] Add integration tests with real projects
- [ ] Add performance regression detection

### Medium Term (3-6 Months)

- [ ] Set up test matrix for all OS versions
- [ ] Add Docker container testing
- [ ] Create user acceptance testing procedures
- [ ] Add automated rollback testing
- [ ] Implement continuous release testing

### Long Term (6+ Months)

- [ ] Multi-region testing infrastructure
- [ ] Automated canary release testing
- [ ] A/B testing framework for releases
- [ ] Performance monitoring in production
- [ ] User feedback collection system

---

## Contributing

### Adding New Test Scripts

1. Follow existing script patterns
2. Include comprehensive error handling
3. Generate structured output files
4. Document in this README
5. Add to GitHub Actions workflow

### Improving Documentation

1. Update templates with lessons learned
2. Add troubleshooting entries
3. Improve procedures clarity
4. Add diagrams if helpful
5. Keep examples current

### Reporting Issues

File issues with:
- `[Release Testing]` prefix in title
- Steps to reproduce
- Expected vs actual behavior
- Platform and version information
- Screenshots or logs

---

## Resources

### Internal Documentation

- [Release Baseline Metrics](../release-baseline-metrics.md)
- [GoReleaser Configuration](../../.goreleaser.yml)
- [GitHub Actions Workflow](../../.github/workflows/release.yml)
- [Task Definition](../plan/E04-task-mgmt-cli-core/E04-F08-distribution-release/tasks/T-E04-F08-004.md)

### External Resources

- [GoReleaser Documentation](https://goreleaser.com)
- [Homebrew Formula Documentation](https://docs.brew.sh/Formula-Cookbook)
- [Scoop Manifest Documentation](https://github.com/ScoopInstaller/Scoop/wiki/App-Manifests)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)

---

## Contact

**Maintained By:** DevOps Team
**Questions:** [Create an issue](https://github.com/jwwelbor/shark-task-manager/issues)
**Documentation Issues:** Tag with `documentation` label

---

**Last Updated:** 2025-12-18
**Version:** 1.0
**Related Task:** T-E04-F08-004
