# E22 Brownfield Research Report: External Orchestration Runner

## Executive Summary

The `shark run <entity-key>` command can be built almost entirely by composing existing infrastructure. The orchestrator action system (E07-F21, E16-F02), template rendering engine (E07-F30), workflow service (E16-F01), and entity service layer (E15, E21) already provide all the data retrieval, template population, status transition, and action resolution needed for the run loop. The only genuinely new code required is: (1) the `os/exec` agent dispatcher interface and two implementations (Claude CLI, Codex CLI), (2) the run loop controller that reads orchestrator actions and dispatches agents, and (3) the Cobra command registration. No new database tables, no new repository methods, and no new service interfaces are required for the core loop.

---

## 1. Existing Implementations Relevant to E22

### 1.1 Orchestrator Action System

The orchestrator action system is the backbone of `shark run`. It already defines what action to take for each workflow status.

**Key files:**

| File | Purpose | Line Count |
|------|---------|------------|
| `internal/config/orchestrator_action.go` | `OrchestratorAction` struct with `Action`, `AgentType`, `Provider`, `Model`, `Skills`, `InstructionTemplate` fields. Validation. `PopulateTemplate()` method. | 245 lines |
| `internal/config/action_service.go` | `ActionService` interface and `DefaultActionService` implementation. `GetStatusAction()`, `GetStatusActionPopulated()`, `GetAllActions()`, `ValidateActions()`. | 210 lines |
| `internal/config/workflow_schema.go` | `WorkflowConfig`, `StatusMetadata` with `OrchestratorAction *OrchestratorAction` field. Status flow, special statuses, phase ordering. | 291 lines |
| `internal/config/template_helpers.go` | `TaskPlaceholders()`, `FeaturePlaceholders()`, `EpicPlaceholders()`, `BugPlaceholders()`, `ChangeCardPlaceholders()` for template variable generation. | ~300 lines |
| `internal/config/mock_action_service.go` | Mock implementation for testing. | ~50 lines |

**Action type constants already defined** (`internal/config/orchestrator_action.go:36-56`):
- `ActionSpawnAgent` = `"spawn_agent"`
- `ActionPause` = `"pause"`
- `ActionWaitForTriage` = `"wait_for_triage"`
- `ActionArchive` = `"archive"`
- `ActionAdvanceStatus` = `"advance_status"`
- `ActionCheckOrResume` = `"check_or_resume"`
- `ActionCascade` = `"cascade"`

**`PopulatedAction` struct** (`internal/config/action_service.go:33-40`): Already contains the exact fields `shark run` needs to dispatch agents: `Action`, `AgentType`, `Provider`, `Model`, `Skills`, `Instruction` (rendered template).

**Template rendering** (`internal/config/orchestrator_action.go:158-184`): `PopulateTemplate()` already supports both `.tmpl` file references (routed to the template engine) and inline string replacement. The template engine at `internal/templates/` handles Go template syntax with partials.

### 1.2 Workflow Service

**Key files:**

| File | Purpose |
|------|---------|
| `internal/workflow/service.go` | `GetValidTransitions()`, `IsValidTransition()`, `IsValidStatus()`, `GetTransitionInfo()`, `GetStatusMetadata()` |

**`GetValidTransitions()`** (`internal/workflow/service.go:249-257`): Returns the list of valid next statuses from the status flow. The run loop needs this to determine what status to advance to after an agent completes.

**Multi-level workflow support**: The workflow service supports `ForLevel()` scoping for epic/feature/task level workflows. The `.sharkworkflow.json` file already defines separate `epic_workflow`, `feature_workflow`, and `task_workflow` sections.

### 1.3 Entity Service and Status Transitions

**Key files:**

| File | Purpose |
|------|---------|
| `internal/services/entity_service.go` | `EntityService.TransitionStatus()` -- the unified status transition engine used by all entity types |
| `internal/services/transition_types.go` | `TransitionOptions`, `TransitionResult`, `TransitionInfoWithAction` structs |
| `internal/services/task_service.go` | `TaskService.TransitionStatus()` -- delegates to `EntityService` |
| `internal/services/feature_service.go` | `FeatureService.TransitionStatus()` -- delegates to `EntityService` |
| `internal/services/epic_service.go` | `EpicService.TransitionStatus()` -- delegates to `EntityService` |

**`EntityService.TransitionStatus()`** (`internal/services/entity_service.go:168-249`): This is the method `shark run` will call to advance status. It already:
1. Gets entity by key
2. Validates the transition
3. Updates status in database
4. Records entity history
5. Creates rejection notes (for backward transitions)
6. Resolves orchestrator action for the new status
7. Returns `TransitionResult` with the populated action

**`TransitionResult`** (`internal/services/transition_types.go:44-58`): Already contains `OrchestratorAction *config.PopulatedAction` -- this is exactly what `shark run` reads to determine the next dispatch.

### 1.4 Entity Models and Polymorphism

**Key files:**

| File | Purpose |
|------|---------|
| `internal/models/entity.go` | `Entity` interface with `GetKey()`, `GetStatus()`, `GetEntityType()`, etc. |
| `internal/models/task.go` | `Task` struct implementing `Entity` |
| `internal/models/feature.go` | `Feature` struct implementing `Entity` |
| `internal/models/epic.go` | `Epic` struct implementing `Entity` |
| `internal/models/bug.go` | `Bug` struct implementing `Entity` |
| `internal/models/change_card.go` | `ChangeCard` struct implementing `Entity` |

The `Entity` interface enables `shark run` to operate polymorphically across entity types.

### 1.5 CLI Command Registration Pattern

**Key files:**

| File | Purpose |
|------|---------|
| `internal/cli/commands/status_group.go` | `runStatusAdvance()` -- reference implementation for status advancement via CLI |
| `internal/cli/commands/commands.go` | Command tree registration |
| `internal/cli/service_accessors.go` | Service accessor wiring adapters |
| `internal/cli/services_global.go` | `GetTaskService()`, `GetFeatureService()`, `GetEpicService()`, etc. |

**`runStatusAdvance()`** (`internal/cli/commands/status_group.go:408+`): This is the closest existing code to what `shark run` does per-iteration. It parses an entity key, detects entity type, dispatches to the correct service, and calls `TransitionStatus()`. The run loop wraps this in a loop with agent dispatch between transitions.

### 1.6 Template System

**Key files:**

| File | Purpose |
|------|---------|
| `shark-templates/task/*.tmpl` | Per-status workflow instruction templates (19 files for advanced workflow) |
| `shark-templates/feature/*.tmpl` | Feature-level templates |
| `shark-templates/epic/*.tmpl` | Epic-level templates |
| `shark-templates/partials/` | Shared template partials (e.g., `_tdd_process`) |
| `internal/templates/` | Template engine with `Render()`, `GetOrchestratorEngine()` |

The templates already contain the full agent instructions for each workflow stage. For example, `shark-templates/task/ready_for_development.tmpl` contains the complete developer instructions including TDD process, exit gates, and status advance commands.

### 1.7 Existing `os/exec` Usage

**`internal/view/service.go:147`**: Already uses `exec.CommandContext()` to launch external processes. This provides a pattern reference for how to invoke CLI tools with context support, stdin/stdout/stderr wiring.

---

## 2. Patterns and Conventions That Must Be Followed

### 2.1 CLI Command Pattern
- Commands are thin wrappers: parse args, call service, format output
- Register via `init()` function
- Use `cli.GlobalConfig.JSON` for JSON output
- Use `cli.Success()`, `cli.Error()`, `cli.Warning()` for human output
- Accept `context.Context` for cancellation

### 2.2 Service Layer Pattern
- Business logic in services, not commands
- Constructor injection for dependencies
- `context.Context` as first parameter
- Return domain models and errors
- Error wrapping with business context: `fmt.Errorf("failed to ...: %w", err)`

### 2.3 Global Service Accessor Pattern
- Defined in `internal/cli/services_global.go`
- Lazy initialization with `sync.Once` or per-call creation
- Panic on DB failure for CLI entry points
- Example: `GetTaskService()`, `GetEpicService()`, etc.

### 2.4 Error Handling
- Exit code 0: success
- Exit code 1: not found
- Exit code 2: database error
- Exit code 3: invalid state
- Use custom error types (`NotFoundError`, `BackwardReasonError`)

### 2.5 Testing
- Service tests use mocked repositories
- CLI tests use mocked services
- Only repository tests use real database
- Table-driven tests for multiple scenarios

---

## 3. Integration Points

### 3.1 Services the Run Loop Will Call

| Service | Method | Purpose |
|---------|--------|---------|
| `TaskService` | `TransitionStatus(ctx, key, targetStatus, opts)` | Advance task status |
| `FeatureService` | `TransitionStatus(ctx, key, targetStatus, opts)` | Advance feature status |
| `EpicService` | `TransitionStatus(ctx, key, targetStatus, opts)` | Advance epic status |
| `ActionService` | `GetStatusActionPopulated(ctx, status, vars)` | Get orchestrator action for current status |
| `workflow.Service` | `GetValidTransitions(currentStatus)` | Determine valid next statuses |
| `workflow.Service` | `IsTerminalStatus(status)` | Check if entity is done |

### 3.2 Configuration
- `.sharkconfig.json`: Workflow configuration with `status_metadata` containing `orchestrator_action` per status
- `.sharkworkflow.json`: Separate workflow definition file with multi-level support
- `shark-templates/`: Template files referenced by `instruction_template` field

### 3.3 Entity Key Parsing
- `internal/cli/commands/helpers.go`: `ParseGetArgs()` -- detects entity type from key format (E##, E##-F##, E##-F##-###, B###, CC-###)
- Existing entity type constants in `internal/models/entity.go`

### 3.4 Template Variable Generation
- `internal/config/template_helpers.go`: `TaskPlaceholders()`, `FeaturePlaceholders()`, `EpicPlaceholders()` generate the `map[string]string` needed by `PopulateTemplate()`
- Enriched variants with related docs: `TaskPlaceholdersWithRelated()`, etc.

---

## 4. What Can Be EXTENDED vs What Needs NEW Code

### 4.1 EXTEND Existing Code (Zero or Minimal Changes)

| Component | What Exists | How to Use |
|-----------|-------------|------------|
| `OrchestratorAction` struct | Full definition with `Provider`, `Model`, `AgentType`, `Skills` | Read directly -- all fields already present for dispatch |
| `ActionService.GetStatusActionPopulated()` | Returns `PopulatedAction` with rendered instruction | Call from run loop to get dispatch instructions |
| `TransitionStatus()` (all services) | Full status transition with history, notes, validation | Call after agent success to advance status |
| `GetValidTransitions()` | Returns valid next statuses | Use to determine target status for advancement |
| `PopulateTemplate()` | Template rendering with `.tmpl` and inline support | Already called by `GetStatusActionPopulated()` |
| `ParseGetArgs()` | Entity type detection from key | Reuse for run command argument parsing |
| Service accessors | `GetTaskService()`, `GetEpicService()`, etc. | Call from run command to get services |
| Template system | 19 task templates, epic templates, feature templates | Templates are the agent instructions -- used as-is |

**Why these cannot be new code**: All the data retrieval, template rendering, status transition, and action resolution logic already exists and is tested. Duplicating any of it would violate DRY and risk inconsistency.

### 4.2 NEW Code Required

| Component | Why Existing Code Cannot Cover It | Estimated Size |
|-----------|-----------------------------------|----------------|
| **Agent Dispatcher Interface** | No existing concept of invoking external CLI processes as agents. The `view/service.go` uses `os/exec` but for a completely different purpose (opening files in a viewer). The dispatcher needs process management, stdout/stderr capture, exit code handling, and tool-specific flag construction. | ~50 lines (interface + types) |
| **Claude CLI Dispatcher** | Claude CLI has specific flags (`-p`, `--disallowedTools`, `--allowedTools`, `--max-turns`, `--output-format`) that don't exist anywhere in the codebase. | ~80 lines |
| **Codex CLI Dispatcher** | Codex CLI has different flags (`exec`, `-m`, `-s`, `--skip-git-repo-check`, `-c`). No overlap with any existing code. | ~60 lines |
| **Run Loop Controller** | The orchestration loop (read action -> dispatch agent -> check exit code -> advance status -> repeat) is a fundamentally new control flow pattern. Nothing in the codebase loops through workflow statuses driving external processes. The closest analog is `runStatusAdvance()` but that's a single-shot operation, not a loop. | ~150 lines |
| **CLI Tool Validation** | Checking `claude`/`codex` binary availability on PATH via `exec.LookPath()`. Trivial but new. | ~20 lines |
| **Run Command (Cobra)** | New `shark run <key>` subcommand registration. Standard pattern, follows existing commands. | ~60 lines |
| **Run Logging** | Structured logging of each stage with timing, exit codes, output capture. No existing logging infrastructure for multi-stage run tracking. | ~80 lines |
| **Worktree Management** (Should Have) | Creating/managing git worktrees for agent isolation. No git worktree code exists in the codebase. Would use `os/exec` to call `git worktree add/remove`. | ~80 lines |

**Total new code estimate**: ~580 lines for Must Have, ~660 lines including Should Have.

### 4.3 Minor Extensions to Existing Code

| Component | Change Needed | Reason |
|-----------|---------------|--------|
| `services_global.go` | Add `GetActionService()` accessor | No global accessor currently exists for `ActionService`; commands that use it create it inline via `config.NewActionService(configPath)` |
| `status_group.go` | None | Run command is a separate command, not a modification |
| `OrchestratorAction` struct | Possibly add `MaxTurns`, `AllowedTools`, `DisallowedTools` fields | For Claude CLI flag passthrough. Alternative: encode in template or config. |

---

## 5. Technical Risks and Feasibility Assessment

### 5.1 Feasibility: CONFIRMED

All core technical requirements are feasible with the existing codebase:

- **Status advancement**: Fully supported via `TransitionStatus()` on all entity services
- **Orchestrator action reading**: Fully supported via `ActionService.GetStatusActionPopulated()`
- **Template rendering**: Fully supported via existing template engine
- **Entity polymorphism**: Supported via `Entity` interface and entity type detection
- **Multi-level workflows**: Supported via `ForLevel()` scoping

### 5.2 Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| **Claude CLI `--disallowedTools` flag format changes** | Low | Medium | Abstract behind dispatcher interface; flag construction is isolated to ~5 lines |
| **Codex CLI interface changes** | Medium | Medium | Same mitigation -- dispatcher interface isolates CLI specifics |
| **Template instruction quality** | Medium | High | Not a code risk but an operational risk. Templates must include verification steps. `shark run` enforces execution, not work quality |
| **Long-running agent processes** | Low | Medium | Use `context.WithTimeout()` or `--max-turns` in Claude CLI. `os/exec` supports context cancellation |
| **Agent exits 0 without completing work** | Medium | Medium | Architectural limit -- `shark run` enforces execution, not correctness. Mitigated by red-team stages (Codex QA) |
| **E21 EntityRegistry dependency** | Medium | Low | E21 is in progress on branch E21. If E22 starts before E21 merges, the run command can use per-entity-type service accessors directly (`GetTaskService()`, etc.) instead of the registry. Adapter can be added later. |

### 5.3 No Showstoppers Identified

Every component in the requirements maps to either existing code or straightforward new Go code using `os/exec`. No external dependencies beyond `claude` and `codex` CLI tools are needed. No database schema changes are required.

---

## 6. Recommended Implementation Approach

### 6.1 Architecture: Dispatcher Interface Pattern

```
shark run <key>
  |
  v
RunController (new)
  |-- reads entity status via services (existing)
  |-- reads orchestrator action via ActionService (existing)
  |-- dispatches to AgentDispatcher interface (new)
  |     |-- ClaudeDispatcher (new)
  |     |-- CodexDispatcher (new)
  |-- advances status via TransitionStatus (existing)
  |-- loops until terminal/pause/failure
```

### 6.2 Recommended File Organization

```
internal/
  runner/                    # NEW package for run loop
    runner.go                # RunController: the orchestration loop
    dispatcher.go            # AgentDispatcher interface + types
    claude_dispatcher.go     # Claude CLI implementation
    codex_dispatcher.go      # Codex CLI implementation
    runner_test.go           # Tests with mocked dispatchers
    dispatcher_test.go       # Dispatcher unit tests
  cli/commands/
    run.go                   # NEW: shark run command registration
```

### 6.3 Dependency Wiring

The `RunController` needs:
- `TaskService` / `FeatureService` / `EpicService` (for `TransitionStatus()`)
- `ActionService` (for `GetStatusActionPopulated()`)
- `workflow.Service` (for `GetValidTransitions()`, `IsTerminalStatus()`)
- `AgentDispatcher` (Claude or Codex, selected per action)

All except `AgentDispatcher` already exist as service accessors.

### 6.4 Key Design Decision: First Valid Transition as Default

The run loop needs to pick the "next" status when advancing. The existing `runStatusAdvance()` uses the first valid transition from `GetValidTransitions()`. The run loop should follow this same convention, using `transitions[0]` as the default forward status.

### 6.5 Implementation Order

1. **Agent Dispatcher Interface + Claude Implementation** (Must Have, no dependencies)
2. **Run Loop Controller** (Must Have, depends on #1)
3. **Cobra Command Registration** (Must Have, depends on #2)
4. **Codex Dispatcher** (Must Have, depends on #1 interface)
5. **Run Logging** (Should Have, enhances #2)
6. **Worktree Support** (Should Have, enhances #2)
7. **Dry Run Mode** (Could Have)
8. **Feature-Level Run** (Could Have)

---

## 7. References

### Existing Epics and Features

- **E07-F21**: Add Actions to Status Transition (orchestrator action system)
- **E16-F01**: Core Workflow Engine (multi-level workflow)
- **E16-F02**: Orchestrator Actions (action service, validation)
- **E07-F30**: Template Engine (template rendering)
- **E15**: Service Layer Architecture Refactoring
- **E21**: Entity Polymorphism and Duplication Reduction (in progress)
- **E17**: CLI Simplification for AI Agents

### Key Source Files (Absolute Paths from Project Root)

- `internal/config/orchestrator_action.go` -- OrchestratorAction struct, PopulateTemplate(), action constants
- `internal/config/action_service.go` -- ActionService interface, DefaultActionService, GetStatusActionPopulated()
- `internal/config/workflow_schema.go` -- WorkflowConfig, StatusMetadata, status flow
- `internal/config/template_helpers.go` -- TaskPlaceholders(), FeaturePlaceholders(), EpicPlaceholders()
- `internal/services/entity_service.go` -- EntityService.TransitionStatus()
- `internal/services/transition_types.go` -- TransitionOptions, TransitionResult, PopulatedAction reference
- `internal/services/task_service.go` -- TaskService.TransitionStatus()
- `internal/services/feature_service.go` -- FeatureService.TransitionStatus()
- `internal/services/epic_service.go` -- EpicService.TransitionStatus()
- `internal/workflow/service.go` -- GetValidTransitions(), IsValidTransition(), IsValidStatus()
- `internal/models/entity.go` -- Entity interface
- `internal/cli/commands/status_group.go` -- runStatusAdvance() reference implementation
- `internal/cli/services_global.go` -- Service accessor pattern
- `internal/cli/commands/helpers.go` -- ParseGetArgs() entity type detection
- `internal/view/service.go` -- Existing os/exec usage pattern
- `shark-templates/task/*.tmpl` -- 19 per-status agent instruction templates

---

*Research completed: 2026-03-21*
