---
feature_key: E07-F21
epic_key: E07
title: Add Orchestrator Actions to Status Transitions
description: Extend Shark's status_metadata configuration to include orchestrator_action instructions, enabling workflow-driven agent spawning without hardcoded orchestrator logic
execution_order: 21
status: active
priority: high
related_documents:
  - docs/plan/E07-enhancements/E07-F21-add-actions-to-status-transition/shark-orchestrator-actions-feature-request.md
  - docs/plan/E07-enhancements/E07-F21-add-actions-to-status-transition/shark-workflow-config-design.md
  - docs/plan/E07-enhancements/E07-F21-add-actions-to-status-transition/shark-workflow-config-summary.md
---

# Add Orchestrator Actions to Status Transitions

**Feature Key**: E07-F21
**Epic**: E07 - Enhancements
**Status**: Active
**Priority**: High

---

## Related Documents

- [Feature Request](./shark-orchestrator-actions-feature-request.md)
- [Workflow Configuration Design](./shark-workflow-config-design.md)
- [Configuration Summary](./shark-workflow-config-summary.md)

---

## Goal

### Problem

AI Agent Orchestrators currently have **hardcoded workflow knowledge** about which agents to spawn for each status transition. This creates tight coupling between Shark's workflow definitions and orchestrator implementation:

```go
// Hardcoded in orchestrator
statusToAgent := map[string]string{
    "ready_for_refinement_ba": "business-analyst",
    "ready_for_refinement_tech": "architect",
    "ready_for_development": "developer",
}
```

**Problems**:
1. Workflow knowledge is duplicated between Shark (state) and Orchestrator (execution)
2. Adding new workflow stages requires orchestrator code changes
3. Modifying agent instructions requires orchestrator rebuilds
4. Cannot support different workflows for different projects
5. Orchestrator must make separate queries after transitions

### Solution

Add `orchestrator_action` metadata to Shark's existing `.sharkconfig.json` `status_metadata` configuration. When tasks transition to new statuses, Shark returns the orchestrator action as part of the transition response, providing execution instructions immediately.

**Key Benefit**: Decouples orchestrator logic from workflow definitions. Workflow configuration becomes the single source of truth for both state management (Shark) and execution instructions (Orchestrator).

### Impact

**Primary Metrics**:
- **Development Velocity**: Adding new workflow stages reduced from ~8 hours (code changes) to ~2 hours (YAML config)
- **API Efficiency**: Eliminate separate queries for orchestrator actions (reduce API calls by 50%)
- **Workflow Flexibility**: Support multiple workflows without code changes

**Secondary Metrics**:
- **Time to Production**: New workflows deployable in minutes vs hours
- **Error Reduction**: Automated validation catches configuration errors before runtime
- **Documentation Quality**: Configuration serves as self-documenting workflow definition

---

## User Personas

### Persona 1: AI Agent Orchestrator Developer

**Profile**:
- **Role/Title**: Software Engineer working on multi-agent orchestration systems
- **Experience Level**: 3-5 years, proficient in Go, distributed systems
- **Key Characteristics**:
  - Builds systems that coordinate multiple AI agents
  - Needs to support different workflow stages and transitions
  - Values clean separation of concerns and maintainability

**Goals Related to This Feature**:
1. Add new workflow stages without modifying orchestrator code
2. Support multiple client workflows from single orchestrator codebase
3. Reduce API queries needed for task coordination

**Pain Points This Feature Addresses**:
- Hardcoded workflow logic makes changes expensive
- Each new workflow stage requires code changes, testing, deployment
- Need to query Shark twice: once for transition, once for action

**Success Looks Like**:
Orchestrator receives complete execution instructions in status transition responses. Can add new workflows by creating YAML config files. No code changes needed for workflow variations.

---

### Persona 2: Workflow Configuration Manager

**Profile**:
- **Role/Title**: Product Manager or Technical Lead defining AI agent workflows
- **Experience Level**: Strong product/technical background, moderate YAML proficiency
- **Key Characteristics**:
  - Defines how tasks flow through different stages
  - Needs to customize workflows per project
  - Wants to validate workflows before deployment

**Goals Related to This Feature**:
1. Define agent behavior and transitions declaratively
2. Modify workflows without requiring engineering changes
3. Validate workflow completeness and correctness

**Pain Points This Feature Addresses**:
- Workflow changes require coordinating with orchestrator team
- No way to validate workflow definitions before deployment
- Documentation drifts from implementation

**Success Looks Like**:
Can define complete workflows in YAML configuration. Shark validates configurations automatically. Workflows are self-documenting and version-controlled.

---

## User Stories

### Must-Have Stories

**Story 1**: As an orchestrator developer, when an agent completes work and transitions a task status, I want to receive the next orchestrator action in the response so that I can immediately spawn the next agent without an additional query.

**Acceptance Criteria**:
- [ ] `shark task update` returns `orchestrator_action` object when status changes
- [ ] Action includes `agent_type`, `skills`, and `instruction_template` fields
- [ ] Template variables (e.g., `{task_id}`) are populated in instructions
- [ ] Response is backward compatible (missing actions don't break existing code)
- [ ] JSON output format is documented and versioned

---

**Story 2**: As a workflow configuration manager, I want to define orchestrator actions for each status in `.sharkconfig.json` so that the workflow configuration is the single source of truth.

**Acceptance Criteria**:
- [ ] `status_metadata` supports optional `orchestrator_action` field
- [ ] Action types supported: `spawn_agent`, `pause`, `wait_for_triage`, `archive`
- [ ] Schema validation ensures required fields present for each action type
- [ ] Shark CLI validates configuration on load
- [ ] Configuration examples provided for common workflows

---

**Story 3**: As an orchestrator developer, I want to query what action would be taken for a status without transitioning so that I can debug workflow configurations.

**Acceptance Criteria**:
- [ ] `shark config get-status-action <status>` returns action definition
- [ ] Command supports `--task` flag to populate template variables
- [ ] JSON output format matches transition response format
- [ ] Error messages are clear for missing or invalid actions
- [ ] Command works with both JSON and human-readable output

---

**Story 4**: As a workflow configuration manager, I want to validate that all actionable statuses have orchestrator actions defined so that I catch configuration errors before deployment.

**Acceptance Criteria**:
- [ ] `shark workflow validate-actions` checks all statuses
- [ ] Warns if "ready_for_*" statuses lack actions
- [ ] `--strict` flag fails on any missing actions
- [ ] Validates action schema correctness
- [ ] Output lists all statuses with validation results

---

### Should-Have Stories

**Story 5**: As an orchestrator developer, I want task list queries to optionally include orchestrator actions so that polling loops can batch-fetch all needed information.

**Acceptance Criteria**:
- [ ] `shark task list` supports `--with-actions` flag
- [ ] Actions included for each task in response
- [ ] Backward compatible (default behavior unchanged)
- [ ] Performance impact is acceptable (<10% slower)

---

**Story 6**: As a workflow configuration manager, I want to view all orchestrator actions in my workflow so that I can understand the complete agent flow.

**Acceptance Criteria**:
- [ ] `shark workflow show-actions` displays all actions
- [ ] Output grouped by workflow phase
- [ ] Shows agent type and action type for each status
- [ ] Supports JSON format for programmatic access

---

### Edge Case & Error Stories

**Error Story 1**: As an orchestrator developer, when a status has no orchestrator action defined, I want to receive a clear response so that I can handle it gracefully.

**Acceptance Criteria**:
- [ ] Missing actions are omitted from response (not null)
- [ ] Orchestrator can detect missing action and use fallback logic
- [ ] Logs indicate when fallback is used
- [ ] Configuration validation warns about missing actions

---

**Error Story 2**: As a workflow configuration manager, when I define an invalid orchestrator action, I want immediate feedback so that I can fix the configuration before deployment.

**Acceptance Criteria**:
- [ ] Shark CLI validates configuration on load
- [ ] Error messages specify exact issue and location
- [ ] Validation catches: missing required fields, invalid action types, bad template syntax
- [ ] Configuration loading fails fast with clear error

---

## Requirements

### Functional Requirements

**Category: Configuration Schema**

1. **REQ-F-001**: Orchestrator Action Schema
   - **Description**: `status_metadata` must support optional `orchestrator_action` object with fields: `action` (enum), `agent_type` (string), `skills` (array), `instruction_template` (string)
   - **User Story**: Links to Story #2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Schema defined in JSON/YAML format
     - [ ] Validation enforces required fields per action type
     - [ ] Backward compatible with existing configs

2. **REQ-F-002**: Action Types
   - **Description**: Support four action types: `spawn_agent` (launch agent), `pause` (wait), `wait_for_triage` (human decision), `archive` (complete)
   - **User Story**: Links to Story #2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Each action type has clear semantics
     - [ ] Required fields validated per type
     - [ ] Documentation includes when to use each type

3. **REQ-F-003**: Template Variable Substitution
   - **Description**: `instruction_template` supports `{task_id}` placeholder, replaced with actual task ID
   - **User Story**: Links to Story #1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Simple string replacement implementation
     - [ ] Works with all output formats
     - [ ] Future extensible to additional variables

---

**Category: CLI Response Enhancement**

4. **REQ-F-004**: Task Update Response Enhancement
   - **Description**: `shark task update` returns `orchestrator_action` in response when status changes
   - **User Story**: Links to Story #1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] JSON output includes `orchestrator_action` object
     - [ ] Human-readable output shows action summary
     - [ ] Missing actions handled gracefully (omit field)
     - [ ] Backward compatible with existing clients

5. **REQ-F-005**: Task List Actions Flag
   - **Description**: `shark task list --with-actions` optionally includes actions for each task
   - **User Story**: Links to Story #5
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] Opt-in via flag (default behavior unchanged)
     - [ ] Actions populated for current task status
     - [ ] Performance acceptable for large result sets

---

**Category: CLI Utility Commands**

6. **REQ-F-006**: Get Status Action Command
   - **Description**: `shark config get-status-action <status>` returns action definition
   - **User Story**: Links to Story #3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Works with status name parameter
     - [ ] Optional `--task` flag populates template
     - [ ] JSON and human-readable formats
     - [ ] Clear error messages

7. **REQ-F-007**: Validate Actions Command
   - **Description**: `shark workflow validate-actions` checks workflow completeness
   - **User Story**: Links to Story #4
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Validates all statuses have actions (with warnings)
     - [ ] `--strict` mode fails on missing actions
     - [ ] Validates schema correctness
     - [ ] Output shows validation results per status

8. **REQ-F-008**: Show Actions Command
   - **Description**: `shark workflow show-actions` displays all workflow actions
   - **User Story**: Links to Story #6
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] Lists all actions grouped by phase
     - [ ] Shows agent type and action type
     - [ ] Supports JSON output
     - [ ] Human-readable format is clear

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Response Time Impact
   - **Description**: Adding orchestrator action to response must not increase API latency by more than 10ms
   - **Measurement**: Benchmark task update command with/without actions
   - **Target**: <10ms additional latency
   - **Justification**: Orchestrator polling loops are latency-sensitive

2. **REQ-NF-002**: Configuration Load Time
   - **Description**: Loading and validating workflow configuration must complete in under 100ms
   - **Measurement**: Time from config read to validation complete
   - **Target**: <100ms
   - **Justification**: Orchestrator startup time affects system availability

**Reliability**

3. **REQ-NF-010**: Backward Compatibility
   - **Description**: Existing Shark installations must work without configuration changes
   - **Implementation**: `orchestrator_action` is optional; missing actions omitted from responses
   - **Compliance**: Semantic versioning - minor version bump
   - **Risk Mitigation**: Prevents breaking changes for existing users

4. **REQ-NF-011**: Configuration Validation
   - **Description**: Invalid configurations must be detected at load time, not runtime
   - **Implementation**: Schema validation on config parse
   - **Compliance**: Fail-fast principle
   - **Risk Mitigation**: Prevents runtime errors from bad configuration

**Maintainability**

5. **REQ-NF-020**: Schema Extensibility
   - **Description**: Configuration schema must support future enhancements without breaking changes
   - **Implementation**: Use optional fields; version configuration files
   - **Standard**: JSON Schema versioning
   - **Testing**: Add test cases for schema evolution

**Documentation**

6. **REQ-NF-030**: Configuration Examples
   - **Description**: Provide complete, working examples for common workflows
   - **Implementation**: Example configs in docs/examples/
   - **Standard**: Include generic, wormwoodGM, and enterprise workflows
   - **Testing**: Examples validated as part of test suite

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Agent Completes Work and Transitions Task**

- **Given** a task is in "in_refinement_ba" status
- **And** "ready_for_refinement_tech" status has orchestrator_action defined
- **When** business analyst runs `shark task update T-E01-F03-002 --status ready_for_refinement_tech --json`
- **Then** response includes `orchestrator_action` object
- **And** action has `action: "spawn_agent"`, `agent_type: "architect"`, `skills: ["architecture", ...]`
- **And** instruction template has `{task_id}` replaced with "T-E01-F03-002"
- **And** orchestrator can immediately spawn architect agent without additional query

---

**Scenario 2: Configuration Validation Catches Errors**

- **Given** a `.sharkconfig.json` with incomplete orchestrator_action (missing `agent_type` for `spawn_agent`)
- **When** user runs `shark workflow validate-actions --strict`
- **Then** validation fails with clear error message
- **And** message indicates which status and which field is missing
- **And** exit code is non-zero

---

**Scenario 3: Orchestrator Polls for Ready Tasks**

- **Given** multiple tasks in various "ready_for_*" statuses
- **When** orchestrator runs `shark task list --status ready_for_development --with-actions --json`
- **Then** response includes array of tasks
- **And** each task includes `orchestrator_action` for its current status
- **And** orchestrator can spawn agents for all tasks from single query

---

**Scenario 4: Backward Compatibility**

- **Given** a Shark installation with no orchestrator_action defined
- **When** user runs `shark task update T-001 --status ready_for_development --json`
- **Then** task status is updated successfully
- **And** response does not include `orchestrator_action` field
- **And** no error is thrown
- **And** existing orchestrator code continues to work

---

**Scenario 5: Error Handling - Status Not Found**

- **Given** a status "invalid_status" does not exist in configuration
- **When** user runs `shark config get-status-action invalid_status`
- **Then** command exits with code 1
- **And** error message: "Status 'invalid_status' not found in config"
- **And** suggests checking available statuses with `shark workflow show-actions`

---

## Out of Scope

### Explicitly Excluded

1. **Dynamic Template Engine (Jinja2, text/template)**
   - **Why**: Adds complexity; simple string replacement sufficient for v1
   - **Future**: Phase 4 enhancement if needed
   - **Workaround**: Use simple `{task_id}` placeholder initially

2. **Database-Backed Workflow Storage**
   - **Why**: File-based YAML is simpler for initial implementation
   - **Future**: Phase 3 - add workflow versioning and history to database
   - **Workaround**: Version control YAML files with git

3. **Workflow Visualization**
   - **Why**: Not required for core functionality
   - **Future**: Phase 4 - generate Mermaid diagrams from config
   - **Workaround**: Manually create workflow diagrams

4. **Conditional Transitions**
   - **Why**: Significant complexity, not needed for initial workflows
   - **Future**: Phase 4 - support complex conditions (test_coverage >= 80%)
   - **Workaround**: Agents make transition decisions in code

5. **Custom Action Types**
   - **Why**: Four action types (spawn_agent, pause, wait_for_triage, archive) cover initial use cases
   - **Future**: Consider `notify`, `webhook`, `custom` if needed
   - **Workaround**: Use existing action types creatively

6. **Per-Task Workflow Assignment**
   - **Why**: Single workflow sufficient for initial implementation
   - **Future**: Phase 2 - add workflow_name column to tasks table
   - **Workaround**: All tasks in project use same workflow

---

### Alternative Approaches Rejected

**Alternative 1: Store Actions in Database Instead of Config File**

- **Description**: Add orchestrator_action as column in status metadata table
- **Why Rejected**:
  - Harder to edit (requires SQL or migration scripts)
  - No git version control
  - More complex to validate
  - File-based config is standard practice for workflow definitions
- **Future**: May revisit for Phase 3 (workflow versioning)

---

**Alternative 2: Separate Workflow Config File (Not in .sharkconfig.json)**

- **Description**: Store workflow config in `~/.config/shark/workflows/*.yml`
- **Why Rejected**:
  - Requires separate config management
  - More moving parts for simple use case
  - Existing `.sharkconfig.json` already has `status_metadata`
- **Future**: May add as option in Phase 3 for complex workflows

---

**Alternative 3: Orchestrator Actions as Separate Shark Commands**

- **Description**: Add `shark orchestrator get-action` command instead of including in responses
- **Why Rejected**:
  - Requires two API calls (transition + get action)
  - Higher latency
  - More complex orchestrator code
- **Benefit of Chosen Approach**: Single API call handles both

---

## Success Metrics

### Primary Metrics

1. **API Call Reduction**
   - **What**: Number of Shark CLI invocations per task transition
   - **Target**: Reduce from 2 calls (update + get action) to 1 call
   - **Timeline**: Immediate upon deployment
   - **Measurement**: Orchestrator telemetry logs

2. **Workflow Configuration Adoption**
   - **What**: Percentage of status_metadata entries with orchestrator_action defined
   - **Target**: 100% of actionable statuses (ready_for_*) within 2 weeks
   - **Timeline**: 2 weeks post-deployment
   - **Measurement**: Configuration validation reports

3. **Time to Add New Workflow Stage**
   - **What**: Developer hours from requirement to production deployment
   - **Target**: <2 hours (vs 8 hours with code changes)
   - **Timeline**: Measured on first 3 new stages added
   - **Measurement**: Time tracking from issue creation to deployment

---

### Secondary Metrics

- **Configuration Errors**: Zero production errors from invalid orchestrator_action configs (validation catches all)
- **Documentation Drift**: Zero discrepancies between workflow docs and config (config is self-documenting)
- **Orchestrator Code Complexity**: Reduction in orchestrator LOC by removing hardcoded mappings (target: -200 LOC)

---

## Dependencies & Integrations

### Dependencies

- **Shark CLI Core**: Requires access to configuration loading and validation
- **Status Metadata**: Builds on existing `status_metadata` structure in `.sharkconfig.json`
- **Task Repository**: Update methods must support returning enriched transition responses

### Integration Requirements

- **AI Agent Orchestrator**: Must parse `orchestrator_action` from task update responses
  - Expected format: JSON with `action`, `agent_type`, `skills`, `instruction` fields
  - Orchestrator should handle missing actions gracefully (fallback to defaults)

- **Configuration Validation**: Pre-commit hooks should run `shark workflow validate-actions`
  - Ensures configurations are valid before merge
  - CI/CD pipeline fails on invalid configs

---

## Implementation Phases

### Phase 1: Configuration Schema Support (Week 1)

**Goal**: Shark can parse and validate `orchestrator_action` field

**Deliverables**:
- [ ] Update config schema to include optional `orchestrator_action`
- [ ] Add validation for `orchestrator_action` structure
- [ ] Update config parser to load `orchestrator_action` data
- [ ] Unit tests for config parsing
- [ ] Documentation: schema reference

---

### Phase 2: Enhance `shark task update` (Week 2)

**Goal**: Return orchestrator action in status transition responses

**Deliverables**:
- [ ] Modify `shark task update` to look up `orchestrator_action` from new status
- [ ] Populate `{task_id}` template variable in instruction
- [ ] Include `orchestrator_action` in JSON output
- [ ] Add action summary to human-readable output
- [ ] Handle missing `orchestrator_action` gracefully
- [ ] Integration tests
- [ ] Documentation: API response format

---

### Phase 3: CLI Utility Commands (Week 3)

**Goal**: Implement utility commands for querying and validating actions

**Deliverables**:
- [ ] Implement `shark config get-status-action`
- [ ] Implement `shark workflow validate-actions`
- [ ] Implement `shark workflow show-actions`
- [ ] Optional: Enhance `shark task list --with-actions`
- [ ] Command help text and examples
- [ ] Integration tests
- [ ] Documentation: CLI command reference

---

### Phase 4: Template Engine Enhancement (Future)

**Goal**: Support dynamic template variables in `instruction_template`

**Initial Support**:
- `{task_id}` - replaced with actual task ID

**Future Support**:
- `{epic_id}`, `{feature_id}`, `{task_title}`, `{priority}`

**Implementation**: Simple string replacement initially; consider template library for complex cases

---

### Phase 5: Documentation & Migration (Week 4)

**Goal**: Document feature for users and orchestrator developers

**Deliverables**:
- [ ] Update Shark README with `orchestrator_action` section
- [ ] Add examples for different action types
- [ ] Document CLI commands in help text
- [ ] Create migration guide for existing configs
- [ ] Example configurations: generic, wormwoodGM, enterprise workflows

---

## Testing Strategy

### Unit Tests

1. **Config Parsing**:
   - Load config with `orchestrator_action`
   - Load config without `orchestrator_action` (backward compat)
   - Malformed `orchestrator_action` (validation)

2. **Template Population**:
   - `{task_id}` replacement
   - Missing task_id (should work, template not populated)

3. **Task Update Response**:
   - Action included when defined
   - Works without action (backward compat)
   - Action populated from new status metadata

4. **CLI Commands**:
   - `get-status-action` with valid status
   - `get-status-action` with invalid status
   - `validate-actions` with complete config
   - `validate-actions` with incomplete config

---

### Integration Tests

1. **Orchestrator Integration**:
   - Mock orchestrator updates task status
   - Verify response includes orchestrator_action
   - Test all action types

2. **Task List with Actions**:
   - Query tasks with `--with-actions`
   - Verify actions included
   - Test without flag (backward compat)

3. **End-to-End Workflow**:
   - Create task in draft
   - Transition to ready_for_development
   - Verify action in response
   - Verify instruction includes task ID

---

### Validation Tests

- Schema validation catches all invalid configs
- Error messages are clear and actionable
- Examples in documentation are valid and tested

---

## Compliance & Security Considerations

**Configuration Security**:
- `.sharkconfig.json` should not contain secrets
- Instructions should not reference credentials
- File permissions: ensure config is not world-writable

**Audit Logging**:
- Log when orchestrator actions are returned
- Log validation failures
- Track configuration changes (via git)

**Data Protection**:
- Task IDs in instructions are not sensitive
- No PII in template variables
- Instructions logged for debugging (ensure no secrets)

---

## Rollout Plan

### Week 1: Preparation
- ✅ Create feature PRD (this document)
- Create example workflow configurations
- Review with stakeholders

### Week 2-3: Implementation
- Phase 1: Configuration schema
- Phase 2: Task update enhancement
- Phase 3: CLI utility commands

### Week 4: Testing & Documentation
- Integration testing with orchestrator
- Documentation updates
- Example configurations

### Week 5: Deployment
- Deploy Shark CLI update
- Migrate `.sharkconfig.json` files
- Monitor for issues

### Week 6: Orchestrator Integration
- Update orchestrator to use actions from responses
- Remove hardcoded workflow mappings
- Validate behavior matches previous implementation

---

## Rollback Plan

**If Issues Arise**:
1. Orchestrator can ignore `orchestrator_action` field (backward compatible)
2. Shark works without `orchestrator_action` in config (optional field)
3. Revert to previous Shark CLI version if needed
4. Keep hardcoded workflow as fallback in orchestrator v1

---

## Example Configuration

### WormwoodGM Workflow (Excerpt)

```json
{
  "status_metadata": {
    "ready_for_refinement_ba": {
      "color": "cyan",
      "description": "Queued for business analysis",
      "phase": "planning",
      "agent_types": ["business-analyst"],
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "business-analyst",
        "skills": [
          "specification-writing",
          "shark-task-management",
          "research"
        ],
        "instruction_template": "Launch a business-analyst agent to refine requirements for task {task_id}. Review task description and user stories. Document clear acceptance criteria and ensure requirements are testable."
      }
    },
    "ready_for_development": {
      "color": "yellow",
      "description": "Queued for implementation",
      "phase": "development",
      "agent_types": ["developer"],
      "orchestrator_action": {
        "action": "spawn_agent",
        "agent_type": "developer",
        "skills": [
          "test-driven-development",
          "implementation",
          "shark-task-management"
        ],
        "instruction_template": "Launch a developer agent to implement task {task_id}. Write tests first, then implement to pass tests following the technical specifications."
      }
    },
    "blocked": {
      "color": "red",
      "description": "Waiting on external dependency",
      "phase": "any",
      "orchestrator_action": {
        "action": "pause",
        "instruction_template": "Task {task_id} is blocked. Do not spawn agent. Blocker reason documented in task notes."
      }
    },
    "completed": {
      "color": "green",
      "description": "Successfully delivered",
      "phase": "done",
      "orchestrator_action": {
        "action": "archive",
        "instruction_template": "Task {task_id} is completed. No further action needed."
      }
    }
  }
}
```

---

## JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "definitions": {
    "OrchestratorAction": {
      "type": "object",
      "required": ["action", "instruction_template"],
      "properties": {
        "action": {
          "type": "string",
          "enum": ["spawn_agent", "pause", "wait_for_triage", "archive"]
        },
        "agent_type": {
          "type": "string",
          "description": "Required if action is spawn_agent"
        },
        "skills": {
          "type": "array",
          "items": { "type": "string" },
          "description": "List of skills for agent"
        },
        "instruction_template": {
          "type": "string",
          "description": "Template string with {task_id} placeholder"
        }
      },
      "if": {
        "properties": { "action": { "const": "spawn_agent" } }
      },
      "then": {
        "required": ["agent_type", "skills"]
      }
    }
  }
}
```

---

*Last Updated*: 2026-01-14
