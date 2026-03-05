# E18-F06 Test Plan: Unified CLI Integration and Key Auto-Detection

**Feature**: E18-F06
**Complexity Tier**: STANDARD
**Date**: 2026-03-03
**Author**: QA Agent
**Status**: Complete

---

## 1. Acceptance Criteria Test Matrix

This matrix maps every acceptance criterion from the PRD to concrete test cases with expected results. Tests are grouped by story and ordered by priority.

### Story 1: Unified Get Command (Must-Have)

| Test ID | Acceptance Criterion | Test Steps | Expected Result | Priority |
|---------|---------------------|------------|-----------------|----------|
| T01-01 | `shark get B001` returns bug details | 1. Create bug B001 with title "Login fails" and severity "high". 2. Run `shark get B001`. | Output shows bug title, status, severity, linked entity. Section header reads "Bug: B001". | P1 |
| T01-02 | `shark get B001 --json` returns JSON | 1. Run `shark get B001 --json`. 2. Run `shark bug get B001 --json`. 3. Diff the JSON output structures. | Both commands produce structurally identical JSON (same keys, same types). Zero structural differences. | P1 |
| T01-03 | `shark get B001 --field status` extracts field | Run `shark get B001 --field status`. | Outputs a single string value (e.g., `reported`), no JSON wrapper, no extra whitespace. | P1 |
| T01-04 | `shark get C001` returns change-card details | 1. Create change-card C001 with title "Add dark mode". 2. Run `shark get C001`. | Output shows title, status, linked entity. Section header reads "Change Card: C001". | P1 |
| T01-05 | `shark get C001 --json` returns JSON | Run `shark get C001 --json`. | Valid JSON with title, status, linked entity fields. Matches `shark change get C001 --json` structure. | P1 |
| T01-06 | `shark get C001 --field status` extracts field | Run `shark get C001 --field status`. | Single string value (e.g., `proposed`). | P1 |

### Story 2: Unified Status Commands (Must-Have)

| Test ID | Acceptance Criterion | Test Steps | Expected Result | Priority |
|---------|---------------------|------------|-----------------|----------|
| T02-01 | `shark status advance B001` advances bug | 1. Bug B001 in status `reported`. 2. Run `shark status advance B001`. | Bug status changes to `triaged`. Success message displayed. | P1 |
| T02-02 | `shark status advance C001` advances change-card | 1. Change-card C001 in status `proposed`. 2. Run `shark status advance C001`. | Change-card status changes to `approved`. Success message displayed. | P1 |
| T02-03 | `shark status set B001 wont_fix` sets status | 1. Bug B001 in status `triaged`. 2. Run `shark status set B001 wont_fix`. | Bug status changes to `wont_fix`. | P1 |
| T02-04 | `shark status set C001 declined` sets status | 1. Change-card C001 in status `proposed`. 2. Run `shark status set C001 declined`. | Change-card status changes to `declined`. | P1 |
| T02-05 | `shark status options B001` shows valid next | 1. Bug B001 in status `reported`. 2. Run `shark status options B001`. | Output lists `triaged`, `duplicate`, `wont_fix` (per bug workflow). Does NOT list bug statuses that are not valid transitions. | P1 |
| T02-06 | `shark status options C001` shows valid next | 1. Change-card C001 in status `proposed`. 2. Run `shark status options C001`. | Output lists `approved`, `declined`. | P1 |
| T02-07 | `shark status history B001` shows history | 1. Advance bug B001 through reported -> triaged -> in_fix. 2. Run `shark status history B001`. | Shows all 3 transitions with timestamps and actor info. | P1 |
| T02-08 | `shark status history C001` shows history | 1. Advance C001 from proposed -> approved. 2. Run `shark status history C001`. | Shows transition with timestamp. | P1 |

### Story 3: Search Extension (Must-Have)

| Test ID | Acceptance Criterion | Test Steps | Expected Result | Priority |
|---------|---------------------|------------|-----------------|----------|
| T03-01 | Unfiltered search includes bugs | 1. Create bug B001 ("Login fails") and task ("Login API"). 2. Run `shark search "login"`. | Both B001 and the task appear in results. | P2 |
| T03-02 | `--type=bug` filters to bugs only | 1. Same data as T03-01. 2. Run `shark search "login" --type=bug`. | Only B001 appears. Task is excluded. | P2 |
| T03-03 | `--type=change` filters to change-cards | 1. Create C001 ("Add dark mode"). 2. Run `shark search "dark" --type=change`. | Only C001 appears. | P2 |
| T03-04 | Bug search results include severity | Run `shark search "login" --type=bug --json`. | Each bug result has `key`, `title`, `status`, `severity` fields. | P2 |
| T03-05 | Change-card search results include key/title/status | Run `shark search "dark" --type=change --json`. | Each result has `key`, `title`, `status` fields. | P2 |
| T03-06 | Invalid `--type` shows valid options | Run `shark search "query" --type=invalid`. | Error message lists valid types: epic, feature, task, bug, change. | P2 |

### Story 4: Delete and Update (Must-Have)

| Test ID | Acceptance Criterion | Test Steps | Expected Result | Priority |
|---------|---------------------|------------|-----------------|----------|
| T04-01 | `shark delete B001 --force` deletes bug | 1. Bug B001 exists. 2. Run `shark delete B001 --force`. 3. Run `shark get B001`. | Step 2 succeeds. Step 3 returns "bug not found: B001". | P1 |
| T04-02 | `shark delete C001 --force` deletes change-card | 1. C001 exists. 2. Run `shark delete C001 --force`. 3. Run `shark get C001`. | Step 2 succeeds. Step 3 returns "change card not found: C001". | P1 |
| T04-03 | `shark update B001 --title="New"` updates bug | 1. B001 exists. 2. Run `shark update B001 --title="Updated bug"`. 3. Run `shark get B001`. | Title shows "Updated bug". | P1 |
| T04-04 | `shark update C001 --title="New"` updates change-card | 1. C001 exists. 2. Run `shark update C001 --title="Updated card"`. 3. Run `shark get C001`. | Title shows "Updated card". | P1 |
| T04-05 | Not-found uses correct entity type name | Run `shark delete B999 --force`. | Error contains "bug" and "B999" (not "task" or "unknown entity"). | P2 |

### Story 5: Context and Notes (Must-Have)

| Test ID | Acceptance Criterion | Test Steps | Expected Result | Priority |
|---------|---------------------|------------|-----------------|----------|
| T05-01 | `shark context set B001` sets context | Run `shark context set B001 --field environment --value "Safari 17.2"`. | No error. Context is stored. | P2 |
| T05-02 | `shark context get B001` returns context | After T05-01, run `shark context get B001`. | Returns JSON containing `{"environment": "Safari 17.2"}`. | P2 |
| T05-03 | `shark context clear B001` clears context | After T05-02, run `shark context clear B001`. Then `shark context get B001`. | Context is empty or returns `{}`. | P2 |
| T05-04 | Context commands work for change-cards | Run `shark context set C001 --field effort --value small`, then `shark context get C001`. | Returns `{"effort": "small"}`. | P2 |
| T05-05 | `shark notes B001` lists bug notes | 1. Add a note: `shark bug note add B001 --type=comment --content="test"`. 2. Run `shark notes B001`. | Note with content "test" appears in output. | P2 |
| T05-06 | `shark notes C001` lists change-card notes | 1. Add a note on C001. 2. Run `shark notes C001`. | Note appears in output. | P2 |

### Story 6: View Command (Must-Have)

| Test ID | Acceptance Criterion | Test Steps | Expected Result | Priority |
|---------|---------------------|------------|-----------------|----------|
| T06-01 | `shark view B001` renders bug markdown | 1. B001 exists with file at `docs/bugs/B001.md`. 2. Run `shark view B001`. | Markdown content of B001 file is rendered. | P2 |
| T06-02 | `shark view C001` renders change-card markdown | 1. C001 exists with file at `docs/changes/C001.md`. 2. Run `shark view C001`. | Markdown content of C001 file is rendered. | P2 |
| T06-03 | Missing file produces clear error | 1. Create a bug entry in DB but no markdown file. 2. Run `shark view B002`. | Error message mentions file not found, not an unknown entity error. | P3 |

### Story 7: History Command (Must-Have)

| Test ID | Acceptance Criterion | Test Steps | Expected Result | Priority |
|---------|---------------------|------------|-----------------|----------|
| T07-01 | `shark history B001` shows history | 1. Advance B001 through 3 status changes. 2. Run `shark history B001`. | Shows all transitions with timestamps. | P2 |
| T07-02 | `shark history C001` shows history | 1. Advance C001. 2. Run `shark history C001`. | Shows transitions. | P2 |
| T07-03 | `shark history B001 --json` returns JSON | Run `shark history B001 --json`. | Valid JSON array of status change records. | P2 |

### Story 8: Resume Command (Should-Have)

| Test ID | Acceptance Criterion | Test Steps | Expected Result | Priority |
|---------|---------------------|------------|-----------------|----------|
| T08-01 | `shark resume B001` shows bug context | 1. B001 exists with context data and notes. 2. Run `shark resume B001`. | Output includes status, severity, context data, recent notes, linked entity. | P3 |
| T08-02 | `shark resume C001` shows change-card context | 1. C001 exists with context and notes. 2. Run `shark resume C001`. | Output includes status, context data, recent notes. | P3 |

### Error Stories

| Test ID | Acceptance Criterion | Test Steps | Expected Result | Priority |
|---------|---------------------|------------|-----------------|----------|
| TE-01 | Invalid key `B` (no digits) | Run `shark get B`. | Error mentioning invalid key format with examples including B### and C###. | P2 |
| TE-02 | Invalid key `C` (no digits) | Run `shark get C`. | Error mentioning invalid key format with examples. | P2 |
| TE-03 | Not-found bug identifies entity type | Run `shark get B999`. | Error contains "bug" and "B999". Exit code is 1. | P1 |
| TE-04 | Not-found change-card identifies entity type | Run `shark get C999`. | Error contains "change card" and "C999". Exit code is 1. | P1 |

---

## 2. Component Test Strategy

F06 touches three architectural layers. Each layer has a distinct testing approach.

### 2.1. Key Detection Layer

**Files**: `internal/keys/service.go`, `internal/keys/validation.go`, `internal/cli/commands/helpers.go`, `internal/cli/scope/interpreter.go`

**Testing approach**: Unit tests with table-driven test cases. No database, no mocks needed -- pure function testing.

**Test cases for `keys.Parse()` and `keys.DetectEntityType()`**:

| Input | Expected EntityType | Notes |
|-------|-------------------|-------|
| `B001` | `EntityTypeBug` | Standard bug key |
| `b001` | `EntityTypeBug` | Case-insensitive |
| `B1` | `EntityTypeBug` | Single digit |
| `B42` | `EntityTypeBug` | Two digits |
| `B1000` | `EntityTypeBug` | Four digits |
| `C001` | `EntityTypeChange` | Standard change-card key |
| `c015` | `EntityTypeChange` | Case-insensitive |
| `C1` | `EntityTypeChange` | Single digit |
| `B` | `EntityTypeUnknown` | No digits -- invalid |
| `C` | `EntityTypeUnknown` | No digits -- invalid |
| `B0` | `EntityTypeBug` | Zero is a valid digit sequence |
| `BABC` | `EntityTypeUnknown` | Letters after B -- not a bug key |
| `E07` | `EntityTypeEpic` | Must NOT match bug or change |
| `F01` | `EntityTypeFeature` | Must NOT match bug or change |
| `T-E07-F01-001` | `EntityTypeTask` | Must NOT match bug or change |

**Test for `IsBugKey()` and `IsChangeKey()` validators**: Same input table as above, returning true/false.

**Test for CLI `DetectEntityType()` parity**: Verify that `commands.DetectEntityType("B001")` returns `"bug"` and is consistent with `keys.KeyService.DetectEntityType("B001")` returning `EntityTypeBug`. This guards against the two detection paths diverging (F06-REQ-004).

**Test for `scope.ParseScope()`**: Verify `ParseScope(["B001"])` returns `ScopeBug` and `ParseScope(["C001"])` returns `ScopeChange`.

**Performance benchmark**: Extend `BenchmarkDetectEntityType` with B### and C### cases. Verify less than 5% regression in median detection time (F06-REQ-NF-003).

### 2.2. Dispatch Routing Layer

**Files**: All 37 dispatch points listed in 02-architecture.md Section 3.

**Testing approach**: The 37 dispatch points fall into three categories:

**Category A -- Requires new cases (25 dispatch points that need action)**:

For each dispatch point requiring action (points 1-13, 16-27, 33), write a unit test verifying:
1. The `"bug"` case is handled (calls the correct service/handler, does not fall through to default)
2. The `"change"` case is handled
3. The error message from the default case includes B### and C### in the valid formats list

Tests use mocked services. Example for `runGet()`:

```
Verify runGet dispatches B001 to runBugGet
Verify runGet dispatches C001 to runChangeGet
Verify runGet returns error with B### example for unknown key "X001"
```

**Category B -- Verified N/A (12 dispatch points documented as not applicable)**:

Dispatch points 28-32, 34-37 are filesystem discovery / pattern matching that do not apply to bug/change entities. These are verified by code review, not runtime tests. The test plan records their N/A status here for completeness:

| Points | File | Reason N/A |
|--------|------|------------|
| 28-30 | `config.go` | Filesystem pattern configuration; bugs/changes not filesystem-discovered |
| 31 | `list.go` | Hierarchical list traversal; bugs/changes are flat entities |
| 32 | `notes_search.go` | Auto-handled via ValidEntityTypes map update (point #15) |
| 34-35 | `patterns/` | Filesystem pattern matching; not applicable |
| 36-37 | `reporting/scan_report.go` | Filesystem scan tracking; not applicable |

**Category C -- Model constants (2 dispatch points)**:

Points 14-15 (`models/entity_note.go`) are tested by verifying:
- `models.EntityTypeBug` constant equals `"bug"`
- `models.EntityTypeChange` constant equals `"change"`
- `models.ValidEntityTypes["bug"]` is `true`
- `models.ValidEntityTypes["change"]` is `true`

### 2.3. Service Dispatch Layer

**Files**: `note_service.go`, `context_service.go`, `resume_service.go`, `view_service.go`

**Testing approach**: Service tests with mocked repositories. Each service method that gains bug/change dispatch is tested with:
1. Mock `BugRepository.GetByKey()` returning a bug entity
2. Mock `ChangeCardRepository.GetByKey()` returning a change-card entity
3. Verify the service correctly delegates to the right repository
4. Verify graceful degradation when bug/change repos are nil (no panic, clear error)

Example test structure for `NoteService.GetEntityDetails()`:

```
TestNoteService_GetEntityDetails_Bug:
  Mock: bugRepo.GetByKey("B001") returns Bug{Key: "B001", Title: "Login fails"}
  Call: svc.GetEntityDetails(ctx, EntityTypeBug, "B001")
  Assert: returns NoteEntityDetails with Key="B001", Title="Login fails"

TestNoteService_GetEntityDetails_Change:
  Mock: changeRepo.GetByKey("C001") returns ChangeCard{Key: "C001", Title: "Dark mode"}
  Call: svc.GetEntityDetails(ctx, EntityTypeChange, "C001")
  Assert: returns NoteEntityDetails with Key="C001", Title="Dark mode"

TestNoteService_GetEntityDetails_BugRepoNil:
  Setup: NoteService created with bugRepo=nil
  Call: svc.GetEntityDetails(ctx, EntityTypeBug, "B001")
  Assert: returns clear error (not panic)
```

---

## 3. Integration Scenarios

These scenarios test cross-feature interactions. F06 depends on F02/F03 (services), F04/F05 (commands), and E17 (unified CLI infrastructure).

### 3.1. Full Bug Lifecycle via Unified Commands

**Preconditions**: F01 schema deployed, F02 bug service implemented, F04 bug CLI commands implemented, E17 unified commands in place.

**Steps**:

1. `shark bug create "Login fails on Safari" --severity=high` -- create via entity command
2. `shark get B001` -- verify unified get dispatches to bug handler
3. `shark get B001 --json` -- verify JSON output matches `shark bug get B001 --json`
4. `shark status options B001` -- verify bug workflow options shown
5. `shark status advance B001` -- advance to triaged
6. `shark status advance B001` -- advance to in_fix
7. `shark context set B001 --field browser --value "Safari 17.2"` -- set context
8. `shark context get B001` -- verify context persisted
9. `shark notes B001` -- list notes (empty or initial)
10. `shark status history B001` -- verify 3 transitions recorded
11. `shark history B001` -- verify unified history works same as status history
12. `shark view B001` -- verify markdown file rendered
13. `shark resume B001` -- verify full context displayed

**Pass criteria**: All 13 steps execute without "unknown entity type" errors. Bug data is consistent across entity-specific and unified command paths.

### 3.2. Full Change-Card Lifecycle via Unified Commands

**Preconditions**: F01 schema, F03 change-card service, F05 change-card CLI commands, E17 unified commands.

**Steps**:

1. `shark change create "Add keyboard shortcuts" --link=E17` -- create via entity command
2. `shark get C001 --json` -- verify unified get
3. `shark status options C001` -- verify change-card workflow options
4. `shark status set C001 declined` -- set terminal status via unified command
5. `shark status history C001` -- verify history
6. `shark update C001 --title="Updated title"` -- update via unified command
7. `shark get C001 --field title` -- verify title updated

**Pass criteria**: All steps execute without error. Change-card transitions follow the change-card workflow (not the bug workflow or task workflow).

### 3.3. Cross-Entity Search

**Preconditions**: Bugs B001 ("Login fails"), B002 ("Signup error"), change-card C001 ("Add dark mode"), task E07-F01-001 ("Login API endpoint") all exist.

**Steps**:

1. `shark search "login"` -- should find B001 and E07-F01-001 (B002 does not match)
2. `shark search "login" --type=bug` -- should find only B001
3. `shark search "login" --type=task` -- should find only E07-F01-001
4. `shark search "dark" --type=change` -- should find only C001
5. `shark search "nonexistent"` -- should return empty result set (not an error)
6. `shark search "login" --type=invalid` -- should error with valid type list

**Pass criteria**: Type filtering correctly isolates entity types. No cross-contamination between entity types.

### 3.4. Workflow Isolation Between Entity Types

**Preconditions**: Bug B001 in status `reported`, change-card C001 in status `proposed`, task E07-F01-001 in status `todo`.

**Steps**:

1. `shark status set B001 triaged` -- valid bug transition
2. `shark status set B001 in_progress` -- should FAIL (not a valid bug status; `in_progress` is a task status)
3. `shark status set C001 approved` -- valid change-card transition
4. `shark status set C001 in_fix` -- should FAIL (not a valid change-card status; `in_fix` is a bug status)
5. `shark status options B001` -- should show bug-specific options only (no task or change-card statuses)

**Pass criteria**: Each entity type's workflow is enforced independently. Status sets from other entity types are rejected with clear error messages.

### 3.5. Delete and Update via Unified Commands (Roundtrip)

**Preconditions**: Bug B001 and change-card C001 exist.

**Steps**:

1. `shark update B001 --title="Updated bug title"` -- update bug title
2. `shark get B001 --field title` -- verify output is "Updated bug title"
3. `shark delete B001 --force` -- delete bug
4. `shark get B001` -- verify "bug not found: B001" error, exit code 1
5. `shark update C001 --title="Updated card title"` -- update change-card
6. `shark get C001 --field title` -- verify
7. `shark delete C001 --force` -- delete change-card
8. `shark get C001` -- verify "change card not found: C001" error, exit code 1

**Pass criteria**: CRUD roundtrip works end-to-end via unified commands. Entity type names in error messages are correct ("bug", "change card").

---

## 4. Dispatch Inventory Verification Procedure

This section defines the procedure for verifying F06-REQ-019 (Dispatch Point Inventory) and F06-REQ-NF-004 (Dispatch Exhaustiveness).

### Pre-Implementation Baseline

Before implementation begins, run:

```bash
grep -rn 'case "epic"\|case "feature"\|case "task"\|EntityTypeEpic\|EntityTypeFeature\|EntityTypeTask' internal/ \
  | grep -v '_test.go' \
  | grep -v 'vendor/' \
  > dispatch-inventory-baseline.txt
```

Store the output as the baseline checklist. Each line is a dispatch point.

### Post-Implementation Verification

After all dispatch points are updated, run:

```bash
grep -rn 'case "epic"\|case "feature"\|case "task"' internal/ \
  | grep -v '_test.go' \
  | grep -v 'vendor/' \
  > dispatch-inventory-post.txt
```

Then verify coverage:

```bash
# Every line with case "epic" should have a nearby case "bug" and case "change"
# in the same switch block (or be documented as N/A)
grep -rn 'case "bug"\|case "change"' internal/ \
  | grep -v '_test.go' \
  > dispatch-bug-change-coverage.txt
```

Compare the line counts. The number of files containing `case "bug"` should match the number of files containing `case "epic"` minus the N/A exclusions documented in Section 2.2 Category B.

### Drift Detection

If other features add new dispatch points between the baseline and completion of F06, the diff will surface them:

```bash
diff dispatch-inventory-baseline.txt dispatch-inventory-post.txt
```

New lines in the post-implementation file that are NOT in the original 37-point inventory must be investigated and addressed.

---

## 5. Quality Gates

### Gate 1: Key Detection (must pass before dispatch work begins)

- [ ] All key detection unit tests pass (Section 2.1 table)
- [ ] `keys.Parse()` correctly identifies B### and C### patterns
- [ ] CLI `DetectEntityType()` agrees with `keys.DetectEntityType()` for all inputs
- [ ] Performance benchmark shows less than 5% regression

### Gate 2: Dispatch Completeness (must pass before integration testing)

- [ ] All 25 actionable dispatch points have `"bug"` and `"change"` cases
- [ ] All 12 N/A dispatch points are documented with rationale
- [ ] Default cases in switch statements include B### and C### in error messages
- [ ] `make fmt && make lint && make test` passes

### Gate 3: Integration (must pass before feature approval)

- [ ] Integration scenarios 3.1-3.5 all pass
- [ ] Dispatch inventory verification procedure produces zero missed points
- [ ] JSON output parity verified (unified vs. entity-specific commands)
- [ ] Error messages use correct entity type names ("bug", "change card")

### Gate 4: UAT Alignment (must pass before feature approval)

- [ ] Epic UAT scenario CE-1 (Unified Command Dispatch) steps 1-7 pass
- [ ] Epic UAT scenario J3-UNIFIED steps 1-6 pass
- [ ] No regressions in existing epic/feature/task unified commands

---

## 6. Test Data Requirements

### Minimum Test Entities

| Entity | Key | Title | Status | Notes |
|--------|-----|-------|--------|-------|
| Bug | B001 | Login fails on Safari | reported | Severity: high, linked to E07-F01 |
| Bug | B002 | Signup error on mobile | triaged | Severity: medium |
| Change-card | C001 | Add dark mode toggle | proposed | Linked to E17 |
| Change-card | C002 | Keyboard shortcuts | approved | |
| Epic | E07 | User Management | - | Pre-existing |
| Feature | E07-F01 | Authentication | - | Pre-existing |
| Task | E07-F01-001 | Login API endpoint | todo | Pre-existing |

### Test Data Setup

Test data must be created using shark CLI commands (not direct DB inserts) to validate the full creation pipeline:

```bash
shark bug create "Login fails on Safari" --severity=high --link=E07-F01
shark bug create "Signup error on mobile" --severity=medium
shark change create "Add dark mode toggle" --link=E17
shark change create "Keyboard shortcuts"
```

---

## 7. Risk-Based Test Priorities

Aligned with epic UAT Priority 1 (CE-1: Unified Command Dispatch) and Research Risk 1 (Cross-Cutting Entity Type Changes).

| Priority | Test Area | Rationale |
|----------|-----------|-----------|
| P1 | Key detection (B### and C###) | Foundation -- all dispatch depends on correct detection |
| P1 | Get command dispatch | Most commonly used unified command |
| P1 | Status advance/set dispatch | Core workflow interaction |
| P1 | Not-found error entity type names | User-facing quality, prevents confusion |
| P2 | Delete and update dispatch | CRUD completeness |
| P2 | Context and notes dispatch | Secondary interaction commands |
| P2 | Search type filter | Discovery feature |
| P2 | View and history dispatch | Auxiliary commands |
| P2 | Error messages (invalid key, invalid type) | Edge case handling |
| P3 | Resume dispatch (should-have) | Non-critical convenience command |
| P3 | Performance benchmark | Non-functional, low regression risk |
| P3 | Missing markdown file error | Edge case |

---

*This test plan covers 53 test cases across 3 component layers and 5 integration scenarios. It maps to 19 functional requirements, 4 non-functional requirements, and aligns with epic UAT scenarios CE-1 and J3-UNIFIED.*
