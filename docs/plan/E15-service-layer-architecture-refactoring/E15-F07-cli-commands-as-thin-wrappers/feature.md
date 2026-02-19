---
feature_key: E15-F07-cli-commands-as-thin-wrappers
epic_key: E15
title: CLI Commands as Thin Wrappers
description: Refactor task.go (2,664 lines), feature.go (2,252 lines), and epic.go (1,938 lines) from fat controllers to thin wrappers (200-400 lines each). Commands must only parse arguments, call service methods, and format output. All business logic must reside in services. Global service accessors already exist and must be used consistently.
---

# CLI Commands as Thin Wrappers

**Feature Key**: E15-F07-cli-commands-as-thin-wrappers

---

## Epic

- **Epic PRD**: [Service Layer Architecture Refactoring](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)
- **Epic Personas**: [User Personas](../../personas.md)
- **Epic Success Metrics**: [Success Metrics](../../success-metrics.md)

**Epic Requirement Traceability**: This feature implements FR6 (CLI Commands as Thin Wrappers) from the epic requirements and is a direct dependency of the epic's primary success criterion: "No CLI command file exceeds 500 lines."

---

## Goal

### Problem

The three largest CLI command files in `internal/cli/commands/` act as fat controllers, embedding 40-45% of all Shark business logic directly in the CLI layer:

| File | Current Lines | Business Logic % | Problems |
|------|--------------|-----------------|---------|
| `task.go` | 2,664 | ~70% | Workflow validation, dependency checks, status cascading, key generation, file creation |
| `feature.go` | 2,252 | ~65% | Progress calculation, health derivation, action item aggregation, work breakdown |
| `epic.go` | 1,938 | ~60% | Feature rollups, task aggregations, impediment analysis, progress calculation |
| **Total** | **6,854** | | **~4,500 lines of misplaced business logic** |

This violates the clean architecture layering required by Epic E15 and creates three concrete problems:

1. **Business logic is untestable**: CLI tests require real database connections or extremely complex repository mocking because commands bypass the service layer and call repositories directly (351 repository instantiations across 46 files)
2. **HTTP API cannot reuse logic**: The HTTP server (`cmd/server/`) must duplicate logic that exists only in CLI commands, creating feature parity gaps and code duplication
3. **Services exist but are bypassed**: Global service accessors (`cli.GetTaskService()`, `cli.GetFeatureService()`, `cli.GetEpicService()`) already exist in `internal/cli/services_global.go` and `internal/cli/service_accessors.go` but are not consistently used by the command files

### Solution

Apply the thin wrapper pattern to `task.go`, `feature.go`, and `epic.go` by redirecting all business logic through the existing service accessors. Each command handler is refactored to:
1. Parse CLI arguments and flags into a DTO or primitive inputs
2. Call the appropriate service method via the global accessor
3. Format and output the service result

The services (TaskService, FeatureService, EpicService) absorb all extracted business logic and have already been built or expanded by earlier features in Epic E15 (F01-F06).

### Impact

- **Line reduction**: 6,854 lines total → target 1,200 lines (~82% reduction in command files)
- **Zero repository calls from commands**: 351 direct repository instantiations → 0
- **Architecture compliance**: CLI layer contains ZERO business logic post-refactor
- **Testability**: All CLI command handlers become testable with mocked services (no database required)
- **HTTP API parity**: Service layer is fully reachable by both CLI and HTTP API consumers

---

## User Personas

### Primary Persona: Shark Core Developer

**Reference**: [Persona 1: Shark Core Developer](../../personas.md#persona-1-shark-core-developer-human)

**Goals for This Feature**:
- Navigate `task.go`, `feature.go`, and `epic.go` in minutes, not hours
- Understand any command's behavior by reading a 15-30 line handler
- Write CLI command tests that use mocked services (no database setup required)
- Add a new command by following a simple 3-step template

**Pain Points This Feature Addresses**:
- Opens `task.go` to change a behavior and must read 2,664 lines to find the relevant code
- Must mock repositories and database connections in CLI tests because commands bypass services
- Sees different error handling patterns in different commands due to code duplication
- Cannot determine if a change to a command will break the HTTP API (logic is not shared)

**Success Looks Like**: Developer needs to change how `shark task start` validates dependencies. They open `task.go`, find `runTaskStart` (15-20 lines), see it calls `svc.StartTask()`, open `task_service.go`, find `StartTask()` with the business logic. Change is made in the service, automatically applies to both CLI and HTTP API.

### Secondary Persona: Shark AI Developer Agent

**Reference**: [Persona 2: AI Developer Agent](../../personas.md#persona-2-ai-developer-agent-ai)

**Goals for This Feature**:
- Receive predictable, consistent CLI command output (not affected by refactoring)
- Find all business logic for a domain in one file (`task_service.go`, not scattered across `task.go`)
- Generate tests for CLI commands that mock services (simple, reliable, fast)

**Pain Points This Feature Addresses**:
- Must analyze 2,664 lines of `task.go` to understand task operations before making changes
- CLI tests fail intermittently due to database state dependencies
- Unclear whether changes to `task.go` affect HTTP API behavior

---

## User Stories (MoSCoW)

### Must-Have Stories

**Story 1: Global Service Accessors Available for All Three Entity Types**

As a Shark developer, I want global service accessor functions for TaskService, FeatureService, and EpicService to be complete and fully wired so that command handlers can call services without managing dependency construction.

**Acceptance Criteria**:
- Given the `internal/cli/` package
- When a developer writes a command handler for any task, feature, or epic operation
- Then they can call `cli.GetTaskService()`, `cli.GetFeatureService()`, or `cli.GetEpicService()` to get a fully wired service instance
- And each accessor correctly wires repositories, workflow service, note repository, and optional dependencies
- And no TODO comments remain in accessor implementations that indicate missing wiring

**Story 2: task.go Refactored to Thin Wrapper**

As a Shark developer, I want `task.go` refactored to a thin wrapper so that all task business logic is in TaskService and the command file is navigable without extensive reading.

**Acceptance Criteria**:
- Given `internal/cli/commands/task.go`
- When the refactoring is complete
- Then the file is at most 400 lines (excluding test files)
- And every command handler function follows the pattern: parse args → call service method → format output
- And no command handler contains workflow validation, status checks, dependency validation, or progress calculation
- And no command handler instantiates repository objects directly (`repository.New*` calls absent)
- And all task commands produce identical output to pre-refactoring behavior (verified by regression tests)

**Story 3: feature.go Refactored to Thin Wrapper**

As a Shark developer, I want `feature.go` refactored to a thin wrapper so that all feature business logic is in FeatureService and the command file is easy to navigate.

**Acceptance Criteria**:
- Given `internal/cli/commands/feature.go`
- When the refactoring is complete
- Then the file is at most 350 lines (excluding test files)
- And every command handler follows: parse args → call service method → format output
- And no command handler contains progress calculation, health derivation, or action item aggregation
- And no command handler instantiates repository objects directly
- And all feature commands produce identical output to pre-refactoring behavior

**Story 4: epic.go Refactored to Thin Wrapper**

As a Shark developer, I want `epic.go` refactored to a thin wrapper so that all epic business logic is in EpicService and the command file is easy to navigate.

**Acceptance Criteria**:
- Given `internal/cli/commands/epic.go`
- When the refactoring is complete
- Then the file is at most 300 lines (excluding test files)
- And every command handler follows: parse args → call service method → format output
- And no command handler contains feature rollup logic, impediment analysis, or progress aggregation
- And no command handler instantiates repository objects directly
- And all epic commands produce identical output to pre-refactoring behavior

**Story 5: Zero Behavior Regression**

As a Shark user, I want all CLI commands to behave identically after the refactoring so that nothing breaks in my existing scripts and workflows.

**Acceptance Criteria**:
- Given the existing CLI test suite and any integration tests
- When the refactoring is complete and `make test` is run
- Then all tests pass with exit code 0
- And `shark task next`, `shark task start`, `shark task complete`, `shark feature get`, `shark epic get` produce byte-identical output to pre-refactor behavior for the same inputs
- And exit codes are unchanged for all error scenarios

### Should-Have Stories

**Story 6: Centralized Error Handler for CLI Commands**

As a Shark developer, I want a shared `HandleServiceError()` helper function so that all command handlers translate service errors to exit codes consistently rather than duplicating 47 variations of the same error-handling code.

**Acceptance Criteria**:
- Given a command handler that receives a service error
- When `HandleServiceError(err, entityKey)` is called
- Then `NotFoundError` results in `cli.Error(...)` + `os.Exit(1)`
- And `WorkflowError` results in `cli.Error(...)` + `os.Exit(3)`
- And all other errors result in `cli.Error(...)` + `os.Exit(2)`
- And this helper is used consistently across task, feature, and epic command handlers

**Story 7: Argument Parsing Helper Functions**

As a Shark developer, I want argument parsing helper functions extracted from command handlers so that complex positional argument logic is reusable and independently testable.

**Acceptance Criteria**:
- Given a command with positional argument parsing (e.g., task create with 3-arg, 2-arg, or flag formats)
- When the argument parsing is refactored
- Then parsing logic is extracted to a named helper function (e.g., `parseCreateTaskInput()`)
- And the helper function accepts `cmd *cobra.Command` and `args []string`
- And the helper function returns a typed DTO or struct
- And the main command handler is reduced to: `input := parseX(cmd, args); result, err := svc.Method(ctx, input); ...`

### Could-Have Stories

**Story 8: Unused Repository Import Cleanup**

As a Shark developer, I want all unused repository imports removed from refactored command files so that the import statements only show the actual dependencies of the command file.

**Acceptance Criteria**:
- Given a refactored command file
- When `make lint` is run
- Then no "imported and not used" warnings exist for repository packages
- And `make fmt` produces no changes to import sections

---

## Functional Requirements

### FR-F07-001: Service Accessor Completeness

**Description**: All three global service accessors must be fully implemented with complete dependency wiring before any command refactoring begins.

**Specific Requirements**:
- `cli.GetTaskService()` must wire: TaskRepository, workflow.Service, taskcreation.Creator (or nil if not yet implemented), EntityNoteRepository
- `cli.GetFeatureService()` must wire: FeatureRepository, workflow.Service, EntityNoteRepository (with correct interface), TaskRepository for task counting
- `cli.GetEpicService()` must wire: EpicRepository, workflow.Service, EntityNoteRepository (with correct interface), FeatureRepository, TaskRepository
- All TODO comments indicating missing wiring must be resolved or explicitly deferred with a tracked issue

**Acceptance Criteria**:
- `cli.GetTaskService()` returns a usable `*services.TaskService` with all operations functional
- `cli.GetFeatureService()` returns a usable `*services.FeatureService` with progress and health methods functional
- `cli.GetEpicService()` returns a usable `*services.EpicService` with rollup methods functional
- No nil pointer dereferences occur during normal command execution

**Traceability**: Epic FR1 (Service Layer Architecture), FR6 (CLI Commands as Thin Wrappers)

---

### FR-F07-002: task.go Thin Wrapper Implementation

**Description**: Each command handler in `task.go` must follow the three-step thin wrapper pattern with no business logic.

**Specific Requirements**:

The following command handlers must be refactored (each to 15-30 lines):
- `runTaskCreate` - parse to `CreateTaskInput` DTO → `svc.CreateTask(ctx, input)`
- `runTaskList` - parse to `TaskFilters` DTO → `svc.ListTasks(ctx, filters)`
- `runTaskGet` - parse key → `svc.GetTask(ctx, key)`
- `runTaskStart` - parse key + agent → `svc.StartTask(ctx, key, agent)`
- `runTaskComplete` - parse key + notes → `svc.CompleteTask(ctx, key, notes)`
- `runTaskApprove` - parse key + notes → `svc.ApproveTask(ctx, key, notes)`
- `runTaskReopen` - parse key + notes → `svc.ReopenTask(ctx, key, notes)`
- `runTaskBlock` - parse key + reason → `svc.BlockTask(ctx, key, reason)`
- `runTaskUnblock` - parse key → `svc.UnblockTask(ctx, key)`
- `runTaskNext` - parse filters → `svc.GetNextTask(ctx, filters)`
- `runTaskDelete` - parse key → `svc.DeleteTask(ctx, key)` (if applicable)
- Any remaining task sub-commands in task.go

**Prohibited Patterns** (must not appear in any task.go handler after refactoring):
- `repository.NewTaskRepository(...)`, `repository.NewEpicRepository(...)`, `repository.NewFeatureRepository(...)`
- `cli.GetDB(...)` followed by direct repository construction
- `workflow.Service` access (must be in service layer)
- `status.NewCalculationService(...)` calls
- Filtering loops (`for _, t := range tasks { if ... }`)
- Sorting logic (`sort.Slice(...)`)

**Acceptance Criteria**:
- File `task.go` is at most 400 lines
- `grep -c "repository\.New" internal/cli/commands/task.go` returns 0
- All task commands produce identical output before and after refactoring

**Traceability**: Epic FR2 (TaskService Implementation), FR6

---

### FR-F07-003: feature.go Thin Wrapper Implementation

**Description**: Each command handler in `feature.go` must follow the three-step thin wrapper pattern.

**Specific Requirements**:

The following command handlers must be refactored:
- `runFeatureCreate` - parse to `CreateFeatureInput` DTO → `svc.CreateFeature(ctx, input)`
- `runFeatureList` - parse filters → `svc.ListFeatures(ctx, filters)`
- `runFeatureGet` - parse key → `svc.GetFeature(ctx, key)`
- `runFeatureUpdate` - parse key + updates → `svc.UpdateFeature(ctx, key, updates)`
- Any feature sub-commands (status transitions, progress display, etc.)

**Prohibited Patterns** (must not appear after refactoring):
- Direct repository construction from within handlers
- Progress calculation logic (`completed / total * 100`)
- Health status derivation (if/else chains based on task counts)
- Action item aggregation (loops over tasks to find blocked/ready states)
- Work breakdown calculations

**Acceptance Criteria**:
- File `feature.go` is at most 350 lines
- `grep -c "repository\.New" internal/cli/commands/feature.go` returns 0
- All feature commands produce identical output before and after refactoring

**Traceability**: Epic FR3 (FeatureService Implementation), FR6

---

### FR-F07-004: epic.go Thin Wrapper Implementation

**Description**: Each command handler in `epic.go` must follow the three-step thin wrapper pattern.

**Specific Requirements**:

The following command handlers must be refactored:
- `runEpicCreate` - parse to `CreateEpicInput` DTO → `svc.CreateEpic(ctx, input)`
- `runEpicList` - parse filters → `svc.ListEpics(ctx, filters)`
- `runEpicGet` - parse key → `svc.GetEpic(ctx, key)`
- `runEpicUpdate` - parse key + updates → `svc.UpdateEpic(ctx, key, updates)`
- Any epic sub-commands (feature rollup display, impediment display, etc.)

**Prohibited Patterns** (must not appear after refactoring):
- Direct repository construction from within handlers
- Feature rollup aggregation (loops across features to compute counts)
- Impediment analysis (filtering and grouping blocked tasks)
- Progress aggregation across features

**Acceptance Criteria**:
- File `epic.go` is at most 300 lines
- `grep -c "repository\.New" internal/cli/commands/epic.go` returns 0
- All epic commands produce identical output before and after refactoring

**Traceability**: Epic FR4 (EpicService Implementation), FR6

---

### FR-F07-005: Consistent Error Handling

**Description**: Error handling in command handlers must follow the project's exit code conventions and be consistent across all three command files.

**Specific Requirements**:
- Exit code 0: Successful operation
- Exit code 1: Entity not found (NotFoundError from service)
- Exit code 2: Database or system error
- Exit code 3: Invalid state / business rule violation (workflow transition error)
- Error messages use `cli.Error(message)` before calling `os.Exit(n)`

**Acceptance Criteria**:
- Given any task, feature, or epic command that encounters an error
- When the error is a `*repository.NotFoundError`, exit code is 1
- When the error is a workflow/transition error, exit code is 3
- When the error is any other error, exit code is 2
- Error messages are human-readable and include the entity key

**Traceability**: Epic NFR5 (Backward Compatibility)

---

### FR-F07-006: Output Format Preservation

**Description**: JSON and table output formats must be identical before and after refactoring.

**Specific Requirements**:
- `--json` flag output: Same JSON fields, same field names, same data types
- Table output: Same column headers, same column order, same row data
- Success messages: Same wording and format
- Error messages: Same wording, same exit codes

**Acceptance Criteria**:
- Given `shark task get E07-F01-001 --json` before and after refactoring
- When the JSON output is compared
- Then the JSON structures are identical (same keys, same values for same data)
- And `shark task list --json` output structure is identical
- And `shark feature get E07-F01 --json` output structure is identical
- And `shark epic get E07 --json` output structure is identical

**Traceability**: Epic NFR5 (Backward Compatibility)

---

## Non-Functional Requirements

### NFR-F07-001: Line Count Targets

| File | Current | Target | Maximum Allowed |
|------|---------|--------|----------------|
| `task.go` | 2,664 | 300-400 | 500 |
| `feature.go` | 2,252 | 250-350 | 500 |
| `epic.go` | 1,938 | 200-300 | 500 |

These targets align with Epic FR6 which specifies no CLI command file may exceed 500 lines.

### NFR-F07-002: Test Coverage for Thin Wrappers

Command handlers, after refactoring, must be testable with mocked services. New or updated CLI tests must:
- Use mocked service implementations (not real database)
- Verify argument parsing produces correct service inputs
- Verify output formatting matches expected JSON/table format
- Verify error handling maps errors to correct exit codes

### NFR-F07-003: No Performance Regression

CLI command execution time must not increase by more than 10% after refactoring.
- Baseline: `shark task next --agent=backend` ~ 50ms
- Acceptable range: 45-55ms after refactoring

The service accessor pattern (creating a new service instance per command) adds negligible overhead compared to direct repository access.

### NFR-F07-004: Incremental Delivery

Each command file (task.go, feature.go, epic.go) must be refactorable independently. The intermediate state where some commands use services and others use direct repository access is explicitly valid. Main branch must remain deployable after each partial refactoring.

---

## Acceptance Criteria (Feature-Level)

### Scenario 1: Task Command File Audit

**Given** the refactoring of `task.go` is complete
**When** a developer audits `internal/cli/commands/task.go`
**Then** the file is at most 400 lines
**And** every handler function follows the pattern: parse → service call → format output
**And** no handler contains repository instantiation, workflow validation, or business calculations
**And** `make lint` passes with no warnings on the file

### Scenario 2: Feature Command File Audit

**Given** the refactoring of `feature.go` is complete
**When** a developer audits `internal/cli/commands/feature.go`
**Then** the file is at most 350 lines
**And** `grep "repository\.New" internal/cli/commands/feature.go` returns no results
**And** `make test` passes all feature command tests

### Scenario 3: Epic Command File Audit

**Given** the refactoring of `epic.go` is complete
**When** a developer audits `internal/cli/commands/epic.go`
**Then** the file is at most 300 lines
**And** `grep "repository\.New" internal/cli/commands/epic.go` returns no results
**And** `make test` passes all epic command tests

### Scenario 4: Regression-Free Output

**Given** a recorded baseline of CLI output for key commands before refactoring
**When** those same commands are run after refactoring with identical inputs
**Then** `shark task next --json` output structure is identical
**And** `shark task start E07-F01-001` success message is identical
**And** `shark feature get E07-F01 --json` output structure is identical
**And** `shark epic get E07 --json` output structure is identical
**And** `make test` exits with code 0

### Scenario 5: Error Code Consistency

**Given** a thin-wrapper command handler
**When** the service returns a `NotFoundError`
**Then** the command exits with code 1
**And** when the service returns a workflow transition error
**Then** the command exits with code 3
**And** when the service returns any other error
**Then** the command exits with code 2

### Scenario 6: Repository Import Absence

**Given** the three refactored command files
**When** their imports are examined
**Then** `internal/cli/commands/task.go` does not import `repository` package
**And** `internal/cli/commands/feature.go` does not import `repository` package
**And** `internal/cli/commands/epic.go` does not import `repository` package

---

## Out of Scope

### Explicitly Excluded

1. **Other CLI Command Files**
   - **What**: Files other than `task.go`, `feature.go`, `epic.go` (e.g., `init.go`, `cloud.go`, `config.go`, `task_deps.go`, `helpers.go`, etc.)
   - **Why**: This feature focuses on the three largest files, which account for 33% of all command code. Other files will be addressed in a future feature or E15-F11.
   - **Workaround**: Continue using direct repository access in non-targeted files during this feature; E15-F11 addresses remaining commands

2. **New Service Methods**
   - **What**: Implementing new TaskService, FeatureService, or EpicService methods not covered by E15-F01 through E15-F06
   - **Why**: This feature consumes existing services; service implementation is in F01-F05. If a needed service method is missing, create a blocking issue.
   - **Workaround**: If a command needs a service method that doesn't exist, escalate to have it added to the service feature before refactoring that command handler

3. **HTTP API Wiring**
   - **What**: Creating HTTP API endpoints that call the service layer
   - **Why**: HTTP API wiring is in E15-F11. This feature's thin wrapper pattern makes HTTP API wiring easy, but the wiring itself is out of scope here.
   - **Workaround**: HTTP API already calls some services; remaining wiring follows naturally from this refactoring

4. **QueryService Implementation**
   - **What**: Building a `QueryService` for smart dispatcher commands (`get.go`, `list.go`, `status.go`)
   - **Why**: Smart dispatchers are separate files with different patterns; this feature targets the three entity command files only
   - **Workaround**: Smart dispatchers continue using existing dispatch logic during this feature

5. **Test File Refactoring**
   - **What**: Updating existing test files to use mocked services
   - **Why**: Test refactoring is incremental; new tests written during this feature use mocked services, existing tests are not required to be rewritten (though improvements are welcome)
   - **Workaround**: Existing tests may continue to use database connections if needed for regression coverage

6. **Command Behavior Changes**
   - **What**: Changing the user-facing behavior of any CLI command as part of this refactoring
   - **Why**: This is a refactoring only. Any discovered behavior improvements must be deferred to separate features.
   - **Workaround**: Document any identified improvements as separate tasks

---

## Success Metrics

### Primary Metrics

1. **Line Count Reduction**
   - **What**: Total lines across the three refactored files
   - **Baseline**: 6,854 lines (task.go: 2,664 + feature.go: 2,252 + epic.go: 1,938)
   - **Target**: ≤1,200 lines total (≤400 + ≤350 + ≤300 + command registration + helpers)
   - **Minimum Acceptable**: ≤1,500 lines total
   - **Measurement**: `wc -l internal/cli/commands/task.go internal/cli/commands/feature.go internal/cli/commands/epic.go | tail -1`

2. **Repository Instantiation Elimination**
   - **What**: Direct `repository.New*` calls in the three targeted files
   - **Baseline**: 351 repository instantiations across all command files (portion in task.go, feature.go, epic.go)
   - **Target**: 0 repository instantiations in task.go, feature.go, epic.go
   - **Measurement**: `grep -c "repository\.New" internal/cli/commands/task.go internal/cli/commands/feature.go internal/cli/commands/epic.go`

3. **Test Suite Health**
   - **What**: All existing tests pass after refactoring
   - **Target**: `make test` exits with code 0
   - **Measurement**: CI/CD pipeline result

### Secondary Metrics

4. **Service Accessor Adoption**
   - **What**: Count of `cli.Get*Service()` calls in the three files
   - **Target**: At least one service accessor call per command handler category (task/feature/epic)
   - **Measurement**: `grep -c "cli\.Get.*Service" internal/cli/commands/task.go internal/cli/commands/feature.go internal/cli/commands/epic.go`

5. **Average Handler Size**
   - **What**: Average lines per command handler function after refactoring
   - **Target**: 15-30 lines per handler
   - **Measurement**: Manual code review count of handler function sizes

---

## Dependencies and Integrations

### Dependencies (Blocking)

- **E15-F05: Epic and Feature Service Expansion** - FeatureService must have progress, health, and action item methods. EpicService must have feature rollup, task rollup, and impediment methods.
- **E15-F06: Repository Layer Cleanup** - Repository layer must be pure data access before this feature lands; removing business logic from commands after removing it from repositories ensures no accidental duplication.
- **E15-F01/F02/F03/F04: TaskService Implementation** - TaskService must implement all task CRUD, lifecycle, querying, and dependency methods that `task.go` currently handles inline.

### Blocked Features

- **E15-F11: Service Layer Completion and CLI Integration** - Integration testing and final validation depend on this feature's completion. E15-F11 verifies the entire Epic E15 refactoring end-to-end.

### Integration Points

- **`internal/cli/services_global.go`**: Existing file with `GetTaskService()`, `GetNoteService()`, `GetContextService()`, `GetResumeService()`. This file is the primary source of service accessors for task commands.
- **`internal/cli/service_accessors.go`**: Existing file with `GetEpicService()`, `GetFeatureService()`, `GetDisplayService()`. This file provides epic and feature accessors. Note: Both files contain `TODO` comments about note repository interface mismatches that must be resolved before accessors can be fully used.
- **`internal/services/task_service.go`**: The TaskService implementation that `task.go` handlers will call.
- **`internal/services/feature_service.go`**: The FeatureService implementation that `feature.go` handlers will call.
- **`internal/services/epic_service.go`**: The EpicService implementation that `epic.go` handlers will call.

---

## Compliance and Security Considerations

**Not Applicable**: This is an internal architectural refactoring with no changes to user-facing behavior, data access patterns, data storage, or security controls. CLI commands continue to authenticate with the same database backend. No new data is read, written, or transmitted.

---

## Open Questions and Assumptions

**Resolved**:

1. **Are service accessors already implemented?** YES. `cli.GetTaskService()`, `cli.GetFeatureService()`, and `cli.GetEpicService()` all exist in `internal/cli/services_global.go` and `internal/cli/service_accessors.go`. Both files have TODO comments about note repository interface mismatches that must be resolved as part of FR-F07-001 (service accessor completeness).

2. **Is this a STANDARD or COMPLEX feature?** STANDARD. The pattern is well-defined, the service infrastructure exists, and the work is repetitive application of the thin wrapper pattern rather than novel design. The main challenge is thoroughness and regression avoidance, not architectural novelty.

3. **Can command files be refactored independently?** YES. NFR-F07-004 explicitly permits intermediate states where some commands use services and others still use direct repository access. This enables incremental delivery with smaller PRs.

**Remaining Assumptions**:

- TaskService, FeatureService, and EpicService have all methods needed by the three command files (covered by F01-F06). If methods are missing, the dependency (E15-F05 or earlier) must be updated before the affected handler can be refactored.
- The note repository interface mismatch flagged in TODO comments in `service_accessors.go` is minor and can be resolved during this feature without requiring changes to the service interface design.
- Output format preservation is verified by running the existing test suite plus a manual comparison of JSON output for representative commands before and after.

---

*Last Updated*: 2026-02-18
