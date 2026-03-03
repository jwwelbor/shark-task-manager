# E18-F07: Dashboard and Analytics Integration -- Test Plan

**Feature Key**: E18-F07
**Complexity Tier**: STANDARD
**Date**: 2026-03-03
**Author**: QA Agent

---

## 1. Acceptance Criteria Test Matrix

Each acceptance criterion from the PRD is mapped to concrete test cases with inputs, expected outputs, and edge cases. Test IDs use the format `TC-F07-NNN`.

---

### FR-F07-001: Bug Status Section in Dashboard

#### TC-F07-001: Dashboard shows bug counts by status when bugs exist

**Story**: US-F07-001 (Dashboard Bug Summary)
**Priority**: High (UAT P2-5, J3-DASHBOARD)

**Preconditions**:
- Database contains bugs in multiple statuses
- Bug repository is wired into StatusService

**Test Steps**:
1. Seed database with 3 bugs: 1 `reported`, 1 `in_fix`, 1 `resolved`
2. Run `shark status`

**Expected Result**:
- Output contains a "BUGS" section
- Section shows "Total: 3"
- Status breakdown shows "reported: 1, in_fix: 1, resolved: 1"
- Statuses with zero count are omitted from display

**Test Data**:
- Bug 1: title="Login crash", severity=high, status=reported
- Bug 2: title="CSS misalignment", severity=medium, status=in_fix
- Bug 3: title="Typo in help text", severity=low, status=resolved

---

#### TC-F07-002: Dashboard omits bug section when no bugs exist

**Story**: US-F07-008 (Conditional Section Display)
**Priority**: High

**Preconditions**:
- Database contains zero bugs
- Bug repository returns zero total

**Test Steps**:
1. Ensure no bugs exist in database
2. Run `shark status`

**Expected Result**:
- No "BUGS" section appears in output
- Existing dashboard sections (epics, features, tasks) render normally

---

#### TC-F07-003: Dashboard bug section in JSON output

**Story**: US-F07-001
**Priority**: Medium

**Preconditions**:
- Database contains bugs

**Test Steps**:
1. Seed 2 bugs (1 reported, 1 triaged)
2. Run `shark status --json`

**Expected Result**:
- JSON output contains `"bugs"` key
- `bugs.total` equals 2
- `bugs.by_status` contains `{"reported": 1, "triaged": 1}`
- `bugs.open_by_severity` contains severity distribution

---

#### TC-F07-004: JSON output omits bugs key when no bugs exist

**Story**: US-F07-008
**Priority**: Medium

**Preconditions**:
- Zero bugs in database

**Test Steps**:
1. Run `shark status --json`

**Expected Result**:
- JSON output does NOT contain `"bugs"` key (omitempty behavior)
- Existing JSON structure unchanged

---

### FR-F07-002: Bug Severity Breakdown in Dashboard

#### TC-F07-005: Severity breakdown shows open bugs only

**Story**: US-F07-002 (Dashboard Bug Severity Breakdown)
**Priority**: High (UAT Metric 5)

**Preconditions**:
- Mix of open and terminal-status bugs with various severities

**Test Steps**:
1. Seed 5 bugs:
   - Bug A: severity=critical, status=reported (open)
   - Bug B: severity=high, status=in_fix (open)
   - Bug C: severity=high, status=triaged (open)
   - Bug D: severity=medium, status=resolved (terminal)
   - Bug E: severity=low, status=wont_fix (terminal)
2. Run `shark status`

**Expected Result**:
- Severity breakdown shows: "critical: 1, high: 2, medium: 0, low: 0"
- Terminal bugs (D, E) are excluded from severity breakdown
- Total bug count still shows 5 (includes terminal)

---

#### TC-F07-006: Severity breakdown omitted when all bugs are terminal

**Story**: US-F07-002
**Priority**: Medium

**Preconditions**:
- All bugs in terminal status (resolved, wont_fix, duplicate)

**Test Steps**:
1. Seed 3 bugs all in resolved/wont_fix/duplicate status
2. Run `shark status`

**Expected Result**:
- Bug section appears (total > 0)
- Open Bug Severity subsection is omitted or shows all zeros

---

### FR-F07-003: Change-Card Status Section in Dashboard

#### TC-F07-007: Dashboard shows change-card counts by status

**Story**: US-F07-003 (Dashboard Change-Card Summary)
**Priority**: High (UAT P2-5, J3-DASHBOARD)

**Preconditions**:
- Database contains change-cards in multiple statuses

**Test Steps**:
1. Seed 5 change-cards: 2 proposed, 1 approved, 1 in_progress, 1 completed
2. Run `shark status`

**Expected Result**:
- Output contains "CHANGE CARDS" section
- Section shows "Total: 5"
- Status breakdown shows counts for each status

---

#### TC-F07-008: Dashboard omits change-card section when none exist

**Story**: US-F07-008
**Priority**: High

**Preconditions**:
- Zero change-cards in database

**Test Steps**:
1. Run `shark status`

**Expected Result**:
- No "CHANGE CARDS" section in output
- JSON output does NOT contain `"change_cards"` key

---

### FR-F07-004: Feature-Level Bug Context

#### TC-F07-009: Feature status shows linked bug summary

**Story**: US-F07-007 (Feature-Level Bug Context)
**Priority**: Medium (UAT J3-DASH)

**Preconditions**:
- Feature E07-F01 exists with tasks
- 3 bugs linked to E07-F01: 2 open (1 high, 1 medium), 1 resolved

**Test Steps**:
1. Run `shark status E07-F01`

**Expected Result**:
- Output includes "Linked Bugs: 3 (2 open -- 1 high, 1 medium)"
- Resolved bugs counted in total but excluded from severity

---

#### TC-F07-010: Feature status omits bug section when no bugs linked

**Story**: US-F07-007
**Priority**: Medium

**Preconditions**:
- Feature exists with no linked bugs

**Test Steps**:
1. Run `shark status E07-F01`

**Expected Result**:
- No bug information appears in feature status output
- JSON output does NOT contain `"linked_bugs"` key

---

### FR-F07-005 / FR-F07-006: Bug Analytics

#### TC-F07-011: Bug analytics shows all metrics

**Story**: US-F07-004 (Analytics Bug Metrics)
**Priority**: High (UAT J3-ANALYTICS, P3-1)

**Preconditions**:
- 10 bugs with mixed statuses and severities
- At least 3 bugs in terminal status with known creation times

**Test Steps**:
1. Seed 10 bugs (3 reported, 2 triaged, 1 in_fix, 1 in_verification, 2 resolved, 1 wont_fix)
2. Run `shark analytics --type=bug`

**Expected Result**:
- Output shows "Total Bugs: 10"
- Status breakdown lists all statuses with counts
- Severity distribution shows critical/high/medium/low counts
- "Avg Resolution Time" shows a computed duration (not N/A, since resolved bugs exist)
- "Resolved: 3" (resolved + wont_fix count)

---

#### TC-F07-012: Bug analytics with zero resolved bugs

**Story**: US-F07-004
**Priority**: Medium

**Preconditions**:
- All bugs in non-terminal status (no resolved, wont_fix, duplicate)

**Test Steps**:
1. Seed 3 bugs all in reported/triaged status
2. Run `shark analytics --type=bug`

**Expected Result**:
- "Avg Resolution Time: N/A"
- "Resolved: 0"

---

#### TC-F07-013: Bug analytics JSON output

**Story**: US-F07-009 (JSON Output for Analytics)
**Priority**: High (UAT CI/CD persona)

**Preconditions**:
- Bugs exist in database

**Test Steps**:
1. Run `shark analytics --type=bug --json`

**Expected Result**:
- Valid JSON output
- Contains keys: `total_bugs`, `bugs_by_status`, `bugs_by_severity`, `resolved_count`, `avg_resolution_time_seconds`
- `avg_resolution_time_seconds` is null/nil when no resolved bugs, numeric otherwise

---

#### TC-F07-014: Bug analytics type filter excludes other data

**Story**: US-F07-006 (Analytics Type Filtering)
**Priority**: Medium

**Preconditions**:
- Both bugs and change-cards exist

**Test Steps**:
1. Run `shark analytics --type=bug`

**Expected Result**:
- Output shows ONLY bug analytics
- No change-card data, no session analytics

---

### FR-F07-007: Bug Analytics -- Average Resolution Time

#### TC-F07-015: Resolution time calculation accuracy

**Story**: US-F07-004
**Priority**: High (UAT Metric 2)

**Preconditions**:
- 3 resolved bugs with known creation and resolution timestamps

**Test Steps**:
1. Create 3 bugs with controlled timestamps:
   - Bug X: created at T, resolved at T+2h
   - Bug Y: created at T, resolved at T+4h
   - Bug Z: created at T, resolved at T+6h
2. Run `shark analytics --type=bug`

**Expected Result**:
- "Avg Resolution Time: 4h 0m" (average of 2h, 4h, 6h)

**Test Data**:
- Mock repository returns `avg_resolution_seconds: 14400.0` (4 hours)

---

### FR-F07-008 / FR-F07-009: Change-Card Analytics

#### TC-F07-016: Change-card analytics shows all metrics

**Story**: US-F07-005 (Analytics Change-Card Metrics)
**Priority**: High (UAT J3-ANALYTICS, P3-1)

**Preconditions**:
- 8 change-cards with mixed statuses

**Test Steps**:
1. Seed 8 change-cards: 2 proposed, 1 approved, 2 in_progress, 2 completed, 1 declined
2. Run `shark analytics --type=change`

**Expected Result**:
- "Total Change Cards: 8"
- Status breakdown with counts for each status
- "Approval Rate: 83.3%" (5 approved-or-further / 6 decided)
- "Avg Completion Time" shows a computed duration
- "Decided: 6" (approved+declined+completed+in_progress)
- "Completed: 2"

---

#### TC-F07-017: Change-card analytics with zero completed cards

**Story**: US-F07-005
**Priority**: Medium

**Preconditions**:
- Change-cards exist but none are completed

**Test Steps**:
1. Seed 2 change-cards: 1 proposed, 1 approved
2. Run `shark analytics --type=change`

**Expected Result**:
- "Avg Completion Time: N/A"
- "Completed: 0"

---

#### TC-F07-018: Change-card analytics with zero decided cards

**Story**: US-F07-005
**Priority**: Medium

**Preconditions**:
- Only proposed change-cards (none approved or declined)

**Test Steps**:
1. Seed 2 proposed change-cards
2. Run `shark analytics --type=change`

**Expected Result**:
- "Approval Rate: N/A"
- "Decided: 0"

---

#### TC-F07-019: Change-card analytics JSON output

**Story**: US-F07-009
**Priority**: High

**Preconditions**:
- Change-cards exist

**Test Steps**:
1. Run `shark analytics --type=change --json`

**Expected Result**:
- Valid JSON output
- Contains keys: `total_change_cards`, `change_cards_by_status`, `approval_rate`, `decided_count`, `completed_count`, `avg_completion_time_seconds`
- `approval_rate` is null when no decided cards, float otherwise
- `avg_completion_time_seconds` is null when no completed cards

---

### FR-F07-010: Analytics JSON Output (Combined)

#### TC-F07-020: Combined analytics includes both bug and change-card data

**Story**: US-F07-004, US-F07-005
**Priority**: Medium

**Preconditions**:
- Both bugs and change-cards exist

**Test Steps**:
1. Run `shark analytics --json`

**Expected Result**:
- JSON contains `"bugs"` section with bug metrics
- JSON contains `"change_cards"` section with change-card metrics
- Both sections present in single output

---

#### TC-F07-021: Combined analytics omits sections when entities absent

**Story**: US-F07-008
**Priority**: Medium

**Preconditions**:
- Bugs exist but no change-cards

**Test Steps**:
1. Run `shark analytics --json`

**Expected Result**:
- `"bugs"` section present
- `"change_cards"` key absent (omitempty)

---

## 2. API Contract Test Cases

This feature extends existing commands rather than adding new APIs, but the JSON output contracts serve as API contracts for AI agent consumers.

### Dashboard JSON Contract Tests

#### TC-F07-030: Dashboard JSON structure with all entity types

**Validates**: JSON contract from architecture doc section 8.1

**Test Steps**:
1. Populate database with epics, features, tasks, bugs, and change-cards
2. Run `shark status --json`
3. Parse output as JSON

**Expected Result**:
- Top-level keys include: `summary`, `epics`, `active_tasks`, `blocked_tasks`
- `bugs` key present with structure: `{total, by_status, open_by_severity}`
- `change_cards` key present with structure: `{total, by_status}`
- `by_status` maps contain only string keys and integer values
- `open_by_severity` map contains severity level keys

**Implementation**: Unit test on `StatusDashboard` JSON marshaling (extends existing `TestStatusDashboardJSONMarshaling`)

---

#### TC-F07-031: Bug analytics JSON contract validation

**Validates**: JSON contract from architecture doc section 8.2

**Test Steps**:
1. Mock BugSummaryRepository to return known data
2. Call `DashboardAnalyticsService.GetBugAnalytics()`
3. Marshal result to JSON

**Expected Result**:
- JSON keys exactly match contract: `total_bugs`, `bugs_by_status`, `bugs_by_severity`, `resolved_count`, `avg_resolution_time_seconds`
- `avg_resolution_time_seconds` serializes as float when non-nil, absent/null when nil
- All map values are integers

---

#### TC-F07-032: Change-card analytics JSON contract validation

**Validates**: JSON contract from architecture doc section 8.3

**Test Steps**:
1. Mock ChangeCardSummaryRepository to return known data
2. Call `DashboardAnalyticsService.GetChangeCardAnalytics()`
3. Marshal result to JSON

**Expected Result**:
- JSON keys exactly match contract: `total_change_cards`, `change_cards_by_status`, `approval_rate`, `decided_count`, `completed_count`, `avg_completion_time_seconds`
- `approval_rate` serializes as float (0.0-1.0 range)

---

#### TC-F07-033: Feature status JSON with linked bugs

**Validates**: JSON contract from architecture doc section 8.4

**Test Steps**:
1. Mock BugDashboardRepository.GetFeatureBugSummary() to return data
2. Build FeatureStatusInfo with LinkedBugs populated
3. Marshal to JSON

**Expected Result**:
- `linked_bugs` key present with structure: `{total_linked, open_count, open_by_severity}`
- When LinkedBugs is nil, `linked_bugs` key absent from JSON

---

## 3. Component Test Strategy

### 3.1 DashboardAnalyticsService (Service Layer -- Mocked Repos)

**File**: `internal/services/dashboard_analytics_service_test.go`
**Pattern**: Function-field mocks per codebase testing architecture

| Test Case | Mock Setup | Assertion |
|-----------|-----------|-----------|
| GetBugAnalytics happy path | Return BugStatusSummary(total=10, by_status={...}), BugResolutionStats(resolved=3, avg=14400) | Result contains all fields, no error |
| GetBugAnalytics nil repo | bugRepo=nil | Returns error "bug analytics not available" |
| GetBugAnalytics repo error on status | GetStatusSummary returns error | Error propagated with wrapping |
| GetBugAnalytics repo error on resolution | GetResolutionStats returns error | Error propagated with wrapping |
| GetBugAnalytics zero bugs | Return BugStatusSummary(total=0, by_status={}) | Result has total=0, empty maps |
| GetBugAnalytics no resolved bugs | Return BugResolutionStats(resolved=0, avg=nil) | AvgResolutionTimeSecs is nil |
| GetChangeCardAnalytics happy path | Return ChangeCardStatusSummary, ChangeCardThroughputStats | Result contains all fields |
| GetChangeCardAnalytics nil repo | changeCardRepo=nil | Returns error |
| GetChangeCardAnalytics zero decided | Return ThroughputStats(decided=0, approval=nil) | ApprovalRate is nil |
| GetChangeCardAnalytics zero completed | Return ThroughputStats(completed=0, avg=nil) | AvgCompletionTimeSecs is nil |

### 3.2 StatusService Dashboard Extension (Mocked Repos)

**File**: `internal/status/status_test.go` (extend existing)

| Test Case | Mock Setup | Assertion |
|-----------|-----------|-----------|
| Dashboard with nil bug repo (backward compat) | bugRepo=nil, changeCardRepo=nil | Dashboard renders without bug/cc sections, no error |
| Dashboard with bug repo returning zero | bugRepo returns total=0 | BugSummary is nil in result |
| Dashboard with bug repo returning data | bugRepo returns total=5 | BugSummary populated in result |
| Dashboard with change-card repo returning data | ccRepo returns total=3 | ChangeCardSummary populated |
| Dashboard with bug repo error (graceful degrade) | bugRepo returns error | Dashboard still renders, BugSummary is nil, no error returned |
| Feature status with linked bugs | bugRepo.GetFeatureBugSummary returns data | LinkedBugs populated |
| Feature status with no linked bugs | bugRepo.GetFeatureBugSummary returns total=0 | LinkedBugs is nil |

### 3.3 Formatter Functions (Pure Unit Tests)

**File**: `internal/status/formatter_test.go` (extend existing)

| Test Case | Input | Assertion |
|-----------|-------|-----------|
| formatBugSummary nil input | nil | Returns empty string |
| formatBugSummary with data | BugDashboardSummary{Total:5, ...} | Contains "BUGS", "Total: 5", status lines |
| formatBugSummary status ordering | Multiple statuses | Statuses appear in defined order (reported, triaged, in_fix, ...) |
| formatBugSummary severity section | OpenBySeverity with counts | Contains "Open Bug Severity:" header, severity lines |
| formatBugSummary no open severity | OpenBySeverity all zeros | Severity subsection omitted |
| formatChangeCardSummary nil input | nil | Returns empty string |
| formatChangeCardSummary with data | ChangeCardDashboardSummary{Total:3, ...} | Contains "CHANGE CARDS", "Total: 3" |
| formatDurationFromSecs hours | 14400.0 | Returns "4h 0m" |
| formatDurationFromSecs days | 259200.0 | Returns "3d 0h" |
| formatDurationFromSecs minutes | 3660.0 | Returns "1h 1m" |
| formatDurationFromSecs zero | 0.0 | Returns "0h 0m" |

---

## 4. Integration Scenarios

### 4.1 Cross-Feature: F02 (Bug Entity) Repository Contract

**Risk**: F07 consumes BugRepository aggregate methods defined by F02. If F02 changes the DTO structure or method signature, F07 breaks.

**Test**: Compile-time interface satisfaction check. The `BugSummaryRepository` interface in F07 must be satisfied by F02's concrete `BugRepository`. This is verified by Go's type system at compile time:

```go
// In F07's test file or a build tag file:
var _ BugSummaryRepository = (*repository.BugRepository)(nil)
```

**Pass Criteria**: Project compiles without errors. If F02 changes the method signature, the build fails immediately.

### 4.2 Cross-Feature: F03 (Change-Card Entity) Repository Contract

Same pattern as 4.1 but for `ChangeCardSummaryRepository` and F03's `ChangeCardRepository`.

### 4.3 Cross-Feature: F06 (Unified CLI) Dashboard Rendering

**Risk**: F06 introduces B### and C### key auto-detection in unified commands. The dashboard must render bug/change-card sections without conflicting with F06's entity routing.

**Test**: After F06 and F07 are both implemented, verify:
1. `shark status` renders all 5 entity sections (epics, features, tasks, bugs, change-cards)
2. `shark get B001` still works (F06 dispatch does not interfere with dashboard)
3. `shark status E07-F01` shows linked bug context (F07) and task breakdown (existing)

**Pass Criteria**: Both features coexist without interference. UAT scenario J3-DASHBOARD covers this.

### 4.4 StatusService Option Pattern Backward Compatibility

**Risk**: Adding `WithBugRepository()` / `WithChangeCardRepository()` options to `NewStatusService()` could break existing callers.

**Test**: Verify that `NewStatusService(db)` with zero options still works identically to the pre-F07 behavior.

```go
func TestStatusService_BackwardCompatibility(t *testing.T) {
    // No options -- should work as before
    svc := NewStatusService(mockDB)
    dashboard, err := svc.GetDashboard(ctx, &StatusRequest{})
    assert.NoError(t, err)
    assert.Nil(t, dashboard.BugSummary)
    assert.Nil(t, dashboard.ChangeCardSummary)
}
```

**Pass Criteria**: All existing status tests pass without modification after F07 changes.

---

## 5. UAT Scenario Traceability

This section maps epic-level UAT scenarios to feature-level test cases, ensuring complete coverage.

| UAT Scenario | Feature Test Cases | Coverage |
|-------------|-------------------|----------|
| J3-DASHBOARD Step 1 (dashboard shows all entity types) | TC-F07-001, TC-F07-007, TC-F07-030 | Full |
| J3-DASHBOARD Step 2 (conditional display) | TC-F07-002, TC-F07-004, TC-F07-008 | Full |
| J3-DASH (feature status with bug context) | TC-F07-009, TC-F07-010, TC-F07-033 | Full |
| J3-ANALYTICS Step 1 (combined analytics) | TC-F07-020, TC-F07-021 | Full |
| J3-ANALYTICS Step 2 (--type=bug filter) | TC-F07-011, TC-F07-013, TC-F07-014 | Full |
| J3-ANALYTICS Step 3 (--type=change filter) | TC-F07-016, TC-F07-019 | Full |
| Metric 2 (avg resolution time) | TC-F07-015 | Full |
| Metric 3 (change-card throughput) | TC-F07-016, TC-F07-017 | Full |
| Metric 5 (dashboard visibility score) | TC-F07-001, TC-F07-005, TC-F07-007 | Full |

---

## 6. Edge Cases and Boundary Conditions

| Edge Case | Test Case | Expected Behavior |
|-----------|----------|-------------------|
| Zero bugs, zero change-cards | TC-F07-002, TC-F07-008 | Both sections omitted from dashboard |
| Single bug only | Variant of TC-F07-001 | Bug section shows, "Total: 1" |
| All bugs in one status | Variant of TC-F07-001 | Only that status line shown |
| 1000 bugs (performance boundary) | Performance test | Aggregate queries complete in under 100ms |
| Approval rate = 0% (all declined) | Variant of TC-F07-016 | "Approval Rate: 0.0%" |
| Approval rate = 100% (none declined) | Variant of TC-F07-016 | "Approval Rate: 100.0%" |
| Bug repo returns error | StatusService component test | Dashboard degrades gracefully, no crash |
| Change-card repo returns error | StatusService component test | Dashboard degrades gracefully |
| Resolution time includes wont_fix and duplicate | TC-F07-015 variant | wont_fix and duplicate bugs counted in resolution stats |
| Feature with zero tasks but linked bugs | TC-F07-009 variant | Bug context shown even with no tasks |
| formatDurationFromSecs with very large value | Formatter test | Renders as "Xd Yh" format |
| formatDurationFromSecs with fractional seconds | Formatter test | Rounds to nearest minute |

---

## 7. Test Implementation Notes

### Testing Patterns to Follow

1. **Service tests**: Use function-field mocks (matches existing `MockTaskRepository` pattern in codebase). Define `MockBugSummaryRepository` and `MockChangeCardSummaryRepository` with `GetStatusSummaryFunc`, `GetResolutionStatsFunc`, etc.

2. **Formatter tests**: Pure unit tests with no mocks. Pass structs directly to formatter functions and assert on string output using `strings.Contains()` (matches existing `TestRenderProgressBar` pattern).

3. **JSON contract tests**: Use `json.Marshal` + `json.Unmarshal` into `map[string]interface{}` to verify key presence and types (matches existing `TestStatusDashboardJSONMarshaling` pattern).

4. **Table-driven tests**: Use for formatter edge cases and analytics scenarios with multiple input variations (matches existing test patterns).

### Test File Locations

| Test Area | File | New/Extend |
|-----------|------|-----------|
| DashboardAnalyticsService | `internal/services/dashboard_analytics_service_test.go` | New |
| StatusService dashboard extension | `internal/status/status_test.go` | Extend |
| Formatter functions | `internal/status/formatter_test.go` | Extend |
| Analytics CLI command | `internal/cli/commands/analytics_test.go` | Extend |
| JSON contract validation | `internal/services/dashboard_analytics_service_test.go` | New (within service tests) |

### Mock Definitions Needed

```go
// MockBugSummaryRepository for service tests
type MockBugSummaryRepository struct {
    GetStatusSummaryFunc      func(ctx context.Context) (*BugStatusSummary, error)
    GetResolutionStatsFunc    func(ctx context.Context) (*BugResolutionStats, error)
    GetFeatureBugSummaryFunc  func(ctx context.Context, featureKey string) (*BugFeatureSummary, error)
}

// MockChangeCardSummaryRepository for service tests
type MockChangeCardSummaryRepository struct {
    GetStatusSummaryFunc     func(ctx context.Context) (*ChangeCardStatusSummary, error)
    GetThroughputStatsFunc   func(ctx context.Context) (*ChangeCardThroughputStats, error)
}
```

---

## Exit Gate Checklist

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Every acceptance criterion has at least one test case | PASS | 10 FRs mapped to 21+ test cases (TC-F07-001 through TC-F07-021) |
| API contracts tested | PASS | JSON contracts validated in TC-F07-030 through TC-F07-033 |
| Integration points identified | PASS | 4 cross-feature integration scenarios documented (F02, F03, F06, backward compat) |
| Edge cases documented | PASS | 12 edge cases with expected behaviors |
| Test implementation patterns reference existing codebase | PASS | References to existing formatter_test.go, status_test.go, and mock patterns |
| UAT scenarios traced to feature tests | PASS | Traceability matrix covers all J3-DASHBOARD, J3-ANALYTICS, J3-DASH scenarios |
| Actionable for TDD | PASS | Mock definitions, file locations, and test structures specified |

---

*Traces to*: [E18-F07 PRD](./prd.md) | [E18-F07 Architecture](./02-architecture.md) | [E18 UAT Acceptance Plan](../E18-UAT-ACCEPTANCE-PLAN.md)
*Last Updated*: 2026-03-03
