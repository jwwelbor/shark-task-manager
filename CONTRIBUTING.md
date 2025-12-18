# Contributing to Shark Task Manager

Thank you for your interest in contributing to Shark Task Manager! This document provides guidelines and processes for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)
- [Style Guidelines](#style-guidelines)

## Code of Conduct

Be respectful, inclusive, and constructive. We aim to maintain a welcoming environment for all contributors.

## Getting Started

### Prerequisites

- Go 1.23 or later
- SQLite3
- Git
- Make

### Setting Up Development Environment

1. Fork and clone the repository:
   ```bash
   git clone https://github.com/YOUR_USERNAME/shark-task-manager.git
   cd shark-task-manager
   ```

2. Install dependencies:
   ```bash
   make install
   ```

3. Build the CLI:
   ```bash
   make shark
   ```

4. Run tests to verify setup:
   ```bash
   make test
   ```

## Development Workflow

### Creating a Feature Branch

```bash
git checkout main
git pull origin main
git checkout -b feature/your-feature-name
```

### Making Changes

1. Write tests first (TDD approach)
2. Implement the feature
3. Run tests: `make test`
4. Format code: `make fmt`
5. Run linter: `make vet`

### Commit Message Format

Use conventional commit format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build process or tool changes
- `ci`: CI/CD changes

**Example**:
```
feat(cli): add epic list command with progress

- Implement epic listing with JSON output
- Calculate progress from completed tasks
- Add filtering by status
- Include comprehensive tests

Closes #42
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test ./internal/repository -v

# Run integration tests
make test-db
```

### Writing Tests

- Place tests in `*_test.go` files
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test both success and error cases

**Example**:
```go
func TestEpicRepository_Create(t *testing.T) {
    tests := []struct {
        name    string
        epic    *models.Epic
        wantErr bool
    }{
        {
            name:    "valid epic",
            epic:    &models.Epic{Key: "E01", Title: "Test Epic"},
            wantErr: false,
        },
        {
            name:    "duplicate key",
            epic:    &models.Epic{Key: "E01", Title: "Duplicate"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := repo.Create(tt.epic)
            if (err != nil) != tt.wantErr {
                t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Submitting Changes

### Pull Request Process

1. Push your branch to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Open a pull request against `main` branch

3. Fill out the PR template with:
   - Clear description of changes
   - Link to related issues
   - Testing performed
   - Screenshots (if UI changes)

4. Wait for review and address feedback

5. Once approved, a maintainer will merge your PR

### PR Checklist

- [ ] Tests pass locally (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] Linter passes (`make vet`)
- [ ] Commit messages follow convention
- [ ] Documentation updated (if needed)
- [ ] CHANGELOG.md updated (for significant changes)

## Release Process

This section documents the complete release process for maintainers.

### Overview

Shark uses semantic versioning (SemVer) and automated releases via GitHub Actions and GoReleaser. Releases are triggered by pushing Git tags.

### Semantic Versioning (SemVer)

Format: `MAJOR.MINOR.PATCH` (e.g., `v1.2.3`)

**When to Bump**:
- **MAJOR** (v2.0.0): Breaking changes, incompatible API changes
  - Example: Changing command syntax, removing commands, database schema changes requiring migration
- **MINOR** (v1.1.0): New features, backward-compatible additions
  - Example: Adding new commands, new flags, new features without breaking existing functionality
- **PATCH** (v1.0.1): Bug fixes, backward-compatible improvements
  - Example: Fixing crashes, improving error messages, documentation fixes

**Pre-Release Versions**:
- Alpha: `v1.0.0-alpha.1` (early testing, unstable)
- Beta: `v1.0.0-beta.1` (feature complete, testing)
- Release Candidate: `v1.0.0-rc.1` (final testing before release)

### Prerequisites for Releasing

Before creating a release, ensure:

1. **All tests pass**:
   ```bash
   go test ./...
   ```

2. **Code is clean**:
   ```bash
   make fmt
   make vet
   make lint
   ```

3. **CHANGELOG.md is updated** with all changes since last release

4. **Documentation is current**:
   - README.md reflects new features
   - CLI documentation is accurate
   - Examples work correctly

5. **Local build succeeds**:
   ```bash
   make shark
   ./bin/shark --version
   ```

### Release Workflow

#### Step 1: Prepare Release Branch (Optional for Major Releases)

For patch and minor releases, work directly on `main`. For major releases, consider a release branch:

```bash
git checkout main
git pull origin main
git checkout -b release/v1.0.0
```

#### Step 2: Update Version Information

1. Update CHANGELOG.md with release date:
   ```markdown
   ## [1.0.0] - 2025-12-18

   ### Added
   - New epic listing command with progress
   - JSON output support for all commands

   ### Fixed
   - Database connection handling in concurrent scenarios

   ### Changed
   - Improved error messages for missing dependencies
   ```

2. Commit version updates:
   ```bash
   git add CHANGELOG.md
   git commit -m "chore: prepare v1.0.0 release"
   git push origin main  # or release branch
   ```

#### Step 3: Create and Push Git Tag

```bash
# Ensure you're on the correct branch
git checkout main
git pull origin main

# Create annotated tag with release notes
git tag -a v1.0.0 -m "Release v1.0.0

This release includes:
- Epic and feature management with progress tracking
- Comprehensive CLI for task lifecycle management
- Multi-platform distribution (macOS, Linux, Windows)
- Package manager support (Homebrew, Scoop)

See CHANGELOG.md for detailed changes."

# Push tag to trigger release workflow
git push origin v1.0.0
```

**Important**: Use annotated tags (`-a` flag), not lightweight tags. Annotated tags include metadata and are required for proper release notes.

#### Step 4: Monitor Release Workflow

1. Go to GitHub Actions: `https://github.com/jwwelbor/shark-task-manager/actions`

2. Find the "Release" workflow triggered by your tag

3. Monitor the workflow steps:
   - **Test Job**: Runs all tests (must pass)
   - **Release Job**: Builds binaries and creates release

4. Check for errors:
   - **Test failures**: Fix issues, delete tag, fix code, re-tag
   - **Build failures**: Usually CGO or dependency issues
   - **Token errors**: Check GitHub Secrets configuration

**Expected Duration**: 5-7 minutes for complete release

#### Step 5: Review Draft Release

After the workflow completes:

1. Go to: `https://github.com/jwwelbor/shark-task-manager/releases`

2. Find the draft release for your tag

3. Review the automatically generated content:
   - Release title
   - Auto-generated changelog
   - Release assets (binaries for all platforms)
   - Checksums file

4. Edit the release notes:
   - Add highlights at the top
   - Categorize changes (Added, Fixed, Changed)
   - Add upgrade instructions if needed
   - Link to relevant PRs or issues
   - Add breaking changes section (if MAJOR release)

**Example Release Notes**:
```markdown
# Shark v1.0.0 - Production Release

This is the first production-ready release of Shark Task Manager!

## Highlights

- Complete task lifecycle management for AI-driven development
- Multi-platform support (macOS, Linux, Windows)
- Package manager distribution (Homebrew, Scoop)
- Comprehensive dependency tracking and progress calculation

## Installation

**macOS (Homebrew)**:
```bash
brew install jwwelbor/shark/shark
```

**Windows (Scoop)**:
```powershell
scoop bucket add shark https://github.com/jwwelbor/scoop-shark
scoop install shark
```

See [Installation Guide](https://github.com/jwwelbor/shark-task-manager#installation) for more options.

## What's Changed

### Added
- Epic and feature management with progress tracking (#42)
- Task lifecycle commands (start, complete, approve) (#45)
- Dependency validation and blocking (#48)
- JSON output for all commands (#50)
- Sync command for Git workflow integration (#52)

### Fixed
- Database locking in concurrent operations (#43)
- Task key generation edge cases (#46)

Full Changelog: v0.9.0...v1.0.0
```

#### Step 6: Publish Release

1. Click "Publish release" button

2. Verify the release is now public (not draft)

3. Check release assets are downloadable:
   ```bash
   # Test download
   wget https://github.com/jwwelbor/shark-task-manager/releases/download/v1.0.0/shark_1.0.0_linux_amd64.tar.gz
   ```

#### Step 7: Verify Package Manager Updates

GitHub Actions automatically updates package manager repositories via GoReleaser, but verify they work:

**Homebrew** (updates within 5 minutes):
```bash
# Check formula was updated
curl https://raw.githubusercontent.com/jwwelbor/homebrew-shark/main/Formula/shark.rb | grep version

# Test installation
brew uninstall shark  # if already installed
brew install jwwelbor/shark/shark
shark --version  # Should show v1.0.0
```

**Scoop** (updates within 5 minutes):
```powershell
# Check manifest was updated
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/jwwelbor/scoop-shark/main/bucket/shark.json"

# Test installation
scoop uninstall shark  # if already installed
scoop install shark
shark --version  # Should show v1.0.0
```

#### Step 8: Test Manual Installation

Test manual download and verification on each platform:

**Linux**:
```bash
./scripts/verify-release.sh v1.0.0 linux amd64
```

**macOS**:
```bash
./scripts/verify-release.sh v1.0.0 darwin arm64
```

**Windows**:
```powershell
.\scripts\verify-release.ps1 -Version "v1.0.0" -Platform "windows" -Arch "amd64"
```

#### Step 9: Post-Release Tasks

1. **Update main branch** with any release-specific changes:
   ```bash
   git checkout main
   git pull origin main
   # Start next development cycle
   ```

2. **Close related GitHub issues** linked to this release

3. **Announce the release** (optional):
   - Project README (add "Latest Release" badge)
   - Social media
   - Community channels
   - Mailing lists

4. **Monitor for issues**:
   - Watch GitHub issues for bug reports
   - Check download counts
   - Monitor discussion forums

### Troubleshooting Release Issues

#### Test Failures During Release

If tests fail during the release workflow:

1. **Do NOT publish the draft release**
2. Delete the tag:
   ```bash
   git tag -d v1.0.0
   git push origin :refs/tags/v1.0.0
   ```
3. Fix the failing tests locally
4. Push fixes to main
5. Re-create the tag (Step 3)

#### Build Failures

**CGO Cross-Compilation Errors**:
```
Error: Failed to build for darwin/arm64: cgo: C compiler not found
```

**Solution**: GoReleaser should handle this automatically. If it fails, check:
- GoReleaser version in `.github/workflows/release.yml`
- Build configuration in `.goreleaser.yml`
- GitHub Actions runner OS (should be `ubuntu-latest`)

**GoReleaser Configuration Errors**:
```
Error: invalid config: builds[0].goos: darwin/windows should be darwin or windows
```

**Solution**: Review `.goreleaser.yml` syntax, check GoReleaser documentation for breaking changes.

#### Token Authentication Failures

**Homebrew/Scoop Update Failures**:
```
Error: failed to create file on homebrew-shark: 403 Forbidden
```

**Solution**:
1. Check token expiration: Settings → Developer settings → Personal access tokens
2. Verify token permissions (Contents: Read and write)
3. Regenerate token if expired
4. Update GitHub Secrets:
   - Repository → Settings → Secrets and Variables → Actions
   - Update `HOMEBREW_TAP_TOKEN` or `SCOOP_BUCKET_TOKEN`

#### Package Manager Update Delays

**Homebrew/Scoop not showing latest version**:

This is normal. Updates can take 5-15 minutes. If it persists:

1. Check the package repository was updated:
   - Homebrew: https://github.com/jwwelbor/homebrew-shark/commits/main
   - Scoop: https://github.com/jwwelbor/scoop-shark/commits/main

2. Check GoReleaser logs in GitHub Actions for errors

3. Manually trigger update if needed (requires direct push access to tap/bucket repos)

### Token Management

#### GitHub Personal Access Tokens

**Required Tokens**:
1. `GITHUB_TOKEN`: Automatically provided by GitHub Actions (no setup needed)
2. `HOMEBREW_TAP_TOKEN`: Fine-grained PAT for `homebrew-shark` repository
3. `SCOOP_BUCKET_TOKEN`: Fine-grained PAT for `scoop-shark` repository

#### Creating Fine-Grained PATs

1. Go to: Settings → Developer settings → Personal access tokens → Fine-grained tokens

2. Click "Generate new token"

3. Configure token:
   ```
   Token name: Homebrew Tap Update (Shark CLI)
   Expiration: 90 days
   Resource owner: jwwelbor
   Repository access: Only select repositories
   Selected repositories: homebrew-shark
   Permissions:
     Repository permissions:
       Contents: Read and write
       Metadata: Read-only (automatic)
   ```

4. Generate and copy the token

5. Add to repository secrets:
   - Go to: Repository → Settings → Secrets and Variables → Actions
   - Click "New repository secret"
   - Name: `HOMEBREW_TAP_TOKEN`
   - Secret: Paste the token
   - Click "Add secret"

6. Repeat for `SCOOP_BUCKET_TOKEN` with `scoop-shark` repository

#### Token Rotation Schedule

- **Expiration**: Set tokens to expire after 90 days
- **Rotation**: Renew tokens before expiration
- **Calendar Reminder**: Set reminders 1 week before token expiration

**Renewal Process**:
1. Generate new token (same process as creation)
2. Update GitHub Secret with new value
3. Revoke old token after confirming new token works
4. Test with a release or dry-run

### Emergency Rollback

If a release has critical issues after publishing:

1. **Do NOT delete the GitHub release** (breaks existing downloads)

2. **Create a patch release immediately**:
   ```bash
   git checkout main
   # Fix the critical issue
   git commit -m "fix: critical issue in v1.0.0"
   git tag -a v1.0.1 -m "Emergency patch for v1.0.0"
   git push origin v1.0.1
   ```

3. **Update release notes** on v1.0.0 to indicate users should upgrade to v1.0.1

4. **Notify users** via GitHub discussions, issues, or announcements

### Release Checklist

Use this checklist for every release:

- [ ] All tests pass (`go test ./...`)
- [ ] Code is formatted and linted (`make fmt && make vet`)
- [ ] CHANGELOG.md is updated with release notes
- [ ] Version number follows SemVer
- [ ] Documentation is current (README, CLI docs)
- [ ] Local build succeeds (`make shark`)
- [ ] Release branch created (if major release)
- [ ] Git tag created and pushed (`git tag -a vX.Y.Z`)
- [ ] GitHub Actions workflow completes successfully
- [ ] Draft release reviewed and edited
- [ ] Release published (converted from draft)
- [ ] Manual downloads tested and verified
- [ ] Homebrew installation tested (`brew install shark`)
- [ ] Scoop installation tested (`scoop install shark`)
- [ ] Verification scripts tested (all platforms)
- [ ] Related issues closed
- [ ] Release announced (if applicable)

## Style Guidelines

### Go Code Style

- Follow standard Go conventions: https://golang.org/doc/effective_go
- Use `gofmt` for formatting: `make fmt`
- Run `go vet` to catch common issues: `make vet`
- Use meaningful variable names
- Add comments for exported functions and types
- Keep functions focused and small

### Documentation Style

- Use clear, concise language
- Include code examples
- Test all command examples
- Use proper Markdown formatting
- Add table of contents for long documents

## Questions?

- Open a [GitHub Discussion](https://github.com/jwwelbor/shark-task-manager/discussions)
- Check existing [Issues](https://github.com/jwwelbor/shark-task-manager/issues)
- Review [Documentation](docs/)

Thank you for contributing to Shark Task Manager!
