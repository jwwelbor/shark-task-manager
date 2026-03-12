# E07-F33 Test Plan: Unified Template Variables and Entity Coverage

**Date**: 2026-03-11
**Tier**: STANDARD
**Status**: Draft

---

## 1. Acceptance Criteria Test Matrix

### AC-1: TaskPlaceholders produces correct canonical variable set (REQ-F-001, T2)

| TC-ID | Test Case | Input | Expected Output | Priority |
|-------|-----------|-------|-----------------|----------|
| TC-TP-01 | Task with full key produces `key`, `task_key`, `epic_key`, `feature_key` | Task with Key=`T-E07-F01-001` | `key`=`T-E07-F01-001`, `task_key`=`T-E07-F01-001`, `epic_key`=`E07`, `feature_key`=`E07-F01` | High |
| TC-TP-02 | Task with short key format | Task with Key=`E07-F01-001` | `epic_key`=`E07`, `feature_key`=`E07-F01` | High |
| TC-TP-03 | `id` alias preserved for backward compat | Task with Key=`T-E07-F01-001` | `id`=`T-E07-F01-001` | High |
| TC-TP-04 | Removed: `task_id`, `epic_id`, `feature_id` no longer present | Task with Key=`T-E07-F01-001` | Map does NOT contain keys `task_id`, `epic_id`, `feature_id` | High |
| TC-TP-05 | `description` field populated when present | Task with Description=`"My desc"` | `description`=`My desc` | Medium |
| TC-TP-06 | `description` absent when nil | Task with Description=nil | Map does NOT contain `description` OR `description`=`""` | Medium |
| TC-TP-07 | All optional pointer fields populated | Task with Slug, FilePath, AgentType, ExecutionOrder, BlockedReason, DependsOn, CompletionNotes, FilesChanged all set | All corresponding keys present with correct values | Medium |
| TC-TP-08 | Nil task returns empty map | nil | `len(map) == 0` | High |
| TC-TP-09 | Common fields present: `title`, `status`, `priority`, `created_at`, `updated_at` | Task with Title=`"Test"`, Status=`"todo"`, Priority=5 | All keys present with correct string values | High |

### AC-2: FeaturePlaceholders produces correct canonical variable set (REQ-F-001, T3)

| TC-ID | Test Case | Input | Expected Output | Priority |
|-------|-----------|-------|-----------------|----------|
| TC-FP-01 | Feature produces `key` and `epic_key` | Feature with Key=`E07-F01` | `key`=`E07-F01`, `epic_key`=`E07` | High |
| TC-FP-02 | `id` alias preserved | Feature with Key=`E07-F01` | `id`=`E07-F01` | High |
| TC-FP-03 | Removed: `feature_id` no longer present | Feature with Key=`E07-F01` | Map does NOT contain `feature_id` | High |
| TC-FP-04 | Feature with slug suffix key | Feature with Key=`E07-F01-my-feature` | `epic_key`=`E07` (parsed from key) | Medium |
| TC-FP-05 | Nil feature returns empty map | nil | `len(map) == 0` | High |
| TC-FP-06 | `description` populated when present | Feature with Description=`"Design doc"` | `description`=`Design doc` | Medium |

### AC-3: EpicPlaceholders produces correct canonical variable set (REQ-F-001, T4)

| TC-ID | Test Case | Input | Expected Output | Priority |
|-------|-----------|-------|-----------------|----------|
| TC-EP-01 | Epic produces `key` | Epic with Key=`E07` | `key`=`E07` | High |
| TC-EP-02 | `id` alias preserved | Epic with Key=`E07` | `id`=`E07` | High |
| TC-EP-03 | Removed: `epic_id` no longer present | Epic with Key=`E07` | Map does NOT contain `epic_id` | High |
| TC-EP-04 | `business_value` present when set | Epic with BusinessValue=`"high"` | `business_value`=`high` | Medium |
| TC-EP-05 | Nil epic returns empty map | nil | `len(map) == 0` | High |

### AC-4: BugPlaceholders expanded with full field set (REQ-F-002, T5)

| TC-ID | Test Case | Input | Expected Output | Priority |
|-------|-----------|-------|-----------------|----------|
| TC-BP-01 | Bug produces all common fields | Bug with Key=`B001`, Title=`"Login crash"`, Status=`"open"` | `key`=`B001`, `id`=`B001`, `title`=`Login crash`, `status`=`open` | High |
| TC-BP-02 | Bug produces `severity` | Bug with Severity=`"critical"` | `severity`=`critical` | High |
| TC-BP-03 | Bug produces `linked_entity_type` and `linked_entity_key` | Bug with LinkedEntityType=`"task"`, LinkedEntityKey=`"E07-F01-001"` | `linked_entity_type`=`task`, `linked_entity_key`=`E07-F01-001` | High |
| TC-BP-04 | Bug produces `description` when present | Bug with Description=`"Crash on login"` | `description`=`Crash on login` | Medium |
| TC-BP-05 | Bug produces `slug`, `created_at`, `updated_at` | Bug with Slug=`"login-crash"`, timestamps set | All three keys present | Medium |
| TC-BP-06 | Nil bug returns empty map | nil | `len(map) == 0` | High |
| TC-BP-07 | Bug with nil optional fields degrades gracefully | Bug with nil Description, nil Slug | No panic; missing fields absent or empty string | Medium |

### AC-5: ChangeCardPlaceholders expanded with full field set (REQ-F-003, T6)

| TC-ID | Test Case | Input | Expected Output | Priority |
|-------|-----------|-------|-----------------|----------|
| TC-CC-01 | ChangeCard produces all common fields | ChangeCard with Key=`CC-001`, Title=`"DB migration"` | `key`=`CC-001`, `id`=`CC-001`, `title`=`DB migration` | High |
| TC-CC-02 | ChangeCard produces `requested_by` | ChangeCard with RequestedBy=`"john"` | `requested_by`=`john` | High |
| TC-CC-03 | ChangeCard produces `assigned_to` | ChangeCard with AssignedTo=`"jane"` | `assigned_to`=`jane` | High |
| TC-CC-04 | ChangeCard produces `justification` | ChangeCard with Justification=`"Performance fix"` | `justification`=`Performance fix` | High |
| TC-CC-05 | ChangeCard produces `impact_analysis` and `rollback_plan` | ChangeCard with both fields set | Both keys present with correct values | High |
| TC-CC-06 | ChangeCard produces `slug`, `created_at`, `updated_at` | ChangeCard with all fields set | All three keys present | Medium |
| TC-CC-07 | Nil change card returns empty map | nil | `len(map) == 0` | High |
| TC-CC-08 | ChangeCard with nil optional fields | ChangeCard with nil Description | No panic; `description` absent or empty | Medium |

### AC-6: GetStatusActionPopulated signature change (REQ-F-004, T7)

| TC-ID | Test Case | Input | Expected Output | Priority |
|-------|-----------|-------|-----------------|----------|
| TC-AS-01 | Accepts `map[string]string` instead of `taskID string` | `vars = map[string]string{"key": "E07-F01-001", "epic_key": "E07"}` | PopulatedAction with vars substituted in instruction | High |
| TC-AS-02 | Full task variable map populates all placeholders | TaskPlaceholders(task) passed as vars | All `{key}`, `{epic_key}`, `{feature_key}` replaced in instruction | High |
| TC-AS-03 | Bug variable map works | BugPlaceholders(bug) passed as vars | `{key}`, `{severity}` replaced in instruction | High |
| TC-AS-04 | ChangeCard variable map works | ChangeCardPlaceholders(cc) passed as vars | `{key}`, `{requested_by}` replaced in instruction | High |
| TC-AS-05 | Empty vars map does not panic | `map[string]string{}` | PopulatedAction returned with unreplaced placeholders | Medium |
| TC-AS-06 | Status not found returns StatusNotFoundError | status=`"nonexistent"` | `StatusNotFoundError` returned | High |
| TC-AS-07 | Status with no action returns nil PopulatedAction | status with no orchestrator_action | `nil, nil` returned | High |

### AC-7: Key parsing helpers (T1)

| TC-ID | Test Case | Input | Expected Output | Priority |
|-------|-----------|-------|-----------------|----------|
| TC-PK-01 | Parse epic key from task key (T- prefix) | `T-E07-F01-001` | `E07` | High |
| TC-PK-02 | Parse epic key from task key (short format) | `E07-F01-001` | `E07` | High |
| TC-PK-03 | Parse epic key from feature key | `E07-F01` | `E07` | High |
| TC-PK-04 | Parse epic key from slugged task key | `T-E07-F01-001-implement-jwt` | `E07` | Medium |
| TC-PK-05 | Parse epic key from epic key (identity) | `E07` | `E07` | Medium |
| TC-PK-06 | Parse feature key from task key (T- prefix) | `T-E07-F01-001` | `E07-F01` | High |
| TC-PK-07 | Parse feature key from task key (short format) | `E07-F01-001` | `E07-F01` | High |
| TC-PK-08 | Parse feature key from slugged task key | `E07-F01-001-my-task` | `E07-F01` | Medium |
| TC-PK-09 | Parse feature key from non-task key returns empty | `E07` | `""` | Medium |
| TC-PK-10 | Empty string input returns empty | `""` | `""` | Medium |
| TC-PK-11 | Case insensitive parsing | `e07-f01-001` | `E07` or `e07` (based on original casing) | Low |

---

## 2. API Contract Test Cases

### 2.1 ActionService Interface Change

**Before (current)**:
```go
GetStatusActionPopulated(ctx context.Context, status string, taskID string) (*PopulatedAction, error)
```

**After (target)**:
```go
GetStatusActionPopulated(ctx context.Context, status string, vars map[string]string) (*PopulatedAction, error)
```

#### Contract Tests

| TC-ID | Contract Test | Verification |
|-------|---------------|--------------|
| TC-CT-01 | `DefaultActionService` implements updated `ActionService` interface | Compile-time: `var _ ActionService = &DefaultActionService{}` |
| TC-CT-02 | `MockActionService` implements updated `ActionService` interface | Compile-time: `var _ ActionService = &MockActionService{}` |
| TC-CT-03 | `MockActionService.GetStatusActionPopulatedFunc` accepts `map[string]string` | Mock function field signature matches `func(ctx context.Context, status string, vars map[string]string) (*PopulatedAction, error)` |
| TC-CT-04 | `DefaultActionService.GetStatusActionPopulated` passes `vars` directly to `PopulateTemplate` | Verify no hardcoded 4-var map construction inside implementation |
| TC-CT-05 | No callers still pass `taskID string` | `grep` for old signature pattern returns 0 matches outside of docs/plan |

### 2.2 Placeholder Function Return Contracts

Each placeholder function's return map must satisfy the canonical variable set defined in ADR-2.

| TC-ID | Function | Required Keys | Forbidden Keys |
|-------|----------|---------------|----------------|
| TC-RC-01 | `TaskPlaceholders()` | `key`, `id`, `task_key`, `epic_key`, `feature_key`, `title`, `status`, `priority`, `created_at`, `updated_at` | `task_id`, `epic_id`, `feature_id` |
| TC-RC-02 | `FeaturePlaceholders()` | `key`, `id`, `epic_key`, `title`, `status`, `created_at`, `updated_at` | `feature_id` |
| TC-RC-03 | `EpicPlaceholders()` | `key`, `id`, `title`, `status`, `priority`, `created_at`, `updated_at` | `epic_id` |
| TC-RC-04 | `BugPlaceholders()` | `key`, `id`, `title`, `status`, `severity`, `file_path` | (none removed) |
| TC-RC-05 | `ChangeCardPlaceholders()` | `key`, `id`, `title`, `status`, `priority`, `file_path`, `description` | (none removed) |

---

## 3. Component Test Strategy

### 3.1 Key Parsing Helpers (Pure Unit Tests)

**File**: `internal/config/template_helpers_test.go`
**Pattern**: Table-driven tests, no mocks, no database.

```go
func TestParseEpicKeyFromEntityKey(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"task key with T- prefix", "T-E07-F01-001", "E07"},
        {"task key short format", "E07-F01-001", "E07"},
        {"feature key", "E07-F01", "E07"},
        {"epic key identity", "E07", "E07"},
        {"slugged task key", "T-E07-F01-001-implement-jwt", "E07"},
        {"empty string", "", ""},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := parseEpicKeyFromEntityKey(tt.input)
            if got != tt.expected {
                t.Errorf("parseEpicKeyFromEntityKey(%q) = %q, want %q", tt.input, got, tt.expected)
            }
        })
    }
}
```

Similarly for `parseFeatureKeyFromTaskKey`.

**Test count**: ~20 test cases across both helpers (TC-PK-01 through TC-PK-11).

### 3.2 Placeholder Functions (Pure Unit Tests)

**File**: `internal/config/template_helpers_test.go`
**Pattern**: Table-driven tests constructing model structs, asserting map contents.

Each test creates a model instance with known field values and verifies:
1. All required keys are present with correct values
2. Forbidden keys (removed aliases) are absent
3. Optional keys present only when source field is non-nil
4. Nil model input returns empty map

**Test count**: ~40 test cases (TC-TP-01 through TC-CC-08).

### 3.3 GetStatusActionPopulated (Unit Test with Config)

**File**: `internal/config/action_service_test.go`
**Pattern**: Uses `testConfigDir` helper to create temp config with orchestrator_action templates containing placeholder variables.

Updated test config should include an orchestrator_action with template placeholders:
```json
{
    "status_metadata": {
        "ready_for_development": {
            "color": "blue",
            "phase": "development",
            "orchestrator_action": {
                "action": "develop",
                "agent_type": "developer",
                "instruction": "Work on {key} in epic {epic_key} feature {feature_key}"
            }
        }
    }
}
```

Tests verify:
- Passing `TaskPlaceholders(task)` correctly replaces `{key}`, `{epic_key}`, `{feature_key}` in instruction
- Passing `BugPlaceholders(bug)` replaces `{key}`, `{severity}`
- Passing `ChangeCardPlaceholders(cc)` replaces `{key}`, `{requested_by}`
- Empty map does not panic
- Old `{task_id}` placeholder remains unreplaced (confirming removal)

**Test count**: ~7 test cases (TC-AS-01 through TC-AS-07).

### 3.4 MockActionService (Compile-Time Verification)

**File**: `internal/config/mock_action_service.go` (updated) + test file
**Pattern**: Interface satisfaction check.

```go
// Compile-time verification
var _ ActionService = &MockActionService{}
var _ ActionService = &DefaultActionService{}
```

---

## 4. Integration Scenarios

### 4.1 Service Layer Integration (PopulateTemplate flow)

The 7 service call sites that call `meta.OrchestratorAction.PopulateTemplate(placeholders)` directly are not affected by the `GetStatusActionPopulated` signature change. However, they ARE affected by the placeholder key changes.

| Scenario | Service | Current Call | Impact |
|----------|---------|-------------|--------|
| IS-01 | TaskService `resolveAction()` | `PopulateTemplate(TaskPlaceholders(task))` | TaskPlaceholders now returns different keys; templates using `{task_id}` will stop resolving |
| IS-02 | FeatureService `resolveAction()` | `PopulateTemplate(FeaturePlaceholders(feature))` | `{feature_id}` stops resolving; `{key}` and `{epic_key}` start working |
| IS-03 | EpicService `resolveAction()` | `PopulateTemplate(EpicPlaceholders(epic))` | `{epic_id}` stops resolving; `{key}` starts working |
| IS-04 | BugService `resolveAction()` | `PopulateTemplate(BugPlaceholders(bug))` | New fields available: `{linked_entity_type}`, `{linked_entity_key}`, etc. |
| IS-05 | ChangeCardService `resolveAction()` | `PopulateTemplate(ChangeCardPlaceholders(cc))` | New fields available: `{requested_by}`, `{assigned_to}`, etc. |
| IS-06 | DisplayService (2 call sites) | Various placeholder calls | Same key change impact |

### 4.2 Template File Migration (T9)

| Scenario | File Pattern | Search For | Replace With |
|----------|-------------|------------|--------------|
| TM-01 | `*.tmpl` files | `{task_id}` | `{key}` or `{task_key}` |
| TM-02 | `*.tmpl` files | `{epic_id}` | `{epic_key}` |
| TM-03 | `*.tmpl` files | `{feature_id}` | `{feature_key}` |
| TM-04 | `.sharkconfig.json` | `{task_id}` in orchestrator_action instructions | `{key}` |
| TM-05 | `.sharkconfig.json` | `{epic_id}` in orchestrator_action instructions | `{epic_key}` |
| TM-06 | `.sharkconfig.json` | `{feature_id}` in orchestrator_action instructions | `{feature_key}` |

**Verification**: After migration, `grep -r '{task_id}\|{epic_id}\|{feature_id}' --include='*.tmpl' --include='*.json'` returns 0 matches in production code.

### 4.3 WithRelated Variants Unaffected

The `TaskPlaceholdersWithRelated`, `FeaturePlaceholdersWithRelated`, and `EpicPlaceholdersWithRelated` functions call their base functions (`TaskPlaceholders`, etc.) and extend the map. After the base functions are updated:

| Scenario | Verification |
|----------|--------------|
| WR-01 | `TaskPlaceholdersWithRelated` inherits new `key`, `task_key`, `epic_key`, `feature_key` from base |
| WR-02 | `FeaturePlaceholdersWithRelated` inherits new `key`, `epic_key` from base |
| WR-03 | `EpicPlaceholdersWithRelated` inherits new `key` from base |
| WR-04 | `related_docs`, `related_tasks`, `related_features`, `related_epics` keys unchanged |

Existing tests for WithRelated functions (TC-PH-01 through TC-EPH-04) should continue passing after updates, but assertions checking for `task_id`/`epic_id`/`feature_id` in the base map need updating.

### 4.4 Cross-Feature Touchpoints

| Feature | Touchpoint | Risk | Mitigation |
|---------|-----------|------|------------|
| E07-F30 (Template Engine) | `OrchestratorRenderer.Render()` uses `map[string]string` | None -- renderer is map-agnostic | No code changes needed |
| E07-F29 (Related Docs/Tasks) | `WithRelated` functions extend base placeholder functions | Medium -- existing tests may assert old keys | Update test assertions |
| E07-F21 (Status Actions) | `ActionService` interface is defined here | High -- interface signature change is breaking | Update all implementations and callers simultaneously |
| Status cascade (services) | `resolveAction()` methods call placeholder functions | Medium -- template strings in config may reference old vars | Config migration (T9) |

---

## 5. Quality Gates

| Gate | Threshold | Measurement |
|------|-----------|-------------|
| Unit test pass rate | 100% | `make test` exits 0 |
| No old variable names in code | 0 matches | `grep -r 'task_id\|epic_id\|feature_id' internal/config/template_helpers.go` returns empty |
| No old variable names in templates | 0 matches | `grep -r '{task_id}\|{epic_id}\|{feature_id}' --include='*.tmpl'` returns empty |
| Interface satisfaction | Compile-time | `var _ ActionService = &DefaultActionService{}` compiles |
| Lint clean | 0 errors | `make lint` exits 0 |
| No panics on nil input | All nil tests pass | TC-TP-08, TC-FP-05, TC-EP-05, TC-BP-06, TC-CC-07 |

---

## 6. Test Execution Order

Recommended implementation and testing order matching the task breakdown:

1. **T1**: Key parsing helpers + tests (TC-PK-*) -- no dependencies
2. **T4**: EpicPlaceholders update + tests (TC-EP-*) -- simplest, XS
3. **T3**: FeaturePlaceholders update + tests (TC-FP-*) -- uses parseEpicKey
4. **T2**: TaskPlaceholders update + tests (TC-TP-*) -- uses both parsers
5. **T5**: BugPlaceholders expansion + tests (TC-BP-*)
6. **T6**: ChangeCardPlaceholders expansion + tests (TC-CC-*)
7. **T7**: GetStatusActionPopulated signature change + tests (TC-AS-*, TC-CT-*)
8. **T8**: MockActionService update + interface verification
9. **T9**: Template/config migration + integration verification (TM-*, IS-*)
10. **T10**: Documentation update (manual review)

---

*Last Updated*: 2026-03-11
