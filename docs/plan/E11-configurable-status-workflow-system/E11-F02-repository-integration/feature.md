---
feature_key: E11-F02-repository-integration
epic_key: E11
title: Repository Integration
description: Runtime workflow enforcement at the data access layer with force flag escape hatch for emergency overrides
---

# Repository Integration

**Feature Key**: E11-F02-repository-integration

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Requirements**: [Requirements](../../requirements.md)
- **Epic Scope**: [Scope](../../scope.md)

---

## Goal

### Problem

Workflow configuration validation (F01) ensures config correctness at load time, but runtime enforcement is needed to prevent invalid status transitions during actual task operations. Without repository-level enforcement:

1. **No runtime validation**: Task status can be updated to invalid states bypassing workflow rules
2. **Scattered validation**: Status checks are hardcoded across CLI commands, creating inconsistency
3. **No audit trail**: Forced transitions (emergency overrides) are not tracked separately from normal transitions
4. **No centralized control**: Each CLI command reimplements transition logic, risking divergence
5. **No emergency escape**: Critical situations (production hotfixes) cannot bypass validation when necessary

The repository layer is the correct enforcement point because it's the single gateway to database operations, ensuring all status changes (from CLI, API, scripts) are validated consistently.

### Solution

Integrate workflow validation into the repository layer with comprehensive enforcement and audit trail:

1. **Repository-Level Validation**: `TaskRepository.UpdateStatus()` validates transitions against workflow config before database update
2. **Force Flag Support**: `--force` parameter bypasses validation for emergency overrides while logging forced transitions separately
3. **Atomic Enforcement**: Transaction wrapper ensures validation and update occur atomically (check-and-set pattern)
4. **Audit Trail Enhancement**: Task history records forced transitions with `forced=true` flag for compliance monitoring
5. **Helpful Error Messages**: Validation failures include current status, attempted status, valid next statuses, and force flag hint

This approach centralizes validation in the data layer, ensuring consistency across all entry points while providing safety and flexibility.

### Impact

- **Validation Consistency**: 100% of status transitions validated through single enforcement point (eliminates scattered hardcoded checks)
- **Emergency Flexibility**: Force flag enables critical operations while maintaining audit trail (reduces production incident resolution time by estimated 30%)
- **Data Integrity**: Atomic validation prevents race conditions where multiple operations update status concurrently
- **Compliance Ready**: Separate tracking of forced transitions enables abuse detection and security monitoring

---

## User Personas

### Persona 1: System (Data Integrity Guardian)

**Profile**:
- **Role/Title**: Repository layer responsible for enforcing data integrity constraints
- **Experience Level**: Core system component with deep knowledge of database schema and workflow rules
- **Key Characteristics**:
  - Enforces workflow validation for all status transitions
  - Provides single source of truth for what transitions are allowed
  - Maintains atomic operations (transaction management)
  - Tracks all transition attempts for audit compliance

**Goals Related to This Feature**:
1. Validate all status transitions against configured workflow before database update
2. Prevent invalid task states from entering database (data integrity)
3. Provide clear validation errors to calling code (CLI, API)
4. Support emergency overrides while maintaining audit trail

**Pain Points This Feature Addresses**:
- Previously had hardcoded validation scattered across CLI commands
- No way to enforce consistent validation across different entry points
- Forced transitions were indistinguishable from normal transitions in audit trail

**Success Looks Like**:
Repository validates every status transition, rejects invalid ones with actionable errors, allows force overrides with separate logging, and maintains complete audit trail of all transition attempts.

---

### Persona 2: Developer Agent (Workflow-Aware Consumer)

**Profile**:
- **Role/Title**: AI Developer Agent executing status transitions via repository layer
- **Experience Level**: Programmatic interaction, expects consistent validation across all operations
- **Key Characteristics**:
  - Calls repository methods directly (bypassing CLI)
  - Needs clear error responses for invalid transitions
  - Must handle validation failures gracefully
  - Requires machine-readable error information

**Goals Related to This Feature**:
1. Execute status transitions knowing validation will prevent invalid states
2. Receive structured error responses when transitions fail validation
3. Trust that validation is consistent regardless of entry point (CLI vs. programmatic)

**Pain Points This Feature Addresses**:
- Previously had to duplicate validation logic in agent code
- No guarantee that CLI and programmatic access enforced same rules
- Error responses were inconsistent across different access methods

**Success Looks Like**:
Agent calls `TaskRepository.UpdateStatus()`, validation happens automatically, invalid transitions fail with clear error listing valid next statuses, and agent can handle error programmatically.

---

### Persona 3: Tech Lead (Emergency Override User)

**Profile**:
- **Role/Title**: Tech Lead managing production incidents and emergency deployments
- **Experience Level**: Expert Shark user with deep system knowledge
- **Key Characteristics**:
  - Needs to bypass workflow validation during critical incidents
  - Responsible for production system stability
  - Must document emergency actions for post-incident review
  - Balances speed with safety during outages

**Goals Related to This Feature**:
1. Force status transitions during emergencies when workflow blocks critical operations
2. Document reason for forced transitions (notes field)
3. Ensure forced transitions are tracked separately for audit and abuse detection
4. Minimize production incident resolution time while maintaining compliance

**Pain Points This Feature Addresses**:
- Previously could not bypass workflow validation even in emergencies
- No way to document why forced transition was necessary
- Forced transitions were not distinguishable from normal transitions in audit logs

**Success Looks Like**:
Tech lead executes `shark task set-status <key> completed --force --notes="emergency hotfix for P0 incident"`, transition succeeds immediately, warning is displayed, and task history records transition with forced flag and notes for post-incident analysis.

---

## User Stories

### Must-Have Stories

**Story 1**: As a system, I need to enforce workflow transitions at the data layer to prevent invalid status changes.

**Acceptance Criteria**:
- [x] `TaskRepository.UpdateStatus()` validates transitions against workflow config
- [x] Validation occurs before database update (atomic check-and-set)
- [x] Invalid transitions rejected with error code 3 (invalid state)
- [x] Error message includes: current status, attempted status, list of valid next statuses
- [x] Error message includes hint: "Use --force to override validation"
- [x] Validation adds <100ms overhead (P95 latency)

**Implementation Status**: ✅ Completed in T-E11-F02-001 (commit 8cd4583)

---

**Story 2**: As a developer, I want invalid transitions to be blocked so that tasks don't enter impossible states.

**Acceptance Criteria**:
- [x] All status changes go through `UpdateStatus()` method (centralized enforcement)
- [x] Hardcoded status checks removed from CLI commands
- [x] Validation logic reads from workflow config (not hardcoded)
- [x] Same validation applies to CLI, API, and programmatic access

**Implementation Status**: ✅ Completed in T-E11-F02-001 (validation integration)

---

**Story 3**: As a tech lead, I want to force status transitions during emergencies so that I can deploy hotfixes without workflow blocking me.

**Acceptance Criteria**:
- [x] `UpdateStatus(force=true)` bypasses workflow validation
- [x] Forced transitions still validate that status exists in config (prevents typos)
- [x] Forced transitions logged in task_history with `forced=true` flag
- [x] Warning logged when forced transition executes: "⚠️  Forced transition bypassed workflow validation"
- [x] Forced transitions include notes field for documenting reason

**Implementation Status**: ✅ Completed in T-E11-F02-002 (force flag support)

---

**Story 4**: As a security auditor, I want all transition attempts logged so that I can detect workflow violations.

**Acceptance Criteria**:
- [x] Task history records successful transitions (already exists)
- [x] Task history records forced transitions with `forced=true` flag
- [x] System logs (not DB) record failed transition attempts with validation error
- [x] No transition attempt is invisible to audit trail
- [x] Forced flag queryable in task history for abuse detection

**Implementation Status**: ✅ Completed in T-E11-F02-002 (audit trail enhancement)

---

### Should-Have Stories

**Story 5**: As a developer, I want atomic status updates to prevent race conditions.

**Acceptance Criteria**:
- [x] Validation and update occur in same database transaction
- [x] Transaction rolled back if validation fails
- [x] Concurrent updates to same task are serialized (database-level locking)
- [x] No partial updates possible (atomicity guaranteed)

**Implementation Status**: ✅ Completed in T-E11-F02-001 (transaction wrapper)

---

**Story 6**: As a project manager, I want validation errors to include helpful guidance so that users can fix issues quickly.

**Acceptance Criteria**:
- [x] Error format: "Cannot transition from X to Y. Valid transitions: A, B, C"
- [x] Error includes hint about force flag for emergencies
- [x] Error distinguishes "invalid transition" from "status doesn't exist"
- [x] Error message is consistent across all repository methods

**Implementation Status**: ✅ Completed in T-E11-F02-001 (error message formatting)

---

### Edge Case & Error Stories

**Error Story 1**: As a developer, when I attempt an invalid transition, I want to know exactly what went wrong so that I can correct my command.

**Acceptance Criteria**:
- [x] Error shows current task status: "Task is currently in 'in_development'"
- [x] Error shows attempted status: "Cannot transition to 'completed'"
- [x] Error lists valid next statuses: "Valid next statuses: ready_for_review, blocked"
- [x] Error hints at force override: "Use --force to bypass validation"

**Implementation Status**: ✅ Completed (comprehensive error messages)

---

**Error Story 2**: As a system, when workflow config is invalid, I want to fail safely so that database doesn't get corrupted.

**Acceptance Criteria**:
- [x] Invalid workflow config detected at repository initialization
- [x] Repository falls back to default workflow if config invalid
- [x] Warning logged: "Invalid workflow config, using default workflow"
- [x] No database operations proceed with invalid config

**Implementation Status**: ✅ Completed (safe fallback to default workflow)

---

**Error Story 3**: As a tech lead, when I force a transition to a non-existent status, I want validation to prevent the typo.

**Acceptance Criteria**:
- [x] Force flag bypasses transition validation BUT NOT status existence validation
- [x] Forcing transition to undefined status fails with error: "Status 'complted' not defined in workflow config"
- [x] Error suggests similar status names: "Did you mean 'completed'?"

**Implementation Status**: ✅ Completed (force validates status existence)

---

## Requirements Traceability

This feature implements the following epic requirements:

### Functional Requirements

- **REQ-F-004**: Enforce Valid Transitions
  - Implemented by: T-E11-F02-001 (repository-level validation)

- **REQ-F-005**: Support Force Flag Override
  - Implemented by: T-E11-F02-002 (force flag in `UpdateStatus()`)

- **REQ-F-006**: Record All Transition Attempts
  - Implemented by: T-E11-F02-002 (audit trail with forced flag)

### Non-Functional Requirements

- **REQ-NF-001**: Low Latency Status Validation
  - Validation overhead measured at <100ms (P95), <20ms (P50)

- **REQ-NF-011**: Audit Trail Integrity
  - Task history records are immutable (INSERT only, no UPDATE/DELETE)

- **REQ-NF-012**: Force Flag Abuse Detection
  - Forced transitions logged separately, queryable for anomaly detection

- **REQ-NF-021**: Graceful Config Error Handling
  - Invalid workflow config triggers fallback to default workflow

- **REQ-NF-041**: Repository Layer Abstraction
  - Workflow validation isolated in repository, CLI calls repository methods

---

## Out of Scope

### Explicitly Excluded from This Feature

1. **CLI Command Implementation**
   - **Why**: CLI commands are handled in F03 (CLI Commands & Migration)
   - **Separation**: F02 provides repository methods, F03 provides user-facing commands
   - **Integration**: CLI commands call repository `UpdateStatus()` method

2. **Status Metadata Usage**
   - **Why**: Metadata-aware queries are in F04 (Agent Targeting & Metadata)
   - **Separation**: F02 validates transitions, F04 uses metadata for filtering
   - **Integration**: F04 repository extensions use metadata from workflow config

3. **Workflow Validation Command**
   - **Why**: Validation command is in F03 (workflow validate CLI command)
   - **Separation**: F02 validates at runtime, F03 validates config in CLI
   - **Workaround**: Validation logic is in F01, used by both F02 (runtime) and F03 (CLI)

4. **Database Schema Changes**
   - **Why**: No schema changes required for this feature
   - **Rationale**: Status field already TEXT, task_history already exists
   - **Note**: Forced flag added to task_history metadata (JSON field, no migration)

---

## Success Metrics

### Primary Metrics

1. **Validation Coverage**
   - **What**: Percentage of status transitions that go through repository validation
   - **Target**: 100% of transitions validated (no bypasses)
   - **Timeline**: Immediate (on release)
   - **Measurement**: Code audit confirms all status changes use `UpdateStatus()`

2. **Force Flag Usage Rate**
   - **What**: Percentage of transitions using force flag
   - **Target**: <5% of transitions require force override
   - **Timeline**: 30 days after release
   - **Measurement**: Query task_history for `forced=true` entries
   - **Interpretation**: Low usage indicates workflow matches actual process; high usage indicates workflow too restrictive

3. **Validation Performance**
   - **What**: P95 latency overhead for workflow validation
   - **Target**: <100ms (P95), <20ms (P50)
   - **Timeline**: Immediate (benchmarked before release)
   - **Measurement**: Repository benchmark tests with/without validation

---

### Secondary Metrics

- **Error Message Helpfulness**: Developer survey rating of validation error messages (target: 4.5/5.0)
- **Forced Transition Audit**: Weekly report of forced transitions for security review
- **Validation Failures**: Count of rejected transitions (indicates workflow enforcement effectiveness)

---

## Implementation Summary

### Tasks Completed

1. **T-E11-F02-001**: Integrate workflow validation into TaskRepository
   - Status: ✅ Completed (commit 8cd4583)
   - Deliverables:
     - Modified `TaskRepository` to accept workflow config in constructor
     - Added `UpdateStatus()` method with workflow validation
     - Atomic transaction wrapper for validation + update
     - Error messages with current status, attempted status, valid next statuses
     - Removed hardcoded status checks from repository

2. **T-E11-F02-002**: Implement force flag and audit trail
   - Status: ✅ Completed (commit 8cd4583)
   - Deliverables:
     - Force flag parameter in `UpdateStatus()` method
     - Forced transitions bypass workflow validation but validate status existence
     - Task history records forced flag: `{..., "forced": true}`
     - Warning logged when forced transition executes
     - Notes field in `UpdateStatus()` for documenting forced transitions

3. **T-E11-F02-003**: Write repository integration tests
   - Status: ✅ Completed (commit 8cd4583)
   - Deliverables:
     - Integration tests with real database for workflow enforcement
     - Test cases: valid transition, invalid transition, force override
     - Audit trail verification tests
     - Concurrent update tests (transaction isolation)
     - Test coverage >85% for workflow validation code paths

---

## Dependencies & Integrations

### Dependencies

- **F01: Workflow Configuration & Validation** (COMPLETED)
  - Provides `WorkflowConfig` structure and validation logic
  - Provides `CanTransition()` method for validation
  - Provides default workflow fallback

### Integration Points

- **TaskRepository Constructor**: Accepts `WorkflowConfig` parameter for injection
- **UpdateStatus() Method**: Central validation enforcement point
- **Task History**: Records forced flag and transition metadata
- **Database Transactions**: Ensures atomic validation and update

### Integration with Other Features

- **F03: CLI Commands**: CLI calls `TaskRepository.UpdateStatus()` with force flag from command-line
- **F04: Agent Targeting**: Agent queries filtered by metadata use same repository validation
- **HTTP API** (future): API endpoints will call same repository methods, ensuring consistent validation

---

## Testing Strategy

### Integration Tests (Completed)

**Scope**:
- Workflow validation with real database
- Transaction atomicity (validation + update in single transaction)
- Audit trail verification (forced flag recorded correctly)
- Concurrent update handling
- Force flag behavior

**Test Cases**:
1. **Valid Transition**: Task in `todo` → `in_progress` succeeds
2. **Invalid Transition**: Task in `todo` → `completed` fails with validation error
3. **Force Override**: Task in `todo` → `completed --force` succeeds with warning
4. **Force Invalid Status**: Task in `todo` → `complted --force` fails (typo in status name)
5. **Audit Trail**: Forced transition recorded with `forced=true` in task_history
6. **Concurrent Updates**: Two simultaneous updates to same task are serialized
7. **Fallback to Default**: Invalid workflow config triggers default workflow with warning

**Coverage**: >85% line coverage for workflow validation code in repository

### Unit Tests (Completed)

**Scope**:
- Validation logic with mocked workflow config
- Error message formatting
- Force flag parameter handling
- Status existence validation

---

## Documentation Requirements

### Code Documentation (✅ Completed)

- GoDoc comments on `UpdateStatus()` method
- Parameter documentation for force flag and notes
- Error return value documentation
- Example usage in method comments

### Developer Guide (Recommended Follow-Up)

Create guides for:
- "Repository Integration Architecture" - How workflow validation integrates with data layer
- "Using Force Flag Safely" - When to use force overrides and how to document them
- "Audit Trail Analysis" - Querying task history for forced transitions
- "Custom Repository Extensions" - How to extend repository while preserving validation

---

## Compliance & Security Considerations

### Audit Trail Integrity

- **REQ-NF-011**: Task history records are immutable (INSERT only)
- No UPDATE or DELETE operations on task_history table
- Forced transitions logged separately for compliance review
- All transition attempts visible in audit trail (success and failure)

### Force Flag Abuse Detection

- **REQ-NF-012**: Forced transitions logged separately for monitoring
- Recommendation: Weekly report of forced transitions to security team
- High force usage (>5%) indicates workflow doesn't match reality (needs adjustment)
- Anomaly detection: Alert if same user forces >10 transitions in 24 hours

### Transaction Isolation

- Validation and update occur in single transaction (atomic)
- Prevents race conditions where status checked then updated separately
- Database-level locking prevents concurrent modifications to same task

---

## Future Enhancements

Potential enhancements for future iterations (not committed):

1. **Transition Hooks**: Execute custom code on status transitions (webhooks, notifications)
2. **Conditional Transitions**: Allow transitions based on task metadata (e.g., priority, agent type)
3. **Transition Permissions**: Restrict certain transitions to specific user roles
4. **Bulk Transitions**: Efficiently update status for multiple tasks atomically
5. **Transition Rollback**: Undo status change and restore previous state

---

*Last Updated*: 2025-12-29
*Status*: 3 of 3 tasks complete (100% done)
*Phase*: Complete - Runtime enforcement foundation for F03 and F04
