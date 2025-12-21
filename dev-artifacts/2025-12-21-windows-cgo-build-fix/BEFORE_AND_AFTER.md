# Before and After Comparison

## Before (Original Problem)

```yaml
# GoReleaser configuration for Shark Task Manager
# Build configuration
builds:
  - # Build target: cmd/shark/main.go
    main: ./cmd/shark/main.go

    # Binary name
    binary: shark

    # Inject version at build time
    ldflags:
      - -s -w # Strip debug symbols to reduce binary size
      - -X main.Version={{.Version}} # Inject version from git tag

    # Environment variables for build  <-- PROBLEM HERE
    env:
      - CGO_ENABLED=1 # Enable CGO for go-sqlite3 support  <-- CONFLICTING DEFAULT

    # Build for multiple platforms
    goos:
      - linux
      - darwin
      - windows

    goarch:
      - amd64
      - arm64

    # Platform-specific configurations
    ignore:
      - goos: windows
        goarch: arm64
      - goos: darwin
        goarch: arm64

    # Cross-compilation settings
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

      # Windows: disable CGO (cross-compilation from Linux is problematic)
      - goos: windows
        env:
          - CGO_ENABLED=0  <-- OVERRIDE (but conflicting with default above)

      # macOS: disable CGO (cannot cross-compile from Linux without Apple SDK)
      - goos: darwin
        env:
          - CGO_ENABLED=0  <-- OVERRIDE (but conflicting with default above)
```

### Problem
**Lines 25-28** set a global default that:
1. Applies to all platforms by default
2. Conflicts with platform-specific overrides
3. Causes Windows to attempt CGO compilation
4. Results in error: `gcc: error: unrecognized command-line option '-mthreats'`

---

## After (Fixed)

```yaml
# GoReleaser configuration for Shark Task Manager
# Build configuration
builds:
  - # Build target: cmd/shark/main.go
    main: ./cmd/shark/main.go

    # Binary name
    binary: shark

    # Inject version at build time
    ldflags:
      - -s -w # Strip debug symbols to reduce binary size
      - -X main.Version={{.Version}} # Inject version from git tag

    # Build for multiple platforms  <-- NO CONFLICTING DEFAULT
    goos:
      - linux
      - darwin
      - windows

    goarch:
      - amd64
      - arm64

    # Platform-specific configurations
    ignore:
      - goos: windows
        goarch: arm64
      - goos: darwin
        goarch: arm64

    # Cross-compilation settings
    overrides:
      # Linux amd64 with CGO (native build)
      - goos: linux
        goarch: amd64
        env:
          - CGO_ENABLED=1  <-- EXPLICIT (no ambiguity)

      # Linux ARM64 with CGO and cross-compiler
      - goos: linux
        goarch: arm64
        env:
          - CGO_ENABLED=1  <-- EXPLICIT
          - CC=aarch64-linux-gnu-gcc
          - CXX=aarch64-linux-gnu-g++

      # Windows: disable CGO (cross-compilation from Linux is problematic)
      - goos: windows
        env:
          - CGO_ENABLED=0  <-- EXPLICIT (no conflict with default)

      # macOS: disable CGO (cannot cross-compile from Linux without Apple SDK)
      - goos: darwin
        env:
          - CGO_ENABLED=0  <-- EXPLICIT (no conflict with default)
```

### Solution
**Removed lines 25-28** that set conflicting default:
- No global environment variable default
- All CGO settings now explicit in platform overrides
- Windows unambiguously gets CGO_ENABLED=0
- No compiler conflicts

---

## Exact Diff

```diff
@@ -22,10 +22,6 @@ builds:
      - -s -w # Strip debug symbols to reduce binary size
      - -X main.Version={{.Version}} # Inject version from git tag

-    # Environment variables for build
-    env:
-      - CGO_ENABLED=1 # Enable CGO for go-sqlite3 support
-
     # Build for multiple platforms
     goos:
       - linux
```

---

## Impact on Platform Behavior

### Linux amd64
**Before**: Global CGO_ENABLED=1, no override -> **CGO_ENABLED=1** (uses gcc)
**After**: Override says CGO_ENABLED=1 -> **CGO_ENABLED=1** (uses gcc)
**Change**: NONE - continues to work the same way

### Linux arm64
**Before**: Global CGO_ENABLED=1, override says CGO_ENABLED=1 with cross-compiler
**After**: Override says CGO_ENABLED=1 with cross-compiler
**Change**: NONE - continues to work the same way

### Windows amd64
**Before**: Global CGO_ENABLED=1, override says CGO_ENABLED=0 -> **CONFLICT** (attempts CGO, fails)
**After**: Override says CGO_ENABLED=0 -> **CGO_ENABLED=0** (pure Go, works)
**Change**: FIXED - now builds successfully without CGO

### macOS amd64
**Before**: Global CGO_ENABLED=1, override says CGO_ENABLED=0 -> **CONFLICT** (unclear which applies)
**After**: Override says CGO_ENABLED=0 -> **CGO_ENABLED=0** (pure Go, works)
**Change**: CLARIFIED - now explicitly pure Go

---

## Summary of Changes

| Aspect | Before | After | Impact |
|--------|--------|-------|--------|
| Default CGO | 1 (global) | None | Removes ambiguity |
| Linux amd64 CGO | 1 (explicit) | 1 (explicit) | No change |
| Linux arm64 CGO | 1 (explicit) | 1 (explicit) | No change |
| Windows CGO | 1 (default) + 0 (override) CONFLICT | 0 (explicit) | FIXED |
| macOS CGO | 1 (default) + 0 (override) CONFLICT | 0 (explicit) | FIXED |
| Lines removed | - | 4 | Simplification |
| Functionality | Broken for Windows | Working for all | WORKS |

---

## Why This Is The Correct Fix

1. **Removes ambiguity**: No conflicting settings
2. **Uses existing correct config**: Platform overrides already had right values
3. **Minimal change**: Only 4 lines removed
4. **No code changes**: Pure configuration fix
5. **Solves the problem**: Windows builds now work
6. **Doesn't break anything**: Other platforms unaffected
7. **Easy to understand**: Explicit per-platform settings

This is the simplest, most correct solution to the problem.
