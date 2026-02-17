# Success Metrics

**Epic**: [Service Layer Architecture Refactoring](./epic.md)

---

## Overview

This epic's success is measured by **code quality improvements** (reduced duplication, better testability) and **developer productivity gains** (faster feature development, easier maintenance). Metrics are grouped into three categories: Architecture Quality, Developer Experience, and System Health.

---

## Architecture Quality Metrics

### AQ1: Code Reduction in Command Layer

**What**: Measure total lines of code in `internal/cli/commands/` before and after refactoring.

**Baseline** (measured 2026-02-16):
- Total lines: 43,590 (across 47 command files)
- Average file size: 927 lines
- Largest files: `task.go` (2,671), `feature.go` (2,090), `epic.go` (1,739)

**Target**:
- Total lines: <15,000 (65% reduction)
- Average file size: <320 lines (65% reduction)
- Largest files: <500 lines each (70-80% reduction)

**Measurement Method**:
```bash
# Count lines in command layer
find internal/cli/commands -name "*.go" ! -name "*_test.go" -exec wc -l {} + | tail -1

# Before: 43,590
# Target: <15,000
```

**Target Value**: 65% code reduction in command layer
**Success Threshold**: >50% reduction (acceptable), >65% reduction (excellent)
**Collection Frequency**: Per PR, aggregated at epic completion

---

### AQ2: Service Layer Test Coverage

**What**: Measure test coverage of business logic in service layer.

**Baseline** (measured 2026-02-16):
- `internal/services/epic_service.go`: 85.7% coverage
- `internal/services/feature_service.go`: 82.3% coverage
- Task service not yet implemented: 0% coverage

**Target**:
- `TaskService`: >80% coverage
- `FeatureService`: maintain >80% coverage
- `EpicService`: maintain >85% coverage
- Overall `internal/services/`: >80% coverage

**Measurement Method**:
```bash
# Generate coverage report
make test-coverage
# Check services/ coverage in coverage.html
go test -coverprofile=coverage.out ./internal/services/...
go tool cover -func=coverage.out | grep total
```

**Target Value**: >80% coverage across all services
**Success Threshold**: >70% acceptable, >80% excellent
**Collection Frequency**: Per PR, gating for merge

---

### AQ3: Duplication Elimination

**What**: Measure reduction in duplicated code patterns across command files.

**Baseline** (measured via code analysis):
- Key lookup pattern duplicated: 32 times
- Error handling pattern duplicated: 47 times
- Status validation pattern duplicated: 28 times
- Progress calculation pattern duplicated: 15 times

**Target**:
- Key lookup centralized in service layer: 1 canonical implementation
- Error handling centralized: 1-3 canonical implementations
- Status validation centralized in `workflow.Service`: 1 implementation
- Progress calculation centralized in services: 3 implementations (Task/Feature/Epic)

**Measurement Method**:
```bash
# Count pattern occurrences before/after
grep -r "NormalizeKey" internal/cli/commands/*.go | wc -l  # Before: 32
grep -r "GetByKey" internal/cli/commands/*.go | wc -l      # After: <5

# Duplication ratio = (unique patterns / total occurrences)
# Before: 4 patterns / 122 occurrences = 3.3% (high duplication)
# After: 8 patterns / 15 occurrences = 53% (low duplication)
```

**Target Value**: >80% reduction in duplicated patterns
**Success Threshold**: >60% reduction acceptable, >80% excellent
**Collection Frequency**: At epic completion (manual code analysis)

---

### AQ4: Business Logic in Service Layer

**What**: Verify that 100% of business logic is in service layer, not commands or repositories.

**Baseline**:
- Business logic in commands: ~40-45% of total logic
- Business logic in services: ~5% of total logic
- Business logic in repositories: ~10% of total logic (progress calc, status derivation)

**Target**:
- Business logic in commands: 0% (only parsing and formatting)
- Business logic in services: 90%+ (centralized)
- Business logic in repositories: 0% (only data access)

**Measurement Method**:
Manual code review checklist:
- [ ] No `if` statements for business rules in commands (only for formatting)
- [ ] No calculations in commands (delegates to services)
- [ ] No database transactions in commands (orchestrated by services)
- [ ] No progress/health calculations in repositories
- [ ] All validation logic in services via `workflow.Service`

**Target Value**: 100% of business logic in service layer
**Success Threshold**: >95% acceptable, 100% excellent
**Collection Frequency**: Code review per PR, audit at epic completion

---

## Developer Experience Metrics

### DX1: Time to Add New CLI Command

**What**: Measure developer time to implement a new CLI command from scratch.

**Baseline** (from user journey analysis):
- Average time: 8 hours (4h coding, 2h testing, 2h PR review)
- Lines of code: 350 lines (including 200 lines of boilerplate)

**Target**:
- Average time: 2 hours (1h coding, 0.5h testing, 0.5h review)
- Lines of code: 150 lines (80 lines command wrapper, 30 lines service method, 40 lines test)

**Measurement Method**:
- Track 3-5 new commands added post-refactoring
- Measure PR creation time to merge time
- Measure lines of code added
- Developer survey: "How long did this take compared to previous commands?"

**Target Value**: 75% reduction in development time
**Success Threshold**: >50% reduction acceptable, >75% excellent
**Collection Frequency**: Monthly for 3 months post-epic

---

### DX2: PR Review Cycle Time

**What**: Measure time from PR creation to merge for architecture-related feedback.

**Baseline** (from maintainer experience):
- Average review cycles: 2-3 cycles
- Average time to merge: 3-5 days
- Common feedback: "Move business logic out of commands" (60% of PRs)

**Target**:
- Average review cycles: 1 cycle (architecture is clear by construction)
- Average time to merge: 1 day
- Architecture feedback: <10% of PRs

**Measurement Method**:
- GitHub PR metrics: time from open to first review, time to merge
- Review comment analysis: count architecture-related comments
- Maintainer survey: "How often do you give architecture feedback?"

**Target Value**: 70% reduction in review cycles (3→1)
**Success Threshold**: >50% reduction acceptable, >70% excellent
**Collection Frequency**: Monthly for 3 months post-epic

---

### DX3: Test Execution Speed

**What**: Measure service layer test suite execution time.

**Baseline**:
- CLI integration tests (with database): ~450ms per test
- Total CLI test suite: ~45 seconds

**Target**:
- Service unit tests (with mocks): <1ms per test
- Service layer test suite: <100ms total
- Full test suite: <60 seconds (maintains or improves current speed)

**Measurement Method**:
```bash
# Time service tests
time go test -v ./internal/services/...

# Time full suite
make test | grep "ok\|FAIL" | awk '{sum+=$3} END {print sum "s"}'
```

**Target Value**: Service tests <100ms, full suite maintains <60s
**Success Threshold**: Service <500ms acceptable, <100ms excellent
**Collection Frequency**: Per PR (CI gating)

---

### DX4: Onboarding Time for New Contributors

**What**: Measure time for new contributor to understand architecture and make first meaningful PR.

**Baseline** (estimated from maintainer experience):
- Time to first PR merged: 2-3 PRs over 2-3 weeks
- Confusion points: "Where does this code go?" (100% of new contributors)
- Rework: 60% of first PRs require architectural changes

**Target**:
- Time to first PR merged: 1 PR in 1 week
- Confusion points: <20% need architectural guidance
- Rework: <20% of first PRs require architectural changes

**Measurement Method**:
- Track new contributor PRs post-refactoring
- Survey new contributors: "Was the architecture clear?" (1-5 scale)
- Measure PR rework rate (number of review cycles)

**Target Value**: 70% reduction in onboarding time
**Success Threshold**: >50% reduction acceptable, >70% excellent
**Collection Frequency**: Quarterly (requires multiple new contributors)

---

## System Health Metrics

### SH1: HTTP API Feature Parity

**What**: Measure percentage of CLI commands that have equivalent HTTP API endpoints.

**Baseline**:
- CLI commands: 47 total
- API endpoints: ~15 (32% coverage)
- Feature gaps: task filtering, agent queries, dependency checking, progress calculations

**Target**:
- CLI commands: 47 total
- API endpoints: 47 (100% coverage)
- Feature gaps: 0 (all CLI features available via API)

**Measurement Method**:
```bash
# Count CLI commands
ls internal/cli/commands/*.go | grep -v test | wc -l

# Count API endpoints
grep -r "router\." cmd/server/*.go | grep -E "(GET|POST|PUT|DELETE)" | wc -l

# API parity = (API endpoints / CLI commands) × 100%
```

**Target Value**: 100% API feature parity with CLI
**Success Threshold**: >80% acceptable, 100% excellent
**Collection Frequency**: End of epic, quarterly thereafter

---

### SH2: Regression Rate

**What**: Track number of bugs introduced during refactoring.

**Baseline** (typical refactoring):
- Regressions per 1,000 lines changed: ~2-5 bugs
- Expected regressions for 30K LOC refactoring: 60-150 bugs

**Target**:
- Regressions per 1,000 lines changed: <0.5 bugs
- Total regressions for epic: <15 bugs
- Critical regressions (data loss, crashes): 0

**Measurement Method**:
- GitHub issues tagged with "regression" and "E15"
- User reports of broken functionality post-refactoring
- Test failures in main branch during refactoring

**Target Value**: <15 total regressions, 0 critical
**Success Threshold**: <30 total acceptable, <15 excellent
**Collection Frequency**: Weekly during refactoring, monthly for 3 months post-epic

---

### SH3: Performance Stability

**What**: Verify no performance degradation from refactoring.

**Baseline** (measured 2026-02-16):
- `shark task next`: ~50ms average
- `shark feature get E07-F01`: ~35ms average
- API `/tasks/next`: ~45ms average

**Target**:
- All operations within ±10% of baseline
- No operation slower by >20%
- 90th percentile response times unchanged

**Measurement Method**:
```bash
# Benchmark before/after
for i in {1..100}; do
  time shark task next --agent=backend >/dev/null 2>&1
done | awk '{sum+=$1; count++} END {print sum/count}'

# Compare: baseline vs. post-refactoring
```

**Target Value**: All operations within ±10% of baseline
**Success Threshold**: Within ±20% acceptable, ±10% excellent
**Collection Frequency**: Weekly during refactoring, monthly for 3 months post-epic

---

## Leading Indicators (Early Warning Metrics)

### LI1: Service Interface Stability

**What**: Track changes to service interfaces after initial definition.

**Target**: <3 interface changes per service after initial implementation
**Why**: Frequent interface changes indicate poor initial design
**Collection**: Per PR

### LI2: Mock Usage in Tests

**What**: Percentage of service tests using mocks vs. real database.

**Target**: >95% of service tests use mocks
**Why**: Service tests should be unit tests, not integration tests
**Collection**: Per PR (code review)

### LI3: CLI Command File Size Trend

**What**: Track largest command file size over time during refactoring.

**Target**: Monotonically decreasing (no files grow larger)
**Why**: Growing file size indicates refactoring is incomplete
**Collection**: Per PR (automated check)

---

## Success Dashboard

At epic completion, success is validated by this dashboard:

| Category | Metric | Target | Actual | Status |
|----------|--------|--------|--------|--------|
| **Architecture** | Code reduction in commands | 65% | TBD | 🔲 |
| **Architecture** | Service layer test coverage | >80% | TBD | 🔲 |
| **Architecture** | Duplication elimination | 80% | TBD | 🔲 |
| **Architecture** | Business logic in services | 100% | TBD | 🔲 |
| **Developer** | Time to add new command | 75% faster | TBD | 🔲 |
| **Developer** | PR review cycle time | 70% faster | TBD | 🔲 |
| **Developer** | Test execution speed | <100ms | TBD | 🔲 |
| **System** | API feature parity | 100% | TBD | 🔲 |
| **System** | Regression rate | <15 bugs | TBD | 🔲 |
| **System** | Performance stability | ±10% | TBD | 🔲 |

**Overall Success Criteria**: 8/10 metrics meet "excellent" threshold, 10/10 meet "acceptable" threshold

---

## Measurement Plan

### During Refactoring (Weekly)
- Track CLI command file sizes (automated)
- Track test coverage (CI gating)
- Track regression issues (GitHub)
- Track performance benchmarks (manual)

### At Epic Completion
- Full code analysis for duplication
- Developer survey for DX metrics
- API feature parity audit
- Documentation completeness review

### Post-Epic (Monthly for 3 Months)
- Track time to add new commands (via PR metadata)
- Track PR review cycles (via GitHub API)
- Track new contributor onboarding (via surveys)
- Track production regressions (via issue tracker)

---

*See also*: [Requirements](./requirements.md), [User Journeys](./user-journeys.md)
