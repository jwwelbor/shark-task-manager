# Security Design: Distribution & Release Automation

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F08-distribution-release
**Date**: 2025-12-17
**Author**: security-architect

## Executive Summary

This document defines the security architecture for the automated release and distribution system of the Shark CLI tool. The system must protect against supply chain attacks, ensure binary authenticity, manage secrets securely, and provide users with verifiable download integrity.

**Security Posture**: Defense in depth with multiple verification layers
**Threat Model**: Supply chain tampering, credential compromise, binary tampering
**Primary Controls**: Checksum verification, GitHub token scoping, automated auditing

---

## 1. Threat Model

### 1.1 Assets to Protect

| Asset | Value | Impact if Compromised |
|-------|-------|----------------------|
| **Binary Artifacts** | High | Users install malicious code |
| **GitHub Tokens** | Critical | Unauthorized releases, code injection |
| **Build Environment** | High | Compromised binaries in all releases |
| **Package Manager Repositories** | High | Distribution of malware to all users |
| **Source Code Repository** | Critical | Complete project compromise |
| **User Trust** | Critical | Loss of all adoption, reputation damage |

### 1.2 Threat Actors

| Actor | Capability | Motivation | Likelihood |
|-------|-----------|------------|------------|
| **External Attacker** | High (sophisticated) | Data theft, malware distribution | Low |
| **Malicious Contributor** | Medium (insider) | Code injection via PR | Low |
| **Compromised Dependencies** | Medium (supply chain) | Unintentional malware propagation | Medium |
| **Compromised CI/CD** | High (infrastructure) | Build-time injection | Low |
| **Man-in-the-Middle** | Medium (network) | Binary tampering during download | Medium |

### 1.3 Attack Vectors

```
┌─────────────────────────────────────────────────────────────────┐
│                     ATTACK SURFACE                               │
│                                                                  │
│  1. Source Code Injection                                        │
│     │ Malicious PR merged into main                             │
│     │ Direct commit with stolen credentials                     │
│     └─▶ Mitigation: Code review, branch protection              │
│                                                                  │
│  2. Dependency Poisoning                                         │
│     │ Compromised Go module in go.mod                           │
│     │ Typosquatting attack                                      │
│     └─▶ Mitigation: Go checksum database, dependency review     │
│                                                                  │
│  3. Build Environment Compromise                                 │
│     │ GitHub Actions runner tampered                            │
│     │ GoReleaser action compromised                             │
│     └─▶ Mitigation: Pin action versions, use official runners   │
│                                                                  │
│  4. Credential Theft                                             │
│     │ GitHub token leaked                                       │
│     │ Package manager PAT exposed                               │
│     └─▶ Mitigation: Secret scanning, token rotation             │
│                                                                  │
│  5. Man-in-the-Middle Download                                   │
│     │ User downloads binary via HTTP                            │
│     │ Network attacker injects malware                          │
│     └─▶ Mitigation: HTTPS only, checksum verification           │
│                                                                  │
│  6. Package Manager Repository Compromise                        │
│     │ Homebrew tap tampered                                     │
│     │ Scoop bucket malicious manifest                           │
│     └─▶ Mitigation: Token scoping, commit signing (future)      │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 2. Security Controls

### 2.1 Source Code Protection

**Branch Protection Rules** (`main` branch):

```yaml
Require pull request reviews before merging: ✅
  Required approving reviews: 1
  Dismiss stale reviews: ✅
  Require review from Code Owners: ✅

Require status checks to pass before merging: ✅
  Required checks:
    - tests (go test ./...)
    - lint (golangci-lint)

Require branches to be up to date: ✅

Require signed commits: ⚠️ Optional (recommended for future)

Restrict who can push to matching branches: ✅
  Allowed: Repository administrators only

Allow force pushes: ❌ Disabled

Allow deletions: ❌ Disabled
```

**Code Review Process**:
1. All changes via pull requests (no direct commits to `main`)
2. Minimum 1 approval from maintainer
3. Automated tests must pass
4. Security-sensitive changes require 2 approvals

**Dependency Management**:
```bash
# Go module verification via checksum database
export GOSUMDB=sum.golang.org  # Default, always enabled

# Review dependencies before merging
go list -m all
go mod why <suspicious-module>
```

### 2.2 Build Environment Security

**GitHub Actions Runner**:
- Use GitHub-hosted runners (not self-hosted)
  - Ephemeral (destroyed after each run)
  - No persistent state
  - Maintained by GitHub security team

**Action Version Pinning**:

```yaml
# ❌ BAD: Uses latest version (supply chain risk)
- uses: goreleaser/goreleaser-action@v5

# ✅ GOOD: Pin to specific SHA (immutable)
- uses: goreleaser/goreleaser-action@v5.0.0
  # or even better:
  # - uses: goreleaser/goreleaser-action@a82b9783
```

**Implementation Strategy**:
- Phase 1 (F08): Use semantic version tag (`@v5`)
- Phase 2 (Future): Pin to commit SHA (`@a82b9783`)
- Use Dependabot to update pinned actions automatically

**Build Isolation**:
```yaml
jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write  # Minimal permissions
    steps:
      # No secrets exposed in logs
      # No network access during build (except Go module downloads)
```

### 2.3 Credential Management

**Token Hierarchy**:

```
┌──────────────────────────────────────────────────────────────┐
│ GITHUB_TOKEN (automatic)                                      │
│ ├─ Scope: Repository only                                    │
│ ├─ Permissions: contents:write, pull-requests:read           │
│ ├─ Lifetime: Single workflow run                             │
│ └─ Security: Cannot be exfiltrated (GitHub-managed)          │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│ HOMEBREW_TAP_TOKEN (manual PAT)                              │
│ ├─ Scope: homebrew-shark repository only                     │
│ ├─ Permissions: contents:write                               │
│ ├─ Lifetime: 90 days (auto-expire)                           │
│ └─ Security: Stored in GitHub Secrets (encrypted at rest)    │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│ SCOOP_BUCKET_TOKEN (manual PAT)                              │
│ ├─ Scope: scoop-shark repository only                        │
│ ├─ Permissions: contents:write                               │
│ ├─ Lifetime: 90 days (auto-expire)                           │
│ └─ Security: Stored in GitHub Secrets (encrypted at rest)    │
└──────────────────────────────────────────────────────────────┘
```

**Token Creation Process** (Fine-Grained PAT):

1. Navigate to: GitHub Settings → Developer Settings → Personal Access Tokens → Fine-grained tokens
2. Click "Generate new token"
3. Configuration:
   ```
   Token name: Homebrew Tap Update (Shark CLI)
   Expiration: 90 days
   Resource owner: jwwelbor
   Repository access: Only select repositories
     Selected: homebrew-shark
   Permissions:
     Repository permissions:
       Contents: Read and write
       Metadata: Read-only (automatic)
   ```
4. Generate and copy token
5. Add to repository secrets:
   ```
   Repository → Settings → Secrets and Variables → Actions → New repository secret
   Name: HOMEBREW_TAP_TOKEN
   Secret: <paste token>
   ```

**Token Rotation Policy**:
- Rotate every 90 days (when GitHub expires)
- Immediate rotation if:
  - Token accidentally exposed in logs
  - Team member with access leaves project
  - Suspicious activity detected

**Token Security Best Practices**:
- ✅ Use fine-grained tokens (not classic PATs)
- ✅ Minimum scope (single repository, specific permissions)
- ✅ Set expiration (max 90 days)
- ✅ Store in GitHub Secrets (never in code or logs)
- ❌ Never log token values
- ❌ Never commit tokens to repository

### 2.4 Secret Scanning

**GitHub Secret Scanning** (automatic for public repos):
- Scans commits for leaked tokens
- Alerts repository admins
- Revokes GitHub tokens automatically

**Pre-Commit Hook** (developer machines):
```bash
#!/bin/bash
# .git/hooks/pre-commit

# Check for potential secrets
if git diff --cached | grep -E '(github_pat_|ghp_|gho_|ghu_)'; then
  echo "ERROR: Potential GitHub token detected in commit"
  echo "Remove secret before committing"
  exit 1
fi

# Check for other secrets
if git diff --cached | grep -E '(password|secret|token|api_key).*=.*[A-Za-z0-9]{20,}'; then
  echo "WARNING: Potential secret detected"
  echo "Review changes carefully"
fi
```

**GitHub Actions Workflow Protection**:
```yaml
# Prevent secrets from leaking in logs
env:
  HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
  # ❌ DO NOT use: run: echo $HOMEBREW_TAP_TOKEN
  # ✅ Secrets automatically masked in logs
```

---

## 3. Binary Integrity

### 3.1 Checksum Generation and Verification

**Checksum File Format** (`checksums.txt`):
```
# SHA256 checksums generated by GoReleaser
abc123def456... shark_1.0.0_linux_amd64.tar.gz
789ghi012jkl... shark_1.0.0_linux_arm64.tar.gz
345mno678pqr... shark_1.0.0_darwin_amd64.tar.gz
901stu234vwx... shark_1.0.0_darwin_arm64.tar.gz
567yz890abcd... shark_1.0.0_windows_amd64.zip
```

**Generation Process** (automatic via GoReleaser):
```yaml
# .goreleaser.yml
checksum:
  name_template: 'checksums.txt'
  algorithm: sha256
```

**User Verification Process**:

**macOS/Linux**:
```bash
# Download binary and checksums
wget https://github.com/.../shark_1.0.0_linux_amd64.tar.gz
wget https://github.com/.../checksums.txt

# Verify (automatic match)
sha256sum -c checksums.txt --ignore-missing
# Output: shark_1.0.0_linux_amd64.tar.gz: OK

# Or manual verification
sha256sum shark_1.0.0_linux_amd64.tar.gz
# Compare with value in checksums.txt
```

**Windows (PowerShell)**:
```powershell
# Download files
Invoke-WebRequest -Uri "https://github.com/.../shark_1.0.0_windows_amd64.zip" -OutFile "shark.zip"
Invoke-WebRequest -Uri "https://github.com/.../checksums.txt" -OutFile "checksums.txt"

# Calculate hash
$actual = (Get-FileHash shark.zip -Algorithm SHA256).Hash.ToLower()

# Extract expected hash
$expected = (Get-Content checksums.txt | Select-String "windows_amd64").ToString().Split()[0]

# Verify
if ($actual -eq $expected) {
    Write-Host "✅ Checksum verified successfully"
} else {
    Write-Host "❌ Checksum verification FAILED"
    Write-Host "Expected: $expected"
    Write-Host "Actual:   $actual"
    exit 1
}
```

**Package Manager Verification** (automatic):

**Homebrew**:
```ruby
# In Formula/shark.rb
on_macos do
  if Hardware::CPU.arm?
    url "https://github.com/.../shark_1.0.0_darwin_arm64.tar.gz"
    sha256 "abc123..."  # Homebrew verifies this automatically
  end
end
```

**Scoop**:
```json
{
  "architecture": {
    "64bit": {
      "url": "https://github.com/.../shark_1.0.0_windows_amd64.zip",
      "hash": "sha256:abc123..."  // Scoop verifies this automatically
    }
  }
}
```

**Failure Scenarios**:

| Scenario | Detection | User Experience | Mitigation |
|----------|-----------|-----------------|------------|
| Binary tampered during download | Checksum mismatch | Manual: Error message; Package manager: Install aborted | Re-download from HTTPS source |
| Checksum file tampered | N/A (no higher authority) | User unknowingly installs malware | Future: GPG sign checksums.txt |
| GitHub release compromised | Checksum matches (attacker controls both) | User unknowingly installs malware | Future: Code signing, SLSA provenance |

### 3.2 Future Enhancement: Code Signing (Out of Scope for F08)

**macOS**:
- Apple Developer account required ($99/year)
- `codesign` utility for signing
- Notarization via Apple (requires Xcode)
- Users: No Gatekeeper warnings

**Windows**:
- Code signing certificate ($200-500/year)
- `signtool` utility for signing
- Users: No SmartScreen warnings

**GPG Signing** (all platforms):
```bash
# Generate GPG key
gpg --full-generate-key

# Sign checksums.txt
gpg --detach-sign --armor checksums.txt
# Creates checksums.txt.asc

# Users verify
gpg --verify checksums.txt.asc checksums.txt
```

**SLSA Provenance** (supply chain attestation):
- GitHub Actions automatically generates build provenance
- Users can verify entire build chain
- Standard: https://slsa.dev/

---

## 4. Network Security

### 4.1 Transport Security

**HTTPS Enforcement**:

All downloads via HTTPS only:
```
✅ https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_linux_amd64.tar.gz
❌ http://github.com/... (GitHub redirects to HTTPS automatically)
```

**Certificate Pinning** (future consideration):
- Not applicable for CLI tool distribution
- Users rely on OS certificate store for GitHub certificate validation

**DNS Security**:
- GitHub's DNS protected by DNSSEC
- Users should verify domain: `github.com` (not typosquatted domains)

### 4.2 Download Verification Checklist

**Users should verify**:
1. ✅ Download URL is `https://github.com/jwwelbor/shark-task-manager/releases/`
2. ✅ GitHub shows green padlock (valid HTTPS certificate)
3. ✅ Release is tagged (e.g., `v1.0.0`)
4. ✅ Release date is recent and expected
5. ✅ Checksum matches value in `checksums.txt`

**Warning signs of compromise**:
- ❌ Download URL is not `github.com` (e.g., typosquatted domain)
- ❌ Release has no tag or odd tag name (e.g., `latest-build`)
- ❌ Checksum verification fails
- ❌ Binary size significantly different from expected (~10 MB)
- ❌ Antivirus warnings (though false positives common for Go binaries)

---

## 5. Package Manager Security

### 5.1 Homebrew Tap Security

**Repository Protection**:
```yaml
# homebrew-shark repository settings

Branch protection (main):
  Require pull request reviews: ✅ (for manual updates)
  Require status checks: ✅ (if CI added)
  Restrict pushes: ✅ (GoReleaser bot + admins only)

Vulnerability alerts: ✅ Enabled
Dependabot: N/A (no dependencies in tap)
```

**Formula Verification**:

**What Homebrew checks**:
- SHA256 checksum of downloaded archive
- Archive extracts successfully
- Binary runs (`test do` block)

**What Homebrew does NOT check**:
- Binary is not malware (relies on checksum matching GitHub release)
- GitHub release is legitimate (trusts repository owner)

**User Trust Chain**:
```
User trusts → Homebrew tap owner (jwwelbor)
            → GitHub release integrity (checksums)
            → Build process (GitHub Actions)
            → Source code (repository maintainers)
```

**Tap Takeover Prevention**:
- Strong GitHub account password (required)
- Two-factor authentication (required for org repos)
- Repository transfer restrictions
- Token expiration (90 days)

### 5.2 Scoop Bucket Security

**Repository Protection** (same as Homebrew):
```yaml
# scoop-shark repository settings
(Same branch protection as Homebrew tap)
```

**Manifest Verification**:

**What Scoop checks**:
- SHA256 hash of downloaded archive
- JSON manifest is valid
- Binary extracts successfully

**What Scoop does NOT check**:
- Binary is not malware (relies on hash matching)
- GitHub release is legitimate

**Autoupdate Mechanism**:
```json
{
  "checkver": {
    "github": "https://github.com/jwwelbor/shark-task-manager"
  },
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "https://github.com/.../shark_$version_windows_amd64.zip"
      }
    }
  }
}
```

**Security Risk**: Scoop autoupdate relies on GitHub API
- If GitHub API compromised, could serve malicious version
- Mitigation: Scoop still verifies hash (manual update required if hash changes)

---

## 6. Audit and Monitoring

### 6.1 GitHub Actions Audit

**Workflow Run Logs**:
- Retained for 90 days (GitHub default)
- Viewable by repository collaborators
- Shows:
  - Who triggered workflow (Git tag author)
  - Build steps and outputs
  - Errors and warnings
  - Artifact uploads

**Audit Trail**:
```
Actions → Workflows → Release → Run #123
├─ Triggered by: git-tag-push (v1.0.0 by jwwelbor)
├─ Checkout: commit abc123...
├─ Tests: ✅ Passed (42 tests)
├─ GoReleaser: ✅ Built 5 binaries
├─ Upload: ✅ 5 archives, 1 checksum file
└─ Status: ✅ Success (Duration: 4m 32s)
```

**Alerting**:
- Email notification on workflow failure
- GitHub mobile app notification
- Optional: Slack/Discord webhook

### 6.2 Release Monitoring

**GitHub Release Events**:
- RSS feed: `https://github.com/jwwelbor/shark-task-manager/releases.atom`
- Watch repository for "Releases only"
- Email notifications to repository watchers

**Package Manager Monitoring**:

**Homebrew**:
```bash
# Check current version in tap
brew info jwwelbor/shark/shark

# Check for pending updates
brew update
brew outdated
```

**Scoop**:
```powershell
# Check current version
scoop info shark

# Check for updates
scoop update
scoop status
```

### 6.3 Security Incident Response

**Incident Types and Response**:

| Incident | Detection | Response | Recovery Time |
|----------|-----------|----------|---------------|
| **Leaked GitHub token** | Secret scanning alert | Revoke token immediately, rotate | <1 hour |
| **Compromised release** | User report, checksum mismatch | Delete release, delete tag, investigate | <4 hours |
| **Malicious PR merged** | Code review, security scan | Revert commit, cut hotfix release | <24 hours |
| **Package manager compromise** | User report, checksum mismatch | Revert formula/manifest, notify users | <2 hours |
| **Dependency vulnerability** | Dependabot alert | Update dependency, test, release patch | <48 hours |

**Incident Response Playbook**:

**Step 1: Containment**
- Stop ongoing attacks (revoke tokens, delete releases)
- Prevent further damage (disable workflows temporarily)

**Step 2: Investigation**
- Review GitHub Actions logs
- Check commit history for unauthorized changes
- Verify all recent releases

**Step 3: Eradication**
- Remove malicious code
- Rotate all credentials
- Update dependencies

**Step 4: Recovery**
- Cut clean release
- Update package managers
- Notify users via GitHub release notes and security advisory

**Step 5: Post-Incident**
- Document incident in security log
- Update security controls
- Review and improve processes

---

## 7. Compliance and Best Practices

### 7.1 Security Standards Alignment

**NIST Cybersecurity Framework**:
- ✅ Identify: Threat modeling (section 1)
- ✅ Protect: Access controls, encryption
- ✅ Detect: Audit logs, monitoring
- ✅ Respond: Incident response plan
- ✅ Recover: Rollback procedures

**OWASP Top 10 (relevant items)**:
- ✅ A01 Broken Access Control: Token scoping, branch protection
- ✅ A02 Cryptographic Failures: HTTPS, SHA256 checksums
- ✅ A05 Security Misconfiguration: Branch protection, workflow permissions
- ✅ A08 Software and Data Integrity Failures: Checksum verification, dependency management

**Supply Chain Security (SLSA Levels)**:
- Current (F08): SLSA Level 1 (automated build)
- Future: SLSA Level 2 (version control, build service)
- Future: SLSA Level 3 (provenance, non-falsifiable)

### 7.2 Security Checklist for Releases

**Pre-Release**:
- [ ] All tests passing
- [ ] Dependencies reviewed (no known vulnerabilities)
- [ ] Code review completed
- [ ] GoReleaser config validated (`goreleaser check`)
- [ ] Test build successful (`goreleaser build --snapshot`)

**During Release**:
- [ ] GitHub Actions workflow successful
- [ ] All 5 platforms built
- [ ] Checksums generated
- [ ] GitHub release created (draft)

**Post-Release**:
- [ ] Draft release reviewed
- [ ] Release notes accurate
- [ ] Checksums verified (sample 1-2 platforms manually)
- [ ] Homebrew tap updated
- [ ] Scoop bucket updated
- [ ] Installation tested on at least 2 platforms

**Security Verification**:
- [ ] No secrets in workflow logs
- [ ] Checksums.txt present and valid
- [ ] Binary sizes reasonable (~8-12 MB)
- [ ] No antivirus false positives reported

---

## 8. User Security Guidance

### 8.1 Installation Security Recommendations

**Documentation to Include in README.md**:

```markdown
## Security

### Verify Downloads

All releases include SHA256 checksums for verification.

#### macOS / Linux
```bash
# Download binary and checksums
wget https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_linux_amd64.tar.gz
wget https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/checksums.txt

# Verify checksum
sha256sum -c checksums.txt --ignore-missing
```

#### Windows (PowerShell)
```powershell
# Download files
Invoke-WebRequest -Uri "https://github.com/.../shark_1.0.0_windows_amd64.zip" -OutFile "shark.zip"
Invoke-WebRequest -Uri "https://github.com/.../checksums.txt" -OutFile "checksums.txt"

# Verify (script in docs/verify-checksum.ps1)
.\docs\verify-checksum.ps1 -File shark.zip -ChecksumFile checksums.txt
```

### Reporting Security Issues

If you discover a security vulnerability:
1. **Do NOT** open a public GitHub issue
2. Email security@<domain> with details
3. We will respond within 48 hours
4. We will credit reporters (unless anonymity requested)

### Security Advisories

Subscribe to security advisories:
- GitHub: Watch repository → Custom → Security alerts
- RSS: https://github.com/jwwelbor/shark-task-manager/security/advisories
```

### 8.2 Enterprise Security Considerations

**For Enterprise Users**:

**Binary Verification Script** (`scripts/verify-shark.sh`):
```bash
#!/bin/bash
# Verify Shark CLI binary integrity

set -e

VERSION="$1"
PLATFORM="$2"

if [ -z "$VERSION" ] || [ -z "$PLATFORM" ]; then
    echo "Usage: $0 <version> <platform>"
    echo "Example: $0 v1.0.0 linux_amd64"
    exit 1
fi

# Download checksums
wget "https://github.com/jwwelbor/shark-task-manager/releases/download/${VERSION}/checksums.txt"

# Download binary
wget "https://github.com/jwwelbor/shark-task-manager/releases/download/${VERSION}/shark_${VERSION#v}_${PLATFORM}.tar.gz"

# Verify
sha256sum -c checksums.txt --ignore-missing

# Extract
tar -xzf "shark_${VERSION#v}_${PLATFORM}.tar.gz"

echo "✅ Shark CLI verified and extracted successfully"
echo "Binary location: ./shark"
```

**Air-Gapped Environments**:
1. Download binary and checksums on internet-connected machine
2. Verify checksums
3. Transfer verified binary to air-gapped network via approved media
4. Document transfer in audit log

---

## 9. Risk Assessment

### 9.1 Residual Risks (After F08 Implementation)

| Risk | Likelihood | Impact | Severity | Mitigation Status |
|------|-----------|--------|----------|-------------------|
| Compromised GitHub account | Low | Critical | High | ⚠️ Partial (2FA required, but no hardware key enforcement) |
| Malicious dependency | Medium | High | Medium | ⚠️ Partial (Go checksum DB, but no automated scanning) |
| Build environment tampering | Low | Critical | High | ⚠️ Partial (GitHub-hosted runners, but no SLSA provenance) |
| Man-in-the-middle download | Low | High | Medium | ✅ Mitigated (HTTPS + checksums) |
| Package manager compromise | Low | High | Medium | ⚠️ Partial (Token scoping, but no signing) |
| Leaked credentials | Medium | Medium | Medium | ✅ Mitigated (Secret scanning, rotation policy) |

**Risk Acceptance**:
- Residual risks are acceptable for F08 scope
- Future enhancements will address remaining gaps

### 9.2 Future Security Enhancements (Out of Scope)

**Phase 2** (3-6 months post-F08):
1. Code signing (macOS, Windows)
2. GPG signing of checksums.txt
3. SLSA provenance generation
4. Automated dependency scanning (Dependabot, Snyk)

**Phase 3** (6-12 months post-F08):
1. Hardware security key requirement for maintainers
2. Reproducible builds
3. Binary transparency log
4. Bug bounty program

---

## 10. Security Testing

### 10.1 Pre-Release Security Tests

**Automated Tests** (GitHub Actions):
```yaml
security-tests:
  name: Security Tests
  runs-on: ubuntu-latest
  steps:
    - name: Checkout
      uses: actions/checkout@v4

    - name: Run Gosec (Go security scanner)
      run: |
        go install github.com/securego/gosec/v2/cmd/gosec@latest
        gosec ./...

    - name: Check for known vulnerabilities
      run: |
        go install golang.org/x/vuln/cmd/govulncheck@latest
        govulncheck ./...

    - name: Scan for secrets
      uses: trufflesecurity/trufflehog@main
      with:
        path: ./

    - name: Dependency review
      uses: actions/dependency-review-action@v3
```

**Manual Security Review** (pre-v1.0.0):
- [ ] Threat model reviewed
- [ ] Security controls implemented
- [ ] Incident response plan documented
- [ ] User security guidance written
- [ ] Security testing completed

### 10.2 Post-Release Security Validation

**Checksum Verification Test**:
```bash
# Test script: tests/security/verify-release.sh

#!/bin/bash
VERSION="$1"

# Download checksums
wget "https://github.com/jwwelbor/shark-task-manager/releases/download/${VERSION}/checksums.txt"

# Download all platforms
platforms=("linux_amd64" "linux_arm64" "darwin_amd64" "darwin_arm64" "windows_amd64")

for platform in "${platforms[@]}"; do
    if [[ "$platform" == *"windows"* ]]; then
        ext="zip"
    else
        ext="tar.gz"
    fi

    wget "https://github.com/.../shark_${VERSION#v}_${platform}.${ext}"
done

# Verify all checksums
sha256sum -c checksums.txt

echo "✅ All checksums verified successfully"
```

**Installation Test** (per platform):
```bash
# macOS Homebrew
brew install jwwelbor/shark/shark
shark --version
brew uninstall shark

# Linux manual
wget <binary-url>
sha256sum -c checksums.txt --ignore-missing
tar -xzf shark_*.tar.gz
./shark --version

# Windows Scoop
scoop install shark
shark --version
scoop uninstall shark
```

---

## 11. Documentation and Training

### 11.1 Security Documentation

**Required Documentation**:
1. ✅ This security design document (06-security-design.md)
2. ✅ README.md security section (verification instructions)
3. TODO: SECURITY.md (vulnerability reporting)
4. TODO: security verification scripts (scripts/verify-*.sh)

**SECURITY.md Template**:
```markdown
# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0.0 | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability, please email security@<domain> with:
- Description of vulnerability
- Steps to reproduce
- Impact assessment
- Suggested fix (if any)

We will:
- Respond within 48 hours
- Provide timeline for fix
- Credit you in release notes (unless anonymity requested)
- Notify you when fix is released

## Security Advisories

Subscribe to security advisories:
- GitHub: https://github.com/jwwelbor/shark-task-manager/security/advisories
- Email: security@<domain>
```

### 11.2 Maintainer Training

**Security Checklist for Maintainers**:
- [ ] Enable 2FA on GitHub account
- [ ] Use strong, unique password (password manager)
- [ ] Review all PRs for security implications
- [ ] Rotate tokens every 90 days
- [ ] Monitor security alerts
- [ ] Follow incident response playbook if breach detected

**Annual Security Review**:
- Review threat model (any new threats?)
- Update incident response plan
- Test recovery procedures
- Review and rotate all credentials
- Update security documentation

---

## 12. Conclusion

This security design provides defense-in-depth protection for the Shark CLI distribution and release process. Key security controls include:

1. **Checksum verification** for all binary downloads
2. **Scoped GitHub tokens** with minimal permissions
3. **Automated secret scanning** to prevent credential leaks
4. **Branch protection** to prevent unauthorized code changes
5. **Audit trails** via GitHub Actions logs
6. **Incident response plan** for security events

**Security Posture Assessment**:
- ✅ Adequate for open-source CLI tool
- ✅ Meets industry best practices for Go projects
- ⚠️ Some residual risks (acceptable for F08 scope)
- 🔄 Continuous improvement via future enhancements

**Next Steps**:
1. Implement security controls during F08 development
2. Test security verification workflows
3. Document user security guidance
4. Plan for future enhancements (code signing, SLSA)

---

**Security Design Status**: ✅ Ready for Implementation
**Risk Level**: Low (with controls implemented)
**Next Step**: Performance design (07-performance-design.md)
