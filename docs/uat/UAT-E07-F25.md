# UAT Test Guide - CLI Command Surface Redesign

**Feature:** E07-F25 - CLI Command Surface Redesign
**Epic:** E07 - Enhancements
**Generated:** 2026-02-08
**Status:** APPROVED

---

## Epic Context

**Epic Goal:** Continuous improvements to the Shark Task Manager CLI and infrastructure.

**This Feature's Role:** Redesigns the CLI command surface to provide verb-first shortcuts for daily workflow operations, unified create/delete dispatchers, and reorganized help text -- all while preserving 100% backward compatibility with existing commands.

**Design Principles:**
- The key IS the entity type -- auto-detect where possible
- Frequency-weighted ergonomics -- daily loop (next/start/done) should be shortest
- Additive, never destructive -- every existing command works forever

---

## Design Intent

**From Feature PRD:**
> A hybrid approach that adds verb-first shortcuts for the daily workflow (next/start/done/block/unblock), unified dispatchers for create/delete, and promotes existing smart dispatchers in documentation -- all while keeping every existing command working exactly as-is.

**Key Design Decisions:**
- Top-level aliases delegate to existing handlers (not duplicated logic)
- Entity type detection from key format enables smart dispatchers
- Help organized into 6 groups: Quick, Core, Entity, Details, Status, Setup
- Zero breaking changes -- purely additive

---

## Feature Acceptance Validation

| # | Acceptance Criteria | Status |
|---|---------------------|--------|
| AC-1 | `shark next` works identically to `shark task next` | [x] PASS |
| AC-2 | `shark start <KEY>` works identically to `shark task start <KEY>` | [x] PASS |
| AC-3 | `shark done <KEY>` works identically to `shark task complete <KEY>` | [x] PASS |
| AC-4 | `shark block <KEY>` works identically to `shark task block <KEY>` | [x] PASS |
| AC-5 | `shark unblock <KEY>` works identically to `shark task unblock <KEY>` | [x] PASS |
| AC-6 | `shark create epic/feature/task` dispatches correctly | [x] PASS |
| AC-7 | `shark delete <KEY>` auto-detects entity type from key format | [x] PASS |
| AC-8 | `DetectEntityType` correctly classifies all key formats including slugs | [x] PASS |
| AC-9 | Help output shows 6 organized command groups | [x] PASS |
| AC-10 | All existing commands continue to work unchanged | [x] PASS |
| AC-11 | Documentation promotes smart dispatchers as primary | [x] PASS |
| AC-12 | All new command tests pass | [x] PASS |

---

## Test Scenarios

### Scenario 1: Top-Level Aliases (T-003)

**Steps:**
1. Run `shark next --help` and verify it shows task next documentation
2. Run `shark start --help` and verify it shows task start documentation
3. Run `shark done --help` and verify it shows task complete documentation
4. Run `shark block --help` and verify it shows task block documentation
5. Run `shark unblock --help` and verify it shows task unblock documentation
6. Verify all aliases are in "Quick Commands" group in `shark --help`

**Success Criteria:**
- [x] All 5 aliases appear in help output
- [x] Each alias shows correct flags matching original command
- [x] Aliases grouped under "Quick Commands"

### Scenario 2: Unified Create Dispatcher (T-004)

**Steps:**
1. Run `shark create --help` and verify 3 subcommands (epic, feature, task)
2. Run `shark create epic --help` and verify it matches `shark epic create --help`
3. Run `shark create feature --help` and verify it matches `shark feature create --help`
4. Run `shark create task --help` and verify it matches `shark task create --help`
5. Verify `create` is in "Core Commands" group

**Success Criteria:**
- [x] Create dispatcher has 3 subcommands
- [x] Each subcommand has correct flags
- [x] Grouped under "Core Commands"

### Scenario 3: Unified Delete Dispatcher (T-005)

**Steps:**
1. Run `shark delete --help` and verify key format documentation
2. Verify `--force` flag is available
3. Verify `delete` is in "Core Commands" group

**Success Criteria:**
- [x] Delete dispatcher accepts KEY argument
- [x] Help shows key format detection rules
- [x] `--force` flag documented

### Scenario 4: Entity Type Detection (T-002)

**Steps:**
1. Verify `DetectEntityType("E07")` returns "epic"
2. Verify `DetectEntityType("E07-F01")` returns "feature"
3. Verify `DetectEntityType("F01")` returns "feature"
4. Verify `DetectEntityType("E07-F01-001")` returns "task"
5. Verify `DetectEntityType("T-E07-F01-001")` returns "task"
6. Verify slugged keys: `DetectEntityType("E07-F01-auth")` returns "feature"
7. Verify slugged keys: `DetectEntityType("E07-enhancements")` returns "epic"
8. Verify invalid: `DetectEntityType("invalid")` returns "unknown"

**Success Criteria:**
- [x] All key formats correctly detected
- [x] Slugged feature keys not misclassified as epic
- [x] 40 unit tests pass

### Scenario 5: Help Organization (T-006)

**Steps:**
1. Run `shark --help` and verify 6 command groups
2. Verify group ordering: Quick > Core > Entity > Details > Status > Setup
3. Verify aliases in "Quick Commands" not "Core Commands"
4. Verify entity commands (epic, feature, task) in "Entity Management"

**Success Criteria:**
- [x] 6 distinct command groups visible
- [x] Commands in correct groups
- [x] Root command Long description documents both command styles

### Scenario 6: Backward Compatibility (T-001, cross-cutting)

**Steps:**
1. Verify `shark task next --help` still works
2. Verify `shark task start --help` still works
3. Verify `shark task complete --help` still works
4. Verify `shark epic create --help` still works
5. Verify `shark feature create --help` still works
6. Verify `shark task create --help` still works
7. Verify `shark get --help` still works (pre-existing dispatcher)
8. Verify `shark list --help` still works (pre-existing dispatcher)

**Success Criteria:**
- [x] All original noun-first commands unchanged
- [x] Pre-existing smart dispatchers unaffected
- [x] No command collisions or ambiguities

### Scenario 7: Test Coverage (T-007)

**Steps:**
1. Run `go test ./internal/cli/commands/... -v -count=1`
2. Verify aliases_test.go tests pass
3. Verify create_test.go tests pass
4. Verify delete_dispatch_test.go tests pass
5. Verify helpers_test.go DetectEntityType tests pass (40 cases)

**Success Criteria:**
- [x] All test files compile and pass
- [x] No regressions in existing tests

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-02-08 |
| Result | APPROVED |
| Epic Alignment | PASS |
| Cross-Feature Integration | PASS |
| Feature AC | 12/12 passed |
| Test Scenarios | 7/7 passed |
| Results File | [results](results/UAT-E07-F25-20260208-results.md) |

**Previous Sessions:** None (first session)
