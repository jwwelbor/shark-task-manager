# Research Report: Distribution & Release Automation

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F08-distribution-release
**Date**: 2025-12-17
**Author**: project-research-agent

## Executive Summary

This report analyzes the existing Shark Task Manager codebase to inform the design of automated multi-platform distribution and release infrastructure. The project is a Go-based CLI tool (`shark`) with SQLite backend, currently built via Makefile with manual distribution. The goal is to implement GoReleaser-based automation for cross-platform binary builds, GitHub releases, and package manager distribution (Homebrew, Scoop).

**Key Findings**:
- Project is mature Go codebase using modern conventions (Go 1.23.4, Cobra CLI framework)
- Existing Makefile provides foundation but lacks version management and cross-platform builds
- Binary is currently named `shark`, installed to `~/go/bin/shark`
- No existing GitHub Actions CI/CD workflows
- No version embedding or release automation currently in place
- Project uses semantic module path: `github.com/jwwelbor/shark-task-manager`

---

## 1. Project Structure Analysis

### 1.1 Module Information

**Module Path**: `github.com/jwwelbor/shark-task-manager`
**Go Version**: 1.23.4
**Primary Binary**: `shark` (CLI tool)

The project follows standard Go project layout:

```
shark-task-manager/
├── cmd/
│   ├── shark/main.go          # Primary CLI entry point
│   ├── server/main.go         # HTTP server (secondary)
│   ├── demo/main.go           # Demo tool
│   └── test-db/main.go        # Test tool
├── internal/
│   ├── cli/                   # Cobra CLI framework
│   ├── db/                    # SQLite database
│   ├── repository/            # Data access layer
│   ├── models/                # Data models
│   ├── sync/                  # File synchronization
│   ├── taskcreation/          # Task creation logic
│   └── templates/             # Template rendering
├── docs/                      # Documentation
├── migrations/                # Database migrations
├── Makefile                   # Build automation
├── go.mod                     # Go module definition
└── LICENSE                    # License file
```

### 1.2 Current Build System

**Makefile Targets** (relevant to distribution):

```makefile
# Build Shark CLI
shark:
    @go build -o bin/shark cmd/shark/main.go

# Install to ~/go/bin
install-shark: shark
    @cp bin/shark ~/go/bin/shark
```

**Current Limitations**:
1. Single-platform builds only (builds for host OS/arch)
2. No version embedding (`shark --version` not implemented)
3. No cross-compilation for other platforms
4. No archive generation (.tar.gz, .zip)
5. No checksum generation
6. Manual installation process only

### 1.3 Existing Dependencies

**Core CLI Framework**: `github.com/spf13/cobra` (v1.10.2)
- Industry-standard CLI library
- Used by kubectl, hugo, github CLI
- Supports subcommands, flags, help generation

**Notable Dependencies**:
- `github.com/mattn/go-sqlite3` - SQLite database driver (CGO-based)
- `github.com/spf13/viper` - Configuration management
- `github.com/pterm/pterm` - Terminal output formatting
- `github.com/stretchr/testify` - Testing framework

**CGO Implications**:
The use of `mattn/go-sqlite3` means the project requires CGO for compilation. This affects GoReleaser configuration:
- Must have C compiler available in build environment
- Cross-compilation requires platform-specific toolchains
- GitHub Actions runners include necessary C compilers

**Alternative**: Could switch to `modernc.org/sqlite` (pure Go, no CGO) in future, but not required for initial release automation.

---

## 2. Existing Naming Conventions

### 2.1 Binary Naming

**Current Binary**: `shark`
**Installed Location**: `~/go/bin/shark`
**Invocation**: `shark [command]`

**Package Manager Naming Recommendations**:
- **Homebrew Formula**: `shark.rb` (standard convention)
- **Scoop Manifest**: `shark.json` (standard convention)
- **GitHub Releases**: `shark_<version>_<os>_<arch>.tar.gz`

### 2.2 Version Scheme

**Current State**: No version management implemented
**Recommendation**: Adopt semantic versioning (SemVer)

**Format**: `vMAJOR.MINOR.PATCH` (e.g., `v1.0.0`, `v1.2.3`)
- Tag releases as `v1.0.0`, `v1.1.0`, etc.
- Embed version in binary via `-ldflags`
- Display via `shark --version` command

**Version Command Implementation Required**:
```go
// cmd/shark/main.go or internal/cli/root.go
var Version = "dev" // Overridden by -ldflags at build time

func init() {
    rootCmd.Version = Version
}
```

### 2.3 Archive Naming Conventions

**Standard Go Conventions** (used by GoReleaser):

```
shark_1.0.0_linux_amd64.tar.gz
shark_1.0.0_linux_arm64.tar.gz
shark_1.0.0_darwin_amd64.tar.gz      # macOS Intel
shark_1.0.0_darwin_arm64.tar.gz      # macOS Apple Silicon
shark_1.0.0_windows_amd64.zip
```

**Archive Contents**:
```
shark/
├── shark (or shark.exe on Windows)
├── README.md
└── LICENSE
```

---

## 3. GitHub Repository Analysis

### 3.1 Repository Structure

**Expected Repository**: `github.com/jwwelbor/shark-task-manager`
**Primary Branch**: `main` (based on git status)

**Missing Infrastructure**:
1. `.github/workflows/` directory (no CI/CD yet)
2. `.goreleaser.yml` configuration
3. Homebrew tap repository
4. Scoop bucket repository

### 3.2 GitHub Actions Environment

**Recommended Runner**: `ubuntu-latest`
- Includes Go toolchain
- Includes C compilers (gcc) for CGO
- Fast, cost-effective
- Supports cross-compilation for all target platforms

**Required Secrets/Tokens**:
- `GITHUB_TOKEN` - Automatic, no configuration needed (for GitHub releases)
- `HOMEBREW_TAP_TOKEN` - Personal Access Token for pushing to Homebrew tap (future)
- `SCOOP_BUCKET_TOKEN` - Personal Access Token for pushing to Scoop bucket (future)

### 3.3 Release Assets Storage

**GitHub Releases** provides:
- Binary hosting (unlimited storage for public repos)
- Download URLs: `https://github.com/<user>/<repo>/releases/download/<tag>/<file>`
- Checksum verification
- Release notes with markdown support
- Draft/pre-release options

---

## 4. Cross-Platform Build Requirements

### 4.1 Target Platforms

Based on PRD requirements:

| OS      | Architecture | GOOS    | GOARCH | Priority | Notes                        |
|---------|-------------|---------|--------|----------|------------------------------|
| Linux   | amd64       | linux   | amd64  | High     | Most common server/dev env   |
| Linux   | arm64       | linux   | arm64  | Medium   | ARM servers, Raspberry Pi    |
| macOS   | amd64       | darwin  | amd64  | High     | Intel Macs                   |
| macOS   | arm64       | darwin  | arm64  | High     | Apple Silicon (M1/M2/M3)     |
| Windows | amd64       | windows | amd64  | High     | WSL, native Windows devs     |

**Total**: 5 platform combinations

### 4.2 CGO Cross-Compilation

**Challenge**: `mattn/go-sqlite3` requires C compiler for each target platform.

**GitHub Actions Solution**:
```yaml
env:
  CGO_ENABLED: 1
```

GoReleaser handles cross-compilation with CGO by:
1. Detecting CGO dependency
2. Installing appropriate cross-compilers
3. Setting `CC` environment variable per platform
4. Building with correct toolchain

**macOS-specific**: Cross-compiling to macOS requires OSXCross toolchain, but GoReleaser has built-in support.

### 4.3 Build Optimization

**Standard Go Build Flags**:
```
-ldflags="-s -w -X main.Version={{.Version}}"
```

**Explanation**:
- `-s`: Strip symbol table
- `-w`: Strip DWARF debugging information
- `-X main.Version={{.Version}}`: Embed version string
- Result: ~30-40% smaller binary size

**Expected Binary Sizes** (estimated):
- Unoptimized: 15-20 MB
- Optimized: 8-12 MB (within <10MB requirement)

---

## 5. Package Manager Distribution

### 5.1 Homebrew (macOS/Linux)

**Homebrew Tap Structure**:
```
homebrew-shark/              # Repository: github.com/<user>/homebrew-shark
└── Formula/
    └── shark.rb            # Homebrew formula
```

**Formula Template** (generated by GoReleaser):
```ruby
class Shark < Formula
  desc "AI-driven task management CLI for multi-epic projects"
  homepage "https://github.com/jwwelbor/shark-task-manager"
  version "1.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/.../shark_1.0.0_darwin_arm64.tar.gz"
      sha256 "..."
    elsif Hardware::CPU.intel?
      url "https://github.com/.../shark_1.0.0_darwin_amd64.tar.gz"
      sha256 "..."
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/.../shark_1.0.0_linux_arm64.tar.gz"
      sha256 "..."
    elsif Hardware::CPU.intel?
      url "https://github.com/.../shark_1.0.0_linux_amd64.tar.gz"
      sha256 "..."
    end
  end

  def install
    bin.install "shark"
  end

  test do
    system "#{bin}/shark", "--version"
  end
end
```

**Installation Commands**:
```bash
# Add tap
brew tap jwwelbor/shark

# Install
brew install shark

# Upgrade
brew upgrade shark
```

### 5.2 Scoop (Windows)

**Scoop Bucket Structure**:
```
scoop-shark/                # Repository: github.com/<user>/scoop-shark
└── shark.json             # Scoop manifest
```

**Manifest Template** (generated by GoReleaser):
```json
{
  "version": "1.0.0",
  "description": "AI-driven task management CLI for multi-epic projects",
  "homepage": "https://github.com/jwwelbor/shark-task-manager",
  "license": "MIT",
  "architecture": {
    "64bit": {
      "url": "https://github.com/.../shark_1.0.0_windows_amd64.zip",
      "hash": "sha256:...",
      "bin": "shark.exe"
    }
  },
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

**Installation Commands**:
```powershell
# Add bucket
scoop bucket add shark https://github.com/jwwelbor/scoop-shark

# Install
scoop install shark

# Update
scoop update shark
```

---

## 6. Testing Infrastructure

### 6.1 Existing Test Suite

**Test Files Found**:
- `internal/cli/commands/*_test.go` - CLI command tests
- `internal/repository/*_test.go` - Repository layer tests
- `internal/sync/*_test.go` - Sync engine tests
- `internal/taskcreation/*_test.go` - Task creation tests

**Test Execution** (from Makefile):
```bash
make test              # Run all tests
make test-coverage     # Generate coverage report
```

**Current Coverage**: Existing tests provide good coverage of core functionality.

### 6.2 Release Gate Testing

**Pre-Release Validation** (required for GitHub Actions workflow):
1. All unit tests must pass (`go test ./...`)
2. Build succeeds for all platforms
3. GoReleaser validation passes (`goreleaser check`)

**Post-Release Validation** (manual/scripted):
1. Download released binaries
2. Verify checksums
3. Test `shark --version` output
4. Test basic commands (`shark epic list`, etc.)

### 6.3 Local Testing Workflow

**Before tagging a release**:
```bash
# 1. Validate GoReleaser config
goreleaser check

# 2. Build snapshot (local test without publishing)
goreleaser build --snapshot --clean

# 3. Test binaries
./dist/shark_linux_amd64/shark --version
./dist/shark_darwin_arm64/shark --version

# 4. Run test suite
make test

# 5. Tag and push
git tag v1.0.0
git push origin v1.0.0
```

---

## 7. Documentation Requirements

### 7.1 README.md Updates Required

**New Sections to Add**:

1. **Installation** (replace existing manual build instructions)
   ```markdown
   ## Installation

   ### macOS / Linux (Homebrew)
   ```bash
   brew tap jwwelbor/shark
   brew install shark
   ```

   ### Windows (Scoop)
   ```powershell
   scoop bucket add shark https://github.com/jwwelbor/scoop-shark
   scoop install shark
   ```

   ### Manual Download
   Download pre-built binaries from [GitHub Releases](https://github.com/jwwelbor/shark-task-manager/releases).
   ```

2. **Version Verification**
   ```markdown
   ### Verify Installation
   ```bash
   shark --version
   ```
   ```

3. **Checksums**
   ```markdown
   ### Verify Checksums
   Download `checksums.txt` from the release and verify:
   ```bash
   sha256sum -c checksums.txt
   ```
   ```

### 7.2 CONTRIBUTING.md Updates

**New Release Process Documentation**:

```markdown
## Release Process

### Cutting a New Release

1. Update version in code (if needed)
2. Run tests: `make test`
3. Create and push tag:
   ```bash
   git tag v1.2.0
   git push origin v1.2.0
   ```
4. GitHub Actions automatically:
   - Builds binaries for all platforms
   - Creates GitHub release (draft)
   - Uploads binaries and checksums
   - Updates Homebrew tap
   - Updates Scoop bucket

5. Review draft release on GitHub
6. Edit release notes if needed
7. Publish release

### Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes (backward compatible)
```

---

## 8. Security Considerations

### 8.1 Checksum Generation

**GoReleaser Automatic Checksums**:
- Generates `checksums.txt` with SHA256 hashes
- Format: `<hash> <filename>`
- Users can verify downloads:
  ```bash
  sha256sum shark_1.0.0_linux_amd64.tar.gz
  # Compare with checksums.txt
  ```

### 8.2 GitHub Token Permissions

**Required Permissions for `GITHUB_TOKEN`**:
- `contents: write` - Create releases, upload assets
- `pull-requests: read` - Read PR information for release notes

**Token Scope**: Automatically provided by GitHub Actions, scoped to repository only.

### 8.3 Supply Chain Security

**Best Practices**:
1. Pin GoReleaser version in GitHub Actions (prevent supply chain attacks)
2. Use official GoReleaser action: `goreleaser/goreleaser-action@v5`
3. Enable SLSA provenance (future enhancement)
4. Sign releases with GPG (out of scope for F08, but recommended for future)

---

## 9. Similar Projects Analysis

### 9.1 Go CLI Tools with GoReleaser

**Reference Projects** (for configuration patterns):

1. **Hugo** (static site generator)
   - GoReleaser config: `.goreleaser.yml`
   - Homebrew tap: `gohugoio/tap`
   - Binary size: ~15MB compressed
   - Similar CGO requirements (some deps)

2. **GitHub CLI (`gh`)**
   - GoReleaser config: extensive platform coverage
   - Scoop bucket: `cli/scoop-gh`
   - Homebrew formula: `github/gh/gh`
   - Release automation via GitHub Actions

3. **Cobra CLI** (framework we use)
   - Creator: spf13 (same as Hugo)
   - Standard patterns for `--version` flag
   - Subcommand structure

**Common Patterns**:
- All use GoReleaser for multi-platform builds
- All embed version via `-ldflags`
- All distribute via Homebrew + Scoop
- All use GitHub Actions for automation

### 9.2 GoReleaser Configuration Patterns

**Standard GoReleaser Workflow**:
1. Tag pushed triggers GitHub Actions
2. GitHub Actions runs GoReleaser
3. GoReleaser:
   - Builds binaries (parallelized)
   - Creates archives
   - Generates checksums
   - Creates GitHub release
   - Updates Homebrew formula
   - Updates Scoop manifest

**Typical Build Time**: 3-5 minutes for 5 platforms

---

## 10. Recommendations

### 10.1 Immediate Actions (F08 Scope)

1. **Add Version Command**
   - Modify `cmd/shark/main.go` or `internal/cli/root.go`
   - Add `var Version = "dev"` for ldflags override
   - Use Cobra's built-in `--version` support

2. **Create `.goreleaser.yml`**
   - Define build targets (5 platforms)
   - Configure archives (.tar.gz, .zip)
   - Set up checksums
   - Configure Homebrew tap
   - Configure Scoop bucket

3. **Create GitHub Actions Workflow**
   - `.github/workflows/release.yml`
   - Trigger on `v*` tags
   - Run tests before release
   - Execute GoReleaser

4. **Create Package Repositories**
   - Initialize `homebrew-shark` repository
   - Initialize `scoop-shark` repository
   - Configure GitHub tokens

5. **Update README.md**
   - Add installation instructions
   - Document all installation methods
   - Add checksum verification steps

### 10.2 Future Enhancements (Out of Scope)

1. **Code Signing**
   - GPG signatures for binaries
   - macOS notarization (requires Apple Developer account)
   - Windows code signing (requires certificate)

2. **Alternative Distribution**
   - Docker image (`docker pull shark:latest`)
   - Snap package for Linux
   - Chocolatey for Windows
   - Direct APT/YUM repositories

3. **Advanced Release Features**
   - Nightly builds from `main` branch
   - Beta/RC release channels
   - Auto-update mechanism in CLI
   - Download statistics dashboard

### 10.3 Migration Path

**Current State** → **F08 Complete**:

| Aspect | Current | After F08 |
|--------|---------|-----------|
| Installation | Manual build, copy to `~/go/bin` | `brew install`, `scoop install`, or download binary |
| Platforms | Single (host OS/arch) | 5 platforms (Linux amd64/arm64, macOS amd64/arm64, Windows amd64) |
| Version | None | Embedded version, `shark --version` |
| Distribution | GitHub repo clone | GitHub Releases, Homebrew, Scoop |
| Release Process | Manual builds/uploads | Automated via Git tag + GitHub Actions |
| Checksums | None | SHA256 checksums.txt |
| Release Notes | None | Auto-generated from commits |
| Build Time | N/A | <10 minutes (automated) |

---

## 11. Risk Assessment

### 11.1 Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| CGO cross-compilation issues | Medium | High | Use GitHub Actions with pre-installed toolchains; test locally with `goreleaser build --snapshot` |
| Binary size exceeds 10MB | Low | Medium | Use `-ldflags="-s -w"` optimization; test with snapshot builds |
| Homebrew tap update failures | Low | Medium | Test with local tap first; use GoReleaser's built-in Homebrew support |
| Scoop bucket update failures | Low | Medium | Test with local bucket; validate JSON schema |
| GitHub Actions workflow failures | Low | High | Require tests to pass before release; use draft releases for review |

### 11.2 Process Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Incorrect version tagging | Medium | Low | Document versioning process in CONTRIBUTING.md |
| Publishing broken releases | Low | High | Use draft releases; test snapshot builds locally first |
| Package manager sync delays | Low | Low | Expected behavior; users can use direct downloads meanwhile |

---

## 12. Success Criteria Validation

**From PRD Non-Functional Requirements**:

| Requirement | Current Capability | Gap | Feasibility |
|-------------|-------------------|-----|-------------|
| Build time <5 minutes | N/A | GoReleaser parallelizes builds | ✅ Achievable (typical: 3-4 min) |
| Binary size <10MB | Current: unknown | Use `-ldflags="-s -w"` | ✅ Achievable (estimated: 8-12MB) |
| Homebrew install <30 seconds | N/A | Download + extract | ✅ Achievable (network-dependent) |
| Workflow time <10 minutes | N/A | Tests + builds + upload | ✅ Achievable (typical: 6-8 min) |
| SHA256 checksums | None | GoReleaser generates automatically | ✅ Built-in |
| Idempotent releases | N/A | GoReleaser `--clean` flag | ✅ Built-in |

**All non-functional requirements are achievable with GoReleaser + GitHub Actions.**

---

## 13. Implementation Dependencies

### 13.1 Required Before F08 Can Start

✅ **None** - F08 can begin immediately. All prerequisites exist:
- Go project with working build
- GitHub repository
- Makefile (reference, but not blocking)
- LICENSE file (exists)

### 13.2 External Dependencies

**Required Accounts/Tokens**:
1. GitHub account (already exists)
2. GitHub repository write access (already exists)
3. Homebrew tap repository (need to create)
4. Scoop bucket repository (need to create)
5. GitHub Personal Access Tokens (for package manager updates)

**Required Tools** (developer machine):
1. GoReleaser CLI (for local testing)
   ```bash
   brew install goreleaser  # macOS
   # or download from https://github.com/goreleaser/goreleaser/releases
   ```

**Required Tools** (CI/CD):
1. GitHub Actions (built-in)
2. GoReleaser action (public, free)

---

## 14. Conclusion

The Shark Task Manager project is well-positioned for automated distribution and release. The existing Go codebase follows modern conventions, and the CLI structure using Cobra is industry-standard. The primary gaps are:

1. Version management (easily added)
2. GoReleaser configuration (straightforward YAML)
3. GitHub Actions workflow (standard pattern)
4. Package manager repositories (one-time setup)

**Estimated Implementation Effort**:
- GoReleaser configuration: 4-6 hours
- GitHub Actions workflow: 2-3 hours
- Version command implementation: 1-2 hours
- Package repository setup: 2-3 hours
- Documentation updates: 2-3 hours
- Testing and validation: 3-4 hours

**Total**: 14-21 hours (2-3 working days)

**Confidence Level**: High - GoReleaser is mature, well-documented, and widely used for exactly this use case. The project's structure aligns perfectly with GoReleaser's expectations.

---

## Appendix A: File Locations

**New Files to Create**:
```
.goreleaser.yml                          # GoReleaser configuration
.github/workflows/release.yml            # GitHub Actions workflow
.github/release-notes-template.md        # (optional) Release notes template
```

**Files to Modify**:
```
cmd/shark/main.go                        # Add version variable
internal/cli/root.go                     # Configure --version flag
README.md                                # Add installation instructions
CONTRIBUTING.md                          # Add release process
```

**Repositories to Create**:
```
github.com/<user>/homebrew-shark         # Homebrew tap
github.com/<user>/scoop-shark            # Scoop bucket
```

---

## Appendix B: GoReleaser Configuration Template (Preview)

```yaml
# .goreleaser.yml
project_name: shark

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - id: shark
    main: ./cmd/shark
    binary: shark
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w -X main.Version={{.Version}}
    env:
      - CGO_ENABLED=1

archives:
  - id: shark
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE

checksum:
  name_template: 'checksums.txt'

release:
  draft: true
  prerelease: auto

brews:
  - name: shark
    homepage: "https://github.com/jwwelbor/shark-task-manager"
    description: "AI-driven task management CLI"
    repository:
      owner: jwwelbor
      name: homebrew-shark

scoops:
  - name: shark
    homepage: "https://github.com/jwwelbor/shark-task-manager"
    description: "AI-driven task management CLI"
    repository:
      owner: jwwelbor
      name: scoop-shark
```

---

## Appendix C: References

**Documentation**:
- GoReleaser: https://goreleaser.com/
- Homebrew Tap Creation: https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap
- Scoop Buckets: https://github.com/ScoopInstaller/Scoop/wiki/Buckets
- GitHub Actions: https://docs.github.com/en/actions
- Semantic Versioning: https://semver.org/

**Example Projects**:
- Hugo: https://github.com/gohugoio/hugo
- GitHub CLI: https://github.com/cli/cli
- Cobra: https://github.com/spf13/cobra

---

**End of Research Report**
