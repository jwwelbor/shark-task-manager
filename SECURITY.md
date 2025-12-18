# Security Policy

## Supported Versions

We release patches for security vulnerabilities for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

We recommend always using the latest stable release for the best security and feature set.

## Reporting a Vulnerability

We take the security of Shark Task Manager seriously. If you discover a security vulnerability, please follow these steps:

### Reporting Process

1. **DO NOT** open a public GitHub issue for security vulnerabilities
2. Email security reports to: [jwwelbor@example.com] (replace with actual email)
3. Include the following information:
   - Description of the vulnerability
   - Steps to reproduce the issue
   - Affected versions
   - Potential impact
   - Any suggested fixes (if available)

### Response Timeline

- **Initial Response**: Within 48 hours of receiving your report
- **Status Update**: Within 7 days with assessment and planned actions
- **Fix Timeline**: Security patches will be released as soon as possible, typically within 14 days for critical issues

### Disclosure Policy

- We will acknowledge receipt of your vulnerability report
- We will provide regular updates on our progress
- We will notify you when the vulnerability is fixed
- We will publicly disclose the vulnerability after a fix is released (with credit to you if desired)
- We request that you do not publicly disclose the vulnerability until we have released a fix

## Verifying Release Integrity

All official releases include SHA256 checksums to verify download integrity. Always verify checksums before installing Shark CLI.

### Why Verify Checksums?

Verifying checksums ensures:
- The file was not corrupted during download
- The file has not been tampered with
- You received the exact binary we published

### Verification Instructions

#### Linux and macOS

1. Download both the binary archive and checksums file:
   ```bash
   # Example for Linux AMD64
   wget https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_linux_amd64.tar.gz
   wget https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/checksums.txt
   ```

2. Verify the checksum automatically:
   ```bash
   sha256sum -c checksums.txt --ignore-missing
   ```

   Expected output:
   ```
   shark_1.0.0_linux_amd64.tar.gz: OK
   ```

3. If verification fails, DO NOT use the binary. Download again or report the issue.

**Alternative: Manual Verification**

```bash
# Calculate checksum
sha256sum shark_1.0.0_linux_amd64.tar.gz

# Compare output with value in checksums.txt
cat checksums.txt | grep linux_amd64
```

The checksums must match exactly.

#### Windows (PowerShell)

1. Download both files:
   ```powershell
   # Download binary
   Invoke-WebRequest -Uri "https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_windows_amd64.zip" -OutFile "shark.zip"

   # Download checksums
   Invoke-WebRequest -Uri "https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/checksums.txt" -OutFile "checksums.txt"
   ```

2. Run verification script:
   ```powershell
   # Calculate actual hash
   $actual = (Get-FileHash shark.zip -Algorithm SHA256).Hash.ToLower()

   # Extract expected hash from checksums.txt
   $expected = (Get-Content checksums.txt | Select-String "windows_amd64").ToString().Split()[0]

   # Compare
   if ($actual -eq $expected) {
       Write-Host "✅ Checksum verified successfully" -ForegroundColor Green
       Write-Host "The download is authentic and has not been modified."
   } else {
       Write-Host "❌ CHECKSUM VERIFICATION FAILED" -ForegroundColor Red
       Write-Host "DO NOT install this binary. It may be corrupted or tampered with."
       Write-Host ""
       Write-Host "Expected: $expected" -ForegroundColor Yellow
       Write-Host "Actual:   $actual" -ForegroundColor Yellow
       exit 1
   }
   ```

3. Only proceed with installation if verification succeeds.

### Automated Verification Scripts

For convenience, we provide verification scripts:

**Linux/macOS**: Use `scripts/verify-release.sh`
```bash
./scripts/verify-release.sh v1.0.0 linux amd64
```

**Windows**: Use `scripts/verify-release.ps1`
```powershell
.\scripts\verify-release.ps1 -Version "v1.0.0" -Platform "windows" -Arch "amd64"
```

These scripts automatically download, verify, and validate releases.

## Package Manager Security

### Homebrew (macOS)

Homebrew automatically verifies checksums during installation. The formula includes SHA256 hashes:

```ruby
# Formula/shark.rb
sha256 "abc123..." # Homebrew verifies this automatically
```

When you run `brew install shark`, Homebrew will:
1. Download the binary from GitHub Releases
2. Calculate the SHA256 checksum
3. Compare with the hash in the formula
4. Abort installation if checksums don't match

**Manual Verification**:
```bash
brew fetch --force --verbose shark
# Output shows checksum verification
```

### Scoop (Windows)

Scoop also includes automatic checksum verification:

```json
{
  "hash": "sha256:abc123...",
  "url": "https://github.com/.../shark.zip"
}
```

Installation with `scoop install shark` includes automatic verification.

## Security Best Practices

### For Users

1. **Always verify checksums** when downloading manually
2. **Use HTTPS** for all downloads (enforced by GitHub)
3. **Keep Shark updated** to get latest security patches
4. **Use package managers** (Homebrew/Scoop) when possible - they automate verification
5. **Check release signatures** (planned for future releases)

### For Developers

1. **Never commit secrets** to the repository
2. **Use GitHub Secrets** for tokens (HOMEBREW_TAP_TOKEN, SCOOP_BUCKET_TOKEN)
3. **Rotate tokens** every 90 days minimum
4. **Review dependencies** regularly for vulnerabilities
5. **Enable secret scanning** in your repository settings
6. **Use fine-grained PATs** with minimum required permissions

## Known Security Considerations

### SQLite Database Security

Shark uses SQLite for task storage. By default:
- Database file permissions: `0600` (owner read/write only)
- Database location: Current working directory (`shark-tasks.db`)
- No network access (local file only)

**Best Practices**:
- Store project database in project directory (version control)
- Do not store sensitive data in task descriptions
- Use `.gitignore` for local working databases if they contain sensitive info
- Back up database files regularly

### GitHub API Tokens

If you're a maintainer managing releases:

**Token Security**:
- Use **fine-grained personal access tokens** (not classic PATs)
- Set **minimum permissions** (Contents: Read/Write only)
- Set **expiration** (90 days maximum)
- **Rotate immediately** if compromised
- **Never log or commit** token values

**Token Permissions Required**:
- `HOMEBREW_TAP_TOKEN`: Contents: Read and write on `homebrew-shark` repo only
- `SCOOP_BUCKET_TOKEN`: Contents: Read and write on `scoop-shark` repo only
- `GITHUB_TOKEN`: Automatically provided by GitHub Actions (no manual setup)

## Security Updates

Security updates are released as patch versions (e.g., 1.0.1, 1.0.2) and announced via:
- [GitHub Releases](https://github.com/jwwelbor/shark-task-manager/releases)
- [Security Advisories](https://github.com/jwwelbor/shark-task-manager/security/advisories)

Subscribe to repository notifications to receive security alerts.

## Additional Resources

- [GitHub Security Best Practices](https://docs.github.com/en/code-security)
- [GoReleaser Security Documentation](https://goreleaser.com/customization/checksums/)
- [NIST Guidelines on Checksums](https://csrc.nist.gov/projects/hash-functions)

## Questions?

If you have questions about security but don't have a vulnerability to report:
- Open a [GitHub Discussion](https://github.com/jwwelbor/shark-task-manager/discussions)
- Check existing [Security Advisories](https://github.com/jwwelbor/shark-task-manager/security/advisories)

Thank you for helping keep Shark Task Manager secure!
