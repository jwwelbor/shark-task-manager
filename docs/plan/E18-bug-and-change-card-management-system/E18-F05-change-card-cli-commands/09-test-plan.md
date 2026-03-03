# Test Plan: E18-F05 -- Change-Card CLI Commands

**Feature**: E18-F05 - Change-Card CLI Commands
**Epic**: E18 - Bug and Change-Card Management System
**Complexity Tier**: STANDARD
**Test Plan Type**: Focused Test Plan (AC Test Matrix + Component Test Strategy + Integration Scenarios)
**Date**: 2026-03-03
**Author**: QA Agent

---

## Executive Summary

This test plan validates the `shark change` Cobra command group -- 11 subcommands implementing change-card CRUD, approval, notes, and context management. Every subcommand must follow the thin-wrapper pattern (parse, call service, format output) with zero business logic in the CLI layer.

**User Goal This Serves**: Developers need to propose enhancements as change-cards with a single CLI command. Product owners need to approve or decline change-cards without learning new CLI patterns. AI agents need JSON output for machine consumption.

**Critical Success Criteria**:
- All 11 subcommands execute successfully (create, get, list, update, delete, approve, note add, notes, context set, context get, context clear)
- `shark change approve C001` advances from `proposed` to `approved` and rejects all other statuses with exit code 3
- `--json` output on every command returns valid, parseable JSON
- `--link` validation rejects non-existent entities atomically (no partial records)
- All CLI tests use mocked ChangeCardService (zero real database access)
- `shark view C001` resolves the change-card markdown file path

**UAT Traceability**: This feature implements UAT scenarios J2-HP (change-card happy path), J2-ALT-A (declined), J2-ERR-1 (re-approve), J2-ERR-2 (decline from wrong status), and partially J3-UNIFIED (via entity-specific commands; unified dispatch is F06).

---

## 1. Acceptance Criteria Test Matrix

### Story 1: Create a Change-Card

**User Goal**: A developer proposes a small enhancement without interrupting workflow.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC1.1** | `shark change create "Title" --link=E07` creates a change-card in `proposed` status | Call `ChangeCardService.CreateChangeCard` with title and link key | `"Add dark mode toggle" --link=E07` | Change-card returned with auto-generated key (C001), status `proposed`, link to E07 | Empty title ("") should be rejected by service validation |
| **AC1.2** | Command outputs key and file path on success | Verify CLI success message format | Valid create input | `Created change-card C001: Add dark mode toggle` + file path | Verify file path matches `docs/changes/C001.md` pattern |
| **AC1.3** | `--json` outputs full change-card object | Create with `--json` flag | Valid create input with `--json` | JSON object with keys: `key`, `title`, `status`, `linked_entity_type`, `linked_entity_key`, `file_path`, `created_at`, `updated_at` | Verify JSON is valid (parseable by `jq`) |
| **AC1.4** | `--link` to non-existent entity returns clear error | Create with `--link=E99` (non-existent) | `"Title" --link=E99` | Error message: entity E99 not found. No change-card created. | Verify no database record created and no orphaned markdown file |
| **AC1.5** | Omitting `--link` creates change-card without link | Create without `--link` flag | `"Title"` (no link) | Change-card created with null/empty link fields | Verify JSON output shows null linked entity fields |

---

### Story 2: Retrieve Change-Card Details

**User Goal**: A developer or product owner reviews a change-card's current state and linked context.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC2.1** | `shark change get C001` displays full details | Call `ChangeCardService.GetChangeCard` with key | `C001` | Formatted output: key, title, status, linked entity, file path, timestamps | Verify all fields present in display |
| **AC2.2** | `--json` outputs full JSON object | Get with `--json` flag | `C001 --json` | Valid JSON with all change-card fields | Field names match model struct tags |
| **AC2.3** | `--field status` outputs only the status value | Get with `--field` flag | `C001 --field status` | Raw value: `proposed` (no formatting) | Test with multiple field names: `key`, `title`, `status`, `linked_entity_key` |
| **AC2.4** | Non-existent key returns exit code 1 | Get non-existent change-card | `C999` | Error: "change-card not found: C999", exit code 1 | Key format is valid but entity does not exist |

---

### Story 3: List Change-Cards with Filters

**User Goal**: A product owner reviews pending proposals and tracks change-card throughput.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC3.1** | `shark change list` displays non-completed change-cards in table | List with no filters | (none) | Table with columns: Key, Title, Status, Linked Entity, Created At | Verify completed change-cards excluded by default |
| **AC3.2** | `--status=proposed` filters to proposed only | List with status filter | `--status=proposed` | Only change-cards in `proposed` status returned | Verify filtering happens in service, not CLI |
| **AC3.3** | `--link=E07` filters by linked entity | List with link filter | `--link=E07` | Change-cards linked to E07 or entities under E07 | Test with feature-level link: `--link=E07-F01` |
| **AC3.4** | `--json` outputs JSON array | List with JSON flag | `--json` | Valid JSON array of change-card objects | Empty results should return `[]`, not null |
| **AC3.5** | No matching results shows empty table | List with filter matching nothing | `--status=proposed` (when no proposed cards exist) | Empty table with headers, no error | Verify exit code is 0, not 1 |
| **AC3.6** | `--status=invalid_status` returns error listing valid statuses | List with invalid status | `--status=bogus` | Error listing valid statuses: proposed, approved, in_progress, completed, declined | Validate error message is helpful |

---

### Story 4: Update Change-Card Fields

**User Goal**: A developer corrects a title or adds missing information.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC4.1** | `shark change update C001 --title="Revised"` updates title | Update with new title | `C001 --title="Revised title"` | Success message with updated key | Verify title is actually changed (mock assertion) |
| **AC4.2** | `--json` outputs updated change-card | Update with JSON flag | `C001 --title="New" --json` | JSON object with updated title | Verify `updated_at` reflects the update |
| **AC4.3** | Non-existent key returns exit code 1 | Update non-existent key | `C999 --title="New"` | Error: change-card not found, exit code 1 | |

---

### Story 5: Delete Change-Card

**User Goal**: A developer removes proposals no longer relevant.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC5.1** | `shark change delete C001 --force` deletes immediately | Delete with force flag | `C001 --force` | Success message: "Deleted change-card C001" | Verify service.DeleteChangeCard called |
| **AC5.2** | Delete without `--force` prompts for confirmation | Delete without force | `C001` (no --force) | Confirmation prompt displayed | Cannot fully test interactive prompt in unit tests; test that GetChangeCard is called before prompt |
| **AC5.3** | Deleting removes DB record and markdown file | Verify atomic deletion | `C001 --force` | Both DB record and file deleted | Service handles atomicity; CLI just calls service |
| **AC5.4** | Non-existent key returns exit code 1 | Delete non-existent key | `C999 --force` | Error: change-card not found, exit code 1 | |
| **AC5.5** | Deleted key is never reused | Create after delete | Delete C001, then create new | New card gets C002, not C001 | Key auto-increment is service/DB responsibility |

---

### Story 6: Approve Change-Card

**User Goal**: A product owner approves a proposal with a single command, with clear audit trail.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC6.1** | `shark change approve C001` advances from `proposed` to `approved` | Approve a proposed change-card | `C001` (status: proposed) | Success: "Approved change-card C001", status changes to `approved` | Verify ChangeCardService.ApproveChangeCard called |
| **AC6.2** | Non-proposed status returns exit code 3 | Approve already-approved card | `C001` (status: approved) | Error: "cannot approve change-card C001: current status is 'approved', approval requires status 'proposed'", exit 3 | Test with each non-proposed status: approved, in_progress, completed, declined |
| **AC6.3** | Non-existent key returns exit code 1 | Approve non-existent card | `C999` | Error: change-card not found, exit code 1 | |
| **AC6.4** | `--json` outputs approved card | Approve with JSON flag | `C001 --json` (if supported) | JSON object with status `approved` | Verify JSON output is valid |
| **AC6.5** | Status history records transition | Verify audit trail | Approve C001 | Status history contains `proposed -> approved` with timestamp | This is a service-layer concern but UAT verifies end-to-end via `shark status history C001` |

---

### Story 7: Notes on Change-Cards (Should-Have)

**User Goal**: A product owner documents approval rationale and review comments.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC7.1** | `shark change note add C001 --type=decision "content"` adds note | Add a typed note | `C001 --type=decision "Approved: aligns with Q2"` | Success: "Note added to change-card C001" | Verify NoteService.AddNote called with entity_type="change" |
| **AC7.2** | `shark change notes C001` lists notes chronologically | List notes | `C001` | Notes displayed in chronological order | Empty notes list shows informational message, not error |
| **AC7.3** | All existing note types work | Add notes with different types | `--type=comment`, `--type=blocker`, `--type=question` | Each type accepted | Verify invalid type is rejected |

---

### Story 8: Context Fields on Change-Cards (Should-Have)

**User Goal**: A developer stores structured metadata like effort estimate and target area.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC8.1** | `shark change context set C001 --field effort --value "small"` sets field | Set context field | `C001 --field effort --value "small"` | Success message | Verify ContextService called with entity_type="change" |
| **AC8.2** | `shark change context get C001` returns all fields | Get all context | `C001` | JSON object with context fields | Empty context returns `{}`, not error |
| **AC8.3** | `shark change context clear C001 --field effort` removes field | Clear specific field | `C001 --field effort` | Success message | Clearing non-existent field should not error |

---

### Story 9: View Change-Card Markdown (Should-Have)

**User Goal**: A developer reads the full description and justification.

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **AC9.1** | `shark view C001` renders the markdown file | View change-card file | `C001` | Markdown file contents rendered to terminal | Verify view service resolves C### key format |
| **AC9.2** | Missing markdown file returns error | View with missing file | `C001` (file deleted from disk) | Error: file not found | Verify error is clear and mentions file path |

---

### Error Stories

| AC ID | Acceptance Criterion | Test Case | Input | Expected Output | Edge Cases |
|-------|---------------------|-----------|-------|-----------------|------------|
| **ACE1.1** | Invalid link key format returns clear error | Create with invalid link format | `"Title" --link=INVALID` | Error: "invalid link key format: INVALID. Expected E##, E##-F##, or E##-F##-###" | No partial record created |
| **ACE2.1** | Approve completed card shows current status | Approve already-completed card | `C001` (status: completed) | Error includes current status "completed" and requirement for "proposed" | Exit code 3 |
| **ACE3.1** | List with invalid status shows valid values | List with bad status filter | `--status=invalid_status` | Error listing valid statuses | Exit code is non-zero |

---

## 2. Component Test Strategy

### Primary Component: `internal/cli/commands/change.go`

**Test File**: `internal/cli/commands/change_test.go`

**Test Approach**: All CLI tests use **mocked ChangeCardService**. Zero real database access. This follows the testing architecture rule: only repository tests use real database.

#### Mock Interface

```go
type MockChangeCardService struct {
    CreateFunc  func(ctx context.Context, input services.CreateChangeCardInput) (*models.ChangeCard, error)
    GetFunc     func(ctx context.Context, key string) (*models.ChangeCard, error)
    ListFunc    func(ctx context.Context, filters services.ChangeCardFilters) ([]*models.ChangeCard, error)
    UpdateFunc  func(ctx context.Context, key string, updates services.ChangeCardUpdates) (*models.ChangeCard, error)
    DeleteFunc  func(ctx context.Context, key string) error
    ApproveFunc func(ctx context.Context, key string) (*models.ChangeCard, error)
}
```

#### Test Cases per Command

| Command | Happy Path | Not Found (exit 1) | Invalid State (exit 3) | JSON Output | Edge Cases |
|---------|-----------|-------------------|----------------------|-------------|------------|
| `create` | Title + link | N/A | N/A | Full JSON object | No link (null fields), invalid link format |
| `get` | Existing key | C999 | N/A | Full JSON, `--field` extraction | Case-insensitive key |
| `list` | No filters | N/A | N/A | JSON array, empty `[]` | Status filter, link filter, invalid status |
| `update` | Title change | C999 | N/A | Updated JSON | No flags provided |
| `delete` | With --force | C999 | N/A | Deletion JSON | Without --force (confirmation flow) |
| `approve` | Proposed card | C999 | Approved/completed/in_progress | Approved JSON | Each non-proposed status produces exit 3 |
| `note add` | Valid note | C999 | N/A | Success message | Invalid note type |
| `notes` | Has notes | C999 | N/A | JSON list | Empty notes list |
| `context set` | Set field | C999 | N/A | Success message | |
| `context get` | Has context | C999 | N/A | JSON object | Empty context |
| `context clear` | Clear field | C999 | N/A | Success message | Clear non-existent field |

#### Exit Code Coverage

| Exit Code | Meaning | Test Commands |
|-----------|---------|---------------|
| 0 | Success | All commands (happy path) |
| 1 | Not found | get, update, delete, approve, note add, notes, context set/get/clear |
| 2 | Service/DB error | All commands (simulate service error) |
| 3 | Invalid state | approve (non-proposed status) |

#### Architectural Validation Tests

These tests verify F05-NFR-003 (thin wrapper architecture):

| Test | What It Validates | How |
|------|-------------------|-----|
| No repository imports | CLI commands do not call repositories directly | `grep -c "repository\.New" internal/cli/commands/change.go` returns 0 |
| No business logic | Handlers only parse, call, format | Code review: no status checks, no filtering loops, no sort.Slice |
| Service accessor exists | `GetChangeCardService()` is wired | Call accessor, verify non-nil return |
| Global flags inherited | `--json`, `--field`, `--no-color`, `--verbose` available | Verify flags on parent command |

---

### Secondary Component: View Command Extension

**Files Modified**: `internal/view/service.go`, `internal/cli/service_accessors.go` or `services_global.go`

**Test Cases**:

| Test | Input | Expected | Validates |
|------|-------|----------|-----------|
| C### key detected as change-card | `isChangeCardKey("C001")` | true | Key detection regex |
| C###-slug key detected | `isChangeCardKey("C001-add-dark-mode")` | true | Slugged key support |
| Non-change key rejected | `isChangeCardKey("E07")` | false | No false positives |
| View resolves file path | `shark view C001` | Renders markdown from `card.FilePath` | End-to-end view resolution |
| View with missing file | `shark view C001` (file missing) | Error: file not found | Error handling |

---

## 3. Integration Scenarios

### INT-1: Change-Card Lifecycle Through CLI

**What**: Full change-card lifecycle from creation to completion using only `shark change` commands.

**Traces to UAT**: J2-HP (Change-Card Happy Path)

| Step | Command | Expected | Validates |
|------|---------|----------|-----------|
| 1 | `shark change create "Add keyboard shortcuts" --link=E17` | C001 created, status `proposed` | Create + link validation |
| 2 | `shark change get C001` | All fields displayed, status `proposed` | Get retrieval |
| 3 | `shark change list --status=proposed` | C001 appears | List with status filter |
| 4 | `shark change approve C001` | Status changes to `approved` | Approve workflow |
| 5 | `shark change get C001 --field status` | `approved` | Field extraction |
| 6 | `shark change note add C001 --type=decision "Approved for Q2"` | Note added | Notes delegation to NoteService |
| 7 | `shark change notes C001` | Shows decision note | Note retrieval |

**Pass Criteria**: All 7 steps execute without error. Status transitions are correct. Notes are persisted and retrievable.

---

### INT-2: Change-Card Decline Flow

**What**: Product owner declines a change-card with rationale.

**Traces to UAT**: J2-ALT-A (Change-Card Declined)

| Step | Command | Expected | Validates |
|------|---------|----------|-----------|
| 1 | `shark change create "Refactor logging" --link=E07` | C002 created, status `proposed` | Create |
| 2 | `shark status set C002 declined` | Status changes to `declined` | Direct status set (via status command, not change command -- this is F06 scope, but `shark change` users can use `shark status set`) |
| 3 | `shark change note add C002 --type=decision "Not aligned with Q2 priorities"` | Decision note recorded | Notes with decline rationale |
| 4 | `shark change get C002` | Status shows `declined`, notes accessible | Get reflects terminal state |

**Note**: Step 2 uses `shark status set` which is F06 unified dispatch scope. For F05-only testing, the service-level test validates that the ChangeCardService supports the `declined` transition. CLI integration testing of `shark status set C002 declined` is deferred to F06.

---

### INT-3: Invalid Operations Rejected

**What**: Error handling across multiple invalid scenarios.

**Traces to UAT**: J2-ERR-1, J2-ERR-2, J1-ERR-1 (error handling patterns)

| Step | Command | Expected | Validates |
|------|---------|----------|-----------|
| 1 | `shark change approve C001` (status: `approved`) | Error: already approved, exit 3 | Re-approve rejection |
| 2 | `shark change approve C999` | Error: not found, exit 1 | Non-existent key |
| 3 | `shark change create "Title" --link=E99` | Error: E99 not found, no card created | Link validation atomicity |
| 4 | `shark change get C999` | Error: not found, exit 1 | Get non-existent |
| 5 | `shark change delete C999 --force` | Error: not found, exit 1 | Delete non-existent |
| 6 | `shark change list --status=bogus` | Error listing valid statuses | Invalid filter value |

**Pass Criteria**: All 6 steps return appropriate error messages and exit codes. No partial state or orphaned records.

---

### INT-4: JSON Output Consistency

**What**: All commands produce valid, consistent JSON output for AI agent and CI/CD consumption.

**Traces to UAT**: J3-UNIFIED Steps 3-4 (machine-parseable output)

| Step | Command | Expected JSON Structure | Validates |
|------|---------|------------------------|-----------|
| 1 | `shark change create "Title" --json` | `{"key":"C001","title":"...","status":"proposed",...}` | Create JSON |
| 2 | `shark change get C001 --json` | Same structure as create | Get JSON consistency |
| 3 | `shark change list --json` | `[{"key":"C001",...},...]` | List JSON array |
| 4 | `shark change get C001 --field key` | `C001` (raw value) | Field extraction |
| 5 | `shark change approve C001 --json` | `{"key":"C001","status":"approved",...}` | Approve JSON |
| 6 | `shark change update C001 --title="New" --json` | Updated object | Update JSON |

**Pass Criteria**: All JSON output is valid (parseable), consistent field naming across commands, and matches the `models.ChangeCard` struct serialization.

---

### INT-5: NoteService and ContextService Delegation

**What**: Notes and context commands correctly delegate to existing services with `entity_type="change"`.

**Traces to UAT**: CE-4 Steps 4, 6 (pattern consistency with existing entities)

| Step | Command | Service Call Expected | Validates |
|------|---------|----------------------|-----------|
| 1 | `shark change note add C001 --type=decision "content"` | `NoteService.AddNote(ctx, "change", card.ID, "decision", "content")` | entity_type parameter |
| 2 | `shark change notes C001` | `NoteService.ListNotes(ctx, "change", card.ID)` | entity_type parameter |
| 3 | `shark change context set C001 --field effort --value "small"` | `ContextService.SetField(ctx, "change", card.ID, "effort", "small")` | entity_type parameter |
| 4 | `shark change context get C001` | `ContextService.GetContext(ctx, "change", card.ID)` | entity_type parameter |
| 5 | `shark change context clear C001 --field effort` | `ContextService.ClearField(ctx, "change", card.ID, "effort")` | entity_type parameter |

**Pass Criteria**: All delegations pass `entity_type="change"`. Mock assertions verify the exact parameters passed to NoteService and ContextService.

---

### INT-6: Parallel Consistency with F04 (Bug CLI)

**What**: F05 command structure, naming, and patterns are structurally parallel to F04.

**Validation Method**: Code review checklist (not runtime test).

| Check | F04 (Bug CLI) | F05 (Change-Card CLI) | Pass? |
|-------|---------------|----------------------|-------|
| File name | `bug.go` | `change.go` | Match pattern |
| Parent command Use | `"bug"` | `"change"` | Match pattern |
| GroupID | `"advanced"` | `"advanced"` | Must match |
| Service accessor | `GetBugService()` | `GetChangeCardService()` | Match pattern |
| CRUD subcommands | create, get, list, update, delete | create, get, list, update, delete | Must match |
| Domain command | `triage` | `approve` | Both have entity-specific domain commands |
| Note delegation | NoteService entity_type="bug" | NoteService entity_type="change" | Match pattern |
| Context delegation | ContextService entity_type="bug" | ContextService entity_type="change" | Match pattern |
| View extension | B### key detection | C### key detection | Match pattern |
| Test file | `bug_test.go` | `change_test.go` | Match pattern |
| Test approach | Mocked BugService | Mocked ChangeCardService | Must match |

**Pass Criteria**: All 11 structural checks match. Deviations must be documented and justified.

---

## 4. Requirements Traceability

| Epic Requirement | Feature Requirement | Test Coverage | Priority |
|-----------------|---------------------|---------------|----------|
| REQ-F-007 (Change-Card Creation) | F05-REQ-002 | AC1.1-AC1.5, INT-1 Step 1 | Must-Have |
| REQ-F-008 (Change-Card Linking) | F05-REQ-002 (`--link`) | AC1.1, AC1.4, AC1.5, ACE1.1, INT-3 Step 3 | Must-Have |
| REQ-F-009 (Change-Card Workflow) | Via ChangeCardService | AC6.1-AC6.5, INT-1 Steps 4-5, INT-2 | Must-Have |
| REQ-F-010 (Approval Command) | F05-REQ-007 | AC6.1-AC6.5, INT-1 Step 4, INT-3 Steps 1-2 | Must-Have |
| REQ-F-011 (Change-Card CRUD) | F05-REQ-002 to F05-REQ-006 | AC1-AC5 (all), INT-1, INT-3 | Must-Have |
| REQ-F-017 (Notes and Context) | F05-REQ-008, F05-REQ-009 | AC7.1-AC7.3, AC8.1-AC8.3, INT-5 | Should-Have |
| REQ-F-018 (Linked Entity Filter) | F05-REQ-004 (`--link`) | AC3.3, INT-4 | Should-Have |
| REQ-NF-001 (Creation Speed) | F05-NFR-002 | Performance test: `time shark change create "test" --json` < 500ms | Must-Have |
| REQ-NF-005 (CLI Pattern Consistency) | F05-NFR-001, F05-NFR-003 | INT-6 (structural review), Architectural Validation Tests | Must-Have |

---

## 5. UAT Scenario Decomposition

Mapping from epic UAT acceptance plan scenarios to F05 test coverage.

| UAT Scenario | F05 Coverage | F05 Test References | Notes |
|--------------|-------------|---------------------|-------|
| J2-HP (Change-Card Happy Path) | Steps 1-4, 7 covered by F05 entity commands. Steps 5-6 use `shark status advance` (F06 scope). | INT-1 | Steps 5-6 will need `shark status advance C001` via unified dispatch |
| J2-ALT-A (Declined) | Step 2 uses `shark status set` (F06). Steps 1, 3 covered by F05. | INT-2 | Direct status set deferred to F06 integration |
| J2-ERR-1 (Re-approve) | Fully covered by F05 approve command | AC6.2, INT-3 Step 1 | |
| J2-ERR-2 (Decline from wrong status) | Workflow enforcement is service-layer; F05 tests approve exit code 3 | AC6.2, ACE2.1 | `shark status set C001 declined` is F06 scope |
| CE-4 Steps 4, 6 (Pattern Consistency) | Notes and context delegation | INT-5 | Verifies entity_type="change" parameter |

---

## Exit Gate Checklist

| Gate Criterion | Status | Evidence |
|----------------|--------|----------|
| Every acceptance criterion has at least one test case | PASS | AC1.1 through ACE3.1 -- 35 test cases across 9 stories + error stories |
| API contracts tested (service method signatures) | PASS | Mock interface mirrors ChangeCardService contract; INT-5 validates NoteService and ContextService delegation contracts |
| Integration points identified and tested | PASS | INT-1 through INT-6 cover lifecycle, error handling, JSON consistency, service delegation, and F04 structural consistency |
| Test cases are actionable for TDD | PASS | Each test case specifies exact input, expected output, and edge cases. Mock interface defined with function field pattern. |
| Performance requirement has test method | PASS | REQ-NF-001 validated via shell timing of create command (Section 4 traceability) |

---

*This test plan is ready for task generation. All test cases are actionable for TDD implementation.*
