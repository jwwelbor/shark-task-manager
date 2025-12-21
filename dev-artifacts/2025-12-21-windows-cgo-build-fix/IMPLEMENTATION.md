# Implementation Details - Windows CGO Build Fix

## Change Summary

**File Modified**: `.goreleaser.yml`

**Change Type**: Configuration simplification and bug fix

### What Was Changed

Removed the problematic default environment variable configuration that was causing Windows builds to attempt CGO compilation with an incompatible toolchain.

#### Before:
```yaml
    ldflags:
      - -s -w # Strip debug symbols to reduce binary size
      - -X main.Version={{.Version}} # Inject version from git tag

    # Environment variables for build
    env:
      - CGO_ENABLED=1 # Enable CGO for go-sqlite3 support

    # Build for multiple platforms
```

#### After:
```yaml
    ldflags:
      - -s -w # Strip debug symbols to reduce binary size
      - -X main.Version={{.Version}} # Inject version from git tag

    # Build for multiple platforms
```

### Platform-Specific Behavior (Unchanged)

The `overrides` section now provides **all environment configuration** for each platform:

1. **Linux amd64**: `CGO_ENABLED=1` (native Linux compile, uses system gcc)
2. **Linux arm64**: `CGO_ENABLED=1` + ARM64 cross-compiler (`aarch64-linux-gnu-gcc`)
3. **Windows amd64**: `CGO_ENABLED=0` (no CGO, pure Go)
4. **macOS amd64**: `CGO_ENABLED=0` (no CGO, cannot cross-compile to macOS)
5. **Windows arm64**: Excluded from builds (not commonly used)
6. **macOS arm64**: Excluded from builds (cannot cross-compile from Linux)

## Why This Fixes the Windows Build Error

### The Problem Chain

1. **Default `CGO_ENABLED=1`** was applied to all builds
2. Windows override set `CGO_ENABLED=0`, but GoReleaser's env variable inheritance could cause conflicts
3. Windows builds attempted to use CGO despite override
4. CI environment (Ubuntu/Linux) has no Windows MinGW compiler
5. Build tried to use Linux gcc to compile Windows code
6. Linux gcc with `-mthreads` flag (Windows threading) = **compilation error**

### The Solution

By removing the default and relying entirely on explicit per-platform overrides:
- **No ambiguity** about which environment variables apply to which platform
- **Windows builds** explicitly get `CGO_ENABLED=0` only
- **Linux builds** explicitly get `CGO_ENABLED=1` with appropriate compiler
- **macOS builds** explicitly get `CGO_ENABLED=0` (cannot cross-compile)

## Verification Checklist

### Pre-Fix State (Current)
- [ ] Build fails for Windows targets with `-mthreads` error
- [ ] Error indicates CGO is being applied to Windows builds

### Post-Fix State (Expected)
- [ ] Windows amd64 builds successfully without CGO
- [ ] Linux amd64 builds successfully with CGO
- [ ] Linux arm64 cross-compile works with arm64 toolchain
- [ ] macOS amd64 builds successfully without CGO
- [ ] Release artifacts created: `.tar.gz` for Unix, `.zip` for Windows
- [ ] All binaries functional with `--version` and `--help`

## Configuration Details

**File**: `/home/jwwelbor/projects/shark-task-manager/.goreleaser.yml`

**Lines Changed**: 25-28 (removed 4 lines)

**Deletion**:
```yaml
    # Environment variables for build
    env:
      - CGO_ENABLED=1 # Enable CGO for go-sqlite3 support
```

**Retention**:
- Lines 43-67: `overrides` section provides explicit per-platform configuration
- All platform-specific CGO settings remain unchanged

## Build Environment Requirements

The GitHub Actions CI environment already has necessary dependencies:

```yaml
- name: Install build dependencies
  run: |
    sudo apt-get update
    sudo apt-get install -y build-essential gcc-aarch64-linux-gnu libc6-dev-arm64-cross
```

These provide:
- `gcc` and `build-essential` for Linux builds
- `gcc-aarch64-linux-gnu` for Linux ARM64 cross-compilation
- No Windows toolchain needed (CGO disabled for Windows)

## Testing Approach

### Local Verification (Before Release)

1. Validate YAML syntax:
   ```bash
   # Check that .goreleaser.yml is valid YAML
   go run github.com/bmatcuk/doublestar/v4@latest
   ```

2. Test snapshot build (doesn't require git tag):
   ```bash
   # Requires goreleaser installed
   goreleaser release --snapshot --clean
   ```

### CI/CD Verification (Automatic)

1. Unit tests run first (quality gate)
2. GoReleaser builds all platforms
3. Upload artifacts for inspection
4. Verify binary functionality

### Manual Post-Release Verification

Use the existing `./scripts/verify-release.sh` script:
```bash
./scripts/verify-release.sh v0.X.X windows amd64
./scripts/verify-release.sh v0.X.X linux amd64
./scripts/verify-release.sh v0.X.X linux arm64
./scripts/verify-release.sh v0.X.X darwin amd64
```

## Regression Risk Assessment

**Risk Level**: Very Low

**Why**:
1. Only changes environment variable handling in GoReleaser config
2. Explicit per-platform overrides already existed and were correct
3. Removed redundant default that was being overridden anyway
4. No changes to source code or build logic
5. All platforms already had correct settings in overrides

**What Could Go Wrong**:
- None identified; this is removing conflicting configuration

**Rollback Plan**:
- If issues arise, revert this commit
- Revert adds back the 4 lines of default env configuration
- No database changes, no code changes to revert

## Related Issues

This fixes commit 817d293 which attempted to set per-platform CGO but left the problematic default in place.

**Git History**:
- d93ce15: Initial ARM64 exclusion
- 0123ab0: Revert to CGO_ENABLED=0 (attempted global disable)
- d691595: Try ARM64 cross-compile with CGO (partial fix)
- 9ade1a8: Exclude darwin_arm64 (partially working)
- 817d293: Add per-platform overrides (but left default CGO_ENABLED=1)

This fix completes what 817d293 was trying to achieve by removing the conflicting default.

## References

- GoReleaser Docs: https://goreleaser.com/customization/build/
- go-sqlite3: https://github.com/mattn/go-sqlite3 (requires CGO when enabled)
- Cross-compilation guide: https://golang.org/doc/effective_go#cgo
