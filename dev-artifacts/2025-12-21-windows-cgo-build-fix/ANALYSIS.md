# Windows Build CGO Compilation Error - Analysis

## Problem Statement

The release build is failing for the Windows target (windows_amd64_v1) with a CGO compilation error:

```
gcc: error: unrecognized command-line option '-mthreads'; did you mean '-pthread'?
target=windows_amd64_v1
```

The `-mthreads` flag is a **Windows/MinGW-specific compiler flag** that should only be used when compiling for Windows using MinGW. The error indicates that either:
1. CGO is enabled for Windows builds when it should be disabled, OR
2. The Windows build is trying to use a non-Windows compiler toolchain

## Root Cause Analysis

### Configuration Overview

**File**: `.goreleaser.yml` (lines 1-72)

The current configuration attempts to selectively enable CGO:

1. **Default environment** (line 27):
   ```yaml
   env:
     - CGO_ENABLED=1  # Enable CGO for go-sqlite3 support
   ```

2. **Platform-specific overrides** (lines 48-72):
   ```yaml
   overrides:
     # Linux amd64 with CGO (native build)
     - goos: linux
       goarch: amd64
       env:
         - CGO_ENABLED=1

     # Linux ARM64 with CGO and cross-compiler
     - goos: linux
       goarch: arm64
       env:
         - CGO_ENABLED=1
         - CC=aarch64-linux-gnu-gcc
         - CXX=aarch64-linux-gnu-g++

     # Windows: disable CGO
     - goos: windows
       env:
         - CGO_ENABLED=0

     # macOS: disable CGO
     - goos: darwin
       env:
         - CGO_ENABLED=0
   ```

### The Problem

The issue is how **GoReleaser processes environment variable overrides**:

1. GoReleaser applies the default `env` to all builds
2. Platform-specific `overrides` are meant to **replace** environment variables
3. However, the default `CGO_ENABLED=1` might be persisting or conflicting with overrides

**Evidence**: The Windows build shows `-mthreads` which is a MinGW flag, suggesting:
- Windows builds are attempting to use CGO (should be disabled)
- Using Linux compiler (gcc) instead of MinGW compiler
- Compiler is detecting a "threads" library and trying to link with `-mthreads`

### Why This Happens

The `go-sqlite3` library requires CGO when enabled. When CGO is enabled for Windows builds:
1. Go's build system tries to use `gcc` to compile C code
2. On Linux-based CI systems (like GitHub Actions), there's no Windows MinGW compiler available
3. The build tries to use the Linux gcc compiler with Windows target settings
4. This creates a mismatch: Linux compiler trying to compile Windows code
5. The `-mthreads` flag is attempted because Linux is trying to resolve thread libraries for the Windows target

## Solution

The fix is to **explicitly set CGO_ENABLED for each platform**, rather than relying on a default that gets overridden.

### Option 1: Remove Default CGO and Set Per-Platform (RECOMMENDED)

Instead of:
```yaml
env:
  - CGO_ENABLED=1
```

Use explicit per-platform configuration with NO default env.

This ensures:
- Linux amd64: `CGO_ENABLED=1` (with gcc)
- Linux arm64: `CGO_ENABLED=1` (with aarch64-linux-gnu-gcc)
- Windows: `CGO_ENABLED=0` (no compilation needed)
- macOS: `CGO_ENABLED=0` (cross-compilation not supported)
- Darwin arm64: `CGO_ENABLED=0` (excluded from builds anyway)

### Implementation Details

The fix involves:

1. **Remove the default `env` section** that sets `CGO_ENABLED=1`
2. **Keep all the overrides** that explicitly set environment variables per platform
3. Add explicit override for linux/amd64 (already present)
4. Add explicit override for linux/arm64 (already present)
5. Keep Windows and macOS with `CGO_ENABLED=0`

## Dependencies and Build Chain

- **Dependency**: `github.com/mattn/go-sqlite3 v1.14.32` - requires CGO when enabled
- **CI/CD**: GitHub Actions + GoReleaser v2.x
- **Build Host**: Ubuntu-latest (Linux x86_64)
- **Cross-compile Targets**:
  - Linux amd64 (native, CGO enabled)
  - Linux arm64 (cross-compile with aarch64-linux-gnu toolchain)
  - Windows amd64 (no CGO)
  - macOS amd64 (no CGO)

## Files to Modify

1. `.goreleaser.yml` - Remove default CGO_ENABLED=1 from root env

## Testing Strategy

1. Verify Windows build completes without CGO compilation errors
2. Verify Linux amd64 still compiles with CGO support
3. Verify Linux arm64 cross-compilation works
4. Verify macOS builds disable CGO properly
5. Verify resulting binaries are functional with `--version` and `--help`

## Risk Assessment

**Low Risk**:
- Only changes how environment variables are configured in GoReleaser
- Doesn't change any source code
- All platform-specific behavior is already explicitly defined in overrides
- Removing a redundant default that's being overridden anyway
