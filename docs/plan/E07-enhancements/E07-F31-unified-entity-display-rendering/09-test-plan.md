# Test Plan: E07-F31 - Unified Entity Display Rendering

**Feature**: E07-F31 - Unified Entity Display Rendering
**Epic**: E07 - Enhancements
**Complexity Tier**: STANDARD (4/18 points)
**Created**: 2026-02-16

---

## Overview

This test plan covers E07-F31: creating shared rendering infrastructure and making surgical fixes to display related documents and valid transitions in epic/feature planning mode. The plan is scoped for STANDARD tier complexity with focused coverage on acceptance criteria, component testing, and API contract validation.

**Test Strategy Focus**:
- Acceptance criteria test matrix (10 scenarios from PRD)
- Component unit tests (8 helper functions in render_common.go)
- Integration scenarios (planning mode display with related docs)
- API contract tests (JSON backward compatibility for epic and feature)

**Personas Addressed**:
- **AI Agent**: Can discover related docs in planning mode via JSON output
- **Developer**: Sees valid transitions in CLI output for workflow decisions
- **QA Engineer**: Can verify test plan visibility in related documents

---

## 1. Acceptance Criteria Test Matrix

### AC Test Case 1.1: AI Agent - Related Docs in Epic Planning Mode (JSON)

**User Story**: Story 1 - AI agent discovers related documents in planning mode
**Acceptance Criterion**: When epic status is in planning mode, `shark epic get E## --json` includes `related_documents` array

**Test Scenario**:
```
GIVEN epic E07 is in status "ready_for_decomposition" (planning mode)
  AND epic has 2 related documents registered:
      - "Epic Research Report" (research)
      - "Feature Decomposition Plan" (planning)
WHEN AI agent executes `shark epic get E07 --json`
THEN JSON response includes:
  - Field "related_documents" (array)
  - Array has 2 items
  - Each item has: title, type, file_path fields
  - All existing JSON fields preserved (epic, display_mode, phase, orchestrator_action)
```

**Test Data**:
- Epic key: `E07`
- Status: `ready_for_decomposition`
- Related docs:
  1. Title: "Epic Research Report", Type: "research", FilePath: "docs/plan/E07/research.md"
  2. Title: "Feature Decomposition Plan", Type: "planning", FilePath: "docs/plan/E07/decomposition.md"

**Expected JSON Output Structure**:
```json
{
  "epic": {
    "id": 7,
    "key": "E07",
    "status": "ready_for_decomposition"
  },
  "display_mode": "planning",
  "phase": "decomposition",
  "orchestrator_action": { ... },
  "related_documents": [
    {
      "title": "Epic Research Report",
      "type": "research",
      "file_path": "docs/plan/E07/research.md"
    },
    {
      "title": "Feature Decomposition Plan",
      "type": "planning",
      "file_path": "docs/plan/E07/decomposition.md"
    }
  ],
  "valid_transitions": ["in_decomposition", "blocked"]
}
```

**Pass Criteria**:
- ✅ `related_documents` field exists in JSON
- ✅ Array contains exactly 2 items
- ✅ Each item has title, type, file_path fields
- ✅ All existing fields preserved

---

### AC Test Case 1.2: AI Agent - Related Docs in Feature Planning Mode (JSON)

**User Story**: Story 1
**Acceptance Criterion**: When feature status is in planning mode, `shark feature get E##-F## --json` includes `related_documents` array

**Test Scenario**:
```
GIVEN feature E07-F31 is in status "ready_for_refinement_ba" (planning mode)
  AND feature has 3 related documents:
      - "Feature PRD" (prd)
      - "Architecture Design" (architecture)
      - "Complexity Triage Report" (research)
WHEN AI agent executes `shark feature get E07-F31 --json`
THEN JSON response includes:
  - Field "related_documents" (array)
  - Array has 3 items
  - Each item has: title, type, file_path fields
  - Orchestrator action is present
```

**Test Data**:
- Feature key: `E07-F31`
- Status: `ready_for_refinement_ba`
- Related docs:
  1. Title: "Feature PRD", Type: "prd", FilePath: "docs/plan/E07-F31/feature.md"
  2. Title: "Architecture Design", Type: "architecture", FilePath: "docs/plan/E07-F31/02-architecture.md"
  3. Title: "Complexity Triage Report", Type: "research", FilePath: "docs/plan/E07-F31/COMPLEXITY-TRIAGE-REPORT.md"

**Pass Criteria**:
- ✅ `related_documents` field exists
- ✅ Array contains 3 items
- ✅ Each item structure matches expected format
- ✅ Orchestrator action included

---

### AC Test Case 2.1: Developer - Valid Transitions in Epic Human Output

**User Story**: Story 2 - Developer sees valid status transitions
**Acceptance Criterion**: `shark epic get E##` human-readable output displays "Valid Transitions" section

**Test Scenario**:
```
GIVEN epic E07 is in status "active"
  AND workflow config defines transitions: ["completed", "on_hold"]
WHEN developer runs `shark epic get E07` (no --json flag)
THEN human-readable output includes:
  - Section header "Valid Transitions"
  - Bulleted list of transitions:
      - completed
      - on_hold
  - Section appears before "Related Documents"
  - All existing sections still display (features table, rollups)
```

**Test Data**:
- Epic key: `E07`
- Status: `active`
- Workflow config:
  ```json
  "status_flow": {
    "active": ["completed", "on_hold"]
  }
  ```

**Expected Output** (excerpt):
```
Epic: E07
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Title                   Enhancements
Status                  active
Priority                high

Valid Transitions
━━━━━━━━━━━━━━━━━━
  - completed
  - on_hold

Orchestrator Action
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
...

Related Documents
━━━━━━━━━━━━━━━━━━
  - Epic Research Report (docs/plan/E07/research.md)
```

**Pass Criteria**:
- ✅ "Valid Transitions" section header displayed
- ✅ Transitions listed as bullet points
- ✅ Section positioned before "Related Documents"
- ✅ Existing sections unchanged

---

### AC Test Case 2.2: Developer - Valid Transitions in Feature JSON Output

**User Story**: Story 2
**Acceptance Criterion**: `shark feature get E##-F## --json` includes `valid_transitions` array

**Test Scenario**:
```
GIVEN feature E07-F31 is in status "ready_for_refinement_ba"
  AND workflow defines transitions: ["in_refinement_ba", "blocked", "on_hold"]
WHEN developer runs `shark feature get E07-F31 --json`
THEN JSON response includes:
  - Field "valid_transitions" (array of strings)
  - Array contains: ["in_refinement_ba", "blocked", "on_hold"]
  - All existing JSON fields preserved
```

**Test Data**:
- Feature key: `E07-F31`
- Status: `ready_for_refinement_ba`
- Workflow config:
  ```json
  "status_flow": {
    "ready_for_refinement_ba": ["in_refinement_ba", "blocked", "on_hold"]
  }
  ```

**Expected JSON** (excerpt):
```json
{
  "feature": {
    "key": "E07-F31",
    "status": "ready_for_refinement_ba"
  },
  "valid_transitions": ["in_refinement_ba", "blocked", "on_hold"],
  "orchestrator_action": { ... }
}
```

**Pass Criteria**:
- ✅ `valid_transitions` field exists
- ✅ Array contains expected transitions
- ✅ Field added (not replacing existing fields)

---

### AC Test Case 3.1: QA Engineer - Orchestrator Action Consistency

**User Story**: Story 3 - Consistent orchestrator action display
**Acceptance Criterion**: Orchestrator action displayed in both planning and aggregation modes

**Test Scenario**:
```
GIVEN feature E07-F31 is in status "ready_for_test_planning"
  AND orchestrator action is configured for this status:
      - agent_type: "qa"
      - skills: ["quality", "shark"]
      - instruction: "Create feature test plan..."
WHEN QA runs `shark feature get E07-F31`
THEN human-readable output includes:
  - "Orchestrator Action" section
  - Section shows agent_type: "qa"
  - Section shows skills: ["quality", "shark"]
  - Section positioned after "Valid Transitions", before "Related Documents"
```

**Test Data**:
- Feature key: `E07-F31`
- Status: `ready_for_test_planning`
- Orchestrator action:
  - agent_type: `qa`
  - skills: `["quality", "shark"]`
  - instruction: "Create feature test plan for E07-F31..."

**Pass Criteria**:
- ✅ "Orchestrator Action" section displayed
- ✅ agent_type shown
- ✅ Skills listed
- ✅ Consistent position in output

---

### AC Test Case 4.1: Helper Function - Related Docs Display

**User Story**: Story 4 - Helper functions for common sections
**Acceptance Criterion**: `renderRelatedDocuments()` helper handles various input scenarios

**Test Scenarios**:

**Scenario 4.1a: Multiple Documents**
```
GIVEN docs = [
  {Title: "Doc1", FilePath: "path/doc1.md"},
  {Title: "Doc2", FilePath: "path/doc2.md"}
]
WHEN renderRelatedDocuments(docs) is called
THEN output displays:
  - Section header "Related Documents"
  - Bullet list:
      - Doc1 (path/doc1.md)
      - Doc2 (path/doc2.md)
```

**Scenario 4.1b: Empty Array**
```
GIVEN docs = []
WHEN renderRelatedDocuments(docs) is called
THEN no output (section skipped gracefully)
```

**Scenario 4.1c: Nil Input**
```
GIVEN docs = nil
WHEN renderRelatedDocuments(docs) is called
THEN no panic (section skipped)
```

**Pass Criteria**:
- ✅ Multiple docs displayed correctly
- ✅ Empty array skips section
- ✅ Nil input handled without panic

---

### AC Test Case 4.2: Helper Function - Valid Transitions Extraction

**User Story**: Story 4
**Acceptance Criterion**: `GetValidTransitions()` extracts transitions from workflow config

**Test Scenarios**:

**Scenario 4.2a: Valid Status**
```
GIVEN workflow config:
  status_flow = {
    "in_progress": ["ready_for_review", "blocked"]
  }
  AND status = "in_progress"
WHEN GetValidTransitions("in_progress", workflow) is called
THEN returns ["ready_for_review", "blocked"]
```

**Scenario 4.2b: Terminal Status (not in status_flow)**
```
GIVEN workflow config:
  status_flow = {
    "in_progress": ["completed"]
  }
  AND status = "completed"
WHEN GetValidTransitions("completed", workflow) is called
THEN returns [] (empty array)
```

**Scenario 4.2c: Nil Workflow Config**
```
GIVEN workflow = nil
WHEN GetValidTransitions("in_progress", nil) is called
THEN returns [] (empty array, no panic)
```

**Pass Criteria**:
- ✅ Valid status returns correct transitions
- ✅ Terminal status returns empty array
- ✅ Nil config returns empty array without crash

---

### AC Test Case 5.1: Edge Case - Missing Workflow Config

**Error Story**: Error Story 1 - Handle missing workflow config gracefully
**Acceptance Criterion**: Missing `.sharkconfig.json` returns empty valid_transitions array

**Test Scenario**:
```
GIVEN .sharkconfig.json does not exist
  AND feature E07-F31 exists in database
WHEN user runs `shark feature get E07-F31 --json`
THEN JSON response includes:
  - Field "valid_transitions": []
  - Command completes successfully (exit code 0)
  - Other sections display normally
```

**Test Data**:
- Feature key: `E07-F31`
- Config file: Missing/not found

**Pass Criteria**:
- ✅ Command does not crash
- ✅ `valid_transitions` field is empty array
- ✅ Exit code 0
- ✅ Other JSON fields present

---

### AC Test Case 5.2: Edge Case - No Related Documents

**Error Story**: Error Story 2 - Handle entities with no related docs
**Acceptance Criterion**: Entities with no related docs return empty related_documents array in JSON

**Test Scenario**:
```
GIVEN epic E04 has no related documents registered
  AND epic is in status "active" (aggregation mode)
WHEN user runs `shark epic get E04 --json`
THEN JSON response includes:
  - Field "related_documents": []
  - No error or crash
```

**Test Data**:
- Epic key: `E04`
- Related docs: None (empty)

**Expected JSON** (excerpt):
```json
{
  "epic": {
    "key": "E04",
    "status": "active"
  },
  "related_documents": [],
  "valid_transitions": ["completed"]
}
```

**Pass Criteria**:
- ✅ `related_documents` field present
- ✅ Array is empty (not null)
- ✅ Command succeeds

---

### AC Test Case 6.1: Backward Compatibility - Existing JSON Consumers

**User Story**: Implicit requirement - Don't break existing integrations
**Acceptance Criterion**: All existing JSON fields preserved with same types

**Test Scenario**:
```
GIVEN existing AI agent expects JSON fields:
  - epic.id, epic.key, epic.status (object)
  - display_mode (string)
  - phase (string)
  - orchestrator_action (object)
WHEN agent parses `shark epic get E07 --json` response after E07-F31
THEN all existing fields present:
  - epic object unchanged
  - display_mode string unchanged
  - orchestrator_action object unchanged
  AND new fields added:
  - related_documents (array)
  - valid_transitions (array)
```

**Test Data**:
- Epic key: `E07`
- Status: `active`

**Validation Approach**:
1. Capture JSON output from current version (before E07-F31)
2. Capture JSON output from new version (after E07-F31)
3. Assert all fields from (1) exist in (2) with same types
4. Assert new fields are arrays

**Pass Criteria**:
- ✅ All existing fields present
- ✅ All existing field types unchanged
- ✅ New fields are additive only
- ✅ JSON parseable by existing consumers

---

## 2. Component Test Strategy

### 2.1 Unit Tests: render_common.go Helper Functions

**Test File**: `internal/cli/commands/render_common_test.go`
**Coverage Target**: 100% of helper functions

#### Test Suite 2.1.1: renderHeader()

**Function Signature**: `func renderHeader(entityType, key string)`

**Test Cases**:
- TC-renderHeader-001: Epic type capitalizes correctly ("Epic: E07")
- TC-renderHeader-002: Feature type capitalizes correctly ("Feature: E07-F01")
- TC-renderHeader-003: Task type capitalizes correctly ("Task: E07-F01-001")
- TC-renderHeader-004: Empty strings handled (no panic)

**Test Implementation Pattern**:
```go
func TestRenderHeader(t *testing.T) {
    tests := []struct {
        name       string
        entityType string
        key        string
        wantOutput string
    }{
        {
            name:       "epic header",
            entityType: "epic",
            key:        "E07",
            wantOutput: "Epic: E07",
        },
        {
            name:       "feature header",
            entityType: "feature",
            key:        "E07-F01",
            wantOutput: "Feature: E07-F01",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Capture output
            output := captureStdout(func() {
                renderHeader(tt.entityType, tt.key)
            })

            if !strings.Contains(output, tt.wantOutput) {
                t.Errorf("expected output to contain %q, got %q", tt.wantOutput, output)
            }
        })
    }
}
```

---

#### Test Suite 2.1.2: renderBasicInfo()

**Function Signature**: `func renderBasicInfo(info [][]string)`

**Test Cases**:
- TC-renderBasicInfo-001: Multiple rows render as table
- TC-renderBasicInfo-002: Single row renders correctly
- TC-renderBasicInfo-003: Empty array skips rendering (no output)
- TC-renderBasicInfo-004: Nil input skips rendering (no panic)

**Edge Cases**:
- Empty string values in rows
- Very long values (truncation handling)
- Special characters in values

---

#### Test Suite 2.1.3: renderValidTransitions()

**Function Signature**: `func renderValidTransitions(status string, transitions []string)`

**Test Cases**:
- TC-renderValidTransitions-001: Multiple transitions display as bulleted list
- TC-renderValidTransitions-002: Single transition displays correctly
- TC-renderValidTransitions-003: Empty array skips section (no output)
- TC-renderValidTransitions-004: Nil array skips section (no panic)

**Expected Output Pattern**:
```
Valid Transitions
━━━━━━━━━━━━━━━━━━
  - ready_for_review
  - blocked
```

---

#### Test Suite 2.1.4: renderRelatedDocuments()

**Function Signature**: `func renderRelatedDocuments(docs []*models.Document)`

**Test Cases**:
- TC-renderRelatedDocuments-001: Multiple docs display with title and path
- TC-renderRelatedDocuments-002: Single doc displays correctly
- TC-renderRelatedDocuments-003: Empty array skips section
- TC-renderRelatedDocuments-004: Nil array skips section
- TC-renderRelatedDocuments-005: Docs with missing fields handled

**Edge Cases**:
- Document with empty title
- Document with very long file path
- Document with nil fields

---

#### Test Suite 2.1.5: GetValidTransitions()

**Function Signature**: `func GetValidTransitions(status string, workflow *config.WorkflowConfig) []string`

**Test Cases**:
- TC-GetValidTransitions-001: Valid status returns transitions
- TC-GetValidTransitions-002: Terminal status returns empty array
- TC-GetValidTransitions-003: Nil workflow returns empty array
- TC-GetValidTransitions-004: Empty status_flow map returns empty array
- TC-GetValidTransitions-005: Status not in status_flow returns empty array

**Test Implementation Pattern**:
```go
func TestGetValidTransitions(t *testing.T) {
    tests := []struct {
        name     string
        status   string
        workflow *config.WorkflowConfig
        want     []string
    }{
        {
            name:   "nil config returns empty",
            status: "in_progress",
            workflow: nil,
            want:   []string{},
        },
        {
            name:   "valid status returns transitions",
            status: "in_progress",
            workflow: &config.WorkflowConfig{
                StatusFlow: map[string][]string{
                    "in_progress": {"ready_for_review", "blocked"},
                },
            },
            want: []string{"ready_for_review", "blocked"},
        },
        {
            name:   "terminal status returns empty",
            status: "completed",
            workflow: &config.WorkflowConfig{
                StatusFlow: map[string][]string{
                    "in_progress": {"completed"},
                },
            },
            want: []string{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := GetValidTransitions(tt.status, tt.workflow)
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

### 2.2 Unit Tests: renderOrchestratorAction()

**Function**: Wrapper around existing `displayOrchestratorAction()`

**Test Cases**:
- TC-renderOrchestratorAction-001: Non-nil action delegates correctly
- TC-renderOrchestratorAction-002: Nil action shows "None configured"
- TC-renderOrchestratorAction-003: Action with all fields displays completely

**Note**: This is a thin wrapper. Primary testing covered by existing `orchestrator_display_test.go`.

---

### 2.3 Unit Tests: renderNotes() and renderContextData()

**Functions**:
- `func renderNotes(notes []*models.EntityNote)`
- `func renderContextData(contextData *models.ContextData)`

**Test Cases**:
- TC-renderNotes-001: Displays most recent 10 notes
- TC-renderNotes-002: Truncates notes > 80 chars
- TC-renderNotes-003: Empty notes skips section
- TC-renderContextData-001: Displays context fields
- TC-renderContextData-002: Nil context skips section
- TC-renderContextData-003: Empty context skips section

---

## 3. Integration Test Scenarios

### Integration Test 3.1: Epic Planning Mode with Related Docs

**Test File**: `internal/cli/commands/epic_get_integration_test.go`

**Test Scenario**:
```
GIVEN epic E07 exists in database
  AND epic status is "ready_for_decomposition"
  AND 2 related documents registered for epic
WHEN test calls `shark epic get E07`
THEN output includes:
  - "Related Documents" section
  - Both documents listed
  - Orchestrator action section
  - Valid transitions section
```

**Setup**:
1. Create test database with epic E07
2. Insert 2 related documents
3. Set epic status to planning mode status
4. Execute command and capture output

**Assertions**:
- ✅ "Related Documents" section header found
- ✅ 2 document titles displayed
- ✅ File paths included
- ✅ Section positioned after orchestrator action

---

### Integration Test 3.2: Feature Planning Mode with Related Docs

**Test File**: `internal/cli/commands/feature_get_integration_test.go`

**Test Scenario**:
```
GIVEN feature E07-F31 exists in database
  AND feature status is "ready_for_refinement_ba"
  AND 3 related documents registered
WHEN test calls `shark feature get E07-F31`
THEN output includes:
  - "Related Documents" section
  - 3 documents listed
  - Valid transitions section
```

**Setup**:
1. Create test database with feature E07-F31
2. Insert 3 related documents (PRD, architecture, triage)
3. Set feature status to planning mode
4. Execute command and capture output

---

### Integration Test 3.3: JSON Output with New Fields

**Test Scenario**:
```
GIVEN epic E07 in planning mode with related docs
WHEN test calls `shark epic get E07 --json`
THEN JSON response:
  - Contains "related_documents" array
  - Contains "valid_transitions" array
  - All existing fields present
  - JSON is valid and parseable
```

**Validation**:
1. Parse JSON response
2. Assert `related_documents` field exists and is array
3. Assert `valid_transitions` field exists and is array
4. Assert all pre-existing fields present
5. Assert JSON schema valid

---

### Integration Test 3.4: Aggregation Mode Unchanged (Regression)

**Test Scenario**:
```
GIVEN epic E07 in status "active" (aggregation mode)
WHEN test calls `shark epic get E07`
THEN output includes:
  - All existing sections (features table, rollups)
  - Related documents section (existing behavior)
  - Valid transitions section (new)
  - Section ordering consistent
```

**Purpose**: Ensure changes don't break aggregation mode rendering

**Assertions**:
- ✅ Features table displayed
- ✅ Feature status rollup displayed
- ✅ Task status rollup displayed
- ✅ Related documents section present
- ✅ Valid transitions section added
- ✅ No sections removed

---

## 4. API Contract Tests

### Contract Test 4.1: Epic JSON Response Schema

**Purpose**: Ensure JSON output for epic get command remains backward compatible

**Test Approach**: JSON schema validation

**Pre-E07-F31 Schema** (baseline):
```json
{
  "epic": {
    "id": "integer",
    "key": "string",
    "title": "string",
    "status": "string",
    "created_at": "string (ISO 8601)",
    "updated_at": "string (ISO 8601)"
  },
  "display_mode": "string",
  "phase": "string",
  "orchestrator_action": "object|null"
}
```

**Post-E07-F31 Schema** (with new fields):
```json
{
  "epic": { ... },  // unchanged
  "display_mode": "string",  // unchanged
  "phase": "string",  // unchanged
  "orchestrator_action": "object|null",  // unchanged
  "related_documents": "array",  // NEW
  "valid_transitions": "array"  // NEW
}
```

**Test Implementation**:
```go
func TestEpicGetJSONBackwardCompatibility(t *testing.T) {
    // Execute command
    output := execCommand("shark epic get E07 --json")

    // Parse JSON
    var response map[string]interface{}
    if err := json.Unmarshal([]byte(output), &response); err != nil {
        t.Fatalf("JSON parse error: %v", err)
    }

    // Assert existing fields
    assertFieldExists(t, response, "epic", "object")
    assertFieldExists(t, response, "display_mode", "string")
    assertFieldExists(t, response, "phase", "string")
    assertFieldExists(t, response, "orchestrator_action", "object|null")

    // Assert new fields (additive)
    assertFieldExists(t, response, "related_documents", "array")
    assertFieldExists(t, response, "valid_transitions", "array")
}
```

**Pass Criteria**:
- ✅ All existing fields present
- ✅ Field types unchanged
- ✅ New fields are arrays
- ✅ JSON valid and parseable

---

### Contract Test 4.2: Feature JSON Response Schema

**Purpose**: Ensure feature JSON output backward compatible

**Pre-E07-F31 Schema**:
```json
{
  "feature": {
    "id": "integer",
    "key": "string",
    "title": "string",
    "status": "string"
  },
  "display_mode": "string",
  "phase": "string",
  "orchestrator_action": "object|null",
  "path": "string"
}
```

**Post-E07-F31 Schema**:
```json
{
  "feature": { ... },  // unchanged
  "display_mode": "string",  // unchanged
  "phase": "string",  // unchanged
  "orchestrator_action": "object|null",  // unchanged
  "related_documents": "array",  // NEW
  "valid_transitions": "array",  // NEW
  "path": "string"  // unchanged
}
```

**Test Assertions**:
- Same pattern as Contract Test 4.1
- Verify all existing fields unchanged
- Verify new fields added correctly

---

### Contract Test 4.3: Related Documents Array Structure

**Purpose**: Ensure related_documents array items have consistent structure

**Expected Item Schema**:
```json
{
  "title": "string",
  "type": "string",
  "file_path": "string"
}
```

**Test Scenario**:
```
GIVEN epic with 2 related documents
WHEN JSON output parsed
THEN each item in related_documents array:
  - Has "title" field (string)
  - Has "type" field (string)
  - Has "file_path" field (string)
  - No extra fields
```

**Test Implementation**:
```go
func TestRelatedDocumentsArrayStructure(t *testing.T) {
    output := execCommand("shark epic get E07 --json")
    var response map[string]interface{}
    json.Unmarshal([]byte(output), &response)

    docs := response["related_documents"].([]interface{})
    for i, doc := range docs {
        docMap := doc.(map[string]interface{})

        // Assert required fields
        if _, ok := docMap["title"]; !ok {
            t.Errorf("doc[%d] missing 'title' field", i)
        }
        if _, ok := docMap["type"]; !ok {
            t.Errorf("doc[%d] missing 'type' field", i)
        }
        if _, ok := docMap["file_path"]; !ok {
            t.Errorf("doc[%d] missing 'file_path' field", i)
        }
    }
}
```

---

## 5. Test Execution Summary

### Test Coverage Metrics

**Unit Tests** (render_common_test.go):
- 8 helper functions × 4-5 test cases each = ~35 test cases
- Target coverage: 100% of helper functions
- Execution time: < 100ms (no database, pure logic)

**Integration Tests** (epic_get_integration_test.go, feature_get_integration_test.go):
- 4 integration scenarios
- Uses test database (must clean up)
- Execution time: ~500ms

**API Contract Tests**:
- 3 contract validation scenarios
- JSON schema validation
- Execution time: ~200ms

**Total Test Count**: ~42 test cases
**Total Execution Time**: < 1 second

---

### Exit Gate Checklist (STANDARD Tier)

- [ ] **AC Coverage**: All 10 acceptance criteria have test cases
- [ ] **Component Tests**: All 8 helper functions have unit tests
- [ ] **Integration Tests**: Planning mode display verified with related docs
- [ ] **API Contracts**: JSON backward compatibility verified for epic and feature
- [ ] **Edge Cases**: Missing config and empty docs scenarios tested
- [ ] **Regression**: Aggregation mode unchanged (existing tests pass)
- [ ] **Personas Validated**:
  - [ ] AI Agent: Can discover related docs via JSON
  - [ ] Developer: Sees valid transitions in CLI output
  - [ ] QA Engineer: Can verify test plan visibility

---

## 6. Test Execution Instructions

### Running Unit Tests

```bash
# Run all unit tests for render_common.go
go test -v ./internal/cli/commands -run TestRender

# Run specific helper function tests
go test -v ./internal/cli/commands -run TestGetValidTransitions
go test -v ./internal/cli/commands -run TestRenderRelatedDocuments

# Run with coverage
go test -cover ./internal/cli/commands/render_common_test.go
```

### Running Integration Tests

```bash
# Run epic planning mode integration tests
go test -v ./internal/cli/commands -run TestEpicGetIntegration

# Run feature planning mode integration tests
go test -v ./internal/cli/commands -run TestFeatureGetIntegration

# Run all integration tests
go test -v ./internal/cli/commands -run Integration
```

### Running API Contract Tests

```bash
# Run backward compatibility tests
go test -v ./internal/cli/commands -run TestBackwardCompatibility

# Run JSON schema validation tests
go test -v ./internal/cli/commands -run TestJSONSchema
```

### Full Test Suite

```bash
# Run all tests for E07-F31
make test

# Run with coverage report
make test-coverage
# View coverage.html in browser
```

---

## 7. Test Data Management

### Test Database Setup

**Location**: `internal/repository/test-shark-tasks.db`
**Managed by**: `internal/test/testdb.go`

**Setup Pattern**:
```go
func setupTest(t *testing.T) (*DB, func()) {
    db := test.GetTestDB()

    cleanup := func() {
        db.Exec("DELETE FROM documents WHERE epic_id IN (SELECT id FROM epics WHERE key = 'E07')")
        db.Exec("DELETE FROM features WHERE key LIKE 'E07-%'")
        db.Exec("DELETE FROM epics WHERE key = 'E07'")
    }

    // Clean before test
    cleanup()

    return db, cleanup
}
```

### Test Fixtures

**Epic Fixture**:
- Key: `E07`
- Title: "Enhancements"
- Status: `ready_for_decomposition` (planning) or `active` (aggregation)

**Feature Fixture**:
- Key: `E07-F31`
- Title: "Unified Entity Display Rendering"
- Status: `ready_for_refinement_ba`

**Related Documents Fixture**:
1. Epic E07:
   - "Epic Research Report" (research) - docs/plan/E07/research.md
   - "Feature Decomposition Plan" (planning) - docs/plan/E07/decomposition.md

2. Feature E07-F31:
   - "Feature PRD" (prd) - docs/plan/E07-F31/feature.md
   - "Architecture Design" (architecture) - docs/plan/E07-F31/02-architecture.md
   - "Complexity Triage Report" (research) - docs/plan/E07-F31/COMPLEXITY-TRIAGE-REPORT.md

---

## 8. Traceability Matrix

| Test Case ID | User Story | Acceptance Criterion | Test Type | Priority |
|--------------|------------|---------------------|-----------|----------|
| AC-1.1 | Story 1 | Epic planning mode related docs JSON | Integration | Must-Have |
| AC-1.2 | Story 1 | Feature planning mode related docs JSON | Integration | Must-Have |
| AC-2.1 | Story 2 | Epic valid transitions human output | Integration | Must-Have |
| AC-2.2 | Story 2 | Feature valid transitions JSON output | Integration | Must-Have |
| AC-3.1 | Story 3 | Orchestrator action consistency | Integration | Must-Have |
| AC-4.1 | Story 4 | renderRelatedDocuments() helper | Unit | Should-Have |
| AC-4.2 | Story 4 | GetValidTransitions() helper | Unit | Must-Have |
| AC-5.1 | Error Story 1 | Missing workflow config | Unit | Must-Have |
| AC-5.2 | Error Story 2 | No related documents | Integration | Must-Have |
| AC-6.1 | Implicit | JSON backward compatibility | Contract | Must-Have |

---

**Document Status**: Ready for Implementation
**Next Steps**:
1. Create `render_common_test.go` with unit tests for 8 helper functions
2. Add integration tests to `epic_get_integration_test.go`
3. Add integration tests to `feature_get_integration_test.go`
4. Implement API contract validation tests
5. Execute full test suite and verify coverage

**Advance Feature**: `shark feature next-status E07-F31`
