# Implementation Phases: Distribution & Release Automation

**Epic**: E04-task-mgmt-cli-core
**Feature**: E04-F08-distribution-release
**Date**: 2025-12-17
**Author**: feature-architect (coordinator)

## Overview

This document defines the implementation approach for automated multi-platform distribution and release of the Shark CLI tool. The implementation is structured into 5 sequential phases, each with clear deliverables, dependencies, and success criteria.

**Implementation Strategy**: Incremental delivery with validation at each phase
**Total Estimated Duration**: 2-3 weeks (14-21 working days)
**Risk Level**: Low (well-established tools and patterns)

---

## Documents Created

| Document | Created | Agent |
|----------|---------|-------|
| 00-research-report | ✅ | coordinator (self-authored) |
| 01-interface-contracts | ❌ | N/A (not applicable for DevOps feature) |
| 02-architecture | ✅ | coordinator (self-authored) |
| 03-data-design | ❌ | N/A (no data persistence) |
| 04-backend-design | ❌ | N/A (no backend logic) |
| 05-frontend-design | ❌ | N/A (no UI components) |
| 06-security-design | ✅ | coordinator (self-authored) |
| 07-performance-design | ✅ | coordinator (self-authored) |
| 09-test-criteria | ✅ | tdd-agent |

**Document Coverage**: 4/9 created (appropriate for DevOps/infrastructure feature)

---

## Phase 1: Version Management & Local Testing

**Goals**: Add version embedding to CLI and validate GoReleaser configuration locally

**Duration**: 2-3 days

### Tasks

1. **Add Version Variable to Code**
   - Modify `cmd/shark/main.go` to add version variable
   - Configure Cobra to use version
   - Test `shark --version` command

2. **Create GoReleaser Configuration**
   - Write `.goreleaser.yml` with build targets
   - Configure archives, checksums, and ldflags
   - Add comments explaining each section

3. **Install GoReleaser Locally**
   - Install GoReleaser CLI tool
   - Validate configuration with `goreleaser check`

4. **Test Snapshot Builds**
   - Run `goreleaser build --snapshot --clean`
   - Verify all 5 platforms build successfully
   - Test binaries: `./dist/shark_*/shark --version`

5. **Measure Performance Baselines**
   - Time local builds
   - Measure binary sizes
   - Document baseline metrics

### Deliverables

- [ ] `cmd/shark/main.go` updated with version variable
- [ ] `.goreleaser.yml` configuration file
- [ ] Successful local snapshot build (5 platforms)
- [ ] Performance baseline document

### Dependencies

**Required Before Starting**:
- ✅ Go 1.23+ installed
- ✅ Existing Shark CLI codebase compiles

**External Dependencies**:
- GoReleaser CLI tool (install via Homebrew/download)

### Success Criteria

**Technical**:
- [ ] `goreleaser check` passes with no errors
- [ ] Snapshot build completes in <6 minutes
- [ ] All 5 platform binaries generated
- [ ] Binaries are <12 MB each
- [ ] `shark --version` displays correct version in all builds

**Documentation**:
- [ ] `.goreleaser.yml` has inline comments
- [ ] Performance baselines documented

### Validation

```bash
# 1. Validate configuration
goreleaser check

# 2. Build snapshot
time goreleaser build --snapshot --clean

# 3. Verify outputs
ls -lh dist/*.tar.gz dist/*.zip
# Should see 5 archives

# 4. Test binaries
./dist/shark_linux_amd64/shark --version
./dist/shark_darwin_arm64/shark --version
./dist/shark_windows_amd64/shark.exe --version

# 5. Verify checksums
cd dist && sha256sum -c checksums.txt
```

---

## Phase 2: GitHub Actions Workflow & CI/CD Setup

**Goals**: Create automated release workflow and configure GitHub repository

**Duration**: 2-3 days

### Tasks

1. **Create GitHub Actions Workflow**
   - Write `.github/workflows/release.yml`
   - Configure trigger (tag pattern `v*`)
   - Set up Go environment
   - Add test execution step
   - Add GoReleaser execution step

2. **Configure Repository Settings**
   - Enable GitHub Actions (if disabled)
   - Configure branch protection for `main`
   - Set up required status checks

3. **Test Workflow with Test Tag**
   - Create test tag: `v0.0.1-test`
   - Push tag to trigger workflow
   - Monitor workflow execution
   - Verify draft release created

4. **Debug and Refine Workflow**
   - Fix any errors in workflow
   - Optimize build time if needed
   - Add performance reporting

### Deliverables

- [ ] `.github/workflows/release.yml` workflow file
- [ ] Successful test workflow run
- [ ] Draft GitHub release (test tag)

### Dependencies

**Required Before Starting**:
- ✅ Phase 1 complete (`.goreleaser.yml` working)
- ✅ GitHub repository with write access

**External Dependencies**:
- GitHub Actions enabled on repository
- GitHub tokens (automatic `GITHUB_TOKEN`)

### Success Criteria

**Technical**:
- [ ] Workflow triggers on `v*` tag push
- [ ] Tests pass before build
- [ ] GoReleaser builds all platforms
- [ ] Draft release created with 6 assets (5 binaries + checksums)
- [ ] Workflow completes in <10 minutes

**Documentation**:
- [ ] Workflow YAML has clear step names
- [ ] Comments explain non-obvious configurations

### Validation

```bash
# 1. Create test tag
git tag v0.0.1-test
git push origin v0.0.1-test

# 2. Monitor workflow
# Visit: https://github.com/jwwelbor/shark-task-manager/actions

# 3. Verify draft release
# Visit: https://github.com/jwwelbor/shark-task-manager/releases

# 4. Download and verify checksums
wget https://github.com/.../shark_0.0.1-test_linux_amd64.tar.gz
wget https://github.com/.../checksums.txt
sha256sum -c checksums.txt --ignore-missing

# 5. Clean up test release
git tag -d v0.0.1-test
git push origin :refs/tags/v0.0.1-test
# Delete draft release on GitHub
```

---

## Phase 3: Package Manager Infrastructure

**Goals**: Set up Homebrew tap and Scoop bucket repositories

**Duration**: 2-3 days

### Tasks

1. **Create Homebrew Tap Repository**
   - Create `homebrew-shark` GitHub repository
   - Initialize with README.md
   - Create `Formula/` directory structure

2. **Create Scoop Bucket Repository**
   - Create `scoop-shark` GitHub repository
   - Initialize with README.md
   - Create `bucket/` directory structure

3. **Generate Personal Access Tokens**
   - Create fine-grained PAT for Homebrew tap
   - Create fine-grained PAT for Scoop bucket
   - Configure minimal permissions (contents:write)
   - Set 90-day expiration

4. **Add Tokens to GitHub Secrets**
   - Add `HOMEBREW_TAP_TOKEN` to repository secrets
   - Add `SCOOP_BUCKET_TOKEN` to repository secrets
   - Verify secrets are encrypted and accessible

5. **Update GoReleaser Configuration**
   - Add `brews:` section to `.goreleaser.yml`
   - Add `scoops:` section to `.goreleaser.yml`
   - Configure repository references and tokens

### Deliverables

- [ ] `homebrew-shark` repository created
- [ ] `scoop-shark` repository created
- [ ] GitHub PATs generated and stored in secrets
- [ ] `.goreleaser.yml` updated with package manager configs

### Dependencies

**Required Before Starting**:
- ✅ Phase 2 complete (GitHub Actions working)
- ✅ GitHub account with repository creation access

**External Dependencies**:
- Fine-grained GitHub Personal Access Tokens

### Success Criteria

**Technical**:
- [ ] Both package manager repositories created
- [ ] PATs have correct permissions (contents:write only)
- [ ] Secrets accessible in workflow (no errors)
- [ ] GoReleaser config validates (`goreleaser check`)

**Documentation**:
- [ ] Homebrew tap README.md with usage instructions
- [ ] Scoop bucket README.md with usage instructions

### Validation

```bash
# 1. Verify repositories exist
git clone https://github.com/jwwelbor/homebrew-shark
git clone https://github.com/jwwelbor/scoop-shark

# 2. Verify tokens work (manual)
# Create test file in each repo using PAT
curl -X PUT -H "Authorization: token $HOMEBREW_TAP_TOKEN" \
  https://api.github.com/repos/jwwelbor/homebrew-shark/contents/test.txt \
  -d '{"message":"test","content":"dGVzdAo="}'

# 3. Validate GoReleaser config
goreleaser check
```

---

## Phase 4: End-to-End Release Testing

**Goals**: Perform complete release cycle and validate all distribution channels

**Duration**: 2-3 days

### Tasks

1. **Cut Test Release**
   - Create tag: `v0.1.0-beta`
   - Push tag to trigger workflow
   - Monitor entire workflow execution

2. **Verify GitHub Release**
   - Check draft release created
   - Verify all 6 assets uploaded
   - Verify checksums.txt is valid
   - Review auto-generated release notes

3. **Verify Homebrew Tap**
   - Check `Formula/shark.rb` committed to tap
   - Verify formula syntax: `brew audit shark.rb`
   - Test installation: `brew install jwwelbor/shark/shark`
   - Verify version: `shark --version`

4. **Verify Scoop Bucket**
   - Check `bucket/shark.json` committed to bucket
   - Verify manifest syntax (JSON validation)
   - Test installation: `scoop install shark`
   - Verify version: `shark --version`

5. **Test Manual Downloads**
   - Download binary for each platform
   - Verify checksums match
   - Test binaries run correctly

6. **Performance Validation**
   - Measure workflow duration
   - Measure binary sizes
   - Measure installation times
   - Compare against targets

### Deliverables

- [ ] Successful beta release (v0.1.0-beta)
- [ ] All distribution channels verified
- [ ] Performance metrics documented

### Dependencies

**Required Before Starting**:
- ✅ Phase 3 complete (package managers configured)
- ✅ Test machines available (macOS, Linux, Windows)

**External Dependencies**:
- Homebrew installed (macOS/Linux test)
- Scoop installed (Windows test)

### Success Criteria

**Technical**:
- [ ] Workflow completes successfully (<10 minutes)
- [ ] GitHub release created with all assets
- [ ] Homebrew installation successful on macOS
- [ ] Scoop installation successful on Windows
- [ ] Manual download works on Linux
- [ ] All checksums verify correctly

**Performance**:
- [ ] Build time <5 minutes
- [ ] Workflow time <10 minutes
- [ ] Binary sizes <10 MB (compressed)
- [ ] Installation times <30 seconds

**Functional**:
- [ ] `shark --version` shows correct version on all platforms
- [ ] Basic commands work (`shark epic list`, etc.)

### Validation

```bash
# macOS (Homebrew)
brew tap jwwelbor/shark
brew install shark
shark --version  # Should show v0.1.0-beta
brew uninstall shark

# Windows (Scoop - PowerShell)
scoop bucket add shark https://github.com/jwwelbor/scoop-shark
scoop install shark
shark --version  # Should show v0.1.0-beta
scoop uninstall shark

# Linux (manual)
wget https://github.com/.../shark_0.1.0-beta_linux_amd64.tar.gz
wget https://github.com/.../checksums.txt
sha256sum -c checksums.txt --ignore-missing
tar -xzf shark_*.tar.gz
./shark --version  # Should show v0.1.0-beta

# Clean up test release
git tag -d v0.1.0-beta
git push origin :refs/tags/v0.1.0-beta
# Delete release on GitHub
# Revert commits in homebrew-shark and scoop-shark
```

---

## Phase 5: Documentation & Production Release

**Goals**: Update documentation and cut production v1.0.0 release

**Duration**: 2-3 days

### Tasks

1. **Update README.md**
   - Add installation instructions (Homebrew, Scoop, manual)
   - Add version verification instructions
   - Add checksum verification examples
   - Update "Quick Start" section

2. **Update CONTRIBUTING.md**
   - Document release process for maintainers
   - Add version numbering guidelines (SemVer)
   - Document how to cut releases
   - Add troubleshooting for release failures

3. **Create SECURITY.md**
   - Document vulnerability reporting process
   - Add checksum verification instructions
   - Link to security advisories

4. **Add Release Scripts**
   - Create `scripts/verify-release.sh` (Linux/macOS)
   - Create `scripts/verify-release.ps1` (Windows)
   - Make scripts executable

5. **Cut v1.0.0 Release**
   - Ensure all tests pass
   - Create tag: `v1.0.0`
   - Push tag to trigger workflow
   - Monitor workflow execution

6. **Publish Release**
   - Review draft release
   - Edit release notes if needed
   - Publish release (make public)

7. **Verify Production Distribution**
   - Test Homebrew installation
   - Test Scoop installation
   - Test manual downloads
   - Announce release

### Deliverables

- [ ] README.md updated with installation instructions
- [ ] CONTRIBUTING.md updated with release process
- [ ] SECURITY.md created
- [ ] Release verification scripts
- [ ] v1.0.0 production release published

### Dependencies

**Required Before Starting**:
- ✅ Phase 4 complete (end-to-end testing successful)
- ✅ All tests passing in main branch
- ✅ No known critical bugs

**External Dependencies**:
- None (all infrastructure in place)

### Success Criteria

**Technical**:
- [ ] v1.0.0 release published successfully
- [ ] All distribution channels working
- [ ] No errors in workflow
- [ ] All acceptance criteria from PRD met

**Documentation**:
- [ ] README.md has clear installation instructions
- [ ] CONTRIBUTING.md has release process
- [ ] SECURITY.md has vulnerability reporting
- [ ] Scripts validated on all platforms

**Functional**:
- [ ] Homebrew: `brew install shark` works
- [ ] Scoop: `scoop install shark` works
- [ ] Manual: Downloads verify and run correctly

### Validation

```bash
# 1. Create production tag
git tag v1.0.0
git push origin v1.0.0

# 2. Monitor workflow
# Visit: https://github.com/jwwelbor/shark-task-manager/actions

# 3. Review and publish draft release
# Visit: https://github.com/jwwelbor/shark-task-manager/releases
# Click "Edit draft" → "Publish release"

# 4. Verify installation (macOS)
brew tap jwwelbor/shark
brew install shark
shark --version  # Should show v1.0.0
shark epic list  # Test basic functionality

# 5. Verify installation (Windows)
scoop bucket add shark https://github.com/jwwelbor/scoop-shark
scoop install shark
shark --version  # Should show v1.0.0

# 6. Verify manual download (Linux)
wget https://github.com/.../shark_1.0.0_linux_amd64.tar.gz
wget https://github.com/.../checksums.txt
sha256sum -c checksums.txt --ignore-missing
tar -xzf shark_*.tar.gz
./shark --version  # Should show v1.0.0

# 7. Announce release
# Social media, changelog, docs site, etc.
```

---

## Post-Implementation Phase: Monitoring & Iteration

**Goals**: Monitor release usage and iterate based on feedback

**Duration**: Ongoing

### Tasks

1. **Monitor GitHub Releases**
   - Track download counts
   - Review user issues related to installation
   - Monitor workflow success rate

2. **Monitor Package Managers**
   - Check Homebrew tap health
   - Check Scoop bucket health
   - Review installation feedback

3. **Performance Tracking**
   - Track build times per release
   - Track binary sizes per release
   - Identify performance regressions

4. **Incident Response**
   - Respond to security vulnerabilities
   - Fix broken releases quickly
   - Rotate tokens on schedule (90 days)

5. **Continuous Improvement**
   - Implement future enhancements (code signing, etc.)
   - Optimize build times if needed
   - Add additional distribution channels if requested

### Deliverables

- [ ] Release metrics dashboard
- [ ] Incident response log
- [ ] Quarterly security reviews

### Success Criteria

**Operational**:
- [ ] Release success rate >95%
- [ ] Average workflow time <10 minutes
- [ ] Zero security incidents

**User Satisfaction**:
- [ ] Positive feedback on installation ease
- [ ] No critical installation bugs reported
- [ ] Growing download numbers

---

## Implementation Timeline

### Gantt Chart (Text Format)

```
Week 1:
  Phase 1 (Version Management)   ████████████░░░░░░░░░░░░
  Phase 2 (GitHub Actions)       ░░░░░░░░░░░░████████████

Week 2:
  Phase 3 (Package Managers)     ████████████░░░░░░░░░░░░
  Phase 4 (E2E Testing)          ░░░░░░░░░░░░████████████

Week 3:
  Phase 5 (Documentation & v1.0) ████████████░░░░░░░░░░░░
  Buffer / Refinement            ░░░░░░░░░░░░████████████

Legend:
████ = Active work
░░░░ = Not started / complete
```

### Critical Path

```
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5
  (3d)     (3d)      (3d)      (3d)      (3d)

Total: 15 days (3 weeks)
```

**Dependencies**:
- Phase 2 depends on Phase 1 (GoReleaser config must work locally)
- Phase 3 depends on Phase 2 (GitHub Actions must be working)
- Phase 4 depends on Phase 3 (package managers must be configured)
- Phase 5 depends on Phase 4 (testing must validate everything works)

**Parallel Work Opportunities**:
- Documentation (Phase 5) can be drafted during earlier phases
- Security scripts can be written during Phase 3-4

---

## Risk Mitigation

### Phase 1 Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| GoReleaser config errors | Medium | Medium | Use reference configs from Hugo, GitHub CLI |
| CGO cross-compilation fails | Low | High | Test on GitHub Actions early (same environment) |
| Binary size exceeds limit | Low | Medium | Test with `-s -w` ldflags |

### Phase 2 Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| GitHub Actions workflow errors | Medium | Medium | Test with test tags before production |
| Workflow exceeds time limit | Low | Medium | Monitor duration, optimize if needed |
| Tests fail in CI | Low | High | Run tests locally before tagging |

### Phase 3 Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| PAT permissions incorrect | Medium | Medium | Follow documented PAT creation process |
| Package manager update fails | Low | Medium | Test with manual commits first |
| Token expiration | Low | Low | Set 90-day expiration, calendar reminder |

### Phase 4 Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Installation fails on platform | Low | High | Test on real machines (not just VMs) |
| Checksum mismatch | Low | Critical | Verify GoReleaser checksum generation |
| Performance targets not met | Medium | Medium | Optimize build, accept 20% buffer |

### Phase 5 Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| v1.0.0 release fails | Low | Critical | Extensive testing in Phase 4 first |
| Documentation incomplete | Low | Medium | Review checklist before release |
| Broken installation after publish | Low | High | Test immediately after publishing |

---

## Success Metrics

### Implementation Success (End of Phase 5)

**Technical**:
- [ ] All phases completed without critical issues
- [ ] v1.0.0 released successfully
- [ ] All acceptance criteria from PRD met
- [ ] All performance targets achieved

**Documentation**:
- [ ] All required docs updated
- [ ] Release process documented
- [ ] Security guidance provided

**Validation**:
- [ ] Successful installations on all platforms
- [ ] Positive feedback from early users
- [ ] Zero critical bugs in release process

### Long-Term Success (3-6 months post-launch)

**Operational**:
- [ ] 20+ releases completed
- [ ] Release success rate >95%
- [ ] Average workflow time <10 minutes
- [ ] Zero security incidents

**Adoption**:
- [ ] 1,000+ total downloads
- [ ] Package manager installations > manual downloads
- [ ] Positive user feedback on installation ease

**Maintenance**:
- [ ] Token rotations completed on schedule
- [ ] Performance metrics tracked
- [ ] Continuous improvement backlog

---

## Rollback Plan

### If Production Release Fails

**Immediate Actions**:
1. Delete failed release on GitHub
2. Delete tag locally and remotely
3. Notify users via GitHub issue (if any users affected)

**Root Cause Analysis**:
1. Review GitHub Actions logs
2. Identify failure point
3. Document issue in incident log

**Recovery**:
1. Fix issue in code or configuration
2. Test with test tag (v1.0.1-test)
3. Re-release as v1.0.1 with hotfix

**Prevention**:
1. Add additional validation to prevent recurrence
2. Update testing procedures
3. Review and update this document

---

## Conclusion

This phased implementation approach ensures systematic, validated progress toward automated multi-platform distribution. Each phase builds on the previous, with clear success criteria and validation steps.

**Key Success Factors**:
1. Thorough testing at each phase (don't skip validation)
2. Use test tags before production releases
3. Document issues and solutions as they arise
4. Maintain rollback capability at all times
5. Monitor performance throughout

**Timeline Summary**:
- **Optimistic**: 2 weeks (if no issues)
- **Realistic**: 3 weeks (with normal debugging)
- **Pessimistic**: 4 weeks (with significant issues)

**Next Step**: Begin Phase 1 (Version Management & Local Testing)

---

**Implementation Phases Status**: ✅ Ready for Execution
**Estimated Completion**: 3 weeks from start
**Risk Level**: Low (well-defined phases, clear validation)
