# Flaky Tests Summary - Quick Reference

## Test Failure Matrix

| Test Name | File | Uses Shared Data | Root Cause | Fix Complexity |
|-----------|------|------------------|------------|----------------|
| **TestBulkCreate** | `bulk_operations_test.go:119` | E04-F05 | SeedTestData race | Medium |
| **TestBulkCreateValidationFailure** | `bulk_operations_test.go:194` | E04-F05 | SeedTestData race | Medium |
| **TestBulkCreatePerformance** | `bulk_operations_test.go:488` | E04-F05 | DELETE → Seed race | Medium |
| **TestGetByKeys** | `bulk_operations_test.go:233` | E99-F99 | SeedTestData race | Medium |
| **TestGetByKeysPartial** | `bulk_operations_test.go:255` | E99-F99 | DELETE → Seed race | Medium |
| **TestGetByKeysPerformance** | `bulk_operations_test.go:561` | E04-F05 | DELETE → Seed race | Medium |
| **TestUpdateMetadata** | `bulk_operations_test.go:274` | E99-F99 | DELETE → Seed race | Medium |
| **TestUpdateCompletionMetadata** | `task_completion_metadata_test.go:12` | E99-F99 | SeedTestData race | Medium |
| **TestLinkToTask** | `document_repository_test.go` | E99 | SeedTestData race | Medium |
| **TestLinkToFeature** | `document_repository_test.go` | E99 | SeedTestData race | Medium |
| **TestListForEpic** | `document_repository_test.go` | E99 | SeedTestData race | Medium |
| **TestEpicDetailsIntegration** | `epic_feature_integration_test.go:118` | E50-E99 (generated) | Key collision | High |
| **TestFeatureDetailsIntegration** | `epic_feature_integration_test.go:254` | E50-E99 (generated) | Key collision | High |
| **TestProgressCalculationEdgeCases** | `epic_feature_integration_test.go:382` | E50-E99 (generated) | Key collision | High |
| **TestEpicProgress_WeightedAverage** | `progress_calc_test.go:273` | E97 (hardcoded) | Key collision | High |
| **TestGetDashboard_EmptyDatabase** | `status_test.go` | ALL | Global state | Low |
| **TestGetDashboard_WithData** | `status_test.go` | ALL | Global state | Low |
| **TestGetDashboard_MultipleEpicsOrdering** | `status_test.go` | ALL | Global state | Low |

**Total**: 18 flaky tests across 6 test files

---

## Failure Frequency (10 runs)

Each test failed 1 out of 10 parallel runs (~10% failure rate per test)

---

## Root Cause Categories

### 1. SeedTestData Race Conditions (11 tests)
**Pattern**: Test calls `SeedTestData()` expecting E04/E99 data to exist, but another test deleted it

**Example Error**:
```
FOREIGN KEY constraint failed
task not found: T-E99-F99-001
feature not found: E04-F05
```

**Files Affected**:
- `bulk_operations_test.go` (7 tests)
- `task_completion_metadata_test.go` (1 test)
- `document_repository_test.go` (3 tests)

**Fix Strategy**: Each test should create its own isolated epic/feature/task data with unique keys

---

### 2. Epic Key Collisions (4 tests)
**Pattern**: Multiple tests generate the same epic key using timestamp % 50

**Example Error**:
```
Expected 70.0% progress, got 0.0%
Epic already exists
```

**Epic Key Ranges Used**:
- E50-E99: 50 possible values (`generateTestEpicKey()`)
- E90-E99: 10 possible values (`setupProgressTest()`)
- E97: Hardcoded (always collides)

**Files Affected**:
- `epic_feature_integration_test.go` (3 tests)
- `progress_calc_test.go` (1 test)

**Fix Strategy**: Need truly unique keys or per-test database isolation

---

### 3. Global Database State (3 tests)
**Pattern**: Dashboard queries see ALL epics in database, including from other tests

**Example Error**:
```
Expected empty database, got 5 epics
Expected 3 epics, got 7 epics
```

**Files Affected**:
- `status_test.go` (3 tests)

**Fix Strategy**:
- Option A: Clean entire DB before test (breaks parallelism)
- Option B: Filter results by test-specific prefix
- Option C: Use in-memory DB per test

---

## Quick Stats

```
Total Test Files:           ~40 files
Total Tests:                ~250 tests
Flaky Tests:                18 tests (7%)
Sequential Success Rate:    100%
Parallel Success Rate:      ~85-95% (varies)

File Breakdown:
  bulk_operations_test.go:         7 flaky tests
  epic_feature_integration_test.go: 3 flaky tests
  document_repository_test.go:      3 flaky tests
  status_test.go:                   3 flaky tests
  progress_calc_test.go:            1 flaky test
  task_completion_metadata_test.go: 1 flaky test
```

---

## Recommended Fix Priority

### High Priority (Most Impact)
1. **bulk_operations_test.go** - 7 flaky tests
   - Replace `SeedTestData()` with per-test data creation
   - Use unique epic keys (E10-E49 range)

2. **status_test.go** - 3 flaky tests
   - Mock the repository layer instead of using real database
   - OR filter queries by test-specific epic keys

### Medium Priority
3. **epic_feature_integration_test.go** - 3 flaky tests
   - Fix `generateTestEpicKey()` to use nanosecond + PID for uniqueness
   - OR use per-test database

4. **document_repository_test.go** - 3 flaky tests
   - Replace `SeedTestData()` with per-test data

### Low Priority (Single Tests)
5. **progress_calc_test.go** - 1 test
   - Change E97 hardcoded key to use dynamic generation

6. **task_completion_metadata_test.go** - 1 test
   - Replace `SeedTestData()` with per-test data

---

## Implementation Estimate

**Option 1**: Sequential CI/CD (`go test -p=1`)
- Time: 0 minutes (config change)
- Solves: All failures
- Trade-off: Slower CI (2-3x longer)

**Option 2**: Eliminate SeedTestData
- Time: ~4-6 hours
- Affected: 11 tests
- Benefits: Proper test isolation

**Option 3**: Fix Epic Key Generation
- Time: ~2-3 hours
- Affected: 4 tests
- Benefits: Reduces collisions significantly

**Option 4**: Mock Status Tests
- Time: ~1-2 hours
- Affected: 3 tests
- Benefits: Fast, follows testing best practices

**Option 5**: Complete Fix (All Above)
- Time: ~8-12 hours
- Affected: All 18 tests
- Benefits: Robust, parallel-safe test suite
