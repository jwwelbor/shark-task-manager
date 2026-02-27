# E17 Research Report: CLI Simplification for AI Agents

## Executive Summary

This report assesses the feasibility, system-wide impact, and competitive landscape for Epic E17, which aims to reduce Shark's CLI surface from ~45 command paths to ~25, optimized for AI agent consumption. Analysis of 231 real agent interactions confirms the core pain points: unpredictable command discovery (36% defensive error suppression), missing field extraction (15% piping through Python), and inconsistent flag naming. All six Phase 1 features (F01-F06) are technically feasible with the current Go/Cobra stack, and the existing service layer (E15) provides the foundation needed. The primary risk is the `status` namespace collision between the existing smart dispatcher (`shark status <id>`) and the proposed `shark status set/advance` subcommand group, which the phased approach handles through coexistence in Phase 1 and resolution via `shark progress` (F06) in Phase 2.

---

## 1. Market and Competitive Landscape

### 1.1 Industry Context: CLI Tools for AI Agents

The 2025-2026 landscape shows a clear shift toward **tool-first design** for AI agents interacting with CLI tools. Key industry patterns include:

- **Structured output as API contract**: Modern CLI tools treat JSON output as a first-class API, not an afterthought. GitHub CLI (`gh`) established the pattern with `--json` field selection and `--jq` built-in filtering.
- **Machine-readable errors**: Production agent frameworks recommend structured error objects with type, message, and code fields, enabling programmatic error handling without string parsing.
- **Environment-based configuration**: Session-wide output modes (like `SHARK_OUTPUT=json`) align with how AI agents configure their tool environments.
- **Schema-driven outputs**: The Model Context Protocol (MCP) specification recommends tools provide output schemas for validation of structured results.

### 1.2 Competitive Analysis

| Tool | JSON Output | Field Extraction | Structured Errors | Env Config | Agent-Oriented |
|------|-------------|------------------|--------------------|------------|----------------|
| **GitHub CLI (gh)** | `--json` with field selection | `--jq` built-in | Partial (stderr) | `GH_*` env vars | Yes (scripting guide) |
| **Linear CLI** | API-first JSON | GraphQL field selection | Structured API errors | Token-based | Moderate |
| **Jira CLI** | `--output json` | Limited | Unstructured stderr | `JIRA_*` env vars | Low |
| **Shark (current)** | `--json` flag | None | Unstructured stderr | `PM_*` prefix (limited) | Low |
| **Shark (E17 target)** | `--json` + `SHARK_OUTPUT` | `--field` extraction | Structured JSON errors | `SHARK_OUTPUT` env var | High |

**Key competitive insight**: GitHub CLI's `--json` + `--jq` pattern is the de facto standard for agent-friendly CLI tools. The `gh` approach of requiring field names with `--json` is a documented limitation that users have requested be relaxed. Shark's proposed `--field` flag (F02) provides a simpler, more direct alternative that does not require jq knowledge -- a significant advantage for AI agents that benefit from single-step extraction over multi-step piping.

### 1.3 Market Gaps Addressed by E17

1. **No Python post-processing**: Currently 15% of agent interactions pipe through Python for field extraction. The `--field` flag eliminates this entirely.
2. **Predictable error handling**: 36% of interactions use `2>/dev/null` to suppress unparseable errors. Structured JSON errors with error codes let agents handle failures programmatically.
3. **Session-wide JSON mode**: Agents currently must pass `--json` on every invocation. `SHARK_OUTPUT=json` reduces this to a one-time configuration.
4. **Unified status commands**: Status operations are the second-most-common agent action (~35% of invocations) yet are scattered across `shark task start/complete/approve`, `shark epic set-status`, and `shark epic next-status`. Consolidation into `shark status set/advance` matches how agents think about the operation.

---

## 2. Feasibility Assessment by Feature Area

### 2.1 F01: Status Subcommand Group

**Assessment: FULLY FEASIBLE -- Low complexity**

**What exists today**:
- `shark epic set-status <key> <status>` (`/home/jwwel/projects/shark-task-manager/internal/cli/commands/epic_set_status.go`) -- already uses service layer via `cli.GetEpicService().TransitionStatus()`
- `shark feature set-status <key> <status>` (`/home/jwwel/projects/shark-task-manager/internal/cli/commands/feature_set_status.go`) -- identical pattern
- `shark epic next-status <key>` (`/home/jwwel/projects/shark-task-manager/internal/cli/commands/epic_next_status.go`) -- uses `epicSvc.GetNextStatus()` and shared `performEntityTransition()`
- `shark feature next-status <key>` (`/home/jwwel/projects/shark-task-manager/internal/cli/commands/feature_next_status.go`) -- mirror of epic pattern
- `shark task start/complete/approve/block/unblock` -- task-specific lifecycle commands
- Shared types: `EntityNextStatusResult`, `EntityTransitionChoice`, `buildNextStatusResult()`, `performEntityTransition()` in `epic_next_status.go`
- Service types: `TransitionOptions`, `TransitionResult`, `NextStatusInfo` in `/home/jwwel/projects/shark-task-manager/internal/services/transition_types.go`

**What F01 adds**:
- `shark status set <key> <status>` -- auto-detect entity type, delegate to existing `TransitionStatus()` service methods
- `shark status advance <key>` -- auto-detect entity type, delegate to existing `GetNextStatus()` + `TransitionStatus()`
- `shark status options <key>` -- extract the preview logic from `epic_next_status.go`
- `shark status history <key>` -- wrap existing history service calls

**Implementation approach**:
- Create a new `shark status` subcommand group using Cobra's `AddCommand()` pattern
- Auto-detect entity type from key format (same pattern used by existing smart dispatchers in `get.go`, `list.go`)
- Delegate to existing `EpicService.TransitionStatus()`, `FeatureService.TransitionStatus()`, or `TaskService.StartTask()/CompleteTask()` depending on entity type
- The `entityTransitioner` interface in `epic_next_status.go` already abstracts over entity types for transitions

**Namespace conflict**: The existing `shark status` command (`/home/jwwel/projects/shark-task-manager/internal/cli/commands/status.go`) is a dashboard/progress display. In Phase 1, both coexist (Cobra allows a parent command with both a `RunE` handler and subcommands). In Phase 2, F06 introduces `shark progress` to replace the dashboard use case, and `shark status` becomes exclusively the subcommand group.

**Risks**: None. The service layer methods already exist. This is primarily a CLI routing exercise.

### 2.2 F02: --field Flag for Targeted Extraction

**Assessment: FULLY FEASIBLE -- Medium complexity**

**What exists today**:
- `cli.OutputJSON(data)` in `/home/jwwel/projects/shark-task-manager/internal/cli/root.go` uses `json.MarshalIndent()` to serialize any struct
- JSON output is already supported on all `get`, `list`, `next`, and status commands
- The `formatters` package (`/home/jwwel/projects/shark-task-manager/internal/formatters/json.go`) defines response types like `EpicWithProgress`, `FeatureWithTaskCount`, etc.
- No field extraction capability exists today

**What F02 adds**:
- `--field <name>` flag that extracts a single field value from the JSON response
- Raw value output (no JSON wrapping) for direct use in scripts
- Exit code 1 for missing field
- Works on `get`, `list`, `next`, `progress`, `status options`

**Implementation approach**:
1. Marshal the result to `map[string]interface{}` using `json.Marshal` then `json.Unmarshal`
2. Look up the requested field name in the map
3. For nested fields, support dot notation (e.g., `--field status_breakdown.todo`)
4. Output the raw value (string, number) or re-marshal (object, array)
5. Add `--field` as a persistent flag on the root command or on specific command groups

**Technical considerations**:
- Go struct field names use PascalCase but JSON tags use snake_case. Field lookup must use JSON tag names since that is what agents see in output.
- For list commands, `--field` should extract the field from each item (array of values) or require `--field` on single-entity commands only.
- Exit code 1 for missing field conflicts with the existing "not found" exit code 1 convention. The PRD should clarify whether a new exit code (e.g., 4) should be used for "field not found" vs "entity not found".

**Risks**: Low. The main design decision is how `--field` interacts with list commands (extract from each item vs. error).

### 2.3 F03: Structured JSON Error Output

**Assessment: FULLY FEASIBLE -- Medium complexity**

**What exists today**:
- Error formatting in `/home/jwwel/projects/shark-task-manager/internal/cli/commands/errors.go` -- functions like `NotFoundError()`, `InvalidStatusTransitionError()`, `AmbiguousKeyError()` return `fmt.Errorf()` with multi-line human-readable strings
- Exit codes: 0 (success), 1 (not found), 2 (DB error), 3 (invalid state)
- Errors go to stderr via `cli.Error()` (human-readable text)
- Service layer uses typed errors: `ErrReasonRequired`, `ErrForceReasonRequired`, `BackwardReasonError` in `/home/jwwel/projects/shark-task-manager/internal/services/transition_types.go`
- Repository layer has `NotFoundError` struct type

**What F03 adds**:
- In JSON mode, errors output as structured JSON to **stdout** (not stderr)
- Format: `{"error": true, "code": "NOT_FOUND", "message": "...", "entity": "...", "current_status": "...", "valid_transitions": [...]}`
- Consistent error codes: `NOT_FOUND`, `INVALID_TRANSITION`, `VALIDATION_ERROR`, `DATABASE_ERROR`, `PERMISSION_DENIED`

**Implementation approach**:
1. Create a `CLIError` struct with JSON tags matching the PRD format
2. In `cli.Error()` (or a new `cli.ErrorJSON()`), check `cli.GlobalConfig.JSON` and emit JSON to stdout if true
3. Map existing typed errors (`NotFoundError`, `BackwardReasonError`, etc.) to error codes
4. For `INVALID_TRANSITION`, include `current_status` and `valid_transitions` from the workflow service
5. Intercept errors in Cobra's `PersistentPostRunE` or in a wrapper around `RunE` handlers

**Technical considerations**:
- Switching errors from stderr to stdout in JSON mode is a breaking change for scripts that parse stdout for data and stderr for errors. The PRD explicitly requires this for agent simplicity (one stream to parse).
- The existing `InvalidStatusTransitionError()` in `errors.go` already includes allowed transitions in its human-readable output. This data just needs to be structured as JSON instead.

**Risks**: Low. The typed error infrastructure exists. The main work is the JSON serialization layer.

### 2.4 F04: SHARK_OUTPUT Environment Variable

**Assessment: FULLY FEASIBLE -- Low complexity**

**What exists today**:
- `cli.GlobalConfig.JSON` is set by the `--json` flag in `/home/jwwel/projects/shark-task-manager/internal/cli/root.go`
- Viper configuration uses the `PM` env prefix (line in root.go: `v.SetEnvPrefix("PM")`)
- No environment variable currently controls output format

**What F04 adds**:
- `SHARK_OUTPUT=json` environment variable that sets `cli.GlobalConfig.JSON = true` for the entire session
- `--json` flag overrides the env var (explicit > implicit)

**Implementation approach**:
1. In `root.go`'s `PersistentPreRunE`, check `os.Getenv("SHARK_OUTPUT")`
2. If value is `"json"`, set `cli.GlobalConfig.JSON = true`
3. The `--json` flag already takes precedence since Cobra flags are parsed after PersistentPreRun

**Technical considerations**:
- The current env prefix is `PM`, not `SHARK`. F04 proposes `SHARK_OUTPUT` which breaks the prefix convention. Two options: (a) use `SHARK_OUTPUT` as specified (clearer for agents), or (b) use `PM_OUTPUT` for consistency. Recommendation: use `SHARK_OUTPUT` as specified, since it is more discoverable and the `PM` prefix is an internal detail.
- This is a 5-10 line change in `root.go`. Extremely low risk.

**Risks**: None.

### 2.5 F05: Flag Normalization

**Assessment: FULLY FEASIBLE -- Low complexity**

**What exists today**:
- `--execution-order` on task create (`/home/jwwel/projects/shark-task-manager/internal/cli/commands/create.go`, line 130-131): both `--execution-order` and `--order` already exist as aliases
- `--show-all` on task list commands
- Flag definitions spread across individual command files and `shared_flags.go`
- Shared flag infrastructure in `/home/jwwel/projects/shark-task-manager/internal/cli/commands/shared_flags.go` with `AddFlagSet()` pattern

**What F05 adds**:
- `--order` as the primary name (deprecate `--execution-order`)
- `--all` as alias for `--show-all`
- Consistent naming convention across all commands

**Implementation approach**:
1. For `--execution-order`: already has `--order` alias. Just swap which is primary and mark `--execution-order` as deprecated via Cobra's `MarkDeprecated()`
2. For `--show-all`: add `--all` alias, mark `--show-all` as deprecated
3. Use Cobra's built-in deprecation messaging: `cmd.Flags().MarkDeprecated("show-all", "use --all instead")`

**Risks**: None. Cobra has built-in flag deprecation support that shows warnings without breaking backward compatibility.

### 2.6 F06: Progress Command

**Assessment: FULLY FEASIBLE -- Medium complexity**

**What exists today**:
- `shark status [EPIC] [FEATURE]` in `/home/jwwel/projects/shark-task-manager/internal/cli/commands/status.go` -- a dashboard command that shows progress rollups, active tasks, and blocked items
- Uses `cli.GetStatusService().GetDashboard()` with `status.StatusRequest` for filtering
- `status.FormatDashboard()` for terminal rendering, `json.MarshalIndent()` for JSON output
- The `internal/status/` package provides `StatusDashboard`, `EpicSummary`, progress calculation, health indicators

**What F06 adds**:
- `shark progress` command that takes over the "show me progress/dashboard" use case from `shark status`
- Frees `shark status` to be exclusively the subcommand group for status transitions
- Same functionality, different command name

**Implementation approach**:
1. Copy/adapt `status.go` to `progress.go` with `Use: "progress [EPIC] [FEATURE]"`
2. Register `progressCmd` with the root command
3. In Phase 2, deprecate `shark status` (without subcommands) in favor of `shark progress`
4. In Phase 3, remove the `RunE` handler from `statusCmd`, leaving only subcommands

**Risks**: Low. This is a naming change, not a logic change. The service layer (`StatusService.GetDashboard()`) remains unchanged.

---

## 3. System-Wide Impact: Interactions with Other Epics

### 3.1 E15: Service Layer Architecture Refactoring

**Dependency type**: E17 depends on E15 (prerequisite)

**Current state**: E15 is in progress. The service layer is partially built:
- `TaskService`, `FeatureService`, `EpicService` exist in `/home/jwwel/projects/shark-task-manager/internal/services/`
- Service accessor pattern (`cli.GetTaskService()`, etc.) is established in `services_global.go`
- `TransitionStatus()`, `GetNextStatus()` methods exist on both `EpicService` and `FeatureService`
- Task lifecycle methods (`StartTask`, `CompleteTask`, `ApproveTask`, `BlockTask`, etc.) exist on `TaskService`

**Impact analysis**:
- **F01 (status subcommands)**: Depends on `EpicService.TransitionStatus()`, `FeatureService.TransitionStatus()`, and task lifecycle methods. These all exist today. No E15 blocker.
- **F02 (--field flag)**: No service layer dependency. Operates on JSON output, which is a presentation concern.
- **F03 (structured errors)**: No service layer dependency. Operates at the CLI error handling layer. However, it benefits from the typed error infrastructure that E15 establishes.
- **F04 (SHARK_OUTPUT env)**: No service layer dependency. Root command configuration only.
- **F05 (flag normalization)**: No service layer dependency. Flag definitions only.
- **F06 (progress command)**: Depends on `StatusService.GetDashboard()`, which already exists.

**Conclusion**: E17 Phase 1 has **no E15 blockers**. All required service methods exist. E17 commands should follow E15's patterns (thin commands calling services) for consistency. If any new service methods are needed for F01's auto-detect logic, they should be created following E15 conventions.

### 3.2 E16: Multi-Level Workflow

**Dependency type**: E17 benefits from E16 (enhancer, not blocker)

**Current state**: E16 is planned but not started. It will extend workflow configuration to epic and feature levels with distinct `epic_workflow` and `feature_workflow` sections in `.sharkconfig.json`.

**Impact analysis**:
- **F01 (status subcommands)**: Designed to be E16-compatible. The `shark status set/advance` commands use `workflow.Service` for validation. When E16 adds level-specific workflows, the commands will automatically use the correct workflow for each entity type because the service layer already scopes by level (`workflowSvc.ForLevel(workflow.LevelEpic)`).
- **F01 `status options`**: Returns available transitions from `GetNextStatus()`. When E16 enriches the transition data with descriptions, phases, and agent assignments, `status options` automatically reflects this.
- **No breaking changes**: E16 is additive to the workflow configuration. E17's status commands that delegate to the workflow service will pick up E16 enhancements transparently.

**Conclusion**: E17 should be designed with E16 in mind but does **not need to wait** for E16. The current `workflow.Service.ForLevel()` pattern provides the abstraction layer.

### 3.3 Other Epics

No other epics under `docs/plan/` have significant interaction with E17. The CLI simplification is an additive layer that does not modify database schema, service contracts, or repository interfaces.

---

## 4. Existing Capability Overlap with Defined Scope

### 4.1 Direct Overlap (Already Implemented)

| E17 Requirement | Existing Implementation | Gap |
|-----------------|------------------------|-----|
| Entity-type auto-detection from key | Smart dispatchers in `get.go`, `list.go`, `status.go` | Needs reuse in F01's `status set/advance` |
| Epic status transitions | `shark epic set-status` via `EpicService.TransitionStatus()` | Needs unified `shark status set` entry point |
| Feature status transitions | `shark feature set-status` via `FeatureService.TransitionStatus()` | Needs unified `shark status set` entry point |
| Next-status advancement | `shark epic/feature next-status` via `GetNextStatus()` + `performEntityTransition()` | Needs unified `shark status advance` entry point |
| Transition preview | `--preview` flag on `epic/feature next-status` | Rename to `shark status options` |
| `--order` alias | Already exists as alias for `--execution-order` on task create | Just swap primary/deprecated names |
| Unified `create` dispatcher | `shark create epic/feature/task` exists in `create.go` | Fully implemented |
| JSON output on all commands | `--json` flag and `cli.OutputJSON()` | Universal, no gap |
| Shared transition types | `TransitionOptions`, `TransitionResult`, `NextStatusInfo` | Ready for F01 |
| Shared flag infrastructure | `shared_flags.go` with `AddFlagSet()` | Ready for F05 |

### 4.2 Partial Overlap (Needs Extension)

| E17 Requirement | Existing Partial | Extension Needed |
|-----------------|-----------------|------------------|
| Structured errors | Typed errors (`NotFoundError`, `BackwardReasonError`), human-readable formatters in `errors.go` | JSON serialization layer, error code mapping |
| Error with transition context | `InvalidStatusTransitionError()` includes allowed transitions as text | Restructure as JSON with `valid_transitions` array |
| Status dashboard | `shark status` command with `StatusService.GetDashboard()` | Rename to `shark progress` (F06) |
| Task status transitions | `shark task start/complete/approve/block/unblock` | Needs `shark status set <task-key> <status>` wrapper |

### 4.3 No Overlap (Entirely New)

| E17 Requirement | Notes |
|-----------------|-------|
| `--field` flag for field extraction | No existing field extraction capability |
| `SHARK_OUTPUT` env var | No env-based output mode exists (env prefix is `PM`, not `SHARK`) |
| Structured JSON error output to stdout | Errors currently go to stderr as human text |
| Error codes (`NOT_FOUND`, `INVALID_TRANSITION`, etc.) | No machine-readable error taxonomy exists |
| `--all` flag alias | Only `--show-all` exists today |

### 4.4 Overlap Summary

Approximately **60% of the service layer infrastructure** needed for E17 already exists. The remaining 40% is new CLI routing, the `--field` extraction mechanism, structured error serialization, and the environment variable support. No new service methods, repository methods, or database schema changes are required.

---

## 5. Risk Assessment with Mitigations

### Risk 1: Status Namespace Collision (F01, F06)

**Probability**: Certain (design constraint, not a risk)
**Impact**: Medium -- agents using `shark status E07` for progress could be confused when `shark status set E07 active` is introduced

**Description**: The existing `shark status <key>` smart dispatcher shows progress rollups. The proposed `shark status set/advance/options/history` subcommands use the same `status` keyword for a different purpose.

**Mitigation (already in PRD)**:
- Phase 1: Both coexist. Cobra supports a parent command with both `RunE` and subcommands. `shark status E07` continues to show progress. `shark status set E07 active` transitions status.
- Phase 2: F06 introduces `shark progress` to replace the dashboard. `shark status` (without subcommand) shows a deprecation warning pointing to `shark progress`.
- Phase 3: `shark status` (without subcommand) removed. `shark status` becomes exclusively the subcommand group.

**Residual risk**: Low. The phased approach is sound.

### Risk 2: E15 Service Layer Readiness

**Probability**: Low (based on current state analysis)
**Impact**: High if service methods are missing

**Description**: E17 commands must use the service layer pattern (thin commands calling services). If required service methods do not exist, E17 would need to create them.

**Mitigation**:
- Analysis confirms all required service methods exist: `TransitionStatus()`, `GetNextStatus()`, `StartTask()`, `CompleteTask()`, `ApproveTask()`, `BlockTask()`, `UnblockTask()`, `GetDashboard()`
- The `entityTransitioner` interface in `epic_next_status.go` already abstracts entity-type-agnostic transitions
- If any gaps are discovered during implementation, follow E15's constructor injection pattern to create the needed methods

**Residual risk**: Very low. Verified against codebase.

### Risk 3: Backward Compatibility Regression (F05)

**Probability**: Low
**Impact**: Medium -- could break existing agent scripts

**Description**: Renaming flags (`--execution-order` to `--order`, `--show-all` to `--all`) could break existing agent invocations.

**Mitigation (already in PRD)**:
- Use Cobra's `MarkDeprecated()` which keeps old flags working but shows a deprecation warning
- Old flags remain functional through Phase 2
- Phase 3 removes deprecated flags (with advance notice)

**Residual risk**: Very low. Cobra's deprecation mechanism is battle-tested.

### Risk 4: Error Output Stream Change (F03)

**Probability**: Certain (design decision)
**Impact**: Medium -- could break scripts that parse stdout for data and stderr for errors

**Description**: F03 moves errors from stderr to stdout in JSON mode. Scripts expecting `stdout = data, stderr = errors` will break.

**Mitigation**:
- Only applies when `--json` or `SHARK_OUTPUT=json` is active
- Human-readable mode (default) continues using stderr for errors
- This matches the agent use case: agents parse one stream (stdout) for both data and errors
- Document the change prominently in release notes

**Residual risk**: Low. The change is intentional and limited to JSON mode.

### Risk 5: `--field` Design for List Commands (F02)

**Probability**: Medium
**Impact**: Low -- design ambiguity, not a blocker

**Description**: The PRD does not specify how `--field` behaves with list commands. Should `shark list E07 --field=title` return an array of titles, or is `--field` only valid for single-entity commands?

**Mitigation**:
- Recommend: `--field` on list commands returns a newline-separated list of raw values (one per entity). This matches `gh pr list --json title --jq '.[].title'` behavior.
- For single-entity commands, return the raw value.
- Document clearly in command help.

**Residual risk**: Low. A design decision to be made during implementation.

### Risk 6: Exit Code Conflict for --field (F02)

**Probability**: Medium
**Impact**: Low

**Description**: F02 specifies exit code 1 for "field not found", but exit code 1 is already used for "entity not found" (established convention in `.claude/rules/go/error-handling.md`). An agent cannot distinguish between "entity E07-F01-001 does not exist" and "field 'nonexistent' not found on entity E07-F01-001".

**Mitigation**:
- Recommend: Use exit code 4 for "field not found" to distinguish from entity not found (exit code 1)
- Alternative: Include the distinction in the structured error JSON (F03) with different error codes (`NOT_FOUND` vs `FIELD_NOT_FOUND`)

**Residual risk**: Low. Requires a PRD clarification.

---

## 6. Recommendations

### 6.1 Proceed with Phase 1 as Defined

All Phase 1 features (F01-F06) are feasible, have minimal risk, and do not require any blocking dependencies to be resolved. The service layer infrastructure is ready. Recommendation: **proceed immediately**.

### 6.2 Implementation Order

Recommended implementation sequence for Phase 1:

1. **F04 (SHARK_OUTPUT env var)** -- Smallest change (5-10 lines in `root.go`). Immediate value for agents. Enables all subsequent features to be tested with `SHARK_OUTPUT=json`.
2. **F05 (Flag normalization)** -- Mechanical change using Cobra's `MarkDeprecated()`. No logic changes.
3. **F03 (Structured JSON errors)** -- Creates the error infrastructure used by all other features. Implement the `CLIError` struct and JSON error serialization.
4. **F01 (Status subcommand group)** -- The highest-impact feature. Requires the auto-detect routing and delegation to existing service methods.
5. **F02 (--field flag)** -- Requires F03 for consistent error output. Implement field extraction on JSON-serialized output.
6. **F06 (Progress command)** -- Requires F01 to exist so `shark status` can be disambiguated. Copy dashboard logic to `shark progress`.

### 6.3 Design Clarifications Needed

Before implementation, resolve these open questions:

1. **Exit code for `--field` not found**: Recommend exit code 4 (new) rather than reusing exit code 1.
2. **`--field` on list commands**: Recommend newline-separated raw values, one per entity.
3. **Error output stream in JSON mode**: Confirm that errors go to stdout (not stderr) when `--json` or `SHARK_OUTPUT=json` is active.
4. **`SHARK_OUTPUT` vs `PM_OUTPUT`**: Recommend `SHARK_OUTPUT` for discoverability, breaking the `PM_` prefix convention intentionally.

### 6.4 Leverage Existing Infrastructure

The following existing code should be reused, not reimplemented:

- **Entity type auto-detection**: Extract the key-format detection logic from smart dispatchers (`get.go`, `list.go`) into a shared utility function for F01.
- **`entityTransitioner` interface**: Already in `epic_next_status.go`. Extend to include task transitions for F01.
- **`TransitionOptions`/`TransitionResult`**: Already in `transition_types.go`. Use as-is for F01.
- **`performEntityTransition()`**: Already in `epic_next_status.go`. Reuse for F01's `status advance`.
- **`buildNextStatusResult()`**: Already in `epic_next_status.go`. Reuse for F01's `status options`.

### 6.5 E16 Forward Compatibility

Design F01 to be transparent to E16 enhancements:
- Always delegate status validation to `workflow.Service`, never hardcode statuses
- Use `workflow.Service.ForLevel()` to scope validation by entity type
- Include `phase` and `description` fields from `TransitionInfoWithAction` in `status options` output, so E16's richer transition metadata flows through automatically

### 6.6 Testing Strategy

- **F01, F06**: Test with mocked services (following `.claude/rules/services/testing.md`)
- **F02, F03, F04, F05**: Test at the CLI command level with mocked services
- **Integration tests**: After all Phase 1 features are complete, run end-to-end agent workflow scenarios
- **Metric validation**: Use `docs/workflow/activity.jsonl` to measure command path reduction and error suppression rates

---

## References

- [E17 Epic PRD](/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/epic.md)
- [E17 Requirements](/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/requirements.md)
- [E17 Scope](/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/scope.md)
- [E17 Success Metrics](/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/success-metrics.md)
- [E17 Personas](/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/personas.md)
- [E17 User Journeys](/home/jwwel/projects/shark-task-manager/docs/plan/E17-cli-simplification-for-ai-agents/user-journeys.md)
- [E15 Epic PRD](/home/jwwel/projects/shark-task-manager/docs/plan/E15-service-layer-architecture-refactoring/epic.md)
- [E16 Epic PRD](/home/jwwel/projects/shark-task-manager/docs/plan/E16-multi-level-workflow/epic.md)
- [CX Review](/home/jwwel/projects/shark-task-manager/docs/plan/cx-review-cli-ai-agents.md)
- [GitHub CLI Scripting Guide](https://github.blog/engineering/engineering-principles/scripting-with-github-cli/)
- [GitHub CLI --jq Discussion](https://github.com/cli/cli/discussions/7433)
- [Model Context Protocol - Tools Specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- [Agentic AI Design Patterns 2026](https://research.aimultiple.com/agentic-ai-design-patterns/)
