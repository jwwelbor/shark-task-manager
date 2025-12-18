# Architecture Design: Distribution & Release Automation

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F08-distribution-release
**Date**: 2025-12-17
**Author**: backend-architect

## Executive Summary

This document defines the architecture for automated multi-platform distribution and release of the Shark CLI tool. The system uses GoReleaser as the core automation engine, triggered by Git tags via GitHub Actions, producing cross-platform binaries distributed through GitHub Releases, Homebrew (macOS/Linux), and Scoop (Windows).

**Architecture Pattern**: Event-Driven CI/CD Pipeline
**Core Technology**: GoReleaser + GitHub Actions
**Distribution Channels**: 3 (GitHub Releases, Homebrew, Scoop)
**Supported Platforms**: 5 (Linux amd64/arm64, macOS amd64/arm64, Windows amd64)

---

## 1. System Architecture

### 1.1 High-Level Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                          RELEASE TRIGGER                             │
│                                                                      │
│  Developer Machine                                                   │
│  ┌────────────┐         ┌─────────────┐                            │
│  │ Git Tag    │────────▶│ Git Push    │                            │
│  │ v1.0.0     │         │ origin      │                            │
│  └────────────┘         └─────────────┘                            │
│                                │                                     │
└────────────────────────────────┼─────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       GITHUB ACTIONS CI/CD                           │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ Release Workflow (.github/workflows/release.yml)             │  │
│  │                                                               │  │
│  │  1. Trigger on tag push (v*)                                 │  │
│  │  2. Checkout code                                            │  │
│  │  3. Setup Go 1.23+                                           │  │
│  │  4. Run tests (go test ./...)                                │  │
│  │  5. Run GoReleaser                                           │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                │                                     │
│                                ▼                                     │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ GoReleaser Engine (.goreleaser.yml)                          │  │
│  │                                                               │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐             │  │
│  │  │ Build      │  │ Archive    │  │ Checksum   │             │  │
│  │  │ 5 Binaries │─▶│ .tar.gz    │─▶│ SHA256     │             │  │
│  │  │            │  │ .zip       │  │            │             │  │
│  │  └────────────┘  └────────────┘  └────────────┘             │  │
│  │                                                               │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                │                                     │
└────────────────────────────────┼─────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      DISTRIBUTION CHANNELS                           │
│                                                                      │
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────────┐  │
│  │ GitHub Releases  │  │ Homebrew Tap     │  │ Scoop Bucket    │  │
│  │                  │  │                  │  │                 │  │
│  │ • 5 archives     │  │ • shark.rb       │  │ • shark.json    │  │
│  │ • checksums.txt  │  │ • Auto-updated   │  │ • Auto-updated  │  │
│  │ • Release notes  │  │                  │  │                 │  │
│  └──────────────────┘  └──────────────────┘  └─────────────────┘  │
│         │                       │                      │            │
└─────────┼───────────────────────┼──────────────────────┼────────────┘
          │                       │                      │
          ▼                       ▼                      ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         END USERS                                    │
│                                                                      │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────┐        │
│  │ Manual       │     │ Homebrew     │     │ Scoop        │        │
│  │ Download     │     │ Install      │     │ Install      │        │
│  │              │     │              │     │              │        │
│  │ curl/wget    │     │ brew install │     │ scoop install│        │
│  └──────────────┘     └──────────────┘     └──────────────┘        │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Responsibilities

| Component | Responsibility | Owner |
|-----------|----------------|-------|
| **Git Tags** | Version marker, release trigger | Developer |
| **GitHub Actions** | CI/CD orchestration, test execution | GitHub |
| **GoReleaser** | Build automation, artifact generation | GoReleaser Action |
| **GitHub Releases** | Binary hosting, download distribution | GitHub |
| **Homebrew Tap** | macOS/Linux package management | GoReleaser + GitHub |
| **Scoop Bucket** | Windows package management | GoReleaser + GitHub |

---

## 2. Core Components

### 2.1 GoReleaser Configuration (`.goreleaser.yml`)

**Purpose**: Define build targets, archive formats, and distribution channels.

**Location**: Repository root (`/.goreleaser.yml`)

**Key Sections**:

```yaml
# Project metadata
project_name: shark

# Pre-build hooks (validation)
before:
  hooks:
    - go mod tidy
    - go test ./...

# Build configuration (5 platforms)
builds:
  - id: shark
    main: ./cmd/shark
    binary: shark
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64  # Windows ARM64 not common yet
    ldflags:
      - -s -w
      - -X main.Version={{.Version}}
      - -X main.Commit={{.Commit}}
      - -X main.Date={{.Date}}
    env:
      - CGO_ENABLED=1  # Required for mattn/go-sqlite3

# Archive generation
archives:
  - id: shark
    name_template: "shark_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
    files:
      - README.md
      - LICENSE

# Checksum generation
checksum:
  name_template: 'checksums.txt'
  algorithm: sha256

# GitHub release configuration
release:
  github:
    owner: jwwelbor
    name: shark-task-manager
  draft: true  # Allow manual review before publish
  prerelease: auto  # Detect from version (e.g., v1.0.0-beta)
  name_template: "{{.ProjectName}} v{{.Version}}"

# Homebrew tap
brews:
  - name: shark
    homepage: "https://github.com/jwwelbor/shark-task-manager"
    description: "AI-driven task management CLI for multi-epic projects"
    license: "MIT"
    repository:
      owner: jwwelbor
      name: homebrew-shark
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    folder: Formula
    test: |
      system "#{bin}/shark", "--version"
    install: |
      bin.install "shark"

# Scoop bucket
scoops:
  - name: shark
    homepage: "https://github.com/jwwelbor/shark-task-manager"
    description: "AI-driven task management CLI for multi-epic projects"
    license: "MIT"
    repository:
      owner: jwwelbor
      name: scoop-shark
      token: "{{ .Env.SCOOP_BUCKET_TOKEN }}"
```

**Design Decisions**:

1. **CGO_ENABLED=1**: Required for SQLite3 dependency. GitHub Actions runners include C compilers for all target platforms.

2. **Draft Releases**: Prevents accidental publishing of broken releases. Developer can review and edit before making public.

3. **Version Embedding**: Uses GoReleaser template variables (`{{.Version}}`, `{{.Commit}}`, `{{.Date}}`) to embed build metadata into binary.

4. **Windows ARM64 Excluded**: Not common enough to justify complexity. Can add later if demand exists.

5. **Archive Contents**: Minimal (binary + README + LICENSE). Documentation lives in repository and website.

### 2.2 GitHub Actions Workflow (`.github/workflows/release.yml`)

**Purpose**: Orchestrate release process on tag push.

**Trigger**: Git tags matching `v*` pattern (e.g., `v1.0.0`, `v1.2.3`)

**Workflow Structure**:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write  # Required for creating releases

jobs:
  release:
    name: Build and Release
    runs-on: ubuntu-latest

    steps:
      - name: Checkout Code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for release notes

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'
          cache: true

      - name: Run Tests
        run: go test -v ./...

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
          SCOOP_BUCKET_TOKEN: ${{ secrets.SCOOP_BUCKET_TOKEN }}
```

**Design Decisions**:

1. **ubuntu-latest Runner**: Includes all necessary build tools (Go, GCC, cross-compilers). Fast and cost-effective.

2. **fetch-depth: 0**: Full Git history required for:
   - Generating release notes from commits
   - Determining previous tag for changelog
   - GoReleaser version detection

3. **Test Gate**: Tests run before GoReleaser. If tests fail, workflow stops and release is aborted.

4. **Secret Management**:
   - `GITHUB_TOKEN`: Auto-provided by GitHub Actions
   - `HOMEBREW_TAP_TOKEN`: User-created PAT with `repo` scope
   - `SCOOP_BUCKET_TOKEN`: User-created PAT with `repo` scope

5. **GoReleaser Version Pinning**: Using `latest` for simplicity. Can pin to specific version later for stability.

### 2.3 Version Management

**Version Variable Location**: `cmd/shark/main.go` or `internal/cli/root.go`

**Implementation**:

```go
package main

import (
    "os"
    "github.com/jwwelbor/shark-task-manager/internal/cli"
)

// Version information (injected at build time via ldflags)
var (
    Version = "dev"     // Overridden by GoReleaser
    Commit  = "unknown" // Git commit SHA
    Date    = "unknown" // Build date
)

func main() {
    // Set version in root command
    cli.RootCmd.Version = Version

    if err := cli.RootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

**Cobra Integration** (`internal/cli/root.go`):

```go
package cli

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
    Use:   "shark",
    Short: "AI-driven task management CLI",
    Long:  `Shark CLI for managing epics, features, and tasks in multi-epic projects.`,
    // Version set in main.go
}

func init() {
    // Cobra automatically adds --version flag when Version is set
    // Output format: "shark version <version>"
}
```

**Version Display**:
```bash
$ shark --version
shark version v1.0.0

$ shark version  # Can add custom command for detailed info
Shark Task Manager v1.0.0
Commit: a1b2c3d
Built:  2025-12-17T10:30:00Z
```

---

## 3. Cross-Platform Build Architecture

### 3.1 Build Matrix

| Target | GOOS | GOARCH | CGO | Archive Format | Binary Name | Platform Examples |
|--------|------|--------|-----|----------------|-------------|-------------------|
| Linux AMD64 | linux | amd64 | Yes | .tar.gz | shark | Ubuntu, Debian, RHEL (x64) |
| Linux ARM64 | linux | arm64 | Yes | .tar.gz | shark | Raspberry Pi, ARM servers |
| macOS AMD64 | darwin | amd64 | Yes | .tar.gz | shark | Intel Macs |
| macOS ARM64 | darwin | arm64 | Yes | .tar.gz | shark | M1/M2/M3 Macs |
| Windows AMD64 | windows | amd64 | Yes | .zip | shark.exe | Windows 10/11 (x64) |

**Build Order**: Parallelized (all 5 platforms build simultaneously)
**Expected Total Time**: 3-4 minutes (including tests)

### 3.2 CGO Cross-Compilation Strategy

**Challenge**: SQLite3 driver (`mattn/go-sqlite3`) requires C compiler for each target.

**Solution**: GitHub Actions ubuntu-latest includes:
- GCC (for Linux)
- Cross-compilation toolchains (mingw-w64 for Windows, osxcross for macOS)

**GoReleaser Automatic Handling**:
```yaml
builds:
  - env:
      - CGO_ENABLED=1  # GoReleaser detects and configures cross-compilers
```

**Behind the Scenes** (GoReleaser manages this):
```bash
# Linux AMD64 (native)
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build

# Linux ARM64
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build

# macOS AMD64
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 CC=o64-clang go build

# macOS ARM64
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 CC=oa64-clang go build

# Windows AMD64
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build
```

**Fallback Strategy** (if CGO issues arise):
1. Switch to `modernc.org/sqlite` (pure Go, no CGO) - requires code changes
2. Use Docker containers with pre-built toolchains
3. Build on native platforms (slower, more complex)

**Confidence Level**: High - GoReleaser handles CGO cross-compilation for popular projects (Hugo, etc.)

### 3.3 Build Optimization

**Ldflags Configuration**:
```yaml
ldflags:
  - -s                                    # Strip symbol table
  - -w                                    # Strip DWARF debug info
  - -X main.Version={{.Version}}          # Embed version
  - -X main.Commit={{.Commit}}            # Embed commit SHA
  - -X main.Date={{.Date}}                # Embed build date
```

**Size Optimization Results** (estimated):
- Unoptimized binary: ~15 MB
- With `-s -w`: ~10 MB (33% reduction)
- Compressed (.tar.gz): ~3-4 MB (70% reduction from unoptimized)

**Performance Impact**: None (stripping debug symbols doesn't affect runtime)

---

## 4. Distribution Architecture

### 4.1 GitHub Releases

**URL Pattern**:
```
https://github.com/jwwelbor/shark-task-manager/releases/tag/v1.0.0
```

**Release Assets**:
```
shark_1.0.0_linux_amd64.tar.gz       (3.5 MB)
shark_1.0.0_linux_arm64.tar.gz       (3.3 MB)
shark_1.0.0_darwin_amd64.tar.gz      (3.6 MB)
shark_1.0.0_darwin_arm64.tar.gz      (3.4 MB)
shark_1.0.0_windows_amd64.zip        (3.8 MB)
checksums.txt                         (512 bytes)
```

**Download URL Pattern**:
```
https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_linux_amd64.tar.gz
```

**Release Notes Template**:
```markdown
# Shark v1.0.0

## What's Changed
- Feature: Add epic queries with progress calculation (#42)
- Fix: Handle missing config file gracefully (#38)
- Docs: Update installation instructions (#40)

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
Download for your platform:
- [Linux AMD64](https://github.com/.../shark_1.0.0_linux_amd64.tar.gz)
- [Linux ARM64](https://github.com/.../shark_1.0.0_linux_arm64.tar.gz)
- [macOS AMD64](https://github.com/.../shark_1.0.0_darwin_amd64.tar.gz)
- [macOS ARM64](https://github.com/.../shark_1.0.0_darwin_arm64.tar.gz)
- [Windows AMD64](https://github.com/.../shark_1.0.0_windows_amd64.zip)

Verify checksums: [checksums.txt](https://github.com/.../checksums.txt)

**Full Changelog**: https://github.com/.../compare/v0.9.0...v1.0.0
```

**Automation**: GoReleaser generates this automatically from Git commit history.

### 4.2 Homebrew Tap Architecture

**Repository**: `github.com/jwwelbor/homebrew-shark`

**Structure**:
```
homebrew-shark/
├── Formula/
│   └── shark.rb          # Homebrew formula (auto-generated)
└── README.md             # Tap documentation
```

**Formula Structure** (`shark.rb`):
```ruby
class Shark < Formula
  desc "AI-driven task management CLI for multi-epic projects"
  homepage "https://github.com/jwwelbor/shark-task-manager"
  version "1.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_darwin_arm64.tar.gz"
      sha256 "abc123..."
    elsif Hardware::CPU.intel?
      url "https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_darwin_amd64.tar.gz"
      sha256 "def456..."
    end
  end

  on_linux do
    if Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
      url "https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_linux_arm64.tar.gz"
      sha256 "ghi789..."
    elsif Hardware::CPU.intel?
      url "https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_linux_amd64.tar.gz"
      sha256 "jkl012..."
    end
  end

  def install
    bin.install "shark"
  end

  test do
    system "#{bin}/shark", "--version"
    assert_match "v#{version}", shell_output("#{bin}/shark --version")
  end
end
```

**Update Mechanism**:
1. GoReleaser detects new release
2. Generates updated `shark.rb` with new version, URLs, checksums
3. Commits to `homebrew-shark` repository
4. Users run `brew upgrade shark` to get new version

**Installation Flow**:
```
User runs: brew tap jwwelbor/shark
           └─▶ Homebrew clones homebrew-shark repository locally

User runs: brew install shark
           └─▶ Homebrew reads Formula/shark.rb
                └─▶ Detects user platform (macOS ARM64)
                     └─▶ Downloads shark_1.0.0_darwin_arm64.tar.gz
                          └─▶ Verifies SHA256 checksum
                               └─▶ Extracts and installs to /opt/homebrew/bin/shark
```

### 4.3 Scoop Bucket Architecture

**Repository**: `github.com/jwwelbor/scoop-shark`

**Structure**:
```
scoop-shark/
├── bucket/
│   └── shark.json        # Scoop manifest (auto-generated)
└── README.md             # Bucket documentation
```

**Manifest Structure** (`shark.json`):
```json
{
  "version": "1.0.0",
  "description": "AI-driven task management CLI for multi-epic projects",
  "homepage": "https://github.com/jwwelbor/shark-task-manager",
  "license": "MIT",
  "architecture": {
    "64bit": {
      "url": "https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_windows_amd64.zip",
      "hash": "sha256:abc123...",
      "bin": "shark.exe"
    }
  },
  "checkver": {
    "github": "https://github.com/jwwelbor/shark-task-manager"
  },
  "autoupdate": {
    "architecture": {
      "64bit": {
        "url": "https://github.com/jwwelbor/shark-task-manager/releases/download/v$version/shark_$version_windows_amd64.zip"
      }
    }
  }
}
```

**Update Mechanism**:
1. GoReleaser detects new release
2. Generates updated `shark.json` with new version, URL, checksum
3. Commits to `scoop-shark` repository
4. Users run `scoop update shark` to get new version

**Installation Flow**:
```
User runs: scoop bucket add shark https://github.com/jwwelbor/scoop-shark
           └─▶ Scoop clones scoop-shark repository locally

User runs: scoop install shark
           └─▶ Scoop reads bucket/shark.json
                └─▶ Downloads shark_1.0.0_windows_amd64.zip
                     └─▶ Verifies SHA256 checksum
                          └─▶ Extracts and installs to ~/scoop/apps/shark/current/shark.exe
                               └─▶ Adds to PATH via shim
```

---

## 5. Security Architecture

### 5.1 Checksum Verification

**Checksums.txt Format**:
```
abc123... shark_1.0.0_linux_amd64.tar.gz
def456... shark_1.0.0_linux_arm64.tar.gz
ghi789... shark_1.0.0_darwin_amd64.tar.gz
jkl012... shark_1.0.0_darwin_arm64.tar.gz
mno345... shark_1.0.0_windows_amd64.zip
```

**Verification Process**:

**macOS/Linux**:
```bash
# Download archive and checksums
wget https://github.com/.../shark_1.0.0_linux_amd64.tar.gz
wget https://github.com/.../checksums.txt

# Verify
sha256sum -c checksums.txt --ignore-missing
# Output: shark_1.0.0_linux_amd64.tar.gz: OK
```

**Windows (PowerShell)**:
```powershell
# Calculate hash
$hash = Get-FileHash shark_1.0.0_windows_amd64.zip -Algorithm SHA256

# Compare with checksums.txt
$expected = (Get-Content checksums.txt | Select-String "windows_amd64").ToString().Split()[0]
$hash.Hash -eq $expected
```

**Package Manager Automation**:
- Homebrew: Automatically verifies checksums from formula
- Scoop: Automatically verifies checksums from manifest

### 5.2 GitHub Token Security

**Token Types**:

| Token | Scope | Usage | Storage |
|-------|-------|-------|---------|
| `GITHUB_TOKEN` | repo contents | Create releases, upload assets | Auto-provided by Actions |
| `HOMEBREW_TAP_TOKEN` | repo (homebrew-shark) | Commit formula updates | GitHub Secrets |
| `SCOOP_BUCKET_TOKEN` | repo (scoop-shark) | Commit manifest updates | GitHub Secrets |

**Token Permissions**:
```yaml
permissions:
  contents: write    # Create releases
  pull-requests: read  # Read PR info for release notes
```

**Security Best Practices**:
1. Use repository-scoped secrets (not organization-wide)
2. Rotate tokens annually
3. Use fine-grained PATs (GitHub's new token type) when available
4. Limit token scope to specific repositories

### 5.3 Supply Chain Security

**Dependency Pinning**:
```yaml
- uses: goreleaser/goreleaser-action@v5
  # Pin to specific version in production:
  # - uses: goreleaser/goreleaser-action@v5.0.0
```

**Build Reproducibility**:
- GoReleaser `--clean` flag ensures clean builds
- No external network calls during build (all deps in go.mod)
- Idempotent: Re-running same tag produces identical artifacts

**Future Enhancements** (out of scope for F08):
- SLSA provenance generation
- Sigstore signing
- GPG signature verification

---

## 6. Workflow Orchestration

### 6.1 Release Sequence Diagram

```
Developer                GitHub Actions              GoReleaser               GitHub Releases     Package Managers
    │                           │                           │                        │                    │
    │ 1. Create tag v1.0.0      │                           │                        │                    │
    ├──────────────────────────▶│                           │                        │                    │
    │ 2. Push tag               │                           │                        │                    │
    ├──────────────────────────▶│                           │                        │                    │
    │                           │ 3. Trigger workflow       │                        │                    │
    │                           ├──────────────────────────▶│                        │                    │
    │                           │ 4. Checkout code          │                        │                    │
    │                           │ 5. Setup Go               │                        │                    │
    │                           │ 6. Run tests              │                        │                    │
    │                           │    (go test ./...)        │                        │                    │
    │                           │                           │                        │                    │
    │                           │ 7. Execute GoReleaser     │                        │                    │
    │                           ├──────────────────────────▶│                        │                    │
    │                           │                           │ 8. Build 5 binaries    │                    │
    │                           │                           │    (parallel)          │                    │
    │                           │                           │ 9. Create archives     │                    │
    │                           │                           │ 10. Generate checksums │                    │
    │                           │                           │                        │                    │
    │                           │                           │ 11. Create GitHub Release (draft)           │
    │                           │                           ├───────────────────────▶│                    │
    │                           │                           │ 12. Upload assets      │                    │
    │                           │                           ├───────────────────────▶│                    │
    │                           │                           │                        │                    │
    │                           │                           │ 13. Update Homebrew formula                 │
    │                           │                           ├────────────────────────┼───────────────────▶│
    │                           │                           │ 14. Update Scoop manifest                   │
    │                           │                           ├────────────────────────┼───────────────────▶│
    │                           │                           │                        │                    │
    │                           │ 15. Workflow complete     │                        │                    │
    │◀──────────────────────────┤                           │                        │                    │
    │                           │                           │                        │                    │
    │ 16. Review draft release  │                           │                        │                    │
    ├───────────────────────────┼───────────────────────────┼───────────────────────▶│                    │
    │ 17. Publish release       │                           │                        │                    │
    ├───────────────────────────┼───────────────────────────┼───────────────────────▶│                    │
    │                           │                           │                        │                    │
    │                           │                           │                        │ 18. Users install  │
    │                           │                           │                        │◀───────────────────┤
    │                           │                           │                        │                    │
```

### 6.2 Error Handling

**Failure Scenarios and Mitigation**:

| Failure Point | Cause | Mitigation | Recovery |
|--------------|-------|------------|----------|
| Tests fail | Code regression | Workflow stops, no release created | Fix code, re-push tag |
| Build fails (single platform) | CGO cross-compilation error | GoReleaser aborts entire release | Check build logs, fix toolchain issue |
| GitHub API rate limit | Too many releases | Workflow retries with backoff | Wait or use PAT with higher limit |
| Homebrew push fails | Token expired | Workflow continues, manual formula update needed | Update token, re-run or manual commit |
| Scoop push fails | Token expired | Workflow continues, manual manifest update needed | Update token, re-run or manual commit |
| Checksum mismatch | Corrupted build | GoReleaser regenerates | Retry build |

**Rollback Strategy**:
1. Delete failed GitHub release (draft or published)
2. Delete Git tag locally and remotely
3. Fix issue
4. Re-tag and push

**Monitoring**:
- GitHub Actions email notifications on failure
- Review draft releases before publishing
- Monitor GitHub Actions workflow runs

---

## 7. Performance Architecture

### 7.1 Build Performance

**Target**: <5 minutes for all builds

**Optimization Strategies**:

1. **Parallel Builds**: GoReleaser builds all platforms simultaneously (5 parallel jobs)
2. **Go Module Caching**: GitHub Actions caches `go.mod` dependencies
3. **Build Caching**: Reuses Go build cache between runs
4. **Minimal Build**: No unnecessary compilation (only `cmd/shark`)

**Expected Timeline**:
```
0:00 - 0:30   Checkout code, setup Go
0:30 - 1:30   Run tests (go test ./...)
1:30 - 4:30   GoReleaser builds (5 platforms in parallel)
4:30 - 5:00   Upload artifacts to GitHub
────────────
Total: ~5 minutes
```

### 7.2 Distribution Performance

**GitHub Releases**:
- Download speed: Limited by user's network and GitHub CDN
- Expected: 3-5 MB in 1-5 seconds (on fast connection)

**Homebrew Installation**:
- Download + extract + install: <30 seconds
- Cached by Homebrew for offline reinstalls

**Scoop Installation**:
- Download + extract + install: <30 seconds
- Cached by Scoop for offline reinstalls

**Scalability**:
- GitHub Releases: Unlimited storage for public repos
- CDN distribution: GitHub's global CDN handles traffic
- No backend server required (fully static distribution)

---

## 8. Deployment Architecture

### 8.1 Initial Setup (One-Time)

**Steps**:

1. **Create Package Manager Repositories**:
   ```bash
   # Create Homebrew tap
   gh repo create homebrew-shark --public

   # Create Scoop bucket
   gh repo create scoop-shark --public
   ```

2. **Generate Personal Access Tokens**:
   ```
   GitHub Settings → Developer Settings → Personal Access Tokens → Fine-grained tokens
   - Name: Homebrew Tap Update
   - Repository: homebrew-shark
   - Permissions: Contents (Read and Write)

   GitHub Settings → Developer Settings → Personal Access Tokens → Fine-grained tokens
   - Name: Scoop Bucket Update
   - Repository: scoop-shark
   - Permissions: Contents (Read and Write)
   ```

3. **Add Secrets to Repository**:
   ```
   Repository Settings → Secrets and Variables → Actions
   - HOMEBREW_TAP_TOKEN: <paste token>
   - SCOOP_BUCKET_TOKEN: <paste token>
   ```

4. **Create GoReleaser Configuration**:
   ```bash
   # Create .goreleaser.yml in repository root
   # (see configuration in section 2.1)
   ```

5. **Create GitHub Actions Workflow**:
   ```bash
   mkdir -p .github/workflows
   # Create .github/workflows/release.yml
   # (see workflow in section 2.2)
   ```

6. **Add Version Variable to Code**:
   ```go
   // cmd/shark/main.go
   var Version = "dev"
   ```

7. **Test Locally**:
   ```bash
   # Install GoReleaser
   brew install goreleaser

   # Validate configuration
   goreleaser check

   # Test build (doesn't publish)
   goreleaser build --snapshot --clean

   # Verify binaries
   ls -lh dist/
   ./dist/shark_linux_amd64/shark --version
   ```

### 8.2 Ongoing Release Process

**Developer Workflow**:

```bash
# 1. Ensure all changes are committed and pushed
git status

# 2. Run tests locally
make test

# 3. Create and push tag
git tag v1.0.0
git push origin v1.0.0

# 4. Monitor GitHub Actions
# Visit: https://github.com/jwwelbor/shark-task-manager/actions

# 5. Review draft release
# Visit: https://github.com/jwwelbor/shark-task-manager/releases

# 6. Edit release notes if needed

# 7. Publish release

# 8. Verify package managers updated
brew search shark  # Should show new version
scoop search shark # Should show new version

# 9. Test installation
brew install shark
shark --version  # Should show v1.0.0
```

**Hotfix Workflow**:
```bash
# 1. Fix bug on main or release branch
# 2. Tag as patch version
git tag v1.0.1
git push origin v1.0.1

# 3. Automated release process runs
# 4. Users get hotfix via package manager update
```

---

## 9. Monitoring and Observability

### 9.1 Release Metrics

**GitHub Provides**:
- Release download counts (per asset)
- Workflow run duration and success rate
- Repository traffic (views, clones)

**Useful Metrics to Track**:
- Average release build time (target: <5 min)
- Release failure rate (target: <5%)
- Time from tag to published release (target: <10 min)
- Download count by platform (identify popular platforms)

### 9.2 Workflow Monitoring

**GitHub Actions Dashboard**:
```
Actions → Workflows → Release
- View all runs
- Filter by status (success, failure, in progress)
- Review logs for each step
- Re-run failed workflows
```

**Email Notifications**:
- Automatic on workflow failure
- Configure in personal GitHub settings

**Integration Opportunities** (future):
- Slack notifications on release
- Discord webhook for new versions
- Twitter/social media automation

---

## 10. Migration Plan

### 10.1 From Current State to F08

**Phase 1: Configuration (Week 1)**
- Day 1: Create `.goreleaser.yml`
- Day 2: Create `.github/workflows/release.yml`
- Day 3: Add version variable to code
- Day 4: Test locally with `goreleaser build --snapshot`

**Phase 2: Infrastructure (Week 1)**
- Day 5: Create Homebrew tap repository
- Day 6: Create Scoop bucket repository
- Day 7: Generate and configure PATs

**Phase 3: Testing (Week 2)**
- Day 8: Cut test release (v0.1.0-test)
- Day 9: Verify all platforms build successfully
- Day 10: Test Homebrew installation on macOS
- Day 11: Test Scoop installation on Windows
- Day 12: Test manual downloads and checksum verification

**Phase 4: Documentation (Week 2)**
- Day 13: Update README.md with installation instructions
- Day 14: Update CONTRIBUTING.md with release process

**Phase 5: Production Release (Week 3)**
- Day 15: Cut v1.0.0 release
- Day 16: Monitor downloads and user feedback
- Day 17: Announce release (social media, docs site, etc.)

### 10.2 Rollback Plan

If critical issues discovered post-release:

1. **Immediate**: Delete published GitHub release
2. **Git**: Delete tag locally and remotely
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```
3. **Package Managers**:
   - Homebrew: Revert formula commit in tap
   - Scoop: Revert manifest commit in bucket
4. **Fix**: Address issue in code
5. **Re-release**: Tag as v1.0.1 with fix

---

## 11. Future Architecture Enhancements (Out of Scope)

### 11.1 Code Signing (Phase 2)

**macOS**: Notarization with Apple Developer account
**Windows**: Code signing certificate
**GPG**: Sign all releases with GPG key

### 11.2 Additional Distribution Channels (Phase 3)

- **Docker**: Multi-arch images (linux/amd64, linux/arm64)
- **Snap**: Linux universal package
- **Chocolatey**: Alternative Windows package manager
- **APT/YUM**: Native Linux repositories

### 11.3 Release Automation Enhancements (Phase 4)

- Automated changelog generation from PR labels
- Pre-release channels (beta, rc)
- Nightly builds from main branch
- Automated dependency updates (Dependabot integration)

---

## 12. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Build time | <5 minutes | GitHub Actions workflow duration |
| Binary size | <10 MB per platform | Archive file sizes |
| Install time (Homebrew) | <30 seconds | Manual testing |
| Install time (Scoop) | <30 seconds | Manual testing |
| Workflow time | <10 minutes | Tag push to published release |
| Release success rate | >95% | Successful releases / total releases |
| Download verification | 100% | Checksum matches |

---

## 13. Conclusion

This architecture provides a robust, automated, and secure distribution system for the Shark CLI tool. By leveraging industry-standard tools (GoReleaser, GitHub Actions) and proven patterns from successful Go projects, we achieve:

1. **Multi-platform support** with minimal manual effort
2. **Secure distribution** via checksums and package manager verification
3. **Fast releases** (<10 minutes from tag to distribution)
4. **Easy installation** for end users (single command via package managers)
5. **Maintainable pipeline** with clear error handling and monitoring

The architecture is designed to scale with the project's growth and provides a foundation for future enhancements like code signing, Docker distribution, and advanced release automation.

---

**Architecture Status**: ✅ Ready for Implementation
**Next Step**: Security design review (06-security-design.md)
