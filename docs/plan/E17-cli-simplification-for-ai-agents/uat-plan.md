# E17: UAT Acceptance Plan - CLI Simplification for AI Agents

**Date:** 2026-02-25
**Author:** QA Agent
**Epic:** E17 - CLI Simplification for AI Agents
**Status:** in_test_planning

---

## Table of Contents

1. [Test Strategy Overview](#1-test-strategy-overview)
2. [Acceptance Scenarios by User Journey](#2-acceptance-scenarios-by-user-journey)
3. [Success Metrics Validation Plan](#3-success-metrics-validation-plan)
4. [Cross-Epic Integration Test Scenarios](#4-cross-epic-integration-test-scenarios)
5. [Risk-Based Test Priorities](#5-risk-based-test-priorities)
6. [Persona-Based Acceptance Matrix](#6-persona-based-acceptance-matrix)
7. [Non-Functional Requirements Validation](#7-non-functional-requirements-validation)
8. [Phase Gate Criteria](#8-phase-gate-criteria)

---

## 1. Test Strategy Overview

### Scope

This UAT plan covers all 13 features across 3 phases of E17. Testing is organized around:

- **User journey acceptance scenarios** -- end-to-end validation that each journey works as described in user-journeys.md
- **Success metric measurement** -- concrete validation methods for each KPI from success-metrics.md
- **Cross-epic integration** -- E15 (service layer) and E16 (multi-level workflow) interaction verification
- **Risk-targeted scenarios** -- focused tests on the 7 risks identified across the research report, BA review, and tech review
- **Backward compatibility** -- the non-negotiable constraint across all phases

### Test Levels

| Level | Scope | Who Runs | When |
|-------|-------|----------|------|
| Unit tests | Service methods, error types, field extraction logic | Developer during implementation | Per-feature |
| CLI tests | Command parsing, flag handling, output formatting | Developer during implementation | Per-feature |
| Integration tests | End-to-end command execution with real DB | QA after each feature | Feature completion |
| Journey acceptance tests | Full user journey with multiple commands | QA after each phase | Phase gate |
| Metric validation tests | Log analysis, performance benchmarks | QA after Phase 1 deployment | Post-deployment |
| Regression tests | All existing commands produce identical output | QA continuously | Every feature merge |

### Test Data Requirements

- A project with at least 3 epics, 5 features, and 20 tasks in various statuses
- Tasks in advanced workflow profile (19 statuses) to exercise all transitions
- At least 2 blocked tasks with dependencies
- At least 1 feature with all tasks completed
- At least 1 feature with mixed task statuses (for progress calculation testing)

---

## 2. Acceptance Scenarios by User Journey

### Journey 1: AI Agent Daily Task Workflow

**Persona:** DevAgent
**Features Exercised:** F01 (status subcommand), F02 (--field), F03 (structured errors), F04 (SHARK_OUTPUT)

#### Scenario J1-S01: Get Next Task with Field Extraction

**Preconditions:**
- Project initialized with advanced workflow
- At least 1 task in `ready_for_development` status assigned to agent type `developer`
- `SHARK_OUTPUT=json` is set in environment

**Steps:**
1. Run `shark next --agent developer --field key`
2. Capture output

**Expected Outcome:**
- Exit code 0
- Output is a raw task key string (e.g., `E18-F05-003`), no JSON wrapping, no extra whitespace
- No `2>/dev/null` needed -- output is clean

**Pass/Fail:** Output is a single line containing only the task key.

---

#### Scenario J1-S02: Read Task Details with Environment JSON Mode

**Preconditions:**
- `SHARK_OUTPUT=json` is set in environment
- Task key obtained from J1-S01

**Steps:**
1. Run `shark get <task-key>` (no `--json` flag needed)
2. Parse output as JSON

**Expected Outcome:**
- Exit code 0
- Output is valid JSON containing task fields (key, title, status, priority, agent_type)
- JSON format is identical to `shark get <task-key> --json`

**Pass/Fail:** JSON parses successfully and contains expected fields.

---

#### Scenario J1-S03: Start Task via Status Advance

**Preconditions:**
- Task is in `ready_for_development` status
- `SHARK_OUTPUT=json` set

**Steps:**
1. Run `shark status advance <task-key>`
2. Parse JSON response

**Expected Outcome:**
- Exit code 0
- Task status changes to `in_development`
- Response JSON includes the updated entity with new status
- Task history records the transition

**Pass/Fail:** Status is `in_development` and history entry created.

---

#### Scenario J1-S04: Check Current Status via Field Extraction

**Preconditions:**
- Task was advanced in J1-S03

**Steps:**
1. Run `shark get <task-key> --field status`

**Expected Outcome:**
- Exit code 0
- Output is raw string: `in_development`
- No JSON wrapping, no quotes

**Pass/Fail:** Output is exactly `in_development` with trailing newline only.

---

#### Scenario J1-S05: View Valid Transitions

**Preconditions:**
- Task is in `in_development` status

**Steps:**
1. Run `shark status options <task-key> --json`

**Expected Outcome:**
- Exit code 0
- JSON output includes `current_status`, `valid_transitions` array, `phase`, `agent_type`
- `valid_transitions` contains the expected next statuses for `in_development`

**Pass/Fail:** JSON structure matches specification and transitions are correct per workflow config.

---

#### Scenario J1-S06: Complete Task via Status Advance with Notes

**Preconditions:**
- Task is in `in_development` status

**Steps:**
1. Run `shark status advance <task-key> --notes "Implementation complete"`

**Expected Outcome:**
- Exit code 0
- Task advances to next status in workflow (e.g., `ready_for_code_review`)
- Notes are recorded in task history

**Pass/Fail:** Status advanced and notes appear in `shark status history <task-key>`.

---

#### Scenario J1-S07: Full Journey Command Count Validation

**Preconditions:**
- Fresh task in `ready_for_development`

**Steps:**
1. Execute complete journey: next -> get -> status advance (start) -> get --field status -> status options -> status advance (complete)
2. Count total commands executed

**Expected Outcome:**
- Total commands: 5-6
- Zero Python invocations needed
- Zero `2>/dev/null` patterns needed
- Zero fallback chains (no `||` patterns)

**Pass/Fail:** Journey completes in 6 or fewer commands with no external tool dependencies.

---

#### Scenario J1-S08: Structured Error on Invalid Transition (F03)

**Preconditions:**
- Task is in `completed` status
- `SHARK_OUTPUT=json` set

**Steps:**
1. Run `shark status advance <task-key>`

**Expected Outcome:**
- Exit code 3 (invalid state)
- Output to stdout is valid JSON: `{"error": true, "code": "INVALID_TRANSITION", "message": "...", "entity": "<task-key>", "current_status": "completed", "valid_transitions": [...]}`
- No output to stderr

**Pass/Fail:** Error is valid JSON on stdout with all required fields.

---

#### Scenario J1-S09: Structured Error on Entity Not Found (F03)

**Preconditions:**
- `SHARK_OUTPUT=json` set

**Steps:**
1. Run `shark get E99-F99-999`

**Expected Outcome:**
- Exit code 1 (not found)
- Output to stdout is valid JSON: `{"error": true, "code": "NOT_FOUND", "message": "...", "entity": "E99-F99-999"}`
- No output to stderr

**Pass/Fail:** Error is valid JSON on stdout with NOT_FOUND code.

---

#### Scenario J1-S10: Idempotent Status Set

**Preconditions:**
- Task is in `in_development` status

**Steps:**
1. Run `shark status set <task-key> in_development --json`

**Expected Outcome:**
- Exit code 0 (not an error)
- JSON response includes `"changed": false`
- No new history record created

**Pass/Fail:** Returns success with `changed: false`, no history entry.

---

### Journey 2: Orchestrator Batch Workflow Transition

**Persona:** OrchestratorAgent
**Features Exercised:** F01 (status), F07 (batch mode), F06 (progress)

#### Scenario J2-S01: Batch Status Set with Multiple IDs

**Preconditions:**
- 3 tasks in `in_code_review` status within feature E18-F05
- `SHARK_OUTPUT=json` set

**Steps:**
1. Run `shark status set E18-F05-001 E18-F05-002 E18-F05-003 ready_for_qa`
2. Parse JSON response

**Expected Outcome:**
- Exit code 0
- JSON response: `{"updated": 3, "skipped": 0, "failed": 0, "results": [...]}`
- Each result entry shows the entity key and new status

**Pass/Fail:** All 3 tasks updated, batch summary correct.

---

#### Scenario J2-S02: Batch Status Set with Feature-Level Targeting

**Preconditions:**
- Feature E18-F05 has 9 tasks: 7 in `in_code_review`, 2 already `completed`

**Steps:**
1. Run `shark status set --feature E18-F05 --from in_code_review ready_for_qa`

**Expected Outcome:**
- Exit code 0
- JSON response: `{"updated": 7, "skipped": 2, "failed": 0, "results": [...]}`
- Only the 7 `in_code_review` tasks are changed
- The 2 `completed` tasks are skipped (listed in results with reason)

**Pass/Fail:** Exactly 7 updated, 2 skipped, 0 failed.

---

#### Scenario J2-S03: Batch Partial Failure

**Preconditions:**
- 3 task keys provided, 1 does not exist

**Steps:**
1. Run `shark status set E18-F05-001 E18-F05-002 E18-F05-999 ready_for_qa`

**Expected Outcome:**
- Exit code 0 (partial success is not an error)
- JSON response: `{"updated": 2, "skipped": 0, "failed": 1, "results": [...]}`
- Failed entry includes error code `NOT_FOUND` for E18-F05-999
- Successful entries were not rolled back

**Pass/Fail:** 2 succeed, 1 fails, no rollback of successes.

---

#### Scenario J2-S04: Batch Dry Run

**Preconditions:**
- Feature with tasks in mixed statuses

**Steps:**
1. Run `shark status set --feature E18-F05 --from in_code_review ready_for_qa --dry-run`

**Expected Outcome:**
- Exit code 0
- JSON response shows what WOULD happen without actually changing anything
- No status changes in database
- Verify with `shark list E18 F05 --json` that statuses are unchanged

**Pass/Fail:** Dry run produces preview, no actual changes.

---

#### Scenario J2-S05: Feature Progress Check

**Preconditions:**
- Feature E18-F05 has tasks in various statuses

**Steps:**
1. Run `shark progress E18-F05 --json`
2. Parse JSON response

**Expected Outcome:**
- Exit code 0
- JSON includes: `progress_pct` (weighted), `completion_pct`, `total_tasks`, `task_breakdown` (count by status), `health` indicator
- Progress values are correct (verified manually against task statuses)

**Pass/Fail:** Progress data is present, numeric values are correct.

---

#### Scenario J2-S06: Progress Field Extraction

**Preconditions:**
- Feature with known progress

**Steps:**
1. Run `shark progress E18-F05 --field progress_pct`

**Expected Outcome:**
- Exit code 0
- Output is a raw number (e.g., `78.5`)
- No JSON wrapping

**Pass/Fail:** Output is a single numeric value.

---

#### Scenario J2-S07: Full Batch Journey Command Count

**Preconditions:**
- Feature with 9 tasks to transition

**Steps:**
1. Batch transition: `shark status set --feature E18-F05 --from in_code_review ready_for_qa`
2. Verify: `shark progress E18-F05`
3. Count total commands

**Expected Outcome:**
- Total commands: 2
- Baseline (current): ~20 commands (list + filter + 9 individual transitions + verify)
- Reduction: 90%

**Pass/Fail:** Journey completes in 2 commands.

---

### Journey 3: Project Setup (Create Entities)

**Persona:** DevAgent / HumanDev
**Features Exercised:** F05 (flag normalization), F08 (unified create)

#### Scenario J3-S01: Unified Create Feature with Normalized Flags

**Preconditions:**
- Epic E07 exists

**Steps:**
1. Run `shark create feature E07 "User Authentication" --order=1`

**Expected Outcome:**
- Exit code 0
- Feature created with execution_order=1
- `--order` flag accepted (not `--execution-order`)

**Pass/Fail:** Feature created, `--order` works.

---

#### Scenario J3-S02: Unified Create Task

**Preconditions:**
- Feature E07-F01 exists

**Steps:**
1. Run `shark create task E07-F01 "Design auth flow" --agent=architect --order=1`

**Expected Outcome:**
- Exit code 0
- Task created with agent_type=architect, execution_order=1

**Pass/Fail:** Task created with correct agent and order.

---

#### Scenario J3-S03: Deprecated Flag Still Works

**Preconditions:**
- Feature E07-F01 exists

**Steps:**
1. Run `shark create task E07-F01 "Legacy flag test" --execution-order=2`

**Expected Outcome:**
- Exit code 0
- Task created with execution_order=2
- Deprecation warning printed to stderr (not stdout)
- JSON output (if `--json`) is unaffected by deprecation warning

**Pass/Fail:** Old flag works, warning on stderr only.

---

#### Scenario J3-S04: Consistent Create Syntax Across Entity Types

**Preconditions:**
- None (create from scratch)

**Steps:**
1. Run `shark create epic "Test Epic"`
2. Run `shark create feature <epic-key> "Test Feature" --order=1`
3. Run `shark create task <feature-key> "Test Task" --order=1 --agent=backend`

**Expected Outcome:**
- All 3 succeed with exit code 0
- Pattern is consistently: `shark create <type> [parent] "title" [flags]`
- All use `--order` (not `--execution-order`)

**Pass/Fail:** Uniform syntax across all entity types.

---

### Journey 4: Status Check and Decision Making

**Persona:** OrchestratorAgent
**Features Exercised:** F06 (progress), F02 (--field), F01 (status options)

#### Scenario J4-S01: Feature Progress with Task Rollup

**Preconditions:**
- Feature E18-F05 has tasks in various statuses

**Steps:**
1. Run `shark progress E18-F05 --json`

**Expected Outcome:**
- Exit code 0
- JSON contains: progress_pct, completion_pct, total_tasks, task_breakdown by status, health indicator, action_items

**Pass/Fail:** All progress fields present and numerically correct.

---

#### Scenario J4-S02: Epic Progress with Feature Rollup

**Preconditions:**
- Epic E18 has multiple features with different progress levels

**Steps:**
1. Run `shark progress E18 --json`

**Expected Outcome:**
- Exit code 0
- JSON contains: feature_rollup (count by status), task_rollup (aggregated across features), impediments list (blocked tasks)

**Pass/Fail:** Rollup data matches manual calculation.

---

#### Scenario J4-S03: Extract Specific Progress Metric

**Preconditions:**
- Feature with known progress

**Steps:**
1. Run `shark progress E18-F05 --field progress_pct`

**Expected Outcome:**
- Output: raw number (e.g., `78.5`)
- No Python piping needed

**Pass/Fail:** Single numeric value output.

---

#### Scenario J4-S04: Check Available Feature Transitions

**Preconditions:**
- Feature E18-F05 exists with known status

**Steps:**
1. Run `shark status options E18-F05 --json`

**Expected Outcome:**
- Exit code 0
- JSON shows current_status, valid_transitions array
- Transitions are correct per workflow configuration

**Pass/Fail:** Transitions match workflow config.

---

#### Scenario J4-S05: Decision-Making Journey Without Python

**Preconditions:**
- Feature with tasks to evaluate

**Steps:**
1. Run `shark progress E18-F05 --field progress_pct` to get progress number
2. Run `shark status options E18-F05 --json` to check available transitions
3. Count total commands

**Expected Outcome:**
- Total commands: 2 (down from 3 with Python in baseline)
- Zero Python invocations
- All information obtained from native CLI features

**Pass/Fail:** Complete decision-making workflow in 2 commands, no external tools.

---

## 3. Success Metrics Validation Plan

### Metric 1: Command Surface Reduction

| Aspect | Baseline | Target | Pass Criteria |
|--------|----------|--------|---------------|
| Unique command paths | ~45 | ~25 | Count <= 27 (allowing 10% tolerance) |
| Top-level help commands | ~15 | ~10 | Count <= 12 |

**Measurement Method:**
```bash
# Count all leaf commands (non-hidden)
shark --help 2>/dev/null | grep -c "^  [a-z]"
# Recursively count all subcommands
for cmd in $(shark --help 2>/dev/null | grep "^  [a-z]" | awk '{print $1}'); do
  shark $cmd --help 2>/dev/null | grep -c "^  [a-z]"
done
```

**When:** After Phase 3 completion (Phase 1-2 add commands; Phase 3 hides old ones).

**Pass/Fail Gate:** Total leaf command count (excluding hidden) is 27 or fewer. Top-level visible commands is 12 or fewer.

---

### Metric 2: Agent Workflow Efficiency -- Commands per Task Lifecycle

| Aspect | Baseline | Target | Pass Criteria |
|--------|----------|--------|---------------|
| Commands per lifecycle | 8-10 | 5 or fewer | Median <= 5 across 20+ lifecycle completions |

**Measurement Method:**
1. Deploy updated CLI to a test project
2. Run 20+ complete task lifecycles (get next -> start -> complete) using E17 commands
3. Count commands per lifecycle (excluding reads that are optional)
4. Calculate median

**Alternative (agent log analysis):**
```bash
# Parse activity.jsonl for lifecycle patterns
# Group commands by task key
# Count commands between "next" and final "status advance/set" per task
```

**When:** After Phase 1 completion (F01, F02, F04 deliver the efficiency gains).

**Pass/Fail Gate:** Median commands per lifecycle <= 5.

---

### Metric 3: Agent Workflow Efficiency -- Python Post-Processing

| Aspect | Baseline | Target | Pass Criteria |
|--------|----------|--------|---------------|
| Python pipe invocations | ~15% (~30/231) | 0% | Zero `| python3` patterns in 200+ interaction sample |

**Measurement Method:**
1. Collect activity.jsonl from real project usage (minimum 200 interactions over 1 week)
2. Search for pattern: `| python3` or `| python`
3. Calculate percentage

```bash
# Count Python pipes in agent logs
grep -c "python" docs/workflow/activity.jsonl
# Total interactions
wc -l docs/workflow/activity.jsonl
```

**When:** After Phase 1 completion (F02 `--field` eliminates need for Python).

**Pass/Fail Gate:** Zero occurrences of Python piping in post-E17 logs.

---

### Metric 4: Agent Workflow Efficiency -- Defensive Error Suppression

| Aspect | Baseline | Target | Pass Criteria |
|--------|----------|--------|---------------|
| Error suppression patterns | ~36% (~83/231) | <5% | Less than 5% of commands use `2>/dev/null` or `2>&1 ||` |

**Measurement Method:**
1. Collect activity.jsonl (minimum 200 interactions)
2. Count patterns: `2>/dev/null`, `2>&1`, `|| shark`

```bash
grep -cE "2>/dev/null|2>&1|\\|\\| shark" docs/workflow/activity.jsonl
```

**When:** After Phase 1 completion (F03 structured errors + F04 SHARK_OUTPUT eliminate need).

**Pass/Fail Gate:** Error suppression patterns appear in fewer than 5% of commands.

---

### Metric 5: Agent Workflow Efficiency -- Batch For-Loops

| Aspect | Baseline | Target | Pass Criteria |
|--------|----------|--------|---------------|
| Bash for-loop patterns | ~5% (~12/231) | 0% | Zero `for ... do shark ...` patterns |

**Measurement Method:**
1. Collect activity.jsonl (minimum 200 interactions)
2. Search for pattern: `for .* do shark` or `for .* do.*shark`

**When:** After Phase 2 completion (F07 batch mode eliminates for-loops).

**Pass/Fail Gate:** Zero for-loop patterns in post-E17 logs.

---

### Metric 6: Agent Workflow Efficiency -- Status Command Fallback Chains

| Aspect | Baseline | Target | Pass Criteria |
|--------|----------|--------|---------------|
| Fallback chains | ~10% of status changes | 0% | Zero `|| shark task update` after `shark task set-status` patterns |

**Measurement Method:**
1. Analyze agent logs for multi-attempt status change sequences
2. Count instances where agent tries multiple commands for one status change

**When:** After Phase 1 completion (F01 provides single unified status command).

**Pass/Fail Gate:** Zero fallback chain patterns.

---

### Metric 7: Agent Error Rates -- Non-Existent Commands

| Aspect | Baseline | Target | Pass Criteria |
|--------|----------|--------|---------------|
| Unknown command errors | ~3% (~7/231) | 0% | Zero "unknown command" errors in agent logs |

**Measurement Method:**
1. Analyze agent logs for "unknown command", "Error: unknown command"
2. Also count "invalid command" and similar patterns

**When:** After Phase 2 completion (unified commands reduce surface area).

**Pass/Fail Gate:** Zero unknown-command errors in agent logs.

---

### Metric 8: Developer Experience -- Flag Consistency

| Aspect | Target | Pass Criteria |
|--------|--------|---------------|
| Flag name consistency | 100% | Audit of all commands shows no inconsistent flag names |

**Measurement Method:**
1. Audit all commands that accept ordering: all must accept `--order` (not `--execution-order` as primary)
2. Audit all list commands: all must accept `--all` (not `--show-all` as primary)
3. Deprecated flags still work but show warnings

**When:** After Phase 1 (F05).

**Pass/Fail Gate:** Zero inconsistent primary flag names across all commands.

---

### Metric 9: Performance -- Single Command Latency

| Aspect | Target | Pass Criteria |
|--------|--------|---------------|
| Single command | <200ms | Average of 10 runs <= 200ms |

**Measurement Method:**
```bash
for i in $(seq 1 10); do
  /usr/bin/time -f "%e" shark get E07-F01-001 --json 2>&1 | tail -1
done
# Average the results
```

**When:** After each phase completion.

**Pass/Fail Gate:** Average latency <= 200ms.

---

### Metric 10: Performance -- Batch Latency

| Aspect | Target | Pass Criteria |
|--------|--------|---------------|
| Batch (20 entities) | <500ms | Average of 5 runs <= 500ms |

**Measurement Method:**
```bash
for i in $(seq 1 5); do
  /usr/bin/time -f "%e" shark status set --feature E07-F01 --from todo in_progress 2>&1 | tail -1
done
```

**When:** After Phase 2 (F07).

**Pass/Fail Gate:** Average batch latency <= 500ms for 20 entities.

---

### Metric 11: Performance -- Field Extraction Overhead

| Aspect | Target | Pass Criteria |
|--------|--------|---------------|
| --field overhead | <10ms additional vs full JSON | Difference of 10-run averages <= 10ms |

**Measurement Method:**
```bash
# Full JSON
for i in $(seq 1 10); do /usr/bin/time -f "%e" shark get E07-F01-001 --json 2>&1 | tail -1; done
# With --field
for i in $(seq 1 10); do /usr/bin/time -f "%e" shark get E07-F01-001 --field status 2>&1 | tail -1; done
# Compute difference of averages
```

**When:** After Phase 1 (F02).

**Pass/Fail Gate:** Overhead <= 10ms.

---

### Metric 12: Backward Compatibility

| Aspect | Target | Pass Criteria |
|--------|--------|---------------|
| Existing test suite | 100% pass | `make test` returns 0 |
| Old command output | Identical | Snapshot comparison shows no differences |

**Measurement Method:**
1. Run `make test` -- all tests pass
2. Before E17 changes: capture output snapshots for 20 representative commands
3. After E17 changes: compare snapshots for identity

**When:** Continuously -- after every feature merge.

**Pass/Fail Gate:** Zero test failures. Zero snapshot differences.

---

## 4. Cross-Epic Integration Test Scenarios

### 4.1 E15 Integration: Service Layer Architecture

E17 commands must use the E15 service layer pattern (thin wrappers calling services).

#### Scenario CE-E15-01: Status Set Uses Service Layer

**Steps:**
1. Run `shark status set <task-key> in_development --json`
2. Verify the command delegates to `TaskService` (not direct repo calls)

**Validation Method:** Code review of `status_group.go` -- command handler must call `cli.GetTaskService()` or equivalent, not `repository.NewTaskRepository()`.

**Pass/Fail:** No direct repository calls in any E17 command file.

---

#### Scenario CE-E15-02: Status Advance Uses Workflow Service

**Steps:**
1. Run `shark status advance <task-key>`
2. Verify transition validation happens through `workflow.Service`

**Validation Method:** Code review -- `status advance` handler must call service methods that use `workflowSvc.ValidateTransition()` or `GetNextStatus()`.

**Pass/Fail:** All transition logic goes through workflow service.

---

#### Scenario CE-E15-03: New Service Methods Follow E15 Patterns

**Steps:**
1. If E17 creates any new service methods, review them for:
   - Constructor injection (not global state)
   - Context as first parameter
   - Repository interface dependencies (not concrete types)
   - Error wrapping with business context

**Validation Method:** Code review against `.claude/rules/services/service-design.md`.

**Pass/Fail:** All new service methods conform to E15 patterns.

---

### 4.2 E16 Integration: Multi-Level Workflow

E17 status commands must work with E16's multi-level workflow when available.

#### Scenario CE-E16-01: Status Set Respects Entity-Level Workflow

**Steps:**
1. Configure advanced workflow with distinct epic/feature/task status flows
2. Run `shark status set E07 active` (epic-level transition)
3. Run `shark status set E07-F01 active` (feature-level transition)
4. Run `shark status set E07-F01-001 in_development` (task-level transition)

**Expected Outcome:**
- Each transition is validated against the correct entity-level workflow
- Epic statuses do not leak into feature or task validation

**Pass/Fail:** Each entity type uses its own workflow configuration.

---

#### Scenario CE-E16-02: Status Options Shows Level-Appropriate Transitions

**Steps:**
1. Run `shark status options E07 --json` (epic)
2. Run `shark status options E07-F01 --json` (feature)
3. Run `shark status options E07-F01-001 --json` (task)

**Expected Outcome:**
- Each returns transitions appropriate to its entity level
- Epic options show epic-valid transitions, not task transitions

**Pass/Fail:** Transition lists are level-appropriate.

---

#### Scenario CE-E16-03: Forward Compatibility Without E16

**Steps:**
1. Use a project with basic workflow profile (no E16 multi-level config)
2. Run `shark status set E07 active`
3. Run `shark status advance E07-F01`

**Expected Outcome:**
- Commands work with the existing hardcoded epic/feature statuses
- No errors about missing workflow configuration

**Pass/Fail:** E17 commands work whether E16 is deployed or not.

---

### 4.3 E11/E13 Integration: Existing Workflow Commands

#### Scenario CE-E13-01: Old Task Lifecycle Commands Still Work

**Steps:**
1. Run `shark task start <task-key>` (old E13 command)
2. Run `shark task complete <task-key>` (old E13 command)
3. Run `shark task approve <task-key>` (old E13 command)

**Expected Outcome:**
- All produce identical output to pre-E17 behavior
- Exit codes unchanged
- JSON output format identical

**Pass/Fail:** Byte-identical output comparison with pre-E17 snapshots.

---

#### Scenario CE-E13-02: Old and New Commands Produce Equivalent Results

**Steps:**
1. Create two tasks in `ready_for_development`
2. Start task A with: `shark task start <task-A>`
3. Start task B with: `shark status set <task-B> in_development`
4. Compare resulting statuses and history entries

**Expected Outcome:**
- Both tasks end up in `in_development`
- Both have history entries recording the transition
- Agent field is recorded identically

**Pass/Fail:** Equivalent outcomes regardless of which command was used.

---

#### Scenario CE-E13-03: Status History Shows All Transitions Regardless of Command Used

**Steps:**
1. Advance task through several statuses using a mix of old and new commands
2. Run `shark status history <task-key> --json`

**Expected Outcome:**
- All transitions appear in history
- History entries are consistent regardless of which command caused the transition

**Pass/Fail:** Complete, consistent history.

---

## 5. Risk-Based Test Priorities

Priorities derived from the 7 risks identified in the research report, BA feasibility review, and tech feasibility review.

### Priority 1 (CRITICAL): Backward Compatibility Regression (Risk 2)

**Risk:** Changes break existing commands, destroying agent trust.
**Impact:** HIGH
**Test Focus:** Regression suite runs after EVERY feature merge.

| Test ID | Scenario | Priority |
|---------|----------|----------|
| BC-01 | All `make test` passes | BLOCKER |
| BC-02 | Snapshot comparison of 20 representative old commands | BLOCKER |
| BC-03 | Old exit codes unchanged | BLOCKER |
| BC-04 | Old `--json` output format byte-identical | BLOCKER |
| BC-05 | `shark task start/complete/approve` still work | BLOCKER |
| BC-06 | `shark epic/feature set-status` still work | BLOCKER |
| BC-07 | `shark epic/feature next-status` still work | BLOCKER |
| BC-08 | `--execution-order` still accepted (deprecated but functional) | HIGH |
| BC-09 | `--show-all` still accepted (deprecated but functional) | HIGH |

**Execution:** Automated. Run as part of CI on every PR.

---

### Priority 2 (HIGH): Status Namespace Collision (Risk 5)

**Risk:** `shark status <id>` (progress) vs `shark status set <id>` (transition) cause confusion.
**Impact:** MEDIUM

| Test ID | Scenario | Priority |
|---------|----------|----------|
| NS-01 | `shark status E07` shows progress dashboard (existing behavior) | BLOCKER |
| NS-02 | `shark status set E07 active` changes epic status | HIGH |
| NS-03 | `shark status advance E07-F01-001` advances task | HIGH |
| NS-04 | `shark status options E07-F01-001` shows transitions | HIGH |
| NS-05 | `shark status history E07-F01-001` shows history | HIGH |
| NS-06 | `shark status` (no args) shows project dashboard | MEDIUM |
| NS-07 | Cobra help text clearly distinguishes progress vs transition | MEDIUM |

**Execution:** Manual + automated after F01 and F06 implementation.

---

### Priority 3 (HIGH): Error Output Stream Change (Risk 4, F03)

**Risk:** Moving errors from stderr to stdout in JSON mode breaks scripts.
**Impact:** MEDIUM

| Test ID | Scenario | Priority |
|---------|----------|----------|
| ERR-01 | JSON mode: errors go to stdout as structured JSON | HIGH |
| ERR-02 | Non-JSON mode: errors go to stderr as human text (unchanged) | BLOCKER |
| ERR-03 | JSON error includes all required fields (error, code, message, entity) | HIGH |
| ERR-04 | Transition error includes current_status and valid_transitions | HIGH |
| ERR-05 | Exit codes: 0 success, 1 not found, 2 system, 3 invalid state, 4 conflict | HIGH |
| ERR-06 | Deprecation warnings go to stderr only, never stdout | HIGH |
| ERR-07 | No deprecation warnings in JSON mode | HIGH |

**Execution:** Automated CLI tests with output stream capture.

---

### Priority 4 (MEDIUM): Batch Mode Complexity (Risk 4)

**Risk:** Batch operations with partial failures create confusing results.
**Impact:** MEDIUM

| Test ID | Scenario | Priority |
|---------|----------|----------|
| BATCH-01 | Multi-ID batch: all succeed | HIGH |
| BATCH-02 | Multi-ID batch: partial failure (1 not found) | HIGH |
| BATCH-03 | Feature-level batch with --from filter | HIGH |
| BATCH-04 | Feature-level batch WITHOUT --from (should require it or warn) | HIGH |
| BATCH-05 | Dry-run mode previews without changing | MEDIUM |
| BATCH-06 | Batch result JSON structure is parseable | HIGH |
| BATCH-07 | Batch with 20 entities completes in <500ms | MEDIUM |
| BATCH-08 | Individual failures do NOT roll back other successes | HIGH |

**Execution:** Automated after F07 implementation.

---

### Priority 5 (MEDIUM): Agent Adoption (Risk 3)

**Risk:** Agents continue using old commands even after E17 ships.
**Impact:** MEDIUM

| Test ID | Scenario | Priority |
|---------|----------|----------|
| ADOPT-01 | CLAUDE.md references new E17 commands as primary | HIGH |
| ADOPT-02 | `--help` output for status shows new subcommands prominently | MEDIUM |
| ADOPT-03 | Deprecation warnings reference exact replacement command | MEDIUM |
| ADOPT-04 | Agent log analysis shows >80% adoption of new status commands | MEDIUM |

**Execution:** Manual review (CLAUDE.md, help output) + log analysis post-deployment.

---

### Priority 6 (LOW): Service Layer Readiness (Risk 1)

**Risk:** Required service methods don't exist.
**Impact:** LOW (research verified all methods exist)

| Test ID | Scenario | Priority |
|---------|----------|----------|
| SVC-01 | `EpicService.TransitionStatus()` callable from status set | LOW |
| SVC-02 | `FeatureService.TransitionStatus()` callable from status set | LOW |
| SVC-03 | `TaskService.StartTask()` / lifecycle methods callable | LOW |
| SVC-04 | `StatusService.GetDashboard()` callable from progress | LOW |

**Execution:** Verified during implementation. If any method is missing, create it per E15 patterns.

---

### Priority 7 (LOW): Exit Code Conflict for --field (Risk 6)

**Risk:** Exit code 1 means both "entity not found" and "field not found".
**Impact:** LOW

| Test ID | Scenario | Priority |
|---------|----------|----------|
| EC-01 | Entity not found: exit code 1 | MEDIUM |
| EC-02 | Field not found on valid entity: exit code 1 (or 4 per research recommendation) | MEDIUM |
| EC-03 | JSON error code distinguishes NOT_FOUND vs FIELD_NOT_FOUND | MEDIUM |
| EC-04 | Agent can programmatically distinguish the two cases | MEDIUM |

**Execution:** Automated after F02 and F03 implementation.

---

## 6. Persona-Based Acceptance Matrix

### 6.1 DevAgent (Primary Persona -- 70% of CLI usage)

| Need | Feature | Acceptance Scenario(s) | Validates? |
|------|---------|----------------------|------------|
| Predictable status command syntax | F01 | J1-S03, J1-S06 | One command for all status changes |
| Single-field extraction without Python | F02 | J1-S01, J1-S04 | `--field` returns raw value |
| JSON output by default via env var | F04 | J1-S02 | `SHARK_OUTPUT=json` works |
| Structured JSON errors | F03 | J1-S08, J1-S09 | Errors are valid JSON on stdout |
| Idempotent operations | F01 | J1-S10 | Setting current status returns success |
| Consistent flag names | F05 | J3-S03 | `--order` works, `--execution-order` deprecated |
| Full lifecycle in <=5 commands | All Phase 1 | J1-S07 | Command count <= 5 |

**DevAgent Overall Pass Gate:** All scenarios in the "Validates?" column pass. Full lifecycle completes in 5 or fewer commands with zero Python dependency and zero error suppression.

---

### 6.2 OrchestratorAgent (Secondary Persona -- 20% of CLI usage)

| Need | Feature | Acceptance Scenario(s) | Validates? |
|------|---------|----------------------|------------|
| Batch status changes | F07 | J2-S01, J2-S02 | Multi-ID and feature-level batch |
| Feature-level batch with filter | F07 | J2-S02 | `--from` filter selects correct tasks |
| Feature progress and health | F06 | J2-S05, J4-S01 | Progress rollup with breakdown |
| Single command batch advance | F07 | J2-S07 | 2 commands instead of ~20 |
| Partial success reporting | F07 | J2-S03 | Per-entity results in batch response |
| Dry-run for batch | F07 | J2-S04 | Preview without side effects |

**OrchestratorAgent Overall Pass Gate:** Batch operations work with partial success. Feature progress includes health indicator. Batch journey completes in 2 commands.

---

### 6.3 HumanDev (Tertiary Persona -- 10% of CLI usage)

| Need | Feature | Acceptance Scenario(s) | Validates? |
|------|---------|----------------------|------------|
| Clean help output | F09, F11 | ADOPT-02, ADOPT-03 | Fewer top-level commands, clear help |
| Consistent flag names | F05 | J3-S01, J3-S03 | `--order` everywhere |
| Human-readable default output | All | HD-01 (below) | Table output preserved when no JSON mode |
| Old commands still work | Phase 1-2 | BC-05 through BC-09 | Zero breaking changes |

#### Scenario HD-01: Human-Readable Output Preserved

**Steps:**
1. Unset `SHARK_OUTPUT` env var
2. Run `shark get <task-key>` (no `--json`)
3. Run `shark status set <task-key> in_development`

**Expected Outcome:**
- Output is human-readable table/text format with colors
- Not JSON

**Pass/Fail:** Default output is human-readable, not JSON.

---

**HumanDev Overall Pass Gate:** Default output is human-readable. `--help` is clean and discoverable. Old commands work without changes. Consistent flag naming.

---

## 7. Non-Functional Requirements Validation

### NFR-1: Backward Compatibility

**Test:** BC-01 through BC-09 (see Priority 1 above)
**Gate:** 100% -- all existing commands produce identical output and exit codes.

### NFR-2: Performance

**Tests:** Metrics 9, 10, 11 (see Section 3)
**Gate:**
- Single command: <= 200ms average
- Batch 20 entities: <= 500ms average
- `--field` overhead: <= 10ms

### NFR-3: Service Layer Integration

**Tests:** CE-E15-01 through CE-E15-03
**Gate:** Zero direct repository calls in any E17 command file.

### NFR-4: Testing Coverage

**Tests:**
- All new CLI commands have tests with mocked services
- All new service methods have unit tests with mocked repositories
- Backward compatibility snapshot tests exist for old commands

**Gate:** `make test` passes. Coverage for new code >= 80%.

### NFR-5: Non-Interactive Operation

**Test:** All E17 commands must work without stdin.

| Scenario | Steps | Expected |
|----------|-------|----------|
| NI-01 | Pipe `/dev/null` to stdin for all new commands | All succeed without hanging |
| NI-02 | Run from non-TTY context (e.g., `echo "" | shark status set ...`) | Succeeds |

**Gate:** Zero commands require interactive input.

---

## 8. Phase Gate Criteria

### Phase 1 Complete When:

- [ ] F01-F05 all implemented and tested
- [ ] BC-01 through BC-09 all pass (backward compatibility)
- [ ] J1-S01 through J1-S10 all pass (DevAgent daily workflow)
- [ ] NS-01 through NS-07 all pass (namespace coexistence)
- [ ] ERR-01 through ERR-07 all pass (structured errors)
- [ ] EC-01 through EC-04 all pass (exit codes)
- [ ] Metrics 2, 3, 4, 6, 8, 9, 11 validated (workflow efficiency + performance)
- [ ] `make test` returns 0
- [ ] HD-01 passes (human-readable default output preserved)
- [ ] NI-01 and NI-02 pass (non-interactive)

### Phase 2 Complete When:

- [ ] F06-F08 all implemented and tested
- [ ] Phase 1 gate still passing (regression)
- [ ] J2-S01 through J2-S07 all pass (batch operations)
- [ ] J3-S01 through J3-S04 all pass (unified create)
- [ ] J4-S01 through J4-S05 all pass (status check and decision)
- [ ] BATCH-01 through BATCH-08 all pass
- [ ] Metric 5 validated (batch for-loops eliminated)
- [ ] Metric 7 validated (unknown command errors eliminated)
- [ ] Metric 10 validated (batch performance)
- [ ] Agent log analysis (1 week, 200+ interactions): >80% status changes use new commands

### Phase 3 Complete When:

- [ ] F09-F13 all implemented and tested
- [ ] Phase 1+2 gates still passing (regression)
- [ ] Metric 1 validated (command surface <= 27 paths)
- [ ] ADOPT-01 through ADOPT-04 pass
- [ ] Deprecation warnings work correctly (stderr only, suppressed in JSON mode)
- [ ] Old commands hidden from `--help` but still fully functional
- [ ] Agent logs show <5% usage of deprecated command forms

---

## Appendix A: Test Scenario Summary

| Journey | Scenarios | Features Covered | Phase |
|---------|-----------|-----------------|-------|
| J1: DevAgent Daily Workflow | J1-S01 through J1-S10 | F01, F02, F03, F04 | 1 |
| J2: Orchestrator Batch | J2-S01 through J2-S07 | F01, F06, F07 | 2 |
| J3: Project Setup | J3-S01 through J3-S04 | F05, F08 | 1-2 |
| J4: Status Check & Decision | J4-S01 through J4-S05 | F01, F02, F06 | 1-2 |

| Risk Area | Test IDs | Count |
|-----------|----------|-------|
| Backward Compatibility | BC-01 through BC-09 | 9 |
| Namespace Collision | NS-01 through NS-07 | 7 |
| Structured Errors | ERR-01 through ERR-07 | 7 |
| Batch Complexity | BATCH-01 through BATCH-08 | 8 |
| Agent Adoption | ADOPT-01 through ADOPT-04 | 4 |
| Service Layer | SVC-01 through SVC-04 | 4 |
| Exit Code Conflict | EC-01 through EC-04 | 4 |

**Total acceptance scenarios:** 26 journey scenarios + 43 risk-based scenarios + 12 success metric validations = **81 test points**

---

## Appendix B: Exit Gate Verification

| Criterion | Met? |
|-----------|------|
| Every user journey has at least one acceptance scenario | YES -- J1 (10), J2 (7), J3 (4), J4 (5) |
| Every success metric has a validation method | YES -- 12 metrics, each with measurement method and pass/fail criteria |
| Risk areas have targeted scenarios | YES -- 7 risk areas, 43 test scenarios |
| Plan is actionable for feature-level decomposition | YES -- scenarios map to features; phase gates define feature-level readiness |
| Persona needs are traced to acceptance criteria | YES -- Section 6 matrices trace each persona need to specific scenarios |

---

*Last Updated: 2026-02-25*
*Next: Advance to decomposition phase*
