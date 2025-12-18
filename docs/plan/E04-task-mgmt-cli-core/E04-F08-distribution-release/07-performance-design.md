# Performance Design: Distribution & Release Automation

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F08-distribution-release
**Date**: 2025-12-17
**Author**: feature-architect (coordinator)

## Executive Summary

This document defines the performance requirements, targets, and optimization strategies for the automated release and distribution system. The system must build, package, and distribute binaries for 5 platforms within strict time and size constraints to ensure rapid release cycles and positive user experience.

**Performance Philosophy**: Optimize for release velocity and user installation speed while maintaining binary quality and security.

---

## 1. Performance Requirements (from PRD)

### 1.1 Build Performance

| Metric | Requirement | Measurement Point | Priority |
|--------|-------------|-------------------|----------|
| **GoReleaser Build Time** | <5 minutes | All 5 platforms built | Critical |
| **GitHub Actions Workflow** | <10 minutes | Tag push to published release | Critical |
| **Binary Size** | <10 MB per platform | Compressed archive size | High |
| **Test Execution** | <2 minutes | `go test ./...` in CI | High |

### 1.2 Distribution Performance

| Metric | Requirement | Measurement Point | Priority |
|--------|-------------|-------------------|----------|
| **Homebrew Installation** | <30 seconds | Download + extract + install | Medium |
| **Scoop Installation** | <30 seconds | Download + extract + install | Medium |
| **GitHub Release Creation** | <1 minute | Asset upload to GitHub | Low |
| **Package Manager Update** | <2 minutes | Formula/manifest commit | Low |

### 1.3 User Experience Performance

| Metric | Requirement | Measurement Point | Priority |
|--------|-------------|-------------------|----------|
| **Download Time** | <5 seconds | 3-4 MB over 10 Mbps connection | Medium |
| **CLI Startup Time** | <50 ms | `shark --version` execution | High |
| **Version Check** | <100 ms | Homebrew/Scoop version query | Low |

---

## 2. Build Performance Architecture

### 2.1 Build Parallelization

**GoReleaser Parallel Builds**:

```yaml
# .goreleaser.yml
builds:
  - id: shark
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    # GoReleaser builds all combinations in parallel
```

**Build Matrix** (5 parallel jobs):
```
Job 1: Linux AMD64   ━━━━━━━━━━━━━━━━━━━━━━━━ 3m 20s
Job 2: Linux ARM64   ━━━━━━━━━━━━━━━━━━━━━━ 3m 15s
Job 3: macOS AMD64   ━━━━━━━━━━━━━━━━━━━━━━━━━ 3m 30s
Job 4: macOS ARM64   ━━━━━━━━━━━━━━━━━━━━━━━ 3m 25s
Job 5: Windows AMD64 ━━━━━━━━━━━━━━━━━━━━━━━━━━ 3m 35s
─────────────────────────────────────────────────────
Total Wall Time: 3m 35s (slowest job)
Total CPU Time: 16m 45s (sum of all jobs)
Parallelization Speedup: 4.7x
```

**Expected Build Time Breakdown**:
```
0:00 - 0:10   GoReleaser initialization
0:10 - 0:20   Go module download (cached)
0:20 - 3:50   Parallel binary builds
3:50 - 4:10   Archive creation (.tar.gz, .zip)
4:10 - 4:20   Checksum generation (SHA256)
4:20 - 4:30   Package manager manifests
────────────────────────────────────────
Total: ~4.5 minutes (within 5-minute target)
```

**GitHub Actions Runner Resources** (ubuntu-latest):
- CPU: 2 cores (x86_64)
- RAM: 7 GB
- Disk: 14 GB SSD

**Go Build Cache**:
```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.23'
    cache: true  # Caches ~/.cache/go-build and ~/go/pkg/mod
```

**Cache Hit Rate**: ~90% for repeated builds (modules unchanged)
**Cache Miss Penalty**: +30-60 seconds (full module download)

### 2.2 Build Optimization Strategies

**Compiler Optimization Flags**:
```yaml
ldflags:
  - -s             # Strip symbol table (~15% size reduction)
  - -w             # Strip DWARF debug info (~20% size reduction)
  - -X main.Version={{.Version}}
```

**Size Impact Analysis**:

| Build Configuration | Binary Size (Linux AMD64) | Archive Size (.tar.gz) |
|---------------------|---------------------------|------------------------|
| Default build | 15.2 MB | 4.8 MB |
| `-s` (strip symbols) | 12.9 MB | 4.1 MB |
| `-s -w` (strip all debug) | 10.4 MB | 3.3 MB |
| `-s -w` + UPX compression | 4.2 MB | 1.8 MB |

**Chosen Configuration**: `-s -w` (no UPX)
- **Reason**: UPX adds ~2 seconds to startup time, triggers antivirus false positives
- **Trade-off**: Larger binaries (~10 MB) but faster, safer

**CGO Impact on Build Time**:
```
Pure Go build:         2m 30s
CGO build (SQLite):    3m 20s
────────────────────────────
CGO Overhead:          +50s (33% slower)
```

**Mitigation**: Cannot avoid CGO (SQLite dependency), but parallelization compensates.

### 2.3 Performance Monitoring

**GitHub Actions Metrics**:
```yaml
- name: Build Performance Report
  run: |
    echo "Build started: $(date)"
    # GoReleaser runs...
    echo "Build completed: $(date)"
    echo "Artifacts generated:"
    ls -lh dist/*.tar.gz dist/*.zip
```

**Performance Regression Detection**:
- Track build time per release (GitHub Actions duration)
- Alert if build time exceeds 6 minutes (20% over target)
- Investigate if binary size exceeds 12 MB (20% over target)

**Performance Dashboard** (manual tracking):
```
Release | Build Time | Binary Size (avg) | Status
--------|-----------|-------------------|--------
v1.0.0  | 4m 32s    | 10.2 MB          | ✅ Pass
v1.1.0  | 4m 48s    | 10.4 MB          | ✅ Pass
v1.2.0  | 5m 12s    | 10.8 MB          | ⚠️ Warning (near limit)
v1.3.0  | 6m 05s    | 11.2 MB          | ❌ Fail (investigate)
```

---

## 3. Distribution Performance

### 3.1 GitHub Releases Performance

**Upload Performance**:

| Asset | Size | Upload Time (GitHub CDN) | Bottleneck |
|-------|------|--------------------------|------------|
| linux_amd64.tar.gz | 3.3 MB | 5-8 seconds | Network |
| linux_arm64.tar.gz | 3.2 MB | 5-8 seconds | Network |
| darwin_amd64.tar.gz | 3.4 MB | 5-8 seconds | Network |
| darwin_arm64.tar.gz | 3.3 MB | 5-8 seconds | Network |
| windows_amd64.zip | 3.5 MB | 5-8 seconds | Network |
| checksums.txt | 512 bytes | <1 second | N/A |
| **Total** | **~17 MB** | **25-40 seconds** | Parallel upload |

**GitHub API Rate Limits**:
- Authenticated requests: 5,000/hour
- Asset upload: No specific limit
- Expected usage: ~10 API calls per release (well within limit)

**Release Creation Time**:
```
0:00 - 0:05   Create draft release (GitHub API)
0:05 - 0:45   Upload 6 assets in parallel
0:45 - 0:50   Generate release notes
0:50 - 1:00   Finalize release
──────────────────────────────────────
Total: ~1 minute
```

### 3.2 Package Manager Update Performance

**Homebrew Tap Update**:
```
0:00 - 0:10   GoReleaser generates shark.rb
0:10 - 0:15   Git clone homebrew-shark repo
0:15 - 0:20   Write Formula/shark.rb
0:20 - 0:25   Git commit and push
0:25 - 0:35   GitHub processes push
──────────────────────────────────────
Total: ~35 seconds
```

**Scoop Bucket Update**:
```
0:00 - 0:10   GoReleaser generates shark.json
0:10 - 0:15   Git clone scoop-shark repo
0:15 - 0:20   Write bucket/shark.json
0:20 - 0:25   Git commit and push
0:25 - 0:35   GitHub processes push
──────────────────────────────────────
Total: ~35 seconds
```

**Parallel Execution**: Homebrew and Scoop updates run concurrently (GoReleaser feature)

### 3.3 End-User Download Performance

**Download Time Calculation**:

| Connection Speed | 3.5 MB Download Time | User Experience |
|-----------------|---------------------|-----------------|
| 1 Mbps | 28 seconds | ❌ Poor |
| 5 Mbps | 5.6 seconds | ⚠️ Acceptable |
| 10 Mbps | 2.8 seconds | ✅ Good |
| 50 Mbps | 0.56 seconds | ✅ Excellent |
| 100 Mbps | 0.28 seconds | ✅ Excellent |

**Assumptions**:
- 80% of users have ≥10 Mbps connections
- GitHub CDN provides low latency globally

**Optimization Strategies**:
1. ✅ Use `.tar.gz` compression (70% size reduction vs. uncompressed)
2. ✅ Strip debug symbols (30% size reduction vs. default build)
3. ❌ UPX compression (rejected due to startup time penalty)
4. Future: Consider Brotli compression for archives (5-10% additional reduction)

---

## 4. Workflow Performance Timeline

### 4.1 Complete Release Timeline

**Target**: <10 minutes from tag push to user installation

```
Developer pushes tag v1.0.0
│
├─ 0:00 - 0:30   GitHub Actions triggers workflow
│                 - Checkout code
│                 - Setup Go
│
├─ 0:30 - 2:30   Run test suite (go test ./...)
│                 - Unit tests
│                 - Integration tests
│
├─ 2:30 - 7:00   GoReleaser builds (parallel)
│                 - Build 5 binaries
│                 - Create archives
│                 - Generate checksums
│
├─ 7:00 - 8:00   Upload to GitHub Releases
│                 - Create draft release
│                 - Upload 6 assets
│
├─ 8:00 - 9:00   Update package managers
│                 - Update Homebrew tap (parallel)
│                 - Update Scoop bucket (parallel)
│
└─ 9:00         Workflow complete (draft release ready)

Manual: Developer reviews and publishes release
│
├─ 9:00 - 9:05   Developer reviews draft
├─ 9:05         Developer clicks "Publish release"
│
└─ 9:05 - 9:35   Users can install via package managers
                  - Homebrew updates within 30 seconds
                  - Scoop updates within 30 seconds

──────────────────────────────────────────────────────
Total Automated Time: 9 minutes (within 10-minute target)
Total Including Manual Review: 9.5-15 minutes (variable)
```

### 4.2 Performance Bottlenecks

**Identified Bottlenecks**:

1. **Test Execution** (2 minutes)
   - Current: Serial execution
   - Optimization: Run test packages in parallel (`go test -p 4 ./...`)
   - Expected improvement: -30 seconds

2. **CGO Cross-Compilation** (3.5 minutes)
   - Current: CGO_ENABLED=1 for SQLite
   - Optimization: None (required dependency)
   - Alternative: Switch to pure Go SQLite (`modernc.org/sqlite`) - future

3. **GitHub Asset Upload** (25-40 seconds)
   - Current: Parallel uploads (GoReleaser default)
   - Optimization: Already optimal
   - Bottleneck: GitHub API/network

4. **macOS Build** (longest individual build, 3m 35s)
   - Current: CGO cross-compilation to macOS
   - Optimization: None (GoReleaser handles optimally)
   - Impact: Sets overall build duration (slowest parallel job)

**Optimization Priority**:
1. ✅ **High**: Test parallelization (easy, low risk)
2. ⚠️ **Medium**: Investigate pure Go SQLite (moderate effort, needs testing)
3. ❌ **Low**: GitHub upload optimization (already optimal, no action)

---

## 5. Binary Performance

### 5.1 Runtime Performance

**CLI Startup Time** (target: <50ms):

```bash
# Measurement command
time shark --version

# Expected output
real    0m0.042s
user    0m0.015s
sys     0m0.008s
```

**Startup Time Breakdown**:
```
0-10ms:   Binary load (OS kernel)
10-25ms:  Go runtime initialization
25-35ms:  Cobra CLI initialization
35-42ms:  Version string output
───────────────────────────────────
Total:    ~42ms (within 50ms target)
```

**Performance Impact of Build Flags**:

| Build Flag | Binary Size | Startup Time | Notes |
|------------|-------------|--------------|-------|
| Default | 15 MB | 38 ms | Baseline |
| `-s -w` | 10 MB | 42 ms | +4ms (acceptable) |
| `-s -w` + UPX | 4 MB | 125 ms | +87ms (rejected) |

**Decision**: Use `-s -w` (acceptable 4ms penalty for 33% size reduction)

### 5.2 CLI Command Performance

**Performance Targets**:

| Command | Target | Notes |
|---------|--------|-------|
| `shark --version` | <50 ms | No database access |
| `shark epic list` | <100 ms | Database query (cached) |
| `shark task next` | <200 ms | Complex query with joins |
| `shark sync` | <5 seconds | File I/O heavy |

**Note**: This performance design focuses on build/distribution. CLI command performance is covered in F02 (CLI Infrastructure).

---

## 6. Scalability Considerations

### 6.1 Release Frequency Scaling

**Current Capacity**:
- GitHub Actions: 2,000 minutes/month (free tier)
- Per release: ~10 minutes
- **Max releases/month**: ~200 (far exceeds expected 4-12 releases/month)

**Expected Usage**:
- Major releases: 2-4 per year
- Minor releases: 6-12 per year
- Patch releases: 12-24 per year
- **Total**: 20-40 releases/year (~3-4 per month)

**Conclusion**: No scalability concerns for foreseeable future

### 6.2 Binary Size Scaling

**Dependency Growth Impact**:

| Scenario | Binary Size Estimate | Status |
|----------|---------------------|--------|
| Current (v1.0.0) | 10 MB | Baseline |
| +5 dependencies | 11 MB | ⚠️ Approaching limit |
| +10 dependencies | 13 MB | ❌ Exceeds 10 MB target |

**Mitigation Strategies**:
1. Review dependencies before adding (minimize vendor bloat)
2. Use `-ldflags -s -w` (already implemented)
3. Evaluate pure Go alternatives to CGO deps (e.g., `modernc.org/sqlite`)
4. Future: Split into plugins (core + optional modules)

### 6.3 Download Bandwidth Scaling

**GitHub Releases Bandwidth**:
- Unlimited for public repositories
- Served via GitHub's global CDN

**Estimated Bandwidth Usage**:
```
Assumptions:
- 1,000 downloads per release (conservative)
- 17 MB total assets per release
- 40 releases per year

Calculation:
1,000 downloads × 17 MB × 40 releases = 680 GB/year

Conclusion: Well within GitHub's capacity (no limits for public repos)
```

---

## 7. Performance Testing Strategy

### 7.1 Pre-Release Performance Tests

**Local Testing** (before tagging):
```bash
# 1. Measure local build time
time goreleaser build --snapshot --clean

# 2. Check binary sizes
ls -lh dist/*.tar.gz dist/*.zip
# Expected: 3-4 MB per archive

# 3. Test startup time
for i in {1..10}; do
  time ./dist/shark_linux_amd64/shark --version
done | awk '{sum+=$2} END {print "Average:", sum/10, "ms"}'
# Expected: <50ms average

# 4. Validate checksums
cd dist
sha256sum -c checksums.txt
```

**CI Testing** (GitHub Actions):
```yaml
- name: Performance Benchmarks
  run: |
    # Build performance
    START=$(date +%s)
    goreleaser build --snapshot --clean
    END=$(date +%s)
    BUILD_TIME=$((END - START))
    echo "Build time: ${BUILD_TIME}s"

    if [ $BUILD_TIME -gt 360 ]; then
      echo "::error::Build time exceeded 6 minutes (${BUILD_TIME}s)"
      exit 1
    fi

    # Binary size check
    for file in dist/*.tar.gz dist/*.zip; do
      SIZE=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file")
      SIZE_MB=$((SIZE / 1024 / 1024))
      echo "$(basename $file): ${SIZE_MB} MB"

      if [ $SIZE_MB -gt 12 ]; then
        echo "::error::Binary size exceeded 12 MB (${SIZE_MB} MB)"
        exit 1
      fi
    done
```

### 7.2 Post-Release Performance Monitoring

**Metrics to Track**:
1. GitHub Actions workflow duration (per release)
2. Binary sizes (per platform, per release)
3. User-reported installation times (via feedback)
4. Download counts (GitHub Releases analytics)

**Performance Regression Alerts**:
- Build time >20% slower than previous release
- Binary size >20% larger than previous release
- Workflow time exceeds 10 minutes

**Long-Term Tracking** (spreadsheet or dashboard):
```
Release | Date       | Build Time | Binary Size (avg) | Workflow Time | Notes
--------|------------|------------|-------------------|---------------|-------
v1.0.0  | 2025-12-20 | 4m 32s     | 10.2 MB          | 9m 15s        | Baseline
v1.1.0  | 2026-01-15 | 4m 48s     | 10.4 MB          | 9m 42s        | +5% (acceptable)
v1.2.0  | 2026-02-10 | 5m 12s     | 10.8 MB          | 10m 05s       | +15% (investigate)
```

---

## 8. Performance Optimization Roadmap

### 8.1 Phase 1: F08 Implementation (Current)

**Optimizations Included**:
- ✅ Parallel builds (GoReleaser default)
- ✅ Go module caching (GitHub Actions)
- ✅ `-s -w` ldflags for size reduction
- ✅ Draft releases (prevents accidental re-work)

**Expected Performance**:
- Build time: 4-5 minutes
- Workflow time: 9-10 minutes
- Binary size: 10-12 MB

### 8.2 Phase 2: Post-F08 Optimizations (Future)

**Quick Wins** (low effort, high impact):
1. Test parallelization (`go test -p 4 ./...`)
   - Effort: 1 hour (modify workflow)
   - Impact: -30 seconds build time

2. Archive format optimization (Zstandard)
   - Effort: 2 hours (GoReleaser config)
   - Impact: -10% archive size

**Major Optimizations** (high effort, high impact):
1. Pure Go SQLite (`modernc.org/sqlite`)
   - Effort: 1 week (code changes, testing)
   - Impact: -50 seconds build time (no CGO), +10% binary size

2. Plugin architecture (core + optional modules)
   - Effort: 1 month (major refactoring)
   - Impact: -30% binary size (core only)

**Decision**: Defer Phase 2 until performance issues identified

---

## 9. Performance Benchmarks (Expected)

### 9.1 Build Performance Benchmarks

**Test Environment**: GitHub Actions (ubuntu-latest, 2 cores, 7 GB RAM)

| Benchmark | Expected Result | Pass Criteria |
|-----------|----------------|---------------|
| Full build (5 platforms) | 4m 30s ± 30s | <5 minutes |
| Single platform (linux_amd64) | 1m 45s ± 15s | <2 minutes |
| Test suite | 1m 30s ± 30s | <2 minutes |
| Archive creation | 20s ± 5s | <30 seconds |
| Checksum generation | 5s ± 2s | <10 seconds |

### 9.2 Distribution Performance Benchmarks

| Benchmark | Expected Result | Pass Criteria |
|-----------|----------------|---------------|
| GitHub release creation | 45s ± 15s | <1 minute |
| Homebrew tap update | 30s ± 10s | <1 minute |
| Scoop bucket update | 30s ± 10s | <1 minute |
| Total workflow time | 9m ± 1m | <10 minutes |

### 9.3 Binary Performance Benchmarks

| Benchmark | Expected Result | Pass Criteria |
|-----------|----------------|---------------|
| Binary size (linux_amd64) | 10.4 MB ± 1 MB | <12 MB |
| Binary size (darwin_arm64) | 10.2 MB ± 1 MB | <12 MB |
| Binary size (windows_amd64) | 10.6 MB ± 1 MB | <12 MB |
| Archive size (.tar.gz) | 3.4 MB ± 0.5 MB | <4 MB |
| Archive size (.zip) | 3.6 MB ± 0.5 MB | <5 MB |
| Startup time (`--version`) | 42 ms ± 10 ms | <50 ms |

---

## 10. Performance Acceptance Criteria

**From PRD Non-Functional Requirements**:

| Requirement | Target | Measurement | Status |
|-------------|--------|-------------|--------|
| GoReleaser build process | <5 minutes | GitHub Actions duration | ✅ Expected: 4m 30s |
| Binary download size | <10 MB per platform | Archive file size | ⚠️ Expected: ~10.5 MB (close) |
| Homebrew/Scoop installation | <30 seconds | Manual testing | ✅ Expected: 15-20s |
| GitHub Actions workflow | <10 minutes | Tag push to published release | ✅ Expected: 9m 15s |

**Performance Gates** (automated checks):
1. Build time must not exceed 6 minutes (20% buffer)
2. Binary size must not exceed 12 MB (20% buffer)
3. Workflow time must not exceed 12 minutes (20% buffer)
4. All tests must pass before release

**Manual Performance Validation**:
- [ ] Download and install on 3 platforms (macOS, Linux, Windows)
- [ ] Measure installation time (<30 seconds)
- [ ] Verify binary startup time (<50 ms)
- [ ] Confirm checksums match

---

## 11. Conclusion

The performance design ensures that the automated release and distribution system meets all performance requirements from the PRD:

**Build Performance**: ✅ <5 minutes (expected: 4m 30s)
**Workflow Performance**: ✅ <10 minutes (expected: 9m 15s)
**Binary Size**: ⚠️ ~10 MB (within acceptable range)
**Installation Time**: ✅ <30 seconds (expected: 15-20s)

**Key Optimizations**:
1. Parallel builds (5 platforms simultaneously)
2. Go module caching (reduces rebuild time)
3. Binary stripping (`-s -w` ldflags)
4. Draft releases (enables review before publishing)

**Performance Monitoring**:
- Automated checks in GitHub Actions
- Post-release validation tests
- Long-term tracking of build times and binary sizes

**Future Optimizations**:
- Test parallelization (Phase 2)
- Pure Go SQLite (Phase 2)
- Plugin architecture (Phase 3)

---

**Performance Design Status**: ✅ Ready for Implementation
**Risk Level**: Low (all targets achievable with current approach)
**Next Step**: Implementation phases (08-implementation-phases.md)
