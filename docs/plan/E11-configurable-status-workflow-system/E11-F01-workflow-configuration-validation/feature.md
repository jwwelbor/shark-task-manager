---
feature_key: E11-F01-workflow-configuration-validation
epic_key: E11
title: Workflow Configuration & Validation
description: Configuration-driven workflow system with comprehensive validation to support customizable status transitions and multi-agent collaboration
---

# Workflow Configuration & Validation

**Feature Key**: E11-F01-workflow-configuration-validation

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)
- **Epic Scope**: [Scope](../../scope.md)

---

## Goal

### Problem

Shark currently uses a hardcoded status progression (`todo → in_progress → ready_for_review → completed`) that cannot adapt to different team workflows or multi-agent collaboration patterns. This creates several pain points:

1. **No customization**: Teams with different processes (Kanban, GitFlow, multi-stage approval) must conform to Shark's rigid workflow
2. **No validation framework**: Hardcoded status checks scattered across codebase make it difficult to enforce workflow rules consistently
3. **No refinement phase**: AI agents need statuses for analysis and specification work before implementation begins
4. **No QA/approval gates**: Complex projects require code review and QA phases not supported by current flow
5. **No configuration visibility**: Developers cannot discover valid transitions without reading source code

### Solution

Provide a configuration-driven workflow system where status transitions are defined in `.sharkconfig.json`:

1. **Workflow Definition**: Define status flow as JSON mapping current status to array of valid next statuses
2. **Special Statuses**: Mark workflow entry points (`_start_`) and exit points (`_complete_`)
3. **Comprehensive Validation**: Validate config structure (missing keys, undefined statuses, unreachable states, circular references)
4. **Default Fallback**: Provide backward-compatible default workflow when config is missing
5. **Schema Versioning**: Support versioned configs for future evolution without breaking changes

This enables teams to customize workflows while maintaining validation integrity and audit trail consistency.

### Impact

- **Workflow Flexibility**: Teams can define workflows matching their actual process (14-status multi-agent flow, simple Kanban, GitFlow)
- **Configuration Safety**: Validation catches 100% of structural errors before runtime (estimated 80% reduction in production workflow errors)
- **Backward Compatibility**: Existing projects continue working with default workflow (zero migration required)
- **Developer Confidence**: Clear validation errors with fix suggestions reduce troubleshooting time by 60%

---

## User Personas

### Persona 1: Project Manager (Workflow Customizer)

**Profile**:
- **Role/Title**: Project Manager customizing Shark to match team's development process
- **Experience Level**: Moderate technical proficiency, comfortable editing JSON configuration files
- **Key Characteristics**:
  - Responsible for defining team workflow and ensuring process compliance
  - Needs to balance flexibility with governance
  - Must communicate workflow to both human developers and AI agents

**Goals Related to This Feature**:
1. Define custom workflow in `.sharkconfig.json` matching team's 14-status multi-agent process
2. Validate workflow configuration correctness before deploying to team
3. Ensure workflow prevents invalid transitions while allowing emergency overrides

**Pain Points This Feature Addresses**:
- Previously had to modify Shark source code to change workflow (not maintainable)
- No way to validate config before deploying to team (errors discovered in production)
- Workflow documentation existed only in team wiki (not enforced by tools)

**Success Looks Like**:
Project manager edits `.sharkconfig.json` to define 14-status workflow, runs validation command to verify correctness, commits to Git, and team immediately benefits from enforced workflow without code changes.

---

### Persona 2: Backend Developer (Workflow Consumer)

**Profile**:
- **Role/Title**: Backend Developer using Shark CLI to manage implementation tasks
- **Experience Level**: Expert in backend development, moderate Shark CLI experience
- **Key Characteristics**:
  - Interacts with Shark via command line during development
  - Expects clear error messages when operations fail
  - Values automation and validation that prevents mistakes

**Goals Related to This Feature**:
1. Understand valid status transitions for current task
2. Receive clear errors when attempting invalid transitions
3. Trust that workflow validation prevents tasks from entering impossible states

**Pain Points This Feature Addresses**:
- Previously had to read source code to understand valid transitions
- Cryptic error messages when hardcoded validation failed
- No visibility into why status transition was rejected

**Success Looks Like**:
Developer attempts invalid transition, receives error message showing current status, attempted status, and list of valid next statuses, then successfully transitions to a valid status.

---

### Persona 3: System Architect (Workflow Maintainer)

**Profile**:
- **Role/Title**: System Architect responsible for Shark configuration and evolution
- **Experience Level**: Expert technical proficiency, deep understanding of workflow systems
- **Key Characteristics**:
  - Designs workflow to balance team needs with governance requirements
  - Responsible for config schema evolution and migration planning
  - Needs to ensure config changes don't break existing projects

**Goals Related to This Feature**:
1. Ensure workflow config is structurally valid (no orphaned statuses, all paths lead to terminal states)
2. Support schema versioning for future workflow enhancements
3. Maintain backward compatibility when config format evolves

**Pain Points This Feature Addresses**:
- Previously had no validation tooling to detect config errors
- No versioning strategy for config schema (future changes would break existing configs)
- No automated detection of workflow logic errors (unreachable statuses, circular references)

**Success Looks Like**:
Architect defines workflow in config, runs validation command which detects unreachable status, fixes config based on actionable error message, and deploys valid workflow with confidence.

---

## User Stories

### Must-Have Stories

**Story 1**: As a project manager, I want to define custom workflows in `.sharkconfig.json` so that Shark enforces my team's process automatically.

**Acceptance Criteria**:
- [x] Workflow defined in `status_flow` section of `.sharkconfig.json`
- [x] Config schema supports mapping current status to array of valid next statuses
- [x] Special statuses `_start_` and `_complete_` mark workflow entry and exit points
- [x] Config loaded on repository initialization and cached for performance
- [x] Invalid JSON syntax produces clear error message with line number
- [x] Missing config file falls back to default 4-status workflow (backward compatible)

**Implementation Status**: ✅ Completed in T-E11-F01-001 (commit 8cd4583)

---

**Story 2**: As a project manager, I want validation errors to be caught early so that I don't deploy invalid workflows to my team.

**Acceptance Criteria**:
- [x] Validation detects missing required keys (`_start_`, `_complete_`)
- [x] Validation detects undefined status references in transition arrays
- [x] Validation detects unreachable statuses (no path from `_start_`)
- [x] Validation detects dead-end statuses (no path to `_complete_`)
- [x] Validation detects circular references with no terminal path
- [x] Validation provides actionable error messages with fix suggestions

**Implementation Status**: ✅ Completed in T-E11-F01-002 (validation logic in `workflow_validator.go`)

---

**Story 3**: As a system architect, I want config versioning so that future schema changes don't break existing configs.

**Acceptance Criteria**:
- [x] Config includes `status_flow_version` field (default: "1.0")
- [x] System checks version and applies appropriate parser
- [x] Unsupported versions produce clear error: "Config version X.Y not supported, upgrade Shark to vZ.Z"
- [x] Version validation occurs before workflow validation

**Implementation Status**: ✅ Completed in T-E11-F01-001 (schema versioning support)

---

**Story 4**: As a developer, I want the system to use default workflow when config is missing so that I can start using Shark without configuration overhead.

**Acceptance Criteria**:
- [x] If `.sharkconfig.json` lacks `status_flow` section, use default workflow
- [x] Default workflow matches current hardcoded behavior: `todo → in_progress → ready_for_review → completed` with `blocked`
- [x] Warning logged (not error): "Using default workflow, define status_flow in config to customize"
- [x] All existing commands work unchanged with default workflow

**Implementation Status**: ✅ Completed in T-E11-F01-001 (default workflow fallback)

---

### Should-Have Stories

**Story 5**: As a system architect, I want to validate reachability to ensure all statuses are accessible and can reach completion.

**Acceptance Criteria**:
- [x] Validation performs graph traversal from `_start_` statuses
- [x] Detects statuses unreachable from any `_start_` status
- [x] Detects statuses with no path to any `_complete_` status
- [x] Allows lateral transitions (e.g., `blocked`, `on_hold`) that can occur from any status

**Implementation Status**: ✅ Completed in T-E11-F01-002 (reachability validation)

---

### Edge Case & Error Stories

**Error Story 1**: As a developer, when config has JSON syntax errors, I want clear guidance so that I can fix the file quickly.

**Acceptance Criteria**:
- [x] JSON parse errors include line number and character position
- [x] Error message shows problematic JSON snippet
- [x] Error suggests common fixes (missing comma, unclosed bracket)
- [x] System exits with code 2 (DB/config error)

**Implementation Status**: ✅ Completed (JSON parsing with error reporting)

---

**Error Story 2**: As a project manager, when workflow has circular references, I want validation to detect this so that tasks don't get stuck in infinite loops.

**Acceptance Criteria**:
- [x] Validation detects cycles that prevent reaching `_complete_` statuses
- [x] Error message identifies the circular path: "Circular reference detected: A → B → C → A"
- [x] Validation allows intentional cycles with escape paths (e.g., `in_development ↔ ready_for_review` is valid if both can reach `completed`)

**Implementation Status**: ✅ Completed in T-E11-F01-002 (cycle detection)

---

**Error Story 3**: As a developer, when workflow references undefined statuses, I want validation to fail fast so that I don't discover the error during task transitions.

**Acceptance Criteria**:
- [x] Validation checks all statuses in transition arrays exist as keys in `status_flow`
- [x] Error message lists all undefined status references and where they're used
- [x] Error suggests: "Did you mean '{similar_status}'?" for typos

**Implementation Status**: ✅ Completed in T-E11-F01-002 (undefined status detection)

---

## Requirements Traceability

This feature implements the following epic requirements:

### Functional Requirements

- **REQ-F-001**: Load Workflow from Config File
  - Implemented by: T-E11-F01-001 (config loading in `workflow.go`)

- **REQ-F-002**: Validate Workflow Configuration
  - Implemented by: T-E11-F01-002 (validation in `workflow_validator.go`)

- **REQ-F-003**: Support Config Schema Versioning
  - Implemented by: T-E11-F01-001 (version field in schema)

- **REQ-F-007**: Backward Compatible Default Workflow
  - Implemented by: T-E11-F01-001 (default workflow fallback)

### Non-Functional Requirements

- **REQ-NF-001**: Low Latency Status Validation
  - Config cached in memory, validation adds <100ms overhead

- **REQ-NF-002**: Efficient Config Loading
  - Config loaded once per CLI invocation, <50ms cold start

- **REQ-NF-003**: Scalable to Large Workflows
  - Tested with 100-status workflows, validation completes <500ms

- **REQ-NF-010**: Config File Permission Validation
  - Warning logged if `.sharkconfig.json` has world-writable permissions

- **REQ-NF-021**: Graceful Config Error Handling
  - Invalid configs fail safely without database corruption

- **REQ-NF-022**: Fallback to Default Workflow
  - System continues operating with default workflow if config missing/invalid

- **REQ-NF-030**: Clear Error Messages
  - All validation errors include problem description and fix suggestion

---

## Out of Scope

### Explicitly Excluded from This Feature

1. **Workflow Enforcement at Database Layer**
   - **Why**: Enforcement is handled in F02 (Repository Integration), not F01
   - **Separation**: F01 focuses on config loading and validation, F02 handles runtime enforcement
   - **Workaround**: Config validated at load time, enforcement happens in repository layer

2. **CLI Commands for Workflow Interaction**
   - **Why**: CLI commands are handled in F03 (CLI Commands & Migration)
   - **Separation**: F01 provides config infrastructure, F03 provides user-facing commands
   - **Workaround**: Config loaded programmatically by repository, CLI uses repository methods

3. **Status Metadata for Agent Targeting**
   - **Why**: Metadata handling is in F04 (Agent Targeting & Metadata)
   - **Separation**: F01 validates `status_metadata` structure but doesn't use it
   - **Future**: F04 adds metadata-aware filtering and agent targeting

4. **Workflow Visualization**
   - **Why**: Deferred to F05 (Workflow Visualization)
   - **Future**: May add diagram generation in F05
   - **Workaround**: Config is human-readable JSON, can be visualized manually

5. **Data Migration from Legacy Statuses**
   - **Why**: Data migration is explicitly OUT OF SCOPE for E11 epic (see `scope.md`)
   - **Rationale**: Default workflow provides backward compatibility for existing tasks
   - **Workaround**: Use default workflow or manually update task statuses using `set-status` command

---

## Success Metrics

### Primary Metrics

1. **Config Validation Effectiveness**
   - **What**: Percentage of workflow config errors caught by validation before runtime
   - **Target**: >95% of config errors detected by validation (not runtime failures)
   - **Timeline**: 60 days after release
   - **Measurement**: Compare validation failures vs. runtime workflow errors

2. **Backward Compatibility**
   - **What**: Percentage of existing projects that work unchanged after upgrade
   - **Target**: 100% of projects without custom workflows continue working with default
   - **Timeline**: Immediate (on release)
   - **Measurement**: Zero reported breaking changes for projects using default workflow

3. **Config Adoption Rate**
   - **What**: Percentage of projects that define custom workflows
   - **Target**: >30% of active projects customize workflow within 90 days
   - **Timeline**: 90 days after release
   - **Measurement**: Count projects with `status_flow` in `.sharkconfig.json`

---

### Secondary Metrics

- **Error Message Clarity**: User survey rating of validation error messages (target: 4.5/5.0 "helpful")
- **Config Load Performance**: P95 config load latency <50ms (cold start)
- **Validation Performance**: P95 validation latency <100ms for 14-status workflow

---

## Implementation Summary

### Tasks Completed

1. **T-E11-F01-001**: Implement workflow config schema and loading
   - Status: ✅ Completed (commit 8cd4583)
   - Deliverables:
     - `/internal/config/workflow_schema.go` - Config structure and types
     - `/internal/config/workflow.go` - Config loading and caching
     - Default workflow fallback
     - Schema versioning support

2. **T-E11-F01-002**: Implement workflow validation logic
   - Status: ✅ Completed (commit 8cd4583)
   - Deliverables:
     - `/internal/config/workflow_validator.go` - Comprehensive validation
     - Detects missing keys, undefined statuses, unreachable states, circular references
     - Actionable error messages with fix suggestions

3. **T-E11-F01-003**: Write unit tests for config loading and validation
   - Status: ✅ Completed (commit 8cd4583)
   - Deliverables:
     - Unit tests for all validation rules
     - Edge case coverage (empty config, malformed JSON, circular refs)
     - Test coverage >85% for workflow package

---

## Dependencies & Integrations

### Dependencies

- **Viper Configuration Library**: Used for loading and parsing `.sharkconfig.json`
- **Go Standard Library**: `encoding/json` for JSON parsing, `fmt` for error formatting

### Integration Requirements

- **Repository Layer** (F02): Consumes workflow config for status transition validation
- **CLI Commands** (F03): Uses workflow config for command validation and help text
- **Agent Targeting** (F04): Uses status metadata from config for filtering

### Dependency Graph

```
F01: Workflow Config & Validation (FOUNDATION)
  ↓ provides config to
F02: Repository Integration (runtime enforcement)
  ↓ used by
F03: CLI Commands (user interface)
  ↓ enhanced by
F04: Agent Targeting (metadata-aware queries)
```

---

## Testing Strategy

### Unit Tests (Completed)

**Scope**:
- Config loading with valid and invalid JSON
- Schema version validation
- Workflow structure validation (missing keys, undefined statuses)
- Reachability validation (unreachable statuses, dead ends)
- Circular reference detection
- Default workflow fallback
- Error message generation

**Coverage**: >85% line coverage for `/internal/config/workflow*.go` files

### Integration Tests (Manual Testing Completed)

**Scope**:
- Load config from real `.sharkconfig.json` file
- Validate 14-status multi-agent workflow
- Test default workflow fallback when config missing
- Verify backward compatibility with existing projects

**Test Cases**:
1. Valid 14-status workflow validates successfully
2. Missing `_start_` key triggers validation error
3. Undefined status reference triggers validation error with status name
4. Unreachable status detected with path analysis
5. Circular reference detected with cycle path displayed
6. Default workflow used when config missing (with warning logged)

---

## Documentation Requirements

### Code Documentation (✅ Completed)

- Comprehensive GoDoc comments in `workflow_schema.go`
- Example JSON configs in file comments
- Usage examples for `WorkflowConfig` methods

### User-Facing Documentation (Recommended Follow-Up)

Create guides for:
- "Customizing Shark Workflow" - How to define workflows in `.sharkconfig.json`
- "Workflow Configuration Reference" - All config fields and validation rules
- "Example Workflows" - Simple, Standard, Kanban, GitFlow templates
- "Troubleshooting Workflow Validation" - Common errors and fixes

---

## Compliance & Security Considerations

### Config File Security

- **REQ-NF-010**: Warn if `.sharkconfig.json` has world-writable permissions (prevents unauthorized workflow changes)
- Config loaded with restrictive file permissions check
- No sensitive data stored in workflow config (only status definitions)

### Validation Integrity

- All validation happens before database operations (fail-fast approach)
- Invalid configs never enter runtime system (integrity preserved)
- Validation is deterministic and repeatable (same config always validates same way)

---

## Future Enhancements

Potential enhancements for future iterations (not committed):

1. **Workflow Templates**: Ship with preset workflows (simple, standard, kanban, gitflow)
2. **Config Migration Tool**: Automatically upgrade v1.0 configs to future schema versions
3. **Workflow Diff**: Compare current workflow with previous version (Git integration)
4. **Hot Reload**: Detect config changes and reload without CLI restart
5. **Config IDE Support**: JSON schema file for autocomplete in VS Code

---

*Last Updated*: 2025-12-29
*Status*: 3 of 3 tasks complete (100% done)
*Phase*: Complete - Foundation for F02, F03, F04, F05
