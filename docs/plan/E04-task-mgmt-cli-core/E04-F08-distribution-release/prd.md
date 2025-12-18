# Feature: Distribution & Release Automation

## Epic

- [Epic PRD](../epic.md)

## Goal

### Problem

The Shark CLI tool needs to be easily installable across Linux, macOS, and Windows without requiring users to have Go installed or know how to compile binaries. Manual release processes (building binaries for each platform, creating GitHub releases, updating package managers) are error-prone, time-consuming, and inconsistent. Developers expect modern installation workflows: `brew install shark` on macOS, `scoop install shark` on Windows, or downloading a pre-built binary. Without automated distribution, users face complex installation steps (clone repo, install Go, run build commands), significantly reducing adoption and creating support burden.

### Solution

Implement automated multi-platform distribution using GoReleaser, a Go-specific release automation tool. GoReleaser will build binaries for all supported platforms (Linux amd64/arm64, macOS amd64/arm64, Windows amd64), generate checksums, create GitHub releases with release notes, and publish to package managers (Homebrew tap for Mac/Linux, Scoop bucket for Windows). The automation triggers on Git tags (e.g., `v1.0.0`), runs in GitHub Actions CI/CD, and produces installation-ready packages within minutes. Users install via their platform's native package manager or download pre-built binaries from GitHub releases.

### Impact

- **Developer Experience**: One-command installation (`brew install shark` or `scoop install shark`) reduces setup time from 10+ minutes to <30 seconds
- **Adoption**: Eliminates Go dependency requirement, making the tool accessible to non-Go developers and AI agents
- **Release Velocity**: Automated releases reduce release time from 2+ hours (manual builds/uploads) to <10 minutes (tag and wait)
- **Consistency**: All platforms receive identical versions simultaneously, eliminating version skew and platform-specific bugs
- **Trust**: Signed checksums and GitHub release provenance build user confidence in binary authenticity

## User Personas

### Primary Persona: Product Manager / Technical Lead

**Role**: Human developer installing Shark CLI for the first time
**Environment**: macOS, Linux, or Windows workstation

**Key Characteristics**:
- Expects modern package manager installation
- May not have Go installed (and shouldn't need to)
- Values quick setup without complex build steps
- Trusts established package managers (Homebrew, Scoop)

**Goals**:
- Install Shark CLI with single command
- Get updates automatically through package manager
- Verify binary authenticity with checksums
- Read installation instructions clearly

**Pain Points this Feature Addresses**:
- No pre-built binaries available
- Forced to install Go toolchain just to build one tool
- Manual compilation with unclear instructions
- Uncertainty about binary authenticity

### Secondary Persona: CI/CD Pipeline

**Role**: Automated system installing Shark CLI in GitHub Actions or GitLab CI
**Environment**: Linux containers (Docker), ephemeral environments

**Key Characteristics**:
- Needs fast, reliable installation
- Cannot interact with prompts or installers
- Runs on minimal base images
- Requires specific versions (not just "latest")

**Goals**:
- Install specific PM version in <5 seconds
- Use pre-built binaries (no compilation)
- Verify checksums automatically
- Cache binaries for faster builds

**Pain Points this Feature Addresses**:
- No versioned binary downloads
- Slow compilation in CI environments
- No checksum verification
- Unclear installation for automation

### Tertiary Persona: Release Manager

**Role**: Developer responsible for cutting new PM releases
**Environment**: Git workflow, GitHub repository

**Key Characteristics**:
- Creates releases infrequently (monthly or per-milestone)
- Needs release process to be reliable and consistent
- Reviews changelogs and release notes
- Monitors package manager publication status

**Goals**:
- Trigger release with single Git tag
- Automatically build all platform binaries
- Generate release notes from commit history
- Publish to Homebrew and Scoop simultaneously

**Pain Points this Feature Addresses**:
- Manual building for 6+ platform combinations
- Forgetting to update package managers
- Inconsistent release note formatting
- Error-prone multi-step release process

## User Stories

### Must-Have User Stories

**Story 1: One-Command Installation (Homebrew)**
- As a macOS/Linux user, I want to run `brew install <user>/shark/shark`, so that I can install Shark CLI without downloading or compiling anything manually.

**Story 2: One-Command Installation (Scoop)**
- As a Windows user, I want to run `scoop install shark`, so that I can install Shark CLI using my existing package manager.

**Story 3: Download Pre-Built Binary**
- As a user on any platform, I want to download a pre-built binary from GitHub releases, so that I can install without a package manager.

**Story 4: Automated Release on Git Tag**
- As a release manager, I want to create a Git tag (e.g., `v1.2.0`) and push it, so that GoReleaser automatically builds all binaries and publishes the release.

**Story 5: Verify Binary Checksums**
- As a security-conscious user, I want to verify binary checksums from `checksums.txt`, so that I can ensure my download wasn't tampered with.

**Story 6: Cross-Platform Builds**
- As a release manager, I want GoReleaser to build binaries for Linux (amd64, arm64), macOS (amd64, arm64), and Windows (amd64), so that all users get native binaries.

**Story 7: Automated Homebrew Tap**
- As a macOS user, I want the Homebrew tap to update automatically when a new version is released, so that `brew upgrade shark` gets the latest version.

**Story 8: Automated Scoop Bucket**
- As a Windows user, I want the Scoop bucket to update automatically when a new version is released, so that `scoop update shark` gets the latest version.

### Should-Have User Stories

**Story 9: Release Notes Generation**
- As a user, I want to see auto-generated release notes on GitHub releases, so that I know what changed in each version.

**Story 10: Installation Verification**
- As a user, I want to run `pm --version` after installation, so that I can confirm the installation succeeded and check the version.

**Story 11: Installation Documentation**
- As a new user, I want clear installation instructions in README.md for all platforms, so that I know the recommended installation method.

**Story 12: Archive Formats**
- As a user, I want binaries packaged as `.tar.gz` (Linux/macOS) and `.zip` (Windows), so that I can extract them easily on my platform.

### Could-Have User Stories

**Story 13: Snap Package (Linux)**
- As a Linux user, I want to install via `snap install shark`, so that I can use my preferred package manager.

**Story 14: Docker Image**
- As a CI/CD user, I want a Docker image (`pm:latest`), so that I can use PM in containerized environments without installation.

**Story 15: Version Pinning in CI**
- As a CI user, I want to download specific versions via URL pattern (e.g., `releases/download/v1.2.0/pm_linux_amd64`), so that my builds are reproducible.

## Requirements

### Functional Requirements

**GoReleaser Configuration:**

1. The system must define a `.goreleaser.yml` configuration file at repository root

2. The configuration must specify build targets:
   - Linux: amd64, arm64
   - macOS: amd64 (Intel), arm64 (Apple Silicon)
   - Windows: amd64

3. The configuration must specify binary name as `shark` (with `.exe` suffix for Windows)

4. The configuration must generate archives:
   - Format: `.tar.gz` for Linux/macOS, `.zip` for Windows
   - Contents: Binary, README.md, LICENSE

5. The configuration must generate `checksums.txt` with SHA256 hashes for all artifacts

**Build Process:**

6. The system must build all platform binaries from a single Git tag push

7. Builds must use Go cross-compilation: `GOOS=<os> GOARCH=<arch> go build`

8. Builds must embed version information using `-ldflags "-X main.Version=<tag>"`

9. The build process must strip debug symbols for smaller binary size: `-ldflags "-s -w"`

10. The build process must complete in <5 minutes for all platforms

**GitHub Releases:**

11. The system must create a GitHub release automatically when a tag is pushed (via GitHub Actions)

12. The release must include:
    - All platform binaries (6 archives)
    - `checksums.txt` file
    - Auto-generated release notes from commit messages

13. Release notes must include:
    - Version number
    - Date
    - List of commits since last tag
    - Installation instructions

14. The release must be marked as "draft" initially, allowing manual review before publication

**Homebrew Tap:**

15. The system must publish to a Homebrew tap at `https://github.com/<user>/homebrew-shark`

16. GoReleaser must generate a Homebrew formula (`pm.rb`) with:
    - Binary download URLs for macOS (amd64, arm64)
    - SHA256 checksums
    - Installation instructions

17. The formula must support `brew install <user>/pm/shark` syntax

18. The tap must auto-update when GoReleaser publishes a new release

**Scoop Bucket:**

19. The system must publish to a Scoop bucket at `https://github.com/<user>/scoop-shark`

20. GoReleaser must generate a Scoop manifest (`pm.json`) with:
    - Binary download URL for Windows amd64
    - SHA256 checksum
    - Installation instructions

21. The manifest must support `scoop bucket add pm <user>/scoop-shark` followed by `scoop install shark`

22. The bucket must auto-update when GoReleaser publishes a new release

**Version Management:**

23. The system must read version from Git tags (format: `v1.2.3`)

24. The CLI must display version with `pm --version` command (value embedded at build time)

25. The version must follow semantic versioning (MAJOR.MINOR.PATCH)

**CI/CD Integration:**

26. The system must define a GitHub Actions workflow (`.github/workflows/release.yml`)

27. The workflow must trigger on:
    - Push of tags matching `v*` pattern (e.g., `v1.0.0`)

28. The workflow must:
    - Check out code
    - Set up Go environment
    - Run tests
    - Run GoReleaser with `--clean` flag
    - Publish to GitHub releases, Homebrew, and Scoop

29. The workflow must use GitHub token authentication for releases

30. The workflow must fail if tests don't pass (release gate)

**Installation Documentation:**

31. README.md must include installation instructions for all platforms

32. Instructions must cover:
    - Homebrew installation (macOS/Linux)
    - Scoop installation (Windows)
    - Manual binary download
    - Installation verification (`pm --version`)

33. Instructions must include adding package manager repositories (tap/bucket) if needed

### Non-Functional Requirements

**Performance:**

- GoReleaser build process must complete in <5 minutes
- Binary download size must be <10MB per platform
- Homebrew/Scoop installation must complete in <30 seconds
- GitHub Actions workflow must complete in <10 minutes (including tests)

**Reliability:**

- Release process must be idempotent (re-running for same tag produces identical artifacts)
- Checksum verification must catch any corrupted downloads
- Failed releases must not partially publish (all-or-nothing)
- CI workflow must retry transient GitHub API failures

**Security:**

- All binaries must have SHA256 checksums published
- GitHub releases must be created via authenticated GitHub Actions (not manual uploads)
- Homebrew/Scoop manifests must reference checksums for verification
- No secrets or credentials embedded in binaries

**Usability:**

- Installation instructions must be platform-specific (don't show macOS steps to Windows users)
- Error messages from package managers must be clear (e.g., "pm not found" → suggests adding tap/bucket)
- Release notes must be readable by non-technical users
- Version numbering must be consistent across all platforms

**Compatibility:**

- Homebrew: macOS 11+, Linux with Homebrew installed
- Scoop: Windows 10+, PowerShell 5.0+
- Manual downloads: Any platform with supported architecture
- GitHub Actions: Ubuntu latest runner

**Maintainability:**

- `.goreleaser.yml` must have comments explaining each section
- GitHub Actions workflow must have clear step names
- Version bumping process must be documented in CONTRIBUTING.md
- Template files for release notes must be version-controlled

## Acceptance Criteria

### GoReleaser Configuration

**Given** I create `.goreleaser.yml` in repository root
**When** I run `goreleaser check`
**Then** the configuration is valid with no errors
**And** build targets include Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)

### Automated Release Build

**Given** I create and push a Git tag `v1.0.0`
**When** GitHub Actions workflow runs
**Then** GoReleaser builds binaries for all 6 platform combinations
**And** all binaries are uploaded to GitHub release
**And** `checksums.txt` is generated and uploaded

### Homebrew Installation

**Given** a new release `v1.0.0` is published
**When** I run `brew install <user>/pm/shark`
**Then** the `shark` binary is installed to `/opt/homebrew/bin/shark` (Apple Silicon) or `/usr/local/bin/shark` (Intel)
**And** running `pm --version` outputs `v1.0.0`

### Scoop Installation

**Given** a new release `v1.0.0` is published
**When** I run `scoop bucket add pm <user>/scoop-shark` and `scoop install shark`
**Then** the `pm.exe` binary is installed to `~/scoop/apps/pm/current/pm.exe`
**And** running `pm --version` outputs `v1.0.0`

### Manual Binary Download

**Given** I navigate to GitHub releases page
**When** I download `pm_linux_amd64.tar.gz`
**And** I verify checksum matches entry in `checksums.txt`
**And** I extract and run `pm --version`
**Then** the output is `v1.0.0`

### Checksum Verification

**Given** I download `pm_darwin_arm64.tar.gz` and `checksums.txt`
**When** I run `sha256sum pm_darwin_arm64.tar.gz` and compare to `checksums.txt`
**Then** the checksums match exactly

### Release Notes Generation

**Given** I push tag `v1.0.0` with 5 commits since last tag
**When** the GitHub release is created
**Then** release notes include all 5 commit messages
**And** release notes include installation instructions
**And** release notes include checksums section

### Version Embedding

**Given** I build with `-ldflags "-X main.Version=v1.0.0"`
**When** I run `pm --version`
**Then** the output is exactly `v1.0.0` (not "dev" or "unknown")

### GitHub Actions Workflow

**Given** I push tag `v1.0.0`
**When** the release workflow runs
**Then** the workflow completes successfully within 10 minutes
**And** GitHub release is created with all artifacts
**And** Homebrew tap is updated
**And** Scoop bucket is updated

### Cross-Platform Builds

**Given** I run GoReleaser locally with `goreleaser build --snapshot --clean`
**When** the build completes
**Then** `dist/` contains binaries for all 6 platforms:
- `pm_linux_amd64/shark`
- `pm_linux_arm64/shark`
- `pm_darwin_amd64/shark`
- `pm_darwin_arm64/shark`
- `pm_windows_amd64/pm.exe`

## Out of Scope

### Explicitly NOT Included in This Feature

1. **Package Signing** - GPG signatures for binaries (future enhancement for security-critical users)

2. **Notarization** - Apple notarization for macOS binaries (requires Apple Developer account)

3. **APT/YUM Repositories** - Native Linux package repositories (Homebrew on Linux is sufficient)

4. **Snap/Flatpak/AppImage** - Alternative Linux packaging formats (deferred to user demand)

5. **Windows Installer** - MSI or EXE installer (Scoop is sufficient, users can use binary directly)

6. **Auto-Update Mechanism** - Built-in update checking or self-update command (use package manager updates)

7. **Telemetry** - Download statistics or usage metrics (GitHub releases provides download counts)

8. **Multiple Homebrew Taps** - Separate taps for stable/beta versions (single tap with versioning)

9. **Beta/RC Releases** - Pre-release distribution channels (only stable releases in package managers)

10. **Docker Image** - Containerized distribution (deferred to E05 if needed)

11. **Nightly Builds** - Automated daily builds from main branch (only tagged releases)

12. **Mirror Hosting** - CDN or alternative download mirrors (GitHub releases is sufficient)
