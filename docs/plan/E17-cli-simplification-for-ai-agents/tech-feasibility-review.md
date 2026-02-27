# Technical Feasibility Review: E17 - CLI Simplification for AI Agents

**Reviewer:** Architect Agent
**Date:** 2026-02-25
**Status:** APPROVED
**Epic:** E17 - CLI Simplification for AI Agents

---

## 1. Executive Summary

This technical feasibility review validates the research report findings against the actual codebase and the epic PRD. All Phase 1 features (F01-F05) and Phase 2 feature F06 are technically feasible with the existing service layer infrastructure. Approximately 60-70% of the required service-level code already exists. The primary new work is in the CLI command layer (thin wrappers) and structured error formatting.

**Overall Assessment: APPROVED** -- No blocking technical concerns. Proceed to technical specification.

---

## 2. Per-Feature Technical Feasibility

### F01: Status Command Group (`shark status set/advance`)

**Feasibility: FULLY FEASIBLE**

**Codebase Evidence:**
- `internal/services/transition_types.go` already defines `TransitionOptions`, `TransitionResult`, `NextStatusInfo`, and `TransitionInfoWithAction` DTOs -- all with JSON tags, ready for CLI consumption.
- `internal/cli/commands/epic_next_status.go` contains shared utilities `buildNextStatusResult()` and `performEntityTransition()` that already handle `--force`, `--reason`, `--agent`, `--status`, `--preview` flags across entity types.
- `internal/cli/commands/get.go` demonstrates the smart dispatcher pattern (`ParseGetArgs()`) for entity auto-detection from key format -- directly reusable for `status set/advance` entity routing.
- Service accessors (`cli.GetEpicService()`, `cli.GetFeatureService()`, `cli.GetTaskService()`) all expose `TransitionStatus()` and `GetNextStatus()` methods.

**Namespace Collision Resolution:**
- `internal/cli/commands/status.go` registers `statusCmd` with `cli.RootCmd.AddCommand(statusCmd)` and has `RunE: runStatus` for the existing dashboard.
- Cobra supports parent commands with both `RunE` (for `shark status E07`) and subcommands (for `shark status set E07-F01-001 in_progress`). The existing `shark status` dashboard behavior is preserved as the default `RunE` handler; `set` and `advance` become subcommands added to `statusCmd`.
- Phase 1 adds subcommands non-destructively. Phase 3 deprecation of `shark task start/complete/approve/block/unblock` happens only after new commands are validated.

**Estimated Effort:** S-M (service layer exists; new CLI commands are thin wrappers consolidating existing patterns)

---

### F02: Field Extraction (`--field`)

**Feasibility: FULLY FEASIBLE**

**Codebase Evidence:**
- `internal/cli/root.go` shows `OutputJSON()` uses `json.NewEncoder(os.Stdout).Encode(data)` -- the data is already a Go struct with JSON tags.
- Implementation approach: marshal struct to `map[string]interface{}` via `json.Marshal`/`json.Unmarshal`, then extract the requested field key. For nested fields, support dot notation (e.g., `--field=status`).
- No new service code required. This is purely a CLI output formatting concern, added as a global flag in `root.go` alongside `--json`.

**Design Clarification (from BA review):**
- `--field` on single entities returns the raw value (string, number, boolean).
- `--field` on list commands returns a newline-separated list of that field's values.
- If the field does not exist, return exit code 1 with a structured error.

**Estimated Effort:** XS-S (global flag + output interceptor in root.go)

---

### F03: Structured JSON Errors

**Feasibility: FULLY FEASIBLE (most new code required)**

**Codebase Evidence:**
- `internal/cli/commands/errors.go` contains human-readable error formatters: `InvalidEpicKeyError()`, `NotFoundError()`, `InvalidStatusTransitionError()`, `AmbiguousKeyError()`. All return `fmt.Errorf()` with multi-line strings. **No structured JSON error support exists today.**
- `internal/services/transition_types.go` defines typed errors (`BackwardReasonError`, sentinel errors `ErrReasonRequired`, `ErrForceReasonRequired`) with some structure, but no JSON serialization layer.
- The service layer already returns typed errors (e.g., `NotFoundError` from repositories) that can be inspected with `errors.As()`.

**Required Work:**
1. Define a `StructuredError` type with `code`, `message`, `details`, `suggestion` fields and JSON tags.
2. Create an error mapping layer that translates existing typed errors (`NotFoundError`, `BackwardReasonError`, validation errors) into `StructuredError` instances.
3. In `root.go`'s `PersistentPostRunE` or a wrapper around command execution, intercept errors when `GlobalConfig.JSON` is true and output the structured JSON error to stdout instead of stderr.
4. Preserve human-readable error formatting when `--json` is not active.

**Risk:** Low. The error type hierarchy already exists in services; the work is creating the serialization and interception layer.

**Estimated Effort:** M (new error type + mapping layer + output interception)

---

### F04: Environment Variable Output Control (`SHARK_OUTPUT`)

**Feasibility: TRIVIALLY FEASIBLE**

**Codebase Evidence:**
- `internal/cli/root.go` line 103: `viper.SetEnvPrefix("PM")` with `viper.AutomaticEnv()`. The existing convention uses `PM_` prefix for env vars.
- `GlobalConfig.JSON` is set from `viper.GetBool("json")`, meaning `PM_JSON=true` already works today.

**Implementation:**
- Add `SHARK_OUTPUT=json` check in `initConfig()` or `PersistentPreRunE`:
  ```go
  if os.Getenv("SHARK_OUTPUT") == "json" {
      GlobalConfig.JSON = true
  }
  ```
- This is approximately 5-10 lines of code.

**Design Note (from BA review):** The name `SHARK_OUTPUT` was questioned vs `PM_OUTPUT` for consistency with the existing `PM_` prefix. Recommendation: support both `SHARK_OUTPUT` and `PM_OUTPUT` for discoverability, with `SHARK_OUTPUT` taking precedence. Document both.

**Estimated Effort:** XS (5-10 lines in root.go)

---

### F05: Flag Normalization and Deprecation

**Feasibility: TRIVIALLY FEASIBLE**

**Codebase Evidence:**
- Cobra provides `cmd.Flags().MarkDeprecated("old-flag", "use --new-flag instead")` which prints a deprecation warning to stderr while still accepting the old flag.
- Existing commands already use multiple flag formats (positional + flag syntax). The infrastructure for backward compatibility is established.

**Implementation:**
- Audit all commands for inconsistent flag names.
- Add `MarkDeprecated()` calls for flags being renamed.
- Map deprecated flags to new flags in command handlers.

**Estimated Effort:** XS-S (flag audit + deprecation calls)

---

### F06: Progress Command

**Feasibility: FULLY FEASIBLE**

**Codebase Evidence:**
- `internal/services/` contains `display_service.go` (GetDashboard), `analytics_dto.go` (analytics DTOs), and the status calculation infrastructure in `internal/status/`.
- Feature and epic services already expose progress calculation through `GetStatusContext()`, `CalculateProgress()`, and `GetActionItems()`.
- `internal/cli/commands/status.go` already renders a rich dashboard with progress data.

**Implementation:**
- Create `shark progress <KEY>` command as a thin wrapper calling existing services.
- The service layer data is already available; this is primarily a new CLI command with focused JSON output.

**Estimated Effort:** S (new CLI command, services exist)

---

### F07: Batch Operations

**Feasibility: FEASIBLE**

**Codebase Evidence:**
- `TransitionStatus()` in services processes one entity at a time. Batch mode requires iterating and collecting results.
- `internal/services/task_service.go` has `ListTasks()` with filtering, which provides the query foundation for batch targets.

**Implementation:**
- Service method: `BatchTransition(ctx, keys []string, targetStatus string, opts TransitionOptions) ([]TransitionResult, error)` -- iterates, collects successes and failures, returns aggregate result.
- CLI command: `shark status set --batch E07-F01-001,E07-F01-002 in_progress` or `shark status set --from=todo --epic=E07 in_progress`.

**Risk:** Medium-low. The `--from` filter for batch selection needs clear semantics. BA review flagged this -- recommend requiring explicit key list OR explicit `--from` + scope filter, never implicit "all tasks."

**Estimated Effort:** M (new service method + CLI command)

---

### F08-F13: Phase 2-3 Features (Unified Create, Admin, Notes, Deprecation, Update, Delete)

**Feasibility: FEASIBLE (lower priority, less risk)**

These features consolidate existing commands into new groupings. The service layer already handles all underlying operations. The work is primarily CLI command restructuring. No new service logic is required for most of these features.

**Estimated Effort:** S-M each (CLI restructuring, service layer exists)

---

## 3. Architectural Assessment

### 3.1 Alignment with Service Layer Architecture (E15)

**Assessment: STRONG ALIGNMENT**

E17 builds directly on the E15 service layer migration. The thin-wrapper pattern prescribed by E15 is exactly what E17 commands will follow:
- Parse arguments into DTOs
- Call service method
- Format output (JSON or human-readable)

The existing service accessors (`cli.GetTaskService()`, etc.) provide the wiring. No changes to the service layer architecture are needed.

### 3.2 Backward Compatibility Strategy

**Assessment: SOUND**

The 3-phase approach is architecturally correct:
- **Phase 1:** Additive only -- new commands, new flags, no removals. Zero breaking changes.
- **Phase 2:** Promote new commands as recommended, deprecation warnings on old commands via Cobra's `MarkDeprecated()`.
- **Phase 3:** Remove deprecated commands after migration period.

This follows the same pattern used successfully in the slug migration (`shark migrate slugs`).

### 3.3 Status Namespace Design

**Assessment: SOUND with one clarification needed**

The proposed `shark status` dual-purpose (dashboard when called with entity key, subcommands for `set`/`advance`) is supported by Cobra. However, the interaction between `shark status E07` (dashboard) and `shark status set E07 draft` (transition) must be unambiguous. Since `set` and `advance` are not valid entity keys, Cobra's subcommand matching will correctly route these.

**Clarification needed at spec time:** Define behavior for `shark status set` with no arguments (show help) vs `shark status` with no arguments (show project dashboard).

### 3.4 JSON Output Consistency

**Assessment: MINOR GAP -- addressable in F03**

Current JSON output is inconsistent across commands:
- Some commands output raw entity JSON.
- Some commands output wrapper objects with metadata.
- Error output goes to stderr in human-readable format regardless of `--json` flag.

F03 (structured errors) and the `--field` flag (F02) will establish a consistent contract. Recommend defining a standard envelope for JSON mode in the technical specification:
```json
{
  "data": { ... },
  "error": null
}
```
vs error case:
```json
{
  "data": null,
  "error": {
    "code": "NOT_FOUND",
    "message": "...",
    "details": { ... }
  }
}
```

This is a spec-time decision, not a feasibility blocker.

---

## 4. Dependency and Integration Risk Assessment

### 4.1 E15 (Service Layer Migration) -- PRIMARY DEPENDENCY

**Risk: LOW**

E15 is substantially complete. Key evidence:
- `services_global.go` provides all service accessors.
- `TaskService`, `FeatureService`, `EpicService` all exist with transition methods.
- `transition_types.go` DTOs are fully defined with JSON tags.
- Commands like `epic_next_status.go` already follow the thin-wrapper pattern.

Remaining E15 work (migrating legacy fat-controller commands) does NOT block E17. E17 creates new commands that use services; it does not modify legacy commands until Phase 3.

### 4.2 E16 (Multi-Level Workflow) -- SECONDARY DEPENDENCY

**Risk: LOW**

E16 provides multi-level workflow support (epic/feature/task status flows). Evidence:
- `workflow.Service.ForLevel()` already scopes validation by entity level.
- `TransitionOptions` includes entity-level-aware fields.

E17's `shark status set/advance` commands consume workflow services that E16 has already established. No circular dependency.

### 4.3 Existing Command Ecosystem -- INTEGRATION RISK

**Risk: LOW-MEDIUM**

E17 Phase 3 deprecates ~20 existing commands. Risk is user disruption during migration window. Mitigation:
- Deprecation warnings appear for 2+ releases before removal.
- `shark doctor` or `shark migrate` could detect scripts using deprecated syntax.
- All deprecated commands continue working during deprecation period.

---

## 5. Technical Debt Assessment

### 5.1 Debt Introduced

**Minimal.** E17 follows established patterns (service layer, thin wrappers, Cobra conventions). The main temporary debt is:
- Duplicate command paths during Phase 2 (old + new commands coexist). This is intentional and resolved in Phase 3.
- `SHARK_OUTPUT` env var alongside existing `PM_` prefix convention. Manageable with documentation.

### 5.2 Debt Resolved

**Significant.** E17 addresses several existing debt items:
- **Command proliferation:** Reduces ~45 command paths to ~25.
- **Inconsistent error handling:** F03 establishes a structured error contract.
- **Python post-processing dependency:** F02 `--field` eliminates the need for `jq`/Python wrappers.
- **Defensive error suppression:** Structured errors + consistent exit codes reduce the 36% error suppression pattern to <5%.

---

## 6. Recommended Implementation Order

Aligning with the research report recommendation, validated against codebase dependencies:

1. **F04** (SHARK_OUTPUT) -- XS effort, immediate value for agents, zero risk
2. **F05** (Flag normalization) -- XS-S effort, cleanup, zero risk
3. **F03** (Structured errors) -- M effort, foundational for all other features
4. **F02** (--field extraction) -- XS-S effort, high agent value, depends on consistent JSON
5. **F01** (Status command group) -- S-M effort, highest feature value, depends on F03 for error handling
6. **F06** (Progress command) -- S effort, additive, independent

This order maximizes early value delivery while building foundational infrastructure first.

---

## 7. Design Clarifications for Technical Specification

The following items need resolution during technical specification (not blocking feasibility):

1. **Exit code for `--field` when field not found:** Recommend exit code 1 (not found) with structured error in JSON mode.
2. **`--field` behavior on list commands:** Return newline-separated values for the field across all result entities.
3. **`SHARK_OUTPUT` vs `PM_OUTPUT` naming:** Support both, `SHARK_OUTPUT` takes precedence. Document in help text.
4. **Batch `--from` filter semantics:** Require explicit scope (`--epic`, `--feature`) when using `--from` to prevent accidental mass updates.
5. **JSON envelope standard:** Define whether JSON mode wraps all output in `{data, error}` envelope or outputs raw entities (with errors on a separate channel). Recommend envelope for new commands, raw for backward compatibility on existing commands.
6. **`shark status` with no arguments:** Show project-wide dashboard (existing behavior preserved).

---

## 8. Conclusion

All features in E17 are technically feasible. The service layer infrastructure from E15 provides 60-70% of the required backend code. The primary new work is:
- CLI command files (thin wrappers following established patterns)
- Structured error formatting layer (F03)
- Global flag additions (F02, F04)

No architectural changes are required. No blocking dependencies exist. The phased delivery approach correctly manages backward compatibility risk.

**Recommendation: APPROVED. Proceed to technical specification.**

---

## Appendix: Codebase Files Reviewed

| File | Relevance |
|------|-----------|
| `internal/cli/root.go` | GlobalConfig, OutputJSON, env prefix, PersistentPreRunE |
| `internal/cli/services_global.go` | Service accessor pattern, dependency wiring |
| `internal/cli/commands/status.go` | Existing status dashboard, namespace collision |
| `internal/cli/commands/errors.go` | Human-readable error formatters (gap for F03) |
| `internal/cli/commands/get.go` | Smart dispatcher pattern (reusable for F01) |
| `internal/cli/commands/epic_next_status.go` | Shared transition utilities (foundation for F01) |
| `internal/services/transition_types.go` | DTOs with JSON tags (ready for F01) |
| `internal/services/task_service.go` | Task lifecycle methods |
| `internal/services/feature_service.go` | Feature service methods |
| `internal/services/epic_service.go` | Epic service methods |
| `internal/formatters/*.go` | Output formatting infrastructure |
| `internal/cli/commands/*.go` (~90 files) | Full command inventory |
| `internal/services/*.go` (~20 files) | Full service inventory |
