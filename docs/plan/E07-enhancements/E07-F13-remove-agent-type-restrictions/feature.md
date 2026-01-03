---
feature_key: E07-F13-remove-agent-type-restrictions
epic_key: E07
title: Remove Agent Type Restrictions
description: Remove hardcoded agent type validation to enable flexible multi-agent workflows with custom agent types
---

# Remove Agent Type Restrictions

**Feature Key**: E07-F13-remove-agent-type-restrictions

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem

The current system restricts the `--agent` flag to a predefined set of six values: `frontend`, `backend`, `api`, `testing`, `devops`, and `general`. This limitation prevents users from creating tasks for custom agent types needed in multi-agent workflows (e.g., `architect`, `business-analyst`, `product-manager`, `qa`, `tech-lead`, `ux-designer`, etc.). While E07-F01 made the agent field optional, the validation still rejects any non-standard agent type values, blocking modern AI-driven development workflows that require flexible agent assignments.

The validation occurs in three places:
1. `internal/models/validation.go` - `ValidateAgentType()` function restricts to hardcoded list
2. `internal/taskcreation/validator.go` - calls `ValidateAgentType()` during task creation
3. `internal/models/task.go` - `Task.Validate()` method calls `ValidateAgentType()`
4. `internal/templates/renderer.go` - validates agent type when rendering templates

This creates friction when:
- Creating tasks for specialized roles in multi-agent systems
- Using custom agent types that match specific team structures
- Implementing AI agent orchestration workflows (Product Manager, Business Analyst, Architect, etc.)

### Solution

Remove all hardcoded agent type restrictions and allow any non-empty string value for the `--agent` flag. This aligns with the existing database schema (already stores `agent_type` as TEXT with no constraints) and enables flexible multi-agent workflows. The solution involves:

1. **Remove validation restrictions**: Update `ValidateAgentType()` to accept any non-empty string
2. **Deprecate agent type constants**: Mark `AgentType` enum constants as deprecated
3. **Update documentation**: Add recommended agent types (instead of enforced ones)
4. **Maintain backward compatibility**: Existing tasks with standard agent types continue working
5. **Template fallback**: Use `general` template for any unknown agent type

This is a minimal, non-breaking change that removes artificial restrictions without requiring database migrations or major refactoring.

### Impact

**Immediate Benefits**:
- Enable multi-agent workflow orchestration (Product Manager dispatches Business Analyst, Architect, Developer, QA)
- Support custom agent roles specific to team structures (data-engineer, security-auditor, etc.)
- Remove friction from task creation for non-standard roles
- Align validation with database capabilities (TEXT field already accepts any value)

**Measurable Outcomes**:
- Support for 10+ agent types beyond the current 6 hardcoded values
- Zero breaking changes (100% backward compatibility)
- Enable creation of tasks for all agent types in multi-agent SDLC workflows

---

## User Personas

### Persona 1: AI Agent Orchestrator / Product Manager

**Profile**:
- **Role/Title**: Product Manager or AI orchestration system managing multi-agent workflows
- **Experience Level**: Experienced with SDLC workflows and task management systems
- **Key Characteristics**:
  - Coordinates multiple specialized agents (Business Analyst, Architect, Developer, QA, etc.)
  - Needs to assign tasks to agents based on skillset and current workflow phase
  - Works with modern AI-driven development processes

**Goals Related to This Feature**:
1. Create tasks for any agent type in the workflow (architect, business-analyst, qa, tech-lead, etc.)
2. Implement flexible agent assignment without system restrictions
3. Support custom agent roles specific to project needs

**Pain Points This Feature Addresses**:
- Cannot create tasks for `architect` or `business-analyst` agents due to validation errors
- Forced to use inappropriate agent types (e.g., `general`) for specialized roles
- Blocked from implementing multi-agent orchestration workflows

**Success Looks Like**:
Can create tasks for any agent type needed in the workflow. System accepts custom agent values without validation errors. Multi-agent SDLC workflows operate smoothly with tasks assigned to appropriate specialized agents.

---

### Persona 2: Development Team Lead

**Profile**:
- **Role/Title**: Technical Lead managing diverse development team
- **Experience Level**: 5+ years leading engineering teams with varied specializations
- **Key Characteristics**:
  - Manages teams with custom roles beyond standard frontend/backend
  - Needs to track work for specialized roles (data-engineer, ml-engineer, security-auditor)
  - Values flexibility in task management systems

**Goals Related to This Feature**:
1. Assign tasks to team members based on actual roles, not restricted categories
2. Support evolving team structures without system changes
3. Track specialized work that doesn't fit into standard categories

**Pain Points This Feature Addresses**:
- Cannot represent actual team structure in task assignments
- Forced to use generic `general` agent type for specialized roles
- System doesn't support modern development team structures (MLOps, DataOps, SecOps, etc.)

**Success Looks Like**:
Can create tasks for any role on the team. System adapts to team structure instead of forcing team to adapt to system. Custom agent types are stored and displayed correctly throughout the system.

---

## User Stories

### Must-Have Stories

**Story 1**: As an AI orchestrator, I want to create tasks with custom agent types (architect, business-analyst, qa) so that I can implement multi-agent workflows.

**Acceptance Criteria**:
- [ ] `shark task create --agent=architect` succeeds without validation error
- [ ] `shark task create --agent=business-analyst` succeeds without validation error
- [ ] `shark task create --agent=qa` succeeds without validation error
- [ ] Custom agent types are stored correctly in database
- [ ] Custom agent types display correctly in `task list` and `task get` commands

**Story 2**: As a developer, I want to create tasks for any custom agent type so that I can match my team's specific roles.

**Acceptance Criteria**:
- [ ] Any non-empty string is accepted for `--agent` flag
- [ ] Agent types like `data-engineer`, `ml-specialist`, `security-auditor` are accepted
- [ ] Hyphenated and underscore-separated agent names work correctly
- [ ] Agent type is preserved exactly as entered (no normalization)

**Story 3**: As a user, I want existing tasks with standard agent types to continue working unchanged so that there are no breaking changes.

**Acceptance Criteria**:
- [ ] Existing tasks with `frontend`, `backend`, `api`, `testing`, `devops`, `general` work unchanged
- [ ] All existing tests pass without modification
- [ ] No database migration required
- [ ] Template rendering for standard agent types works as before

---

### Should-Have Stories

**Story 4**: As a developer, when I use a custom agent type, I want clear feedback about which template is being used so that I understand the generated file structure.

**Acceptance Criteria**:
- [ ] CLI output indicates which template was selected for custom agent types
- [ ] Message shows fallback to `general` template when custom agent type has no specific template
- [ ] Documentation lists recommended agent types vs. custom agent types

---

### Could-Have Stories

**Story 5**: As a team, we may want to define preferred agent types in `.sharkconfig.json` for autocomplete suggestions (future enhancement).

**Acceptance Criteria**:
- [ ] Out of scope for this feature - note as future enhancement
- [ ] System still accepts any agent type, config only provides suggestions
- [ ] No enforcement of configured agent types

---

### Edge Case & Error Stories

**Error Story 1**: As a user, when I provide an empty string for `--agent`, I want to see a clear error message so that I provide a valid value.

**Acceptance Criteria**:
- [ ] Empty string for agent type is rejected with clear error message
- [ ] Error message: "agent type cannot be empty"
- [ ] Task creation is aborted, no database entry created

**Error Story 2**: As a user, when I use a custom agent type without a specific template, I want the system to fall back gracefully so that task creation succeeds.

**Acceptance Criteria**:
- [ ] Unknown agent types fall back to `general` template automatically
- [ ] No error thrown for unknown agent types
- [ ] Success message indicates which template was used

---

## Requirements

### Functional Requirements

**Category: Agent Type Validation**

1. **REQ-F-001**: Remove Hardcoded Agent Type Restrictions
   - **Description**: The `ValidateAgentType()` function must accept any non-empty string value for agent type
   - **User Story**: Links to Story 1, Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Update `internal/models/validation.go::ValidateAgentType()` to accept any non-empty string
     - [ ] Remove hardcoded map of valid agent types
     - [ ] Add deprecation comment on `AgentType` enum constants in `internal/models/task.go`
     - [ ] All validation call sites work with custom agent types

2. **REQ-F-002**: Template Fallback for Custom Agent Types
   - **Description**: When agent type has no specific template, fall back to `general` template without error
   - **User Story**: Links to Story 2, Error Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] `internal/templates/renderer.go` falls back to `general` template for unknown agent types
     - [ ] No validation error thrown when agent type has no specific template
     - [ ] Template fallback logic tested with custom agent type

3. **REQ-F-003**: Preserve Backward Compatibility
   - **Description**: Existing tasks with standard agent types must continue working unchanged
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] All existing tests pass without modification
     - [ ] Standard agent types (`frontend`, `backend`, `api`, `testing`, `devops`, `general`) work as before
     - [ ] No database migration required
     - [ ] No breaking changes to CLI commands or repository methods

**Category: User Feedback**

4. **REQ-F-004**: Clear Feedback for Template Selection
   - **Description**: CLI output must indicate which template was used when creating task with custom agent type
   - **User Story**: Links to Story 4
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] Success message includes template information
     - [ ] Message indicates when fallback to `general` template occurred
     - [ ] Output is helpful and non-cryptic

---

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF-001**: Zero Breaking Changes
   - **Description**: All existing functionality must work unchanged
   - **Measurement**: 100% test pass rate, no database migration required
   - **Target**: 100% backward compatibility
   - **Justification**: Critical system - cannot break existing workflows

**Performance**

1. **REQ-NF-002**: No Performance Degradation
   - **Description**: Validation changes must not impact task creation performance
   - **Measurement**: Task creation time before/after change
   - **Target**: <1ms difference in validation time
   - **Justification**: Validation is called on every task create/update

**Usability**

1. **REQ-NF-003**: Clear Documentation
   - **Description**: Documentation must explain recommended vs. custom agent types
   - **Implementation**: Update CLI_REFERENCE.md and CLAUDE.md
   - **Justification**: Users should understand which agent types have specific templates vs. fallback to general

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Create Task With Custom Agent Type (Architect)**
- **Given** a user wants to create a task for an architect agent
- **When** they run `shark task create "Design API Architecture" --epic=E10 --feature=F01 --agent=architect`
- **Then** the task is created successfully with agent_type="architect"
- **And** the `general` template is used (no architect-specific template exists)
- **And** success message indicates task was created

**Scenario 2: Create Task With Custom Agent Type (Business Analyst)**
- **Given** a user wants to create a task for a business analyst agent
- **When** they run `shark task create "Elaborate User Stories" --epic=E10 --feature=F01 --agent=business-analyst`
- **Then** the task is created successfully with agent_type="business-analyst"
- **And** the custom agent type is stored in the database
- **And** task displays correctly in `task list` and `task get` commands

**Scenario 3: Backward Compatibility - Standard Agent Types**
- **Given** existing tasks use standard agent types (frontend, backend, etc.)
- **When** a user creates a new task with `--agent=backend`
- **Then** the task is created with the backend-specific template (if available)
- **And** behavior is identical to before this change
- **And** all existing tests pass

**Scenario 4: Empty Agent Type Rejected**
- **Given** a user provides an empty string for agent type
- **When** they run `shark task create "Task" --epic=E10 --feature=F01 --agent=""`
- **Then** an error message is displayed: "agent type cannot be empty"
- **And** task creation is aborted

**Scenario 5: Custom Agent Type in Filtering**
- **Given** tasks exist with custom agent types
- **When** a user runs `shark task list --agent=architect`
- **Then** only tasks with agent_type="architect" are displayed
- **And** filtering works correctly for custom agent types

---

## Technical Analysis

### Current Implementation

**Validation occurs in 4 locations:**

1. **`internal/models/validation.go:159-174`** (PRIMARY)
```go
func ValidateAgentType(agentType string) error {
	validTypes := map[string]bool{
		"frontend": true,
		"backend":  true,
		"api":      true,
		"testing":  true,
		"devops":   true,
		"general":  true,
	}
	if !validTypes[agentType] {
		return fmt.Errorf("invalid agent type '%s'. Valid types are: frontend, backend, api, testing, devops, general", agentType)
	}
	return nil
}
```

2. **`internal/taskcreation/validator.go:77`** (CALLER)
```go
if err := models.ValidateAgentType(input.AgentType); err != nil {
	return nil, err
}
```

3. **`internal/models/task.go:93`** (CALLER)
```go
if t.AgentType != nil {
	if err := ValidateAgentType(string(*t.AgentType)); err != nil {
		return err
	}
}
```

4. **`internal/templates/renderer.go:41`** (CALLER)
```go
if err := models.ValidateAgentType(string(agentType)); err != nil {
	return "", err
}
```

### Recommended Approach: **Option A - Free-Text Agent Field**

**Rationale:**
- Database already stores agent_type as TEXT (no constraints)
- Simplest implementation (minimal code changes)
- Maximum flexibility for users
- No configuration overhead
- Aligns with E07-F01 intent to make agent field flexible

**Implementation:**
1. Update `ValidateAgentType()` to only reject empty strings
2. Add deprecation comments on `AgentType` enum constants
3. Update template renderer to fall back to `general` for unknown agent types
4. Update documentation to list "recommended" agent types (not enforced)

**Rejected: Option B - Config-Based Agent Types**

**Why Rejected:**
- Adds complexity with `.sharkconfig.json` management
- Requires config validation and error handling
- Users must update config for new agent types
- Over-engineered for the use case
- Goes against the simplicity principle

If teams want to standardize agent types, they can document them without enforcing at the system level.

---

## Implementation Tasks Breakdown

### Task 1: Update ValidateAgentType Function (M)
**Description**: Modify `internal/models/validation.go` to accept any non-empty string for agent type

**Details**:
- Remove hardcoded `validTypes` map
- Check only for non-empty string
- Update error message to be generic
- Maintain function signature for backward compatibility

**Acceptance Criteria**:
- [ ] Function accepts any non-empty string
- [ ] Rejects empty string with clear error message
- [ ] All callers continue working without changes
- [ ] Unit tests updated to cover custom agent types

**Estimated Complexity**: M (Medium - straightforward logic change, multiple call sites to verify)

---

### Task 2: Deprecate AgentType Enum Constants (XS)
**Description**: Add deprecation comments to `AgentType` constants in `internal/models/task.go`

**Details**:
- Add deprecation notice to `AgentType` type definition
- Add deprecation notice to each constant (`AgentTypeFrontend`, etc.)
- Suggest using string literals directly or custom agent types
- Do NOT remove constants (breaking change)

**Acceptance Criteria**:
- [ ] Deprecation comments added to type and all constants
- [ ] Comments explain migration path
- [ ] No code functionality changed

**Estimated Complexity**: XS (Extra Small - documentation only)

---

### Task 3: Update Template Renderer Validation (S)
**Description**: Modify `internal/templates/renderer.go` to handle unknown agent types gracefully

**Details**:
- Remove `ValidateAgentType()` call from template renderer
- Add fallback logic to use `general` template for unknown agent types
- Log or inform when fallback occurs (optional)
- Maintain existing behavior for standard agent types

**Acceptance Criteria**:
- [ ] Unknown agent types don't cause validation errors
- [ ] Fallback to `general` template works correctly
- [ ] Standard agent types still use specific templates
- [ ] Template rendering tests updated

**Estimated Complexity**: S (Small - simple fallback logic)

---

### Task 4: Update Task Creation Validator (S)
**Description**: Verify `internal/taskcreation/validator.go` works correctly with updated validation

**Details**:
- Review validator logic after `ValidateAgentType()` change
- Ensure custom agent types are stored correctly
- Test task creation with various agent type values
- Verify default to `general` when agent not provided still works

**Acceptance Criteria**:
- [ ] Task creation succeeds with custom agent types
- [ ] Agent type stored correctly in database
- [ ] Default to `general` when empty still works
- [ ] Integration tests cover custom agent types

**Estimated Complexity**: S (Small - verification and testing, minimal code changes)

---

### Task 5: Update Documentation (S)
**Description**: Update CLAUDE.md, README.md, and CLI_REFERENCE.md with agent type changes

**Details**:
- Update CLAUDE.md section on agent types
- List recommended agent types (not enforced)
- Add examples of custom agent types
- Update CLI_REFERENCE.md with flexible agent type explanation
- Document template fallback behavior

**Acceptance Criteria**:
- [ ] CLAUDE.md updated with agent type flexibility
- [ ] Examples include custom agent types (architect, business-analyst, qa)
- [ ] CLI_REFERENCE.md explains recommended vs. custom agent types
- [ ] Template fallback behavior documented

**Estimated Complexity**: S (Small - documentation updates)

---

### Task 6: Add Tests for Custom Agent Types (M)
**Description**: Add comprehensive tests for custom agent type functionality

**Details**:
- Unit tests for `ValidateAgentType()` with custom values
- Integration tests for task creation with custom agent types
- Tests for template fallback behavior
- Tests for task filtering with custom agent types
- Tests for task list/get display with custom agent types

**Acceptance Criteria**:
- [ ] Unit tests cover custom agent type validation
- [ ] Integration tests verify end-to-end custom agent type workflow
- [ ] Template fallback tested with unknown agent types
- [ ] Filtering tests include custom agent types
- [ ] All tests pass with 100% coverage of new logic

**Estimated Complexity**: M (Medium - comprehensive test coverage across multiple layers)

---

## Out of Scope

### Explicitly Excluded

1. **Config-based agent type whitelist in `.sharkconfig.json`**
   - **Why**: Adds complexity without clear benefit; free-text is simpler and more flexible
   - **Future**: Could add as optional autocomplete suggestion mechanism (non-enforcing)
   - **Workaround**: Teams can document recommended agent types without system enforcement

2. **Agent-specific template auto-creation**
   - **Why**: Out of scope for this feature; focuses on removing restrictions, not adding template management
   - **Future**: Separate feature for custom template management (E07-F14?)
   - **Workaround**: Use `--template` flag for custom agent-specific templates

3. **Agent type normalization or canonicalization**
   - **Why**: Preserves user intent; no reason to normalize agent type strings
   - **Future**: Not planned - exact string match is desired behavior
   - **Workaround**: Users ensure consistency in agent type naming

4. **Migration of existing agent types to new format**
   - **Why**: No database changes needed; existing agent_type TEXT field already supports any value
   - **Future**: N/A - no migration required
   - **Workaround**: N/A

---

### Alternative Approaches Rejected

**Alternative 1: Config-Based Agent Type Registry**
- **Description**: Define allowed agent types in `.sharkconfig.json` with validation
- **Why Rejected**:
  - Adds complexity (config parsing, validation, error handling)
  - Requires users to update config for every new agent type
  - Goes against simplicity principle
  - Database already supports arbitrary strings
  - Creates maintenance burden without clear benefit

**Alternative 2: Agent Type Normalization**
- **Description**: Normalize agent types to lowercase/slug format (e.g., "Business Analyst" → "business-analyst")
- **Why Rejected**:
  - User intent is clearer with exact string preservation
  - Adds complexity for minimal benefit
  - Case-sensitive matching is acceptable (users control consistency)
  - No existing issue with non-normalized agent types

**Alternative 3: Agent Type Plugin System**
- **Description**: Create extensible plugin system for registering custom agent types
- **Why Rejected**:
  - Massive over-engineering for simple use case
  - Adds architectural complexity
  - No need for plugin lifecycle management
  - Free-text approach achieves same goal with zero complexity

---

## Success Metrics

### Primary Metrics

1. **Custom Agent Type Adoption**
   - **What**: Number of distinct agent_type values beyond the original 6 hardcoded types
   - **Target**: >10 distinct custom agent types used in production within 1 month
   - **Measurement**: `SELECT DISTINCT agent_type FROM tasks WHERE agent_type NOT IN ('frontend', 'backend', 'api', 'testing', 'devops', 'general')`

2. **Zero Breaking Changes**
   - **What**: All existing tests pass without modification
   - **Target**: 100% test pass rate
   - **Measurement**: CI/CD test suite results

---

### Secondary Metrics

- **Multi-Agent Workflow Usage**: Count of tasks created with agent types: `architect`, `business-analyst`, `qa`, `tech-lead`, `product-manager`
- **Template Fallback Rate**: Percentage of task creates that use fallback to `general` template due to unknown agent type
- **User-Reported Issues**: Zero validation errors related to agent type restrictions after deployment

---

## Dependencies & Integrations

### Dependencies

- **`internal/models/validation.go`**: Core validation function that must be updated
- **`internal/taskcreation/validator.go`**: Calls `ValidateAgentType()` during task creation
- **`internal/models/task.go`**: Task model validation that calls `ValidateAgentType()`
- **`internal/templates/renderer.go`**: Template selection logic that validates agent types
- **`internal/cli/commands/task.go`**: CLI commands that use agent type filtering

### Integration Requirements

None - this is an internal validation change with no external integrations.

**Affected CLI Commands**:
- `shark task create --agent=<value>` - accepts custom agent types
- `shark task list --agent=<value>` - filters by custom agent types
- `shark task next --agent=<value>` - finds next task for custom agent types
- `shark task update --agent=<value>` - updates task with custom agent type

**Database Impact**:
- Zero database migration required
- `agent_type` column already defined as TEXT (no constraints)
- Existing data remains valid and unchanged

---

## Compliance & Security Considerations

**No specific regulatory, data protection, or audit requirements for this feature.**

**Security Considerations**:
- Agent type is a metadata field (not executable)
- No SQL injection risk (parameterized queries used throughout)
- No XSS risk (CLI tool, not web interface)
- Input validation ensures non-empty string (prevents NULL constraint violations)

**Data Integrity**:
- Agent type stored as TEXT in database (already supports any value)
- Validation ensures data quality (non-empty string)
- No foreign key constraints or referential integrity issues

---

## Implementation Plan

### Phase 1: Core Validation Changes (High Priority)
1. Task 1: Update ValidateAgentType Function
2. Task 2: Deprecate AgentType Enum Constants
3. Task 3: Update Template Renderer Validation
4. Task 4: Update Task Creation Validator

### Phase 2: Testing & Documentation (High Priority)
5. Task 6: Add Tests for Custom Agent Types
6. Task 5: Update Documentation

### Testing Strategy
- Unit tests for validation changes
- Integration tests for end-to-end workflows
- Backward compatibility tests with standard agent types
- Manual testing with multi-agent workflow scenarios

### Deployment
- No database migration required
- Deploy with full test suite passing
- Monitor for any validation errors in logs
- Quick rollback available (revert validation change)

---

*Last Updated*: 2026-01-01
