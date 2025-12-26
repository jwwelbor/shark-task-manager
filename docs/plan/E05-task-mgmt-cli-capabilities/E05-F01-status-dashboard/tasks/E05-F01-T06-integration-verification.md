# Task: E05-F01-T06 - Integration, Verification & Release Preparation

**Feature**: E05-F01 Status Dashboard & Reporting
**Epic**: E05 Task Management CLI Capabilities
**Task Key**: E05-F01-T06

## Description

Final integration and verification task that brings all components together, validates the complete feature end-to-end, performs final optimizations, and prepares the feature for code review and release.

This task:
- Builds the complete feature without errors
- Runs the full test suite successfully
- Performs manual testing with real project data
- Validates all acceptance criteria from PRD
- Optimizes any remaining performance issues
- Verifies no regressions in existing functionality
- Prepares documentation for release
- Performs final code quality checks

**Why This Matters**: Integration testing validates that all pieces work together correctly. Final verification ensures the feature meets all requirements and is ready for production use.

## What You'll Build

No new code, but comprehensive validation and integration:
- Build verification
- Full test suite execution
- Manual testing with realistic data
- Performance validation
- Documentation updates
- Code review preparation

## Success Criteria

**Build & Test**:
- [x] `make build` succeeds without errors
- [x] All tests pass: `make test`
- [x] No race conditions: `go test -race ./...`
- [x] Code coverage >80% for status package
- [x] No linting errors: `make lint`
- [x] Code formatted: `make fmt`

**Manual Testing**:
- [x] Empty database: `shark status` shows "No epics found"
- [x] Full project: All sections display correctly
- [x] `shark status --epic=E01` filters correctly
- [x] `shark status --json` outputs valid JSON
- [x] `shark status --no-color` has no ANSI codes
- [x] `shark status --recent=7d` filters by date range
- [x] Error cases show helpful messages
- [x] Performance target met: <500ms for 100 epics

**Acceptance Criteria**:
- [x] All 43 functional requirements from PRD verified
- [x] Epic progress bars render correctly (20 chars, color-coded)
- [x] Active tasks grouped by agent type
- [x] Blocked tasks show reasons
- [x] Recent completions show relative time
- [x] JSON schema matches documented format
- [x] Color coding follows spec (green/yellow/red)
- [x] Terminal width handling works for narrow/wide displays
- [x] No crashes on edge cases

**Code Quality**:
- [x] No linting errors
- [x] Code formatted with gofmt
- [x] Error handling proper
- [x] Comments on public functions
- [x] Consistent naming
- [x] No dead code

**Documentation**:
- [x] README.md updated with status command
- [x] JSON schema documented
- [x] Examples provided
- [x] Performance characteristics documented
- [x] Code review ready

## Implementation Notes

### Build Verification Checklist

```bash
# Clean build
make clean
make build

# Verify binary created
ls -lh ./bin/shark
file ./bin/shark

# Quick sanity check
./bin/shark help | grep status
./bin/shark status --help
```

### Test Suite Execution

```bash
# Run all tests with verbose output
make test
# Expected: all tests PASS

# Check specific package
go test ./internal/status -v
go test ./internal/cli/commands -run Status -v

# Run with coverage
go test ./internal/status -cover
# Expected: coverage >80%

# Race detection
go test -race ./internal/status ./internal/cli/commands
# Expected: no race conditions
```

### Manual Testing Scenarios

#### Scenario 1: Empty Project
```bash
# Setup
rm shark-tasks.db* 2>/dev/null
./bin/shark init --non-interactive

# Test
./bin/shark status
# Expected: "No epics found. Create epics to get started."
```

#### Scenario 2: Small Project with Sample Data
```bash
# Setup
./bin/shark epic create "Identity Platform" --priority=high
./bin/shark epic create "Task Management" --priority=medium
./bin/shark feature create --epic=E01 "Authentication"
./bin/shark feature create --epic=E01 "Session Management"
./bin/shark feature create --epic=E02 "Task CRUD"

# Create 20 tasks with various statuses
for i in {1..5}; do
  ./bin/shark task create --epic=E01 --feature=F01 "Auth task $i"
  ./bin/shark task create --epic=E01 --feature=F02 "Session task $i"
  ./bin/shark task create --epic=E02 --feature=F03 "CRUD task $i"
done

# Mark some as in_progress
./bin/shark task start T-E01-F01-001 --agent=backend
./bin/shark task start T-E01-F02-001 --agent=frontend

# Mark some as completed
./bin/shark task complete T-E01-F01-002
./bin/shark task complete T-E01-F01-003
./bin/shark task approve T-E01-F01-002
./bin/shark task approve T-E01-F01-003

# Block one
./bin/shark task block T-E01-F02-005 --reason="Waiting for API spec"

# Test
./bin/shark status
# Expected: Dashboard with 2 epics, progress bars, active tasks, blocked task
```

#### Scenario 3: JSON Output
```bash
# Test
./bin/shark status --json | jq '.summary'
# Expected: Valid JSON with counts

./bin/shark status --json | jq '.epics[0]'
# Expected: Epic object with progress, health, tasks

./bin/shark status --json | jq '.active_tasks | keys'
# Expected: Agent type keys ["backend", "frontend"]
```

#### Scenario 4: Filtering by Epic
```bash
# Test
./bin/shark status --epic=E01 | grep -A 20 "EPIC BREAKDOWN"
# Expected: Only E01 in table

./bin/shark status --epic=E01 --json | jq '.epics | length'
# Expected: 1 (only E01)

./bin/shark status --epic=E01 --json | jq '.summary.features.total'
# Expected: Only E01 features (2)
```

#### Scenario 5: No-Color Mode
```bash
# Test
./bin/shark status --no-color 2>&1 | od -c | grep '\033'
# Expected: No output (no ANSI codes)

./bin/shark status --no-color | head -20
# Expected: Plain text, readable
```

#### Scenario 6: Error Handling
```bash
# Invalid epic
./bin/shark status --epic=E999 2>&1
# Expected: "Epic not found: E999"

# Invalid timeframe
./bin/shark status --recent=badformat 2>&1
# Expected: "Invalid timeframe: badformat"
```

### Acceptance Criteria Verification Matrix

Create verification checklist against PRD requirements:

| Requirement | Verification Method | Status |
|-------------|-------------------|--------|
| Display project summary | Manual: `shark status` | ✓ |
| Show epic progress bars | Manual: Check formatting | ✓ |
| Identify blocked tasks | Manual: Block a task, check output | ✓ |
| See active work distribution | Manual: `shark status` active section | ✓ |
| View recent completions | Manual: Complete tasks, check output | ✓ |
| Filter by epic | Manual: `shark status --epic=E01` | ✓ |
| JSON export | Manual: `shark status --json` | ✓ |
| Color indicators | Manual: Verify green/yellow/red | ✓ |
| <500ms performance | Benchmark: `go test -bench=` | ✓ |
| No N+1 queries | Code review of queries | ✓ |

### Performance Validation

```bash
# Create large test project (100 epics, 2000 tasks)
# Using test fixtures or script

# Measure performance
time ./bin/shark status > /dev/null
# Expected: real <1s (includes startup + query + render)

# With JSON (faster without rendering)
time ./bin/shark status --json > /dev/null
# Expected: real <800ms

# Filtered (should be much faster)
time ./bin/shark status --epic=E01 > /dev/null
# Expected: real <300ms
```

### Final Code Quality Checks

```bash
# Format code
make fmt

# Lint
make lint
# Expected: no errors for status package

# Build
make build
# Expected: succeeds

# Test coverage
go test ./internal/status -coverprofile=coverage.out
go tool cover -html=coverage.out
# Expected: >80% coverage

# Race detection
go test -race ./internal/status
# Expected: no races
```

### Documentation Updates

Update project README.md:

```markdown
## Status Dashboard

Display comprehensive project status with a single command:

\`\`\`bash
shark status
\`\`\`

### Features

- **Project Summary**: Overview of all epics, features, and tasks
- **Epic Progress**: Visual progress bars with color-coded health
- **Active Work**: Tasks in progress grouped by agent type
- **Blockers**: Blocked tasks with blocking reasons
- **Recent Completions**: Recently finished work with timestamps

### Usage

\`\`\`bash
# Full dashboard
shark status

# Filter to specific epic
shark status --epic=E01

# JSON format for parsing
shark status --json

# Plain text (no colors)
shark status --no-color

# Custom recent timeframe
shark status --recent=7d

# Combine options
shark status --epic=E01 --json --recent=7d
\`\`\`

### JSON Schema

See [docs/plan/E05-task-mgmt-cli-capabilities/E05-F01-status-dashboard/prd.md](...)
for complete JSON schema documentation.
```

### Code Review Preparation

Create pre-review checklist:

- [ ] All tests pass locally
- [ ] Code formatted with gofmt
- [ ] No linting errors
- [ ] Performance targets met
- [ ] Documentation updated
- [ ] Commit message clear and descriptive
- [ ] Related issues/PRDs linked
- [ ] No breaking changes to existing APIs

Example commit message:

```
feat: implement status dashboard command (E05-F01)

- Implement shark status command for project visibility
- Display epic progress, active tasks, blockers, recent completions
- Support filtering, JSON export, color-coded output
- Achieve <500ms performance for 100 epics
- Add comprehensive test coverage (>80%)

Fixes: #E05-F01
Refs: docs/plan/E05-task-mgmt-cli-capabilities/E05-F01-status-dashboard/
```

## Test Execution Checklist

```bash
# 1. Build check
make clean
make build
echo "Build: PASSED"

# 2. Test check
make test
echo "Tests: PASSED"

# 3. Lint check
make lint
echo "Lint: PASSED"

# 4. Format check
make fmt
git diff --exit-code
echo "Format: PASSED"

# 5. Coverage check
go test ./internal/status -cover
# Verify >80%
echo "Coverage: PASSED"

# 6. Race check
go test -race ./internal/status ./internal/cli/commands
echo "Race: PASSED"

# 7. Manual smoke test
./bin/shark status
./bin/shark status --json | jq '.' | head -5
./bin/shark status --epic=E01 2>/dev/null || echo "Expected error for missing epic"
echo "Manual: PASSED"

# 8. Performance check
go test -bench=LargeProject -benchmem ./internal/status
# Verify <500ms
echo "Performance: PASSED"
```

## Dependencies

No new dependencies. All integration with existing code.

## Related Tasks

- **E05-F01-T01 through T05**: All components being integrated

## Acceptance Criteria

**Build & Test**:
- [ ] Build succeeds without errors
- [ ] All tests pass (unit, integration, benchmark)
- [ ] No race conditions
- [ ] Code coverage >80%
- [ ] No linting errors
- [ ] Code properly formatted

**Manual Testing**:
- [ ] Empty project handled gracefully
- [ ] Full project displays all sections
- [ ] All flags work correctly
- [ ] Error messages are helpful
- [ ] Terminal display is properly formatted
- [ ] Performance meets <500ms target

**Requirements Verification**:
- [ ] All 43 PRD functional requirements verified
- [ ] All acceptance criteria from PRD pass
- [ ] No regressions in existing functionality
- [ ] Edge cases handled

**Documentation**:
- [ ] README updated
- [ ] JSON schema documented
- [ ] Usage examples provided
- [ ] Performance characteristics documented

**Release Ready**:
- [ ] Code review checklist complete
- [ ] No open issues
- [ ] Documentation complete
- [ ] Ready for merge to main

## Verification Steps

```bash
# Execute full verification script
./scripts/verify-e05-f01.sh

# Or manual steps:
make clean && make build && make test && make lint
go test -race ./internal/status ./internal/cli/commands
go test ./internal/status -cover
./bin/shark status
./bin/shark status --json | jq '.'
```

## Timeline Estimate

- Build & test verification: 15 minutes
- Manual testing scenarios: 30 minutes
- Code quality final checks: 10 minutes
- Documentation updates: 15 minutes
- **Total: ~70 minutes**

## Next Steps

After this task completes:
1. Create pull request to main branch
2. Request code review
3. Address review feedback
4. Merge to main
5. Deploy/release

---

## Quick Reference Checklist

```
BUILD & TEST
□ make clean && make build succeeds
□ make test passes (all tests)
□ go test -race passes
□ Code coverage >80%
□ make lint passes
□ make fmt shows no changes

MANUAL TESTING
□ Empty project: "No epics found" message
□ Sample project: All sections display
□ --epic=E01: Filters correctly
□ --json: Valid JSON output
□ --no-color: No ANSI codes
□ Error case: Helpful error message
□ Performance: <500ms for large project

DOCUMENTATION
□ README updated with shark status
□ JSON schema documented
□ Usage examples provided
□ Performance notes documented

QUALITY
□ No dead code
□ All functions documented
□ Error handling proper
□ Consistent style

ACCEPTANCE
□ All PRD requirements verified
□ All test cases pass
□ Performance targets met
□ Ready for code review
```
