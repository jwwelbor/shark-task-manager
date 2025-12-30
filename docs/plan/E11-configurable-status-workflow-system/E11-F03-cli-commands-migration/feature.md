---
feature_key: E11-F03-cli-commands-migration
epic_key: E11
title: CLI Commands & Workflow Enforcement
description: User-facing CLI commands for workflow management, validation, and status transitions with configurable workflow enforcement
---

# CLI Commands & Workflow Enforcement

**Feature Key**: E11-F03-cli-commands-migration

**Note**: This feature was originally titled "CLI Commands & Migration" but the data migration component was removed as it is explicitly out of scope per epic `scope.md`. This feature focuses solely on CLI command implementation and workflow enforcement.

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)
- **Epic Scope**: [Scope](../../scope.md)

---

## Goal

### Problem

AI development agents and human developers need user-friendly CLI commands to interact with the configurable workflow system. Without dedicated commands, users cannot:
- Discover what statuses and transitions are available in their workflow
- Validate workflow configuration before deploying to production
- Perform generic status transitions (only convenience commands like `start`, `complete` exist)
- Understand why a status transition was rejected

The existing status transition commands (`start`, `complete`, `approve`) are hardcoded and don't respect workflow configuration, creating inconsistency between configured workflows and actual enforcement.

### Solution

Provide comprehensive CLI commands for workflow management and status transitions:

1. **Discovery Commands**:
   - `shark workflow list` - Display configured workflow with all statuses and valid transitions
   - `shark workflow validate` - Validate workflow configuration correctness

2. **Generic Status Transition**:
   - `shark task set-status` - Change task to any valid status with workflow validation

3. **Updated Convenience Commands**:
   - Refactor `start`, `complete`, `approve` to use workflow validation instead of hardcoded checks

All commands support `--force` flag for emergency overrides and `--json` output for AI agent consumption.

### Impact

- **Reduced friction**: Agents can discover workflow without reading config files (40% faster workflow understanding)
- **Fewer errors**: Validation command catches config errors before deployment (estimated 80% reduction in production workflow errors)
- **Flexibility**: Generic `set-status` command enables any valid transition, supporting complex workflows beyond simple linear progression
- **Consistency**: All status transitions validated against single source of truth (workflow config), eliminating hardcoded logic scattered across codebase

---

## User Personas

### Persona 1: AI Development Agent (Backend Developer)

**Profile**:
- **Role/Title**: Backend Developer Agent responsible for API implementation tasks
- **Experience Level**: Programmatic CLI interaction, JSON output parsing, workflow-aware task selection
- **Key Characteristics**:
  - Queries tasks by agent type (`--agent=backend`)
  - Needs machine-readable output (`--json`)
  - Must understand valid status transitions to avoid errors

**Goals Related to This Feature**:
1. Query workflow to understand which statuses are relevant for backend work
2. Transition tasks through development phases (todo → in_development → ready_for_review)
3. Handle validation errors gracefully when attempting invalid transitions

**Pain Points This Feature Addresses**:
- Previously had no way to discover valid transitions programmatically
- Hardcoded commands didn't match custom workflow statuses
- No visibility into why status transitions failed

**Success Looks Like**:
Agent can query `shark workflow list --json`, parse valid transitions for current task status, and execute `shark task set-status` with confidence that validation errors include actionable guidance.

---

### Persona 2: Project Manager (Human User)

**Profile**:
- **Role/Title**: Project Manager customizing Shark workflow to match team process
- **Experience Level**: Moderate technical proficiency, comfortable editing JSON config files
- **Key Characteristics**:
  - Defines custom workflow in `.sharkconfig.json`
  - Needs to validate config before deploying to team
  - Wants visual representation of workflow for stakeholder communication

**Goals Related to This Feature**:
1. Validate workflow configuration correctness before committing to Git
2. Visualize workflow to ensure it matches team's mental model
3. Troubleshoot workflow config errors with clear guidance

**Pain Points This Feature Addresses**:
- Previously had to deploy config and test with real tasks to find errors
- No visual representation of workflow (had to parse JSON mentally)
- Error messages were cryptic or non-existent

**Success Looks Like**:
Project manager runs `shark workflow validate` after editing config, receives clear error messages with fix suggestions, and uses `shark workflow list` to visually verify the workflow matches intentions.

---

### Persona 3: Tech Lead (Emergency Hotfix Scenario)

**Profile**:
- **Role/Title**: Tech Lead managing production incidents and emergency deployments
- **Experience Level**: Expert Shark user, understands workflow system deeply
- **Key Characteristics**:
  - Needs to bypass workflow validation in emergencies
  - Requires audit trail of forced transitions
  - Balances speed with safety during incidents

**Goals Related to This Feature**:
1. Force status transitions during emergencies without workflow validation blocking progress
2. Maintain audit trail of forced transitions for post-incident review
3. Use `set-status` for unusual transitions not covered by convenience commands

**Pain Points This Feature Addresses**:
- Previously could not bypass workflow validation in emergencies
- Convenience commands didn't support all necessary transitions
- No way to document reason for forced transitions

**Success Looks Like**:
Tech lead can execute `shark task set-status <key> completed --force --notes="emergency hotfix bypass"` and transition is logged with forced flag and notes for post-incident analysis.

---

## User Stories

### Must-Have Stories

**Story 1**: As an AI development agent, I want to query the configured workflow so that I can understand valid status transitions before attempting them.

**Acceptance Criteria**:
- [x] `shark workflow list` displays all statuses and valid next statuses
- [x] Output highlights special statuses (`_start_`, `_complete_`)
- [x] Human-readable indented tree structure for terminal display
- [x] `--json` flag returns structured data parseable by agents
- [x] Command loads workflow from `.sharkconfig.json`
- [x] Falls back to default workflow if config missing (backward compatible)

**Implementation Status**: ✅ Completed in T-E11-F03-001 (commit b448446)

---

**Story 2**: As a project manager, I want to validate my workflow configuration so that I catch errors before deploying.

**Acceptance Criteria**:
- [x] `shark workflow validate` checks all validation rules (missing keys, unreachable statuses, circular references)
- [x] Exits with code 0 if valid, code 2 if invalid
- [x] Displays summary on success: "✓ X statuses defined, Y start statuses, Z terminal statuses, all statuses reachable"
- [x] On error, displays specific issues with fix suggestions
- [x] Validates against same rules used at runtime (consistency)

**Implementation Status**: ✅ Completed in T-E11-F03-002 (commit b448446)

---

**Story 3**: As a backend developer agent, I want to change task status to any valid workflow status so that I can transition through complex workflows.

**Acceptance Criteria**:
- [x] `shark task set-status <task-key> <new-status>` changes task status
- [x] Validates transition against workflow config (rejects invalid transitions)
- [x] Error message includes current status, attempted status, and list of valid next statuses
- [x] Error message hints: "Use --force to override validation"
- [x] Updates task status in database atomically
- [x] Records transition in task_history with timestamp and notes
- [x] Supports `--json` output for agent consumption
- [x] `--notes` flag allows documenting reason for transition

**Implementation Status**: ✅ Completed in T-E11-F03-003 (commit b448446)

---

**Story 4**: As a developer, I want existing convenience commands to respect workflow config so that behavior is consistent.

**Acceptance Criteria**:
- [x] `shark task start` validates transition using workflow (not hardcoded)
- [x] `shark task complete` validates transition using workflow
- [x] `shark task approve` validates transition using workflow
- [x] All commands support `--force` flag to bypass validation
- [x] Error messages match `set-status` command format (consistency)
- [x] Commands work unchanged with default workflow (backward compatible)

**Implementation Status**: ✅ Completed in T-E11-F03-004 (commit b448446)

---

**Story 5**: As a tech lead, I want to force status transitions during emergencies so that I can deploy hotfixes without workflow blocking me.

**Acceptance Criteria**:
- [x] `--force` flag supported on `set-status`, `start`, `complete`, `approve`
- [x] Forced transitions logged in task_history with `forced=true` flag
- [x] Warning message displayed: "⚠️  Forced transition bypassed workflow validation"
- [x] Forced transitions still validate that status exists in config (prevents typos)
- [x] `--notes` flag allows documenting reason for force override

**Implementation Status**: ✅ Completed in T-E11-F03-003 and T-E11-F03-004 (commit b448446)

---

### Should-Have Stories

**Story 6**: As a QA agent, I want helpful error messages when transitions fail so that I can fix issues quickly.

**Acceptance Criteria**:
- [x] Error message format: "Cannot transition from X to Y. Valid transitions: A, B, C"
- [x] Error includes hint about `--force` flag
- [x] Validation errors occur before database update (fast fail)
- [x] Error messages use consistent format across all commands

**Implementation Status**: ✅ Completed in T-E11-F03-003 and T-E11-F03-004 (commit b448446)

---

### Edge Case & Error Stories

**Error Story 1**: As a developer, when I attempt an invalid transition, I want clear guidance so that I can correct my command.

**Acceptance Criteria**:
- [x] Invalid transition error shows current status, attempted status, valid next statuses
- [x] Error includes hint: "Use --force to override" (if emergency)
- [x] Error distinguishes between "invalid transition" vs "status doesn't exist in config"
- [x] Exit code 3 for invalid state (matches Shark conventions)

**Implementation Status**: ✅ Completed in T-E11-F03-003 (commit b448446)

---

**Error Story 2**: As a project manager, when workflow config is invalid, I want validation to fail fast so that I don't deploy broken configs.

**Acceptance Criteria**:
- [x] `shark workflow validate` detects missing required keys (`_start_`, `_complete_`)
- [x] Detects undefined status references in transition arrays
- [x] Detects unreachable statuses (no path from `_start_`)
- [x] Detects dead-end statuses (no path to `_complete_`)
- [x] Provides actionable error messages with fix suggestions

**Implementation Status**: ✅ Completed in T-E11-F03-002 (commit b448446)

---

**Error Story 3**: As a developer, when config file is missing, I want system to use default workflow so that I can continue working.

**Acceptance Criteria**:
- [x] If `.sharkconfig.json` lacks `status_flow` section, use default workflow
- [x] Default workflow matches legacy behavior: `todo → in_progress → ready_for_review → completed`
- [x] Warning logged (not error): "Using default workflow, define status_flow in config to customize"
- [x] All commands work unchanged with default workflow

**Implementation Status**: ✅ Completed in F01 workflow config loading (dependency for F03)

---

## Requirements Traceability

This feature implements the following epic requirements:

### Functional Requirements

- **REQ-F-010**: Generic Status Transition Command
  - Implemented by: T-E11-F03-003 (`shark task set-status`)

- **REQ-F-011**: Workflow List Command
  - Implemented by: T-E11-F03-001 (`shark workflow list`)

- **REQ-F-012**: Workflow Validation Command
  - Implemented by: T-E11-F03-002 (`shark workflow validate`)

- **REQ-F-013**: Update Existing Convenience Commands
  - Implemented by: T-E11-F03-004 (refactored `start`, `complete`, `approve`)

### Non-Functional Requirements

- **REQ-NF-030**: Clear Error Messages
  - All commands provide actionable error messages with fix suggestions

- **REQ-NF-031**: Comprehensive Help Text
  - All commands have detailed `--help` output with examples

- **REQ-NF-041**: Repository Layer Abstraction
  - CLI commands call repository layer for status updates
  - Workflow validation isolated in repository layer, not CLI

---

## Out of Scope

### Explicitly Excluded from This Feature

1. **Task Data Migration Tooling**
   - **Why**: Data migration is explicitly OUT OF SCOPE for E11 epic (see `scope.md` lines 17-26)
   - **Original Plan**: Tasks T-E11-F03-005 (migration plan), T-E11-F03-006 (migrate command), T-E11-F03-007 (rollback) were initially planned
   - **Decision**: Removed from feature scope after clarification that only "code migration" (refactoring Shark to use workflows) is in scope, not "data migration" (changing existing task statuses)
   - **Rationale**:
     - Code migration is complete (F01-F04 refactored codebase)
     - Data migration adds significant complexity (safety, rollback, testing)
     - Default workflow provides backward compatibility for existing projects
     - No user demand for automated data migration
   - **Future**: May be addressed in separate epic if user demand emerges
   - **Workaround**: Projects can use default workflow (matches legacy statuses) or manually update task statuses using `shark task set-status`

2. **Workflow Visualization as Diagram**
   - **Why**: Deferred to F05 (Workflow Visualization feature)
   - **Workaround**: `shark workflow list` provides text-based visualization

3. **Workflow Template Selection**
   - **Why**: Template library is "Could-Have" requirement (REQ-F-019), not Must-Have
   - **Future**: May be added in F05 or separate enhancement
   - **Workaround**: Users can copy example workflows from documentation

---

## Success Metrics

### Primary Metrics

1. **Command Usage Adoption**
   - **What**: Percentage of workflow-enabled projects using new commands
   - **Target**: >60% of projects use `workflow list` or `workflow validate` within first week
   - **Timeline**: 30 days after release
   - **Measurement**: CLI telemetry (if enabled) or user surveys

2. **Validation Error Prevention**
   - **What**: Percentage of workflow config errors caught by `workflow validate` before production
   - **Target**: >80% of config errors caught in development
   - **Timeline**: 60 days after release
   - **Measurement**: Compare validation failures in dev vs. production runtime errors

3. **Force Flag Usage (Safety Metric)**
   - **What**: Percentage of status transitions using `--force` flag
   - **Target**: <5% of transitions require force override
   - **Timeline**: Ongoing monitoring
   - **Measurement**: Query task_history for `forced=true` entries
   - **Interpretation**: Low force usage indicates workflow matches actual process; high usage indicates workflow is too restrictive

---

### Secondary Metrics

- **Error Message Clarity**: User satisfaction with error messages (survey: "How helpful were validation error messages?")
- **Command Discoverability**: Time to first successful status transition for new users (target: <5 minutes)
- **Backward Compatibility**: Zero breaking changes for projects using default workflow

---

## Implementation Summary

### Tasks Completed

1. **T-E11-F03-001**: Implement `shark workflow list` command
   - Status: ✅ Completed (commit b448446)
   - Deliverables:
     - `/internal/cli/commands/workflow.go` (list command implementation)
     - Human-readable and JSON output formats
     - Help text and examples

2. **T-E11-F03-002**: Implement `shark workflow validate` command
   - Status: ✅ Completed (commit b448446)
   - Deliverables:
     - Validation logic in `/internal/config/workflow_validator.go`
     - CLI command in `/internal/cli/commands/workflow.go`
     - Comprehensive validation rules (missing keys, unreachable statuses, circular references)

3. **T-E11-F03-003**: Implement `shark task set-status` command
   - Status: ✅ Completed (commit b448446)
   - Deliverables:
     - Generic status transition command with workflow validation
     - `--force` flag support
     - `--notes` flag for documenting transitions
     - Error messages with valid next statuses

4. **T-E11-F03-004**: Update existing commands (start, complete, approve)
   - Status: ✅ Completed (commit b448446)
   - Deliverables:
     - Removed hardcoded status validation
     - Integrated with workflow validation layer
     - Maintained backward compatibility with default workflow
     - `--force` flag support

### Tasks Removed

5. **~~T-E11-F03-005~~**: ~~Implement migration plan generation~~ **REMOVED**
   - Reason: Data migration is out of scope (see "Out of Scope" section)

6. **~~T-E11-F03-006~~**: ~~Implement shark migrate workflow command~~ **REMOVED**
   - Reason: Data migration is out of scope (see "Out of Scope" section)

7. **~~T-E11-F03-007~~**: ~~Implement migration rollback mechanism~~ **REMOVED**
   - Reason: Data migration is out of scope (see "Out of Scope" section)

### Tasks Remaining

8. **T-E11-F03-008**: Write CLI command tests with mocked repositories
   - Status: ⏳ Todo
   - Priority: High (required for REQ-NF-040: >85% test coverage)
   - Scope:
     - Unit tests for all workflow commands (list, validate, set-status)
     - Integration tests for refactored convenience commands (start, complete, approve)
     - Mock repository pattern (no real database in CLI tests)
     - Test coverage for error handling, validation failures, force flag behavior

---

## Dependencies & Integrations

### Dependencies

- **F01: Workflow Configuration & Validation** (COMPLETED)
  - Provides workflow config loading from `.sharkconfig.json`
  - Provides workflow validation logic
  - Provides default workflow fallback

- **F02: Repository Integration** (COMPLETED)
  - Provides repository layer with workflow validation
  - Provides `TaskRepository.UpdateStatus()` method
  - Provides task_history audit trail

### Integration Requirements

- **CLI Framework (Cobra)**: All commands integrate with Shark's Cobra command structure
- **Global Flags**: Commands support `--json`, `--no-color`, `--verbose` from root command
- **Exit Codes**: Commands follow Shark conventions (0=success, 1=not found, 2=DB error, 3=invalid state)

---

## Testing Strategy

### Unit Tests (T-E11-F03-008)

**Scope**:
- Command argument parsing (flags, positional args)
- Output formatting (human-readable vs JSON)
- Error message generation
- Help text completeness

**Pattern**: Mock repositories, test command logic in isolation

**Example**:
```go
func TestWorkflowListCommand(t *testing.T) {
    mockRepo := &MockWorkflowRepository{
        LoadFunc: func() (*Workflow, error) {
            return &Workflow{...}, nil
        },
    }
    // Test command execution
    // Verify output format
}
```

### Integration Tests (Manual Testing Required)

**Scope**:
- End-to-end workflow validation with real config files
- Database integration for status transitions
- Task history audit trail verification
- Force flag behavior with real data

**Test Cases**:
1. Valid transition: `shark task set-status T-X todo in_progress` succeeds
2. Invalid transition: `shark task set-status T-X todo completed` fails with helpful error
3. Force override: `shark task set-status T-X todo completed --force` succeeds with warning
4. Default workflow: Delete config, verify commands use default workflow
5. Custom workflow: Define 14-status workflow, verify all transitions validated

### Regression Tests

- Existing projects with default workflow must continue working unchanged
- All convenience commands maintain existing behavior when using default workflow
- No breaking changes to CLI interface or output formats

---

## Documentation Requirements

### Help Text (✅ Completed)

All commands have comprehensive `--help` output:
- `shark workflow --help`
- `shark workflow list --help`
- `shark workflow validate --help`
- `shark task set-status --help`

### User Guide (Recommended Follow-Up)

Create user-facing documentation:
- Guide: "Customizing Shark Workflow"
- Guide: "Understanding Workflow Validation"
- Example configs: Simple, Standard, Kanban, GitFlow workflows
- Troubleshooting: Common validation errors and fixes

---

## Compliance & Security Considerations

### Audit Trail

- All status transitions (successful and failed) logged in task_history
- Forced transitions flagged with `forced=true` for monitoring
- Transition notes captured for forensic analysis

### Force Flag Abuse Detection

- REQ-NF-012: Force flag usage logged separately for abuse detection
- Recommendation: Monitor force flag usage percentage (<5% is healthy)
- High force usage indicates workflow doesn't match actual process (needs adjustment)

### Config File Security

- REQ-NF-010: Warn if `.sharkconfig.json` has world-writable permissions
- Prevents unauthorized workflow changes

---

## Future Enhancements

Potential enhancements for future iterations (not committed):

1. **Workflow Templates**: `shark workflow use <template>` to apply preset workflows
2. **Workflow Diff**: Compare current workflow with previous version (Git integration)
3. **Migration Dry-Run**: If data migration is added in future epic, provide dry-run mode
4. **Transition Suggestions**: When transition fails, suggest most likely intended transition based on current phase
5. **Workflow Analytics**: `shark workflow stats` showing which transitions are most common, where tasks get stuck

---

*Last Updated*: 2025-12-29
*Status*: 4 of 5 tasks complete (80% done)
*Remaining Work*: T-E11-F03-008 (CLI command tests)
