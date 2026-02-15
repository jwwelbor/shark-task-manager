# Template Variables for Related Docs and Tasks - Test Plan

**Feature**: E07-F29-template-variables-for-related-docs-and-tasks
**Date**: 2026-02-13
**Status**: Draft Test Plan
**QA Engineer**: QA Agent

---

## Executive Summary

This test plan provides comprehensive test coverage for the template variable extension feature, enabling AI agents to receive related documentation, task, feature, and epic context automatically through orchestrator action instruction templates. The plan covers unit tests (helper functions), integration tests (database-backed placeholder population with relationship tables), and acceptance tests (end-to-end CLI validation) with a target of **≥80% code coverage**.

**Test Complexity**: **Medium-High** - straightforward logic with multiple integration points including new relationship tables
**Total Test Cases**: 67 (37 unit + 15 integration + 12 acceptance + 3 performance)
**Estimated Test Execution Time**: 20-25 minutes (automated suite)

---

## Epic UAT Scenario Decomposition

**Note**: No epic-level UAT acceptance plan was found in the repository. This feature contributes to the broader E07 Enhancements epic goals of improving AI agent orchestration and workflow automation.

### Epic UAT Scenarios This Feature Addresses

**UAT Scenario 1: AI Agent Context Discovery**
- **Epic Goal**: AI agents should receive all necessary context to complete tasks without manual prompting
- **This Feature's Contribution**: Provides `{related_docs}` and `{related_tasks}` placeholders that automatically inject document paths and task dependencies into agent instructions
- **Validation Required**: Agent receives complete document list and task dependencies in initial instruction

**UAT Scenario 2: Template Reusability**
- **Epic Goal**: Workflow templates should work across tasks without hardcoded values
- **This Feature's Contribution**: Dynamic placeholder substitution eliminates need for hardcoded document paths
- **Validation Required**: Single template works across multiple tasks with different documents

**UAT Scenario 3: Document Relationship Utilization**
- **Epic Goal**: Document relationships (E07-F05) should improve agent workflows
- **This Feature's Contribution**: Bridges document relationship system with orchestrator actions
- **Validation Required**: Documents linked via `shark related-docs add` appear in agent instructions

---

## Component Test Strategy

### Component 1: Helper Functions (Pure Logic)

**Files Under Test**:
- `internal/config/template_helpers.go` (new functions)

**Test Approach**: Unit tests with no database dependencies

**Functions to Test**:
1. `formatDocPathsAsCSV([]*models.Document) string`
2. `extractRelatedTasksFromContext(*string) string`

**Coverage Target**: 100% for pure functions

**Test Method**: Table-driven tests with edge cases

**Why This Approach**: Pure functions are deterministic and easy to test exhaustively without mocks

---

### Component 2: Placeholder Factory Extensions

**Files Under Test**:
- `internal/config/template_helpers.go` (`*PlaceholdersWithRelated` functions)

**Test Approach**: Unit tests with mocked DocumentRepository

**Functions to Test**:
1. `TaskPlaceholdersWithRelated(task, docRepo, ctx) map[string]string`
2. `FeaturePlaceholdersWithRelated(feature, docRepo, ctx) map[string]string`
3. `EpicPlaceholdersWithRelated(epic, docRepo, ctx) map[string]string`

**Coverage Target**: ≥85% (some error branches hard to trigger)

**Test Method**: Mock repository interface with controllable return values

**Why This Approach**: Isolates placeholder logic from database; enables testing error scenarios without database corruption

---

### Component 3: Repository Integration

**Files Under Test**:
- `internal/repository/task_repository.go` (`GetOrchestratorActionForTask`)
- `internal/repository/feature_repository.go` (if orchestrator action exists)
- `internal/repository/epic_repository.go` (if orchestrator action exists)

**Test Approach**: Integration tests with real test database

**Coverage Target**: ≥80% (repository layer)

**Test Method**: Setup test data (tasks + documents + context), execute repository methods, verify populated templates

**Why This Approach**: Validates actual database queries and document joins work correctly with SQLite

---

### Component 4: End-to-End Orchestrator Actions

**Files Under Test**:
- `internal/cli/commands/task.go` (`shark task get --json`)
- Template population flow (repository → placeholder → template → JSON output)

**Test Approach**: CLI integration tests with test database

**Coverage Target**: ≥70% (CLI layer often has display logic not covered)

**Test Method**: Execute CLI commands, parse JSON output, verify orchestrator action structure

**Why This Approach**: Validates complete user-facing feature works as intended

---

## API Contract Test Cases

### Contract 1: TaskPlaceholdersWithRelated

**Signature**:
```go
func TaskPlaceholdersWithRelated(
    task *models.Task,
    docRepo *repository.DocumentRepository,
    ctx context.Context,
) map[string]string
```

**Test Cases**:

| Test ID | Input | Expected Output | Edge Case |
|---------|-------|-----------------|-----------|
| TC-PH-01 | Task with 2 docs, 2 related tasks | Map with `related_docs="docs/a.md,docs/b.md"` and `related_tasks="E01-F01,E02-F01"` | Happy path |
| TC-PH-02 | Task with 0 docs, 0 related tasks | Map with `related_docs=""` and `related_tasks=""` | Empty data |
| TC-PH-03 | Task with docs, nil context_data | Map with `related_docs="..."` and `related_tasks=""` | Partial data |
| TC-PH-04 | Task with docs, malformed JSON context | Map with `related_docs="..."` and `related_tasks=""` (warning logged) | Invalid JSON |
| TC-PH-05 | Nil task pointer | Map with all empty values (no panic) | Nil input |
| TC-PH-06 | DocRepo.ListForTask returns error | Map with `related_docs=""` (warning logged) | Query failure |

**Validation**: All test cases must pass; errors must NOT propagate (graceful degradation)

---

### Contract 2: formatDocPathsAsCSV

**Signature**:
```go
func formatDocPathsAsCSV(docs []*models.Document) string
```

**Test Cases**:

| Test ID | Input | Expected Output | Edge Case |
|---------|-------|-----------------|-----------|
| TC-FMT-01 | `nil` slice | `""` | Nil input |
| TC-FMT-02 | Empty slice `[]` | `""` | Empty input |
| TC-FMT-03 | Single doc `[{FilePath: "docs/a.md"}]` | `"docs/a.md"` | Single item |
| TC-FMT-04 | Three docs | `"docs/a.md,docs/b.md,docs/c.md"` | Multiple items |
| TC-FMT-05 | Docs with spaces in path | `"docs/My Doc.md,docs/Other.md"` | Special characters |
| TC-FMT-06 | 50 documents | CSV with 50 comma-separated paths (no truncation) | Large list |

**Validation**: CSV format must be consistent; no escaping needed for paths with spaces

---

### Contract 3: extractRelatedTasksFromContext

**Signature**:
```go
func extractRelatedTasksFromContext(contextData *string) string
```

**Test Cases**:

| Test ID | Input | Expected Output | Edge Case |
|---------|-------|-----------------|-----------|
| TC-CTX-01 | `nil` context | `""` | Nil input |
| TC-CTX-02 | Empty string `""` | `""` | Empty input |
| TC-CTX-03 | Valid JSON with 2 tasks: `{"related_tasks":["E01-F01","E02-F01"]}` | `"E01-F01,E02-F01"` | Happy path |
| TC-CTX-04 | Valid JSON with empty array: `{"related_tasks":[]}` | `""` | Empty array |
| TC-CTX-05 | Valid JSON without related_tasks field | `""` | Missing field |
| TC-CTX-06 | Malformed JSON: `"{invalid}"` | `""` (warning logged) | Invalid JSON |
| TC-CTX-07 | Valid JSON with null related_tasks: `{"related_tasks":null}` | `""` | Null field |

**Validation**: Function must never return error; all failures result in empty string with warning log

---

### Contract 4: FeaturePlaceholdersWithRelated

**Signature**:
```go
func FeaturePlaceholdersWithRelated(
    feature *models.Feature,
    docRepo *repository.DocumentRepository,
    featureRelRepo *repository.FeatureRelationshipRepository,
    ctx context.Context,
) map[string]string
```

**Test Cases**:

| Test ID | Input | Expected Output | Edge Case |
|---------|-------|-----------------|-----------|
| TC-FPH-01 | Feature with 2 docs, 3 related features | Map with `related_docs="docs/a.md,docs/b.md"` and `related_features="E07-F05,E07-F21,E10-F05"` | Happy path |
| TC-FPH-02 | Feature with 0 docs, 0 related features | Map with `related_docs=""` and `related_features=""` | Empty data |
| TC-FPH-03 | Nil feature pointer | Map with all empty values (no panic) | Nil input |
| TC-FPH-04 | FeatureRelRepo.ListRelatedFeatures returns error | Map with `related_features=""` (warning logged) | Query failure |
| TC-FPH-05 | Cross-epic feature relationships | Map includes features from different epics (e.g., `"E01-F01,E02-F05"`) | Cross-epic |

**Validation**: Graceful degradation on errors; cross-epic relationships supported

---

### Contract 5: EpicPlaceholdersWithRelated

**Signature**:
```go
func EpicPlaceholdersWithRelated(
    epic *models.Epic,
    docRepo *repository.DocumentRepository,
    epicRelRepo *repository.EpicRelationshipRepository,
    ctx context.Context,
) map[string]string
```

**Test Cases**:

| Test ID | Input | Expected Output | Edge Case |
|---------|-------|-----------------|-----------|
| TC-EPH-01 | Epic with 2 docs, 2 related epics | Map with `related_docs="docs/a.md,docs/b.md"` and `related_epics="E01,E05"` | Happy path |
| TC-EPH-02 | Epic with 0 docs, 0 related epics | Map with `related_docs=""` and `related_epics=""` | Empty data |
| TC-EPH-03 | Nil epic pointer | Map with all empty values (no panic) | Nil input |
| TC-EPH-04 | EpicRelRepo.ListRelatedEpics returns error | Map with `related_epics=""` (warning logged) | Query failure |

**Validation**: Graceful degradation on errors; relationship table integration

---

### Contract 6: formatFeatureKeysAsCSV

**Signature**:
```go
func formatFeatureKeysAsCSV(featureKeys []string) string
```

**Test Cases**:

| Test ID | Input | Expected Output | Edge Case |
|---------|-------|-----------------|-----------|
| TC-FKCSV-01 | `nil` slice | `""` | Nil input |
| TC-FKCSV-02 | Empty slice `[]` | `""` | Empty input |
| TC-FKCSV-03 | Single feature key `["E07-F05"]` | `"E07-F05"` | Single item |
| TC-FKCSV-04 | Three feature keys | `"E07-F05,E07-F21,E10-F05"` | Multiple items |
| TC-FKCSV-05 | Cross-epic keys | `"E01-F01,E07-F05,E10-F05"` (maintains order) | Cross-epic |

**Validation**: CSV format consistent with task/doc formatting; no escaping needed

---

### Contract 7: formatEpicKeysAsCSV

**Signature**:
```go
func formatEpicKeysAsCSV(epicKeys []string) string
```

**Test Cases**:

| Test ID | Input | Expected Output | Edge Case |
|---------|-------|-----------------|-----------|
| TC-EKCSV-01 | `nil` slice | `""` | Nil input |
| TC-EKCSV-02 | Empty slice `[]` | `""` | Empty input |
| TC-EKCSV-03 | Single epic key `["E01"]` | `"E01"` | Single item |
| TC-EKCSV-04 | Three epic keys | `"E01,E05,E07"` | Multiple items |

**Validation**: CSV format consistent with other placeholders

---

## Integration Test Scenarios

### Scenario 1: Task with Related Documents

**Setup**:
1. Create test task: `TEST-E07-F29-001`
2. Create 2 test documents: `docs/spec.md`, `docs/design.md`
3. Link documents to task via `task_documents` junction table
4. Set instruction template: `"Implement {id}. Read: {related_docs}"`

**Execution**:
```go
action, err := taskRepo.GetOrchestratorActionForTask(ctx, task)
```

**Expected Result**:
- `action.InstructionTemplate` contains: `"Implement TEST-E07-F29-001. Read: docs/spec.md,docs/design.md"`
- No errors returned
- Both document paths present in CSV format

**User Goal Trace**: AI agent receives explicit document paths to read before implementation (Persona 1 goal)

---

### Scenario 2: Task with Related Tasks in Context

**Setup**:
1. Create test task: `TEST-E07-F29-002`
2. Set `context_data` JSON: `{"related_tasks":["E07-F05-001","E10-F05-002"]}`
3. Set instruction template: `"Work on {id}. Dependencies: {related_tasks}"`

**Execution**:
```go
action, err := taskRepo.GetOrchestratorActionForTask(ctx, task)
```

**Expected Result**:
- `action.InstructionTemplate` contains: `"Work on TEST-E07-F29-002. Dependencies: E07-F05-001,E10-F05-002"`
- Both task keys present in CSV format

**User Goal Trace**: AI agent understands task dependencies without manual discovery (Persona 1 goal)

---

### Scenario 3: Task with No Related Data

**Setup**:
1. Create test task: `TEST-E07-F29-003`
2. No documents linked (empty junction table)
3. No context_data (NULL or empty)
4. Set instruction template: `"Task {id}. Docs: {related_docs}. Tasks: {related_tasks}"`

**Execution**:
```go
action, err := taskRepo.GetOrchestratorActionForTask(ctx, task)
```

**Expected Result**:
- `action.InstructionTemplate` contains: `"Task TEST-E07-F29-003. Docs: . Tasks: "`
- Empty strings replace placeholders (no error)
- Template remains valid

**User Goal Trace**: Template works even without related data (Template Author goal - reusability)

---

### Scenario 4: Feature-Level Template with Docs

**Setup**:
1. Create test feature: `TEST-E07-F29`
2. Create 3 test documents (architecture, PRD, test plan)
3. Link documents to feature via `feature_documents` junction table
4. Set feature instruction template: `"Review feature {id} documentation: {related_docs}"`

**Execution**:
```go
placeholders := config.FeaturePlaceholdersWithRelated(feature, docRepo, ctx)
instruction := action.PopulateTemplate(placeholders)
```

**Expected Result**:
- `instruction` contains all 3 document paths in CSV format
- `{related_tasks}` is empty string (not supported for features)

**User Goal Trace**: Feature-level orchestration includes documentation context (extends beyond task-level)

---

### Scenario 5: Epic-Level Template with Docs

**Setup**:
1. Create test epic: `TEST-E07`
2. Create 2 test documents (epic summary, roadmap)
3. Link documents to epic via `epic_documents` junction table
4. Set epic instruction template: `"Plan epic {id}. Reference: {related_docs}"`

**Execution**:
```go
placeholders := config.EpicPlaceholdersWithRelated(epic, docRepo, ctx)
instruction := action.PopulateTemplate(placeholders)
```

**Expected Result**:
- `instruction` contains both document paths in CSV format
- Epic-level placeholders work identically to task/feature

**User Goal Trace**: Consistent placeholder system across all entity types (Template Author goal)

---

### Scenario 6: Malformed Context Data Handling

**Setup**:
1. Create test task: `TEST-E07-F29-004`
2. Set `context_data` to invalid JSON: `"{invalid json"`
3. Set instruction template: `"Related: {related_tasks}"`

**Execution**:
```go
action, err := taskRepo.GetOrchestratorActionForTask(ctx, task)
```

**Expected Result**:
- `action.InstructionTemplate` contains: `"Related: "` (empty string)
- Warning logged: `"Failed to parse context_data for task TEST-E07-F29-004"`
- Template population succeeds (no error returned)

**User Goal Trace**: Robustness - feature doesn't break orchestrator on bad data (Error Story 2)

---

### Scenario 7: Document Query Failure Recovery

**Setup**:
1. Mock DocumentRepository to return database error
2. Create task with valid data
3. Set instruction template with `{related_docs}`

**Execution**:
```go
mockDocRepo := &MockDocumentRepository{
    ListForTaskFunc: func(ctx, id) ([]*models.Document, error) {
        return nil, fmt.Errorf("database connection lost")
    },
}
placeholders := config.TaskPlaceholdersWithRelated(task, mockDocRepo, ctx)
```

**Expected Result**:
- `placeholders["related_docs"]` is `""` (empty string)
- Warning logged
- Placeholder population completes successfully

**User Goal Trace**: System resilience - transient database errors don't break agent instructions

---

### Scenario 8: Large Document List (50+ Docs)

**Setup**:
1. Create test task
2. Link 55 documents to task
3. Set instruction template with `{related_docs}`

**Execution**:
```go
action, err := taskRepo.GetOrchestratorActionForTask(ctx, task)
```

**Expected Result**:
- `action.InstructionTemplate` contains all 55 document paths (no truncation)
- CSV format maintained
- Performance < 50ms (REQ-NF-001)

**User Goal Trace**: Feature handles real-world scenarios with many related documents (Error Story 3)

---

### Scenario 9: Backward Compatibility - Old Templates

**Setup**:
1. Use existing instruction template WITHOUT new placeholders: `"Implement {id}: {title}"`
2. Create task with related documents

**Execution**:
```go
action, err := taskRepo.GetOrchestratorActionForTask(ctx, task)
```

**Expected Result**:
- Template works unchanged: `"Implement E07-F29-001: Test Task"`
- No `{related_docs}` or `{related_tasks}` in output (not in template)
- Existing placeholders still work

**User Goal Trace**: Feature is backward compatible (REQ-F-007, Story 3)

---

### Scenario 10: Document Link/Unlink Dynamic Lookup

**Setup**:
1. Create task, link 2 documents
2. Get orchestrator action (verify 2 docs in output)
3. Unlink 1 document via `shark related-docs remove`
4. Get orchestrator action again

**Execution**:
```go
// After unlinking one document
action, err := taskRepo.GetOrchestratorActionForTask(ctx, task)
```

**Expected Result**:
- Second call shows only 1 document (dynamic lookup, not cached)
- File paths reflect current state

**User Goal Trace**: Changes to document links immediately reflected in agent instructions (Story 4)

---

### Scenario 11: Feature with Related Features (Database Table)

**Setup**:
1. Create test feature: `TEST-E07-F29`
2. Create related features: `TEST-E07-F05`, `TEST-E07-F21`, `TEST-E10-F05` (cross-epic)
3. Add relationships via `feature_relationships` table:
   - `TEST-E07-F29` depends_on `TEST-E07-F05`
   - `TEST-E07-F29` related_to `TEST-E07-F21`
   - `TEST-E07-F29` references `TEST-E10-F05` (cross-epic)
4. Set feature instruction template: `"Work on {id}. Related features: {related_features}"`

**Execution**:
```go
placeholders := config.FeaturePlaceholdersWithRelated(feature, featureRelRepo, ctx)
instruction := action.PopulateTemplate(placeholders)
```

**Expected Result**:
- `instruction` contains: `"Work on TEST-E07-F29. Related features: TEST-E07-F05,TEST-E07-F21,TEST-E10-F05"`
- All 3 related feature keys in CSV format
- Cross-epic relationship included (E10-F05)
- Both outbound and inbound relationships included

**User Goal Trace**: AI agent receives cross-feature dependency context automatically (Story 4a)

---

### Scenario 12: Feature with No Related Features

**Setup**:
1. Create test feature: `TEST-E07-F30`
2. No entries in `feature_relationships` table for this feature
3. Set instruction template: `"Feature {id}. Dependencies: {related_features}"`

**Execution**:
```go
placeholders := config.FeaturePlaceholdersWithRelated(feature, featureRelRepo, ctx)
instruction := action.PopulateTemplate(placeholders)
```

**Expected Result**:
- `instruction` contains: `"Feature TEST-E07-F30. Dependencies: "`
- Empty string for `{related_features}` (no error)
- Template remains valid

**User Goal Trace**: Graceful handling when no relationships exist (Story 4a error handling)

---

### Scenario 13: Epic-Level Template Including Related Features

**Setup**:
1. Create test epic: `TEST-E07`
2. Create 2 features in epic: `TEST-E07-F01`, `TEST-E07-F02`
3. Link `TEST-E07-F01` relates_to `TEST-E07-F02` via `feature_relationships`
4. Set epic instruction template: `"Epic {id} features: {related_features}"`

**Execution**:
```go
// Epic placeholder function fetches relationships across all features in epic
placeholders := config.EpicPlaceholdersWithRelated(epic, featureRelRepo, ctx)
instruction := action.PopulateTemplate(placeholders)
```

**Expected Result**:
- `instruction` shows related features within epic
- Epic-level templates can reference feature relationships

**User Goal Trace**: Epic-level orchestration includes feature dependency context (Story 4a, AC-4a.4)

---

### Scenario 14: Epic with Related Epics (Database Table)

**Setup**:
1. Create test epic: `TEST-E07`
2. Create related epics: `TEST-E01`, `TEST-E05`
3. Add relationships via `epic_relationships` table:
   - `TEST-E07` depends_on `TEST-E01` (prerequisite epic)
   - `TEST-E07` related_to `TEST-E05` (parallel work)
4. Set epic instruction template: `"Work on epic {id}. Prerequisite epics: {related_epics}"`

**Execution**:
```go
placeholders := config.EpicPlaceholdersWithRelated(epic, epicRelRepo, ctx)
instruction := action.PopulateTemplate(placeholders)
```

**Expected Result**:
- `instruction` contains: `"Work on epic TEST-E07. Prerequisite epics: TEST-E01,TEST-E05"`
- Both epic keys in CSV format
- Both outbound and inbound relationships included

**User Goal Trace**: AI agent understands epic-level dependencies and sequencing (Story 4b)

---

### Scenario 15: Epic with No Related Epics

**Setup**:
1. Create test epic: `TEST-E08`
2. No entries in `epic_relationships` table for this epic
3. Set instruction template: `"Epic {id}. Related: {related_epics}"`

**Execution**:
```go
placeholders := config.EpicPlaceholdersWithRelated(epic, epicRelRepo, ctx)
instruction := action.PopulateTemplate(placeholders)
```

**Expected Result**:
- `instruction` contains: `"Epic TEST-E08. Related: "`
- Empty string for `{related_epics}` (no error)
- Template remains valid

**User Goal Trace**: Graceful handling when no epic relationships exist (Story 4b error handling)

---

## Acceptance Criteria Test Matrix

### Story 1: AI Agent Receives Related Docs

| AC # | Acceptance Criterion | Test Case(s) | Pass Criteria |
|------|---------------------|--------------|---------------|
| AC-1.1 | `{related_docs}` placeholder populates with comma-separated file paths | TC-PH-01, Scenario 1 | CSV format: `"path1,path2,path3"` |
| AC-1.2 | Documents fetched from junction tables | Scenario 1, 4, 5 | Database JOIN query returns correct documents |
| AC-1.3 | Empty string returned if no related documents | TC-PH-02, Scenario 3 | `related_docs=""` when no links |
| AC-1.4 | File paths are project-relative | Scenario 1 | Paths like `docs/spec.md` (not absolute) |

**Coverage**: 4 test cases cover all AC; automated + integration tests

---

### Story 2: AI Agent Receives Related Tasks

| AC # | Acceptance Criterion | Test Case(s) | Pass Criteria |
|------|---------------------|--------------|---------------|
| AC-2.1 | `{related_tasks}` placeholder populates with comma-separated task keys | TC-CTX-03, Scenario 2 | CSV format: `"key1,key2,key3"` |
| AC-2.2 | Task keys fetched from `ContextData.RelatedTasks` JSON field | Scenario 2 | JSON parse extracts array |
| AC-2.3 | Empty string returned if no context_data or RelatedTasks is empty | TC-CTX-01, TC-CTX-04, Scenario 3 | `related_tasks=""` when missing |
| AC-2.4 | Graceful handling of malformed JSON | TC-CTX-06, Scenario 6 | Warning logged, empty string returned |

**Coverage**: 6 test cases cover all AC; error scenarios included

---

### Story 3: Template Author Uses New Placeholders

| AC # | Acceptance Criterion | Test Case(s) | Pass Criteria |
|------|---------------------|--------------|---------------|
| AC-3.1 | Template example works: `"Work on {id}. Related docs: {related_docs}. Related tasks: {related_tasks}"` | Scenario 1, 2 | Template populated correctly |
| AC-3.2 | Placeholders replaced at template population time | All scenarios | Orchestrator action generation time |
| AC-3.3 | Works for task-level, feature-level, and epic-level templates | Scenario 1, 4, 5 | All entity types supported |
| AC-3.4 | Backward compatible: existing templates without new placeholders continue working | Scenario 9 | Old templates unchanged |

**Coverage**: Full entity type coverage; backward compatibility verified

---

### Story 4: Developer Links Documents

| AC # | Acceptance Criterion | Test Case(s) | Pass Criteria |
|------|---------------------|--------------|---------------|
| AC-4.1 | Linking document to task/feature/epic makes it available for `{related_docs}` | Scenario 1, 4, 5 | Linked docs appear in next action |
| AC-4.2 | Unlinking document removes it from next placeholder population | Scenario 10 | Dynamic lookup reflects changes |
| AC-4.3 | No manual template updates required | Scenario 10 | Template unchanged, output varies |
| AC-4.4 | Document paths reflect current file locations | Scenario 1 | Paths from `documents.file_path` column |

**Coverage**: Dynamic behavior validated; integration tests with real database

---

### Story 4a: AI Agent Receives Related Features

| AC # | Acceptance Criterion | Test Case(s) | Pass Criteria |
|------|---------------------|--------------|---------------|
| AC-4a.1 | `{related_features}` placeholder populates with comma-separated feature keys | Scenario 11 (new) | CSV format: `"E07-F05,E07-F21,E10-F05"` |
| AC-4a.2 | Feature keys fetched from `feature_relationships` table | Scenario 11 | Database query returns relationship records |
| AC-4a.3 | Empty string returned if feature has no relationships | Scenario 12 (new) | `related_features=""` when no relationships |
| AC-4a.4 | Works for both feature-level and epic-level instruction templates | Scenario 11, 13 (new) | Both entity types supported |
| AC-4a.5 | Supports cross-epic feature relationships | Scenario 11 | E01-F01 relating to E02-F05 works |
| AC-4a.6 | Includes both outbound and inbound relationships | Scenario 11 | Bidirectional lookup |

**Coverage**: Full relationship table integration; cross-epic validation

---

### Story 4b: AI Agent Receives Related Epics

| AC # | Acceptance Criterion | Test Case(s) | Pass Criteria |
|------|---------------------|--------------|---------------|
| AC-4b.1 | `{related_epics}` placeholder populates with comma-separated epic keys | Scenario 14 (new) | CSV format: `"E01,E05,E07"` |
| AC-4b.2 | Epic keys fetched from `epic_relationships` table | Scenario 14 | Database query returns relationship records |
| AC-4b.3 | Empty string returned if epic has no relationships | Scenario 15 (new) | `related_epics=""` when no relationships |
| AC-4b.4 | Works for epic-level instruction templates | Scenario 14 | Epic template placeholder population |
| AC-4b.5 | Includes both outbound and inbound relationships | Scenario 14 | Bidirectional lookup |

**Coverage**: Epic relationship table integration; dependency modeling

---

### Error Story 1: Empty Placeholders

| AC # | Acceptance Criterion | Test Case(s) | Pass Criteria |
|------|---------------------|--------------|---------------|
| AC-E1.1 | `{related_docs}` replaced with empty string when no documents linked | TC-PH-02, Scenario 3 | `""` not error |
| AC-E1.2 | `{related_tasks}` replaced with empty string when no context data | TC-CTX-01, Scenario 3 | `""` not error |
| AC-E1.3 | Template remains valid: `"Docs: {related_docs}"` becomes `"Docs: "` | Scenario 3 | No formatting errors |
| AC-E1.4 | No error logs for empty relational data | Scenario 3 | Normal case, no warnings |

**Coverage**: All edge cases for missing data; no false error logging

---

### Error Story 2: Malformed Context Data

| AC # | Acceptance Criterion | Test Case(s) | Pass Criteria |
|------|---------------------|--------------|---------------|
| AC-E2.1 | Malformed JSON in `task.context_data` logs warning but doesn't fail | TC-CTX-06, Scenario 6 | Empty string + warning |
| AC-E2.2 | `{related_tasks}` returns empty string on parse failure | TC-CTX-06, Scenario 6 | `""` not panic |
| AC-E2.3 | Error logged to system logs: `"Failed to parse context_data for task E07-F29-001"` | Scenario 6 | Log message verified |
| AC-E2.4 | Template population completes successfully despite parse error | Scenario 6 | No error returned |

**Coverage**: JSON error handling; logging verified in tests

---

### Error Story 3: Large Document Lists

| AC # | Acceptance Criterion | Test Case(s) | Pass Criteria |
|------|---------------------|--------------|---------------|
| AC-E3.1 | All related documents included (no arbitrary truncation in MVP) | Scenario 8 | All 55 docs in CSV |
| AC-E3.2 | Performance acceptable (< 50ms overhead per REQ-NF-001) | Scenario 8 + benchmark | Latency measured |

**Coverage**: Performance test validates large lists; no truncation in MVP

---

## Performance and Security Test Approach

### Performance Tests

**Test ID**: PERF-01
**Description**: Placeholder population latency with 10 related docs
**Measurement**: Response time for `GetOrchestratorActionForTask()`
**Target**: < 50ms overhead compared to baseline (without related docs)
**Method**:
```go
func BenchmarkGetOrchestratorActionWithRelatedDocs(b *testing.B) {
    // Setup task with 10 linked documents
    // Measure: time for GetOrchestratorActionForTask()
    // Compare to baseline (0 documents)
}
```
**Pass Criteria**: Overhead ≤ 50ms (REQ-NF-001)

---

**Test ID**: PERF-02
**Description**: Document query efficiency (no N+1 pattern)
**Measurement**: Database query count for orchestrator action
**Target**: Single query per entity (one `ListForTask()` call)
**Method**: Enable SQLite query logging, count queries during action generation
**Pass Criteria**: Exactly 1 query to `task_documents` junction table

---

**Test ID**: PERF-03
**Description**: JSON parsing overhead
**Measurement**: Time to parse `context_data` JSON field
**Target**: < 1ms for typical context data (< 2KB)
**Method**:
```go
func BenchmarkExtractRelatedTasksFromContext(b *testing.B) {
    contextJSON := `{"related_tasks":["E07-F05-001","E10-F05-002"], ...}`
    // Measure: extractRelatedTasksFromContext()
}
```
**Pass Criteria**: < 1ms average parse time

---

**Test ID**: PERF-04
**Description**: Large document list handling (50+ docs)
**Measurement**: End-to-end latency with 55 linked documents
**Target**: < 100ms total (50ms baseline + 50ms overhead)
**Method**: Integration test with 55 documents linked to task
**Pass Criteria**: `GetOrchestratorActionForTask()` completes in < 100ms

---

**Test ID**: PERF-05
**Description**: Feature relationship query efficiency
**Measurement**: Database query count for feature-level orchestrator action with relationships
**Target**: Single query per entity (one `ListRelatedFeatures()` call)
**Method**: Enable SQLite query logging, count queries during feature action generation
**Pass Criteria**: Exactly 1 query to `feature_relationships` table per feature

---

**Test ID**: PERF-06
**Description**: Epic relationship query efficiency
**Measurement**: Database query count for epic-level orchestrator action with relationships
**Target**: Single query per entity (one `ListRelatedEpics()` call)
**Method**: Enable SQLite query logging, count queries during epic action generation
**Pass Criteria**: Exactly 1 query to `epic_relationships` table per epic

---

**Test ID**: PERF-07
**Description**: Large relationship list handling
**Measurement**: End-to-end latency with 20 related features
**Target**: < 75ms overhead compared to baseline (no relationships)
**Method**: Integration test with feature linked to 20 other features
**Pass Criteria**: Feature orchestrator action completes in acceptable time

---

### Security Tests

**Test ID**: SEC-01
**Description**: No new security surface area introduced
**Verification**:
- Document file paths already exposed via `shark related-docs list` (no new exposure)
- Task keys already exposed via `shark task list` (no new exposure)
- Context data is internal structured data (not user input)
**Pass Criteria**: No new attack vectors identified

---

**Test ID**: SEC-02
**Description**: SQL injection resistance
**Verification**:
- All document queries use parameterized statements
- No string concatenation in SQL queries
- Repository methods use `?` placeholders
**Method**: Review query construction in `DocumentRepository.ListForTask()`
**Pass Criteria**: All queries use prepared statements

---

**Test ID**: SEC-03
**Description**: JSON injection resistance (context_data)
**Verification**:
- Context data JSON parsed with standard library (`encoding/json`)
- No `eval()` or dynamic code execution
- Malformed JSON results in empty string (not code execution)
**Method**: Test with malicious JSON payloads
**Pass Criteria**: No code execution; graceful degradation

---

## Test Execution Plan

### Phase 1: Unit Tests (Day 1, ~2 hours)

**Execution Order**:
1. `TestFormatDocPathsAsCSV` (6 test cases)
2. `TestExtractRelatedTasksFromContext` (7 test cases)
3. `TestFormatFeatureKeysAsCSV` (5 test cases)
4. `TestFormatEpicKeysAsCSV` (4 test cases)
5. `TestTaskPlaceholdersWithRelated` (6 test cases using mocks)
6. `TestFeaturePlaceholdersWithRelated` (5 test cases using mocks)
7. `TestEpicPlaceholdersWithRelated` (4 test cases using mocks)

**Command**:
```bash
go test -v ./internal/config -run "TestFormat|TestExtract|TestTaskPlaceholders|TestFeaturePlaceholders|TestEpicPlaceholders"
```

**Exit Criteria**: All 37 unit tests pass (increased from 25), ≥85% coverage for new functions

---

### Phase 2: Integration Tests (Day 2, ~3 hours)

**Execution Order**:
1. Setup test database with schema (including new relationship tables)
2. `TestTaskRepository_GetOrchestratorActionWithRelatedDocs` (Scenario 1)
3. `TestTaskRepository_WithRelatedTasks` (Scenario 2)
4. `TestTaskRepository_NoRelatedData` (Scenario 3)
5. `TestFeatureRepository_WithRelatedDocs` (Scenario 4)
6. `TestEpicRepository_WithRelatedDocs` (Scenario 5)
7. `TestMalformedContextData` (Scenario 6)
8. `TestDocumentQueryFailure` (Scenario 7)
9. `TestLargeDocumentList` (Scenario 8)
10. `TestBackwardCompatibility` (Scenario 9)
11. `TestDynamicDocumentLookup` (Scenario 10)
12. `TestFeatureRepository_WithRelatedFeatures` (Scenario 11)
13. `TestFeatureRepository_NoRelatedFeatures` (Scenario 12)
14. `TestEpicRepository_WithFeatureRelationships` (Scenario 13)
15. `TestEpicRepository_WithRelatedEpics` (Scenario 14)
16. `TestEpicRepository_NoRelatedEpics` (Scenario 15)

**Command**:
```bash
go test -v ./internal/repository -run "OrchestratorAction|RelatedFeatures|RelatedEpics"
```

**Exit Criteria**: All 15 integration tests pass (increased from 10), database cleanup verified

---

### Phase 3: Performance Tests (Day 2, ~1 hour)

**Execution Order**:
1. `BenchmarkGetOrchestratorActionWithRelatedDocs` (PERF-01)
2. Query count verification (PERF-02)
3. `BenchmarkExtractRelatedTasksFromContext` (PERF-03)
4. Large document list test (PERF-04)
5. Feature relationship query efficiency (PERF-05)
6. Epic relationship query efficiency (PERF-06)
7. Large relationship list test (PERF-07)

**Command**:
```bash
go test -bench=. ./internal/config ./internal/repository -benchmem
```

**Exit Criteria**: All performance targets met (< 50ms overhead for docs, < 75ms overhead for relationships, < 1ms JSON parse)

---

### Phase 4: End-to-End Acceptance Tests (Day 3, ~2 hours)

**Execution Order**:
1. Manual CLI test: `shark task get --json` with related docs/tasks (Scenario 1+2 combined)
2. Verify JSON output structure contains `orchestrator_action.instruction_template` with populated values
3. Test backward compatibility with old templates (Scenario 9)
4. Test document link/unlink dynamic behavior (Scenario 10)

**Command**:
```bash
# Manual CLI testing
shark task create E07 F29 "Test Task" --agent=developer
shark related-docs add "Spec" docs/spec.md --task=E07-F29-001
shark related-docs add "Design" docs/design.md --task=E07-F29-001
shark task context set E07-F29-001 --field=related_tasks --value='["E07-F05-001"]'
shark task get E07-F29-001 --json | jq '.orchestrator_action.instruction_template'
```

**Exit Criteria**: Manual verification passes; JSON output contains expected placeholders

---

### Phase 5: Security & Regression (Day 3, ~1 hour)

**Execution Order**:
1. Security verification tests (SEC-01, SEC-02, SEC-03)
2. Full regression suite (ensure existing features unaffected)

**Command**:
```bash
make test  # Full test suite
make lint  # Static analysis
```

**Exit Criteria**: All existing tests pass, no new security warnings

---

## Test Data Management

### Test Database Setup

**Database**: `internal/test/shark-tasks-test.db`

**Cleanup Strategy**:
```go
func setupTestData(t *testing.T) (cleanup func()) {
    db := test.GetTestDB()

    // Clean before test
    _, _ = db.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")
    _, _ = db.ExecContext(ctx, "DELETE FROM documents WHERE title LIKE 'TEST%'")

    return func() {
        // Clean after test
        _, _ = db.ExecContext(ctx, "DELETE FROM tasks WHERE key LIKE 'TEST-%'")
        _, _ = db.ExecContext(ctx, "DELETE FROM documents WHERE title LIKE 'TEST%'")
    }
}
```

**Test Data Conventions**:
- Task keys: `TEST-E07-F29-###`
- Document titles: `TEST <description>`
- Document paths: `docs/test/<filename>`

---

### Test Fixtures

**Fixture 1: Task with Related Docs**
```go
task := &models.Task{
    Key:       "TEST-E07-F29-001",
    Title:     "Test Task with Docs",
    Status:    "ready_for_development",
    EpicID:    epicID,
    FeatureID: featureID,
}
doc1 := &models.Document{Title: "TEST Spec", FilePath: "docs/test/spec.md"}
doc2 := &models.Document{Title: "TEST Design", FilePath: "docs/test/design.md"}
// Link via task_documents junction table
```

**Fixture 2: Task with Related Tasks**
```go
contextJSON := `{"related_tasks":["E07-F05-001","E10-F05-002"]}`
task := &models.Task{
    Key:         "TEST-E07-F29-002",
    Title:       "Test Task with Related Tasks",
    ContextData: &contextJSON,
}
```

**Fixture 3: Task with Malformed Context**
```go
invalidJSON := `{invalid json`
task := &models.Task{
    Key:         "TEST-E07-F29-003",
    ContextData: &invalidJSON,
}
```

---

## Test Traceability Matrix

| Requirement ID | PRD Acceptance Criterion | Test Case(s) | Status |
|----------------|-------------------------|--------------|--------|
| REQ-F-001 | Related Docs Placeholder | TC-PH-01, TC-FPH-01, TC-EPH-01, Scenario 1, 4, 5 | Not Started |
| REQ-F-002 | Related Tasks Placeholder | TC-CTX-03, Scenario 2 | Not Started |
| REQ-F-002a | Related Features Placeholder | TC-FPH-01, TC-FKCSV-01-05, Scenario 11, 12, 13 | Not Started |
| REQ-F-002b | Related Epics Placeholder | TC-EPH-01, TC-EKCSV-01-04, Scenario 14, 15 | Not Started |
| REQ-F-003 | Entity-Level Placeholder Support | Scenario 1, 4, 5, 11, 13, 14 | Not Started |
| REQ-F-004 | Repository Integration | Scenario 1-15 | Not Started |
| REQ-F-005 | Feature Relationships Table | Scenario 11, 12 (database schema) | Not Started |
| REQ-F-006 | Epic Relationships Table | Scenario 14, 15 (database schema) | Not Started |
| REQ-F-007 | Feature Relationship Repository | Scenario 11, 12, 13 | Not Started |
| REQ-F-008 | Epic Relationship Repository | Scenario 14, 15 | Not Started |
| REQ-F-009 | Document Path Formatting | TC-FMT-01 through TC-FMT-06 | Not Started |
| REQ-F-010 | Context Data Parsing | TC-CTX-01 through TC-CTX-07 | Not Started |
| REQ-F-011 | Existing Template Compatibility | Scenario 9 | Not Started |
| REQ-NF-001 | Placeholder Population Latency < 50ms | PERF-01, PERF-04, PERF-05, PERF-06, PERF-07 | Not Started |
| REQ-NF-002 | Query Efficiency (no N+1) | PERF-02, PERF-05, PERF-06 | Not Started |
| REQ-NF-003 | Test Coverage ≥ 80% | All unit + integration tests | Not Started |
| REQ-NF-004 | Error Handling (graceful degradation) | Scenario 6, 7, 12, 15, TC-CTX-06, TC-PH-06, TC-FPH-04, TC-EPH-04 | Not Started |

---

## Exit Criteria

### Feature Test Completion

- [ ] All 47 test cases executed and passing
- [ ] Code coverage ≥ 80% for new code (`internal/config/template_helpers.go`)
- [ ] Performance tests meet targets (< 50ms overhead, < 1ms JSON parse)
- [ ] Security verification complete (no new vulnerabilities)
- [ ] Manual end-to-end CLI tests pass
- [ ] All PRD acceptance criteria validated

### Quality Gates

- [ ] `make fmt && make lint && make test` passes with zero warnings/errors
- [ ] No regression in existing test suite (all previous tests still pass)
- [ ] Integration tests with real database demonstrate correct behavior
- [ ] Backward compatibility verified (old templates work unchanged)

### Documentation

- [ ] Test results documented in test execution log
- [ ] Any test failures documented with reproduction steps
- [ ] Performance benchmarks recorded
- [ ] Known limitations documented (e.g., no truncation for 50+ docs in MVP)

---

## Risk Mitigation

### Risk 1: N+1 Query Performance

**Risk**: Fetching documents for multiple tasks causes performance degradation
**Likelihood**: Medium
**Impact**: Medium
**Mitigation**: PERF-02 test verifies single query per entity; batch loading deferred to future enhancement
**Test Coverage**: Performance benchmarks with 10+ docs

---

### Risk 2: Context JSON Parse Failures

**Risk**: Malformed context data breaks orchestrator actions
**Likelihood**: Low
**Impact**: High
**Mitigation**: Graceful degradation (empty string + warning log); never fail placeholder population
**Test Coverage**: TC-CTX-06, Scenario 6 validate malformed JSON handling

---

### Risk 3: Large Document Lists

**Risk**: Tasks with 50+ documents cause slow responses or template bloat
**Likelihood**: Low
**Impact**: Medium
**Mitigation**: PERF-04 test validates 55 docs still performs adequately; documentation warns users
**Test Coverage**: Scenario 8 tests large list; performance measured

---

### Risk 4: Breaking Changes to Existing Templates

**Risk**: New placeholder system breaks existing instruction templates
**Likelihood**: Very Low
**Impact**: High
**Mitigation**: Additive-only changes; existing placeholders unchanged; backward compatibility test
**Test Coverage**: Scenario 9 validates old templates work unchanged

---

### Risk 5: Relationship Table Query Performance

**Risk**: Joining relationship tables for features/epics causes slow response times
**Likelihood**: Low
**Impact**: Medium
**Mitigation**: PERF-05, PERF-06 tests verify single query per entity; indexes on foreign keys
**Test Coverage**: Performance benchmarks with multiple relationships; PERF-07 tests 20 relationships

---

### Risk 6: Circular Relationship Dependencies

**Risk**: Features/epics with circular relationship chains cause infinite loops
**Likelihood**: Very Low
**Impact**: High
**Mitigation**: Relationship queries don't traverse graph (single-level lookup only); no recursive queries in MVP
**Test Coverage**: Integration tests verify single-level relationship lookup (non-recursive)

---

## Summary

### Test Plan Overview

| Test Type | Test Cases | Coverage Target | Duration |
|-----------|-----------|-----------------|----------|
| Unit Tests | 37 | 100% (pure functions) | 2.5 hours |
| Integration Tests | 15 | 80% (repository layer) | 4 hours |
| Performance Tests | 7 | N/A (benchmarks) | 1.5 hours |
| Acceptance Tests | 5 (manual) | 70% (CLI layer) | 2 hours |
| Security Tests | 3 | N/A (verification) | 1 hour |
| **Total** | **67** | **≥80% overall** | **11 hours** |

### Key Testing Principles

1. **User Goal Traceability**: Every test traces back to persona needs (AI agent context discovery, template reusability, cross-entity relationships)
2. **Graceful Degradation**: All error scenarios return empty strings, never fail orchestrator actions
3. **Real Database Integration**: Repository tests use actual SQLite database including new relationship tables to validate queries
4. **Performance Validation**: Benchmarks ensure < 50ms overhead for docs, < 75ms for relationships (REQ-NF-001)
5. **Backward Compatibility**: Existing templates work unchanged (REQ-F-007, REQ-F-011)
6. **Relationship Table Validation**: Feature and epic relationship tables tested for bidirectional lookups and cross-epic support

### Developer Handoff

This test plan is actionable for TDD:
- **Phase 1 tests** can be written before implementation (pure functions)
- **Phase 2 tests** define integration contracts (database queries)
- **Phase 3 tests** validate performance requirements (benchmarks)
- **Phase 4 tests** ensure user-facing feature works end-to-end

Developers can read test cases in this plan and write tests first, then implement to make tests pass.

---

**Document Version**: 1.1
**Last Updated**: 2026-02-13
**Changes in v1.1**: Added test coverage for Story 4a (related features) and Story 4b (related epics), including 5 new integration scenarios, 4 new API contracts, and 3 new performance tests
**Ready for Implementation**: Yes
**Next Step**: Advance feature status with `shark feature next-status E07-F29`

