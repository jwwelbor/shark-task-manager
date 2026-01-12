---
feature_key: E13-F02-enhanced-work-assignment
epic_key: E13
title: Enhanced Work Assignment
description: Make task next workflow-aware (query by agent type and phase). Add feature next command for PM work assignment.
---

# Enhanced Work Assignment

**Feature Key**: E13-F02-enhanced-work-assignment

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem

The current `shark task next` command is hardcoded to only query tasks with status `todo` (line 700 in task.go). This creates critical limitations for custom workflows:

1. **AI orchestrators cannot find work**: When workflows use statuses like `ready_for_development`, `ready_for_code_review`, or `ready_for_qa`, these tasks are invisible to `task next` even though they represent available work.

2. **No phase-aware filtering**: The command doesn't understand workflow phases or which agent types can work on tasks in specific statuses, making multi-agent orchestration impossible.

3. **PMs lack feature-level queries**: Product managers need to find features with available work to assign to teams, but no such command exists.

### Solution

1. **Make `task next` workflow-aware**: Query all statuses matching the `ready_for_*` pattern instead of hardcoded `todo`. Filter results by `status_metadata[status].agent_types` from workflow configuration.

2. **Add `feature next` command**: New command that queries features and returns the first one with tasks available for a specific agent type. Enables PM workflow of assigning features to teams.

### Impact

**For AI Orchestrators:**
- Can find work across all workflow phases (development, code review, QA, approval, etc.)
- Properly route tasks to appropriate agent types based on workflow configuration
- Enable fully automated multi-agent task handoff

**For Product Managers:**
- Discover next feature that needs development/review/testing work
- Better workload distribution across features within an epic

**Expected Outcomes:**
- 100% workflow phase coverage (AI can find work in any custom phase)
- Zero hardcoded status assumptions
- Enable multi-agent orchestration workflows

---

## User Personas

### Persona 1: Atlas (AI Orchestrator Agent)

**Profile**:
- **Role/Title**: AI orchestrator agent coordinating multi-agent development workflows
- **Experience Level**: Automated system running continuously
- **Key Characteristics**:
  - Polls shark CLI every 30 seconds for available work
  - Must route tasks to appropriate agent types (backend, frontend, qa, etc.)
  - Operates across all workflow phases (development, review, testing, approval)
  - Requires JSON output for programmatic processing

**Goals Related to This Feature**:
1. Find next available task for a specific agent type (e.g., backend developer)
2. Understand which workflow phase the task is in
3. Route work to the correct agent without manual intervention

**Pain Points This Feature Addresses**:
- Cannot find tasks in custom workflow statuses (only sees `todo`)
- No way to filter by workflow phase or agent type assignment
- Must hardcode status queries for each custom workflow

**Success Looks Like**:
Atlas queries `shark task next --agent=backend --json` and receives tasks in `ready_for_development` status without any workflow-specific configuration changes.

---

### Persona 2: Sarah (Product Manager / Scrum Master)

**Profile**:
- **Role/Title**: Product Manager at software development company
- **Experience Level**: 5+ years managing development teams
- **Key Characteristics**:
  - Plans sprint work across multiple features
  - Needs to assign features to development teams
  - Wants visibility into which features have work ready
  - Uses shark to track epic/feature/task progress

**Goals Related to This Feature**:
1. Find next feature in an epic that has work available for backend developers
2. Understand which features are blocked vs. ready for work
3. Balance workload across features efficiently

**Pain Points This Feature Addresses**:
- No command to query features by work availability
- Must manually check each feature for available tasks
- Can't filter by agent type (backend, frontend, QA)

**Success Looks Like**:
Sarah runs `shark feature next E07 --agent=backend` and immediately sees which feature needs backend development work, with task count included.

---

## User Stories

### Must-Have Stories

**Story 1**: As Atlas (AI orchestrator), I want `task next` to query workflow-aware statuses so that I can find tasks in any custom workflow phase.

**Acceptance Criteria**:
- [ ] Command queries all statuses matching `ready_for_*` pattern (not just `todo`)
- [ ] Results include tasks in `ready_for_development`, `ready_for_code_review`, `ready_for_qa`, etc.
- [ ] Backward compatible: still works with default workflow using `todo` status
- [ ] JSON output includes workflow phase information

**Story 2**: As Atlas, I want to filter `task next` by agent type so that I only see tasks assigned to the agent I'm coordinating.

**Acceptance Criteria**:
- [ ] `--agent` flag filters by `status_metadata[status].agent_types` from workflow config
- [ ] Tasks returned only if agent type matches status metadata
- [ ] If no workflow config exists, falls back to task.agent_type field (current behavior)
- [ ] JSON output indicates which agent types can work on the task

**Story 3**: As Sarah (PM), I want a `feature next` command to find features with available work so that I can assign work to my team.

**Acceptance Criteria**:
- [ ] `shark feature next <epic-key>` returns first feature with available tasks
- [ ] `--agent` flag filters features by agent type (only features with tasks for that agent)
- [ ] Output includes feature details and count of available tasks
- [ ] Features ordered by execution_order and priority
- [ ] JSON output includes task breakdown by status

---

### Should-Have Stories

**Story 4**: As Atlas, I want `task next` to return workflow phase context so that I know which phase the task is entering.

**Acceptance Criteria**:
- [ ] JSON output includes `workflow_phase` field (e.g., "development", "code_review", "qa")
- [ ] Phase extracted from status name using pattern matching (`ready_for_X` → phase `X`)
- [ ] Falls back to "general" phase if status doesn't match pattern

---

### Edge Case & Error Stories

**Error Story 1**: As Atlas, when no workflow configuration exists, I want `task next` to fall back to default behavior so that the command doesn't fail.

**Acceptance Criteria**:
- [ ] If `.sharkconfig.json` missing, query `todo` status (current behavior)
- [ ] If `status_flow` section missing, query `todo` status
- [ ] Warning logged when using fallback behavior
- [ ] No breaking changes for existing users

**Error Story 2**: As Sarah, when no features have available work, I want a clear message so that I know work assignment is complete.

**Acceptance Criteria**:
- [ ] Command returns "No features with available work" message
- [ ] JSON output: `{"message": "No features with available work for agent type: backend"}`
- [ ] Exit code 0 (not an error condition)

---

## Requirements

### Functional Requirements

**Category: Workflow-Aware Task Queries**

1. **REQ-F-006**: Workflow-Aware `shark task next` Command (from epic requirements)
   - **Description**: Enhance existing `next` command to filter by workflow phase and agent type
   - **User Story**: Story 1, Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Accepts `--agent=<type>` flag to filter by agent assignment
     - [ ] Queries tasks with status matching `ready_for_*` pattern (not just `todo`)
     - [ ] Filters results where `status_metadata[status].agent_types` contains requested agent type
     - [ ] Returns highest priority task matching criteria (using existing selectNextTasks logic)
     - [ ] Returns empty/null if no tasks available
     - [ ] JSON output includes task details and workflow phase info
     - [ ] Backward compatible (works without workflow config using current logic)

2. **REQ-F-007**: Implement `shark feature next` Command (from epic requirements)
   - **Description**: New command to get next feature with available work for an agent type
   - **User Story**: Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Command accepts epic key and optional `--agent=<type>` flag
     - [ ] Queries features in epic ordered by execution_order and priority
     - [ ] For each feature, checks if any tasks match agent type and are in `ready_for_*` state
     - [ ] Returns first feature with available work
     - [ ] JSON output includes feature details and count of available tasks
     - [ ] If no features have work, returns clear message

**Category: Workflow Configuration Integration**

3. **REQ-F-006.1**: Status Pattern Matching
   - **Description**: Query tasks by status pattern instead of hardcoded value
   - **User Story**: Story 1
   - **Priority**: Must-Have
   - **Implementation**:
     - [ ] Replace hardcoded `todoStatus := models.TaskStatusTodo` with pattern query
     - [ ] Query all tasks where `status LIKE 'ready_for_%'` OR `status = 'todo'`
     - [ ] Filter results by workflow config if available
     - [ ] Preserve existing dependency filtering logic

4. **REQ-F-006.2**: Agent Type Filtering via Workflow Config
   - **Description**: Filter tasks by agent types defined in workflow status metadata
   - **User Story**: Story 2
   - **Priority**: Must-Have
   - **Implementation**:
     - [ ] Load workflow config using workflow service
     - [ ] Read `status_metadata[task.status].agent_types`
     - [ ] Filter tasks where agent type matches metadata
     - [ ] Fall back to `task.agent_type` field if no workflow config

---

### Non-Functional Requirements

**Performance**

1. **REQ-NF-001**: Query Performance
   - **Description**: Workflow-aware queries must execute within acceptable latency
   - **Measurement**: Measure time from command invocation to output display
   - **Target**: < 1 second for database with 10,000 tasks (same as current implementation)
   - **Justification**: AI orchestrator polls every 30 seconds; slow queries block assignment loop

2. **REQ-NF-002**: Workflow Config Caching
   - **Description**: Cache workflow configuration to avoid repeated file reads
   - **Measurement**: Count file system reads during command execution
   - **Target**: Config read once per process, cached for lifetime
   - **Implementation**: Use workflow service singleton (already exists in codebase)

**Backward Compatibility**

1. **REQ-NF-003**: Fallback Behavior
   - **Description**: Commands must work without workflow config (default workflow)
   - **Implementation**:
     - [ ] If `.sharkconfig.json` missing or `status_flow` not defined, query `todo` status only
     - [ ] If `status_metadata` missing, filter by `task.agent_type` field
     - [ ] Log warning when using fallback behavior
     - [ ] All tests run against both custom and default workflows

---

## Acceptance Criteria

### Feature-Level Acceptance

**Given/When/Then Format**:

**Scenario 1: AI Orchestrator Finds Work in Custom Workflow Phase**
- **Given** workflow config defines `ready_for_development` status with `agent_types: ["backend"]`
- **And** task T-E07-F20-001 has status `ready_for_development` and agent_type `backend`
- **When** Atlas runs `shark task next --agent=backend --json`
- **Then** task T-E07-F20-001 is returned in results
- **And** JSON includes `workflow_phase: "development"`
- **And** JSON includes `agent_types: ["backend"]`

**Scenario 2: Task Next Filters by Agent Type via Workflow Config**
- **Given** workflow config defines `ready_for_code_review` with `agent_types: ["tech-lead", "code-reviewer"]`
- **And** task T-E07-F20-002 has status `ready_for_code_review`
- **When** Atlas runs `shark task next --agent=backend --json`
- **Then** task T-E07-F20-002 is NOT returned (wrong agent type)
- **When** Atlas runs `shark task next --agent=tech-lead --json`
- **Then** task T-E07-F20-002 IS returned

**Scenario 3: Feature Next Returns Feature with Available Work**
- **Given** epic E07 has features F01 and F02
- **And** F01 has no tasks in `ready_for_*` status
- **And** F02 has 3 tasks in `ready_for_development` status with agent_type `backend`
- **When** Sarah runs `shark feature next E07 --agent=backend --json`
- **Then** feature F02 is returned
- **And** JSON includes `available_tasks: 3`
- **And** JSON includes task breakdown by status

**Scenario 4: Backward Compatibility Without Workflow Config**
- **Given** no `.sharkconfig.json` file exists
- **And** task T-E07-F20-001 has status `todo` and agent_type `backend`
- **When** Atlas runs `shark task next --agent=backend`
- **Then** task T-E07-F20-001 is returned (fallback to current behavior)
- **And** warning logged: "No workflow config found, using default"

**Scenario 5: No Available Work**
- **Given** all tasks in epic E07 are in `in_development` or `completed` status
- **When** Sarah runs `shark feature next E07 --agent=backend --json`
- **Then** JSON output is `{"message": "No features with available work for agent type: backend"}`
- **And** exit code is 0

---

## Out of Scope

### Explicitly Excluded

1. **Bulk Operations (finish multiple tasks at once)**
   - **Why**: Complexity and edge case handling; not needed for MVP
   - **Future**: Could be added in REQ-F-015 (Could Have requirement)
   - **Workaround**: Run `task finish` in a loop for each task

2. **Interactive Mode (prompt for next status if multiple options)**
   - **Why**: Not needed for AI orchestrator; adds complexity for humans
   - **Future**: Could be added as separate enhancement
   - **Workaround**: Use explicit commands or check workflow config manually

3. **Advanced Analytics (time-in-phase reports)**
   - **Why**: Requires session tracking (E13-F04); not core to work assignment
   - **Future**: Will be addressed in separate analytics epic
   - **Workaround**: Query `task_history` table directly

4. **Multi-Tenancy (different workflows per epic/feature)**
   - **Why**: Significant complexity; no current use case
   - **Future**: May be added if demand emerges
   - **Workaround**: Use single project-level workflow

---

### Alternative Approaches Rejected

**Alternative 1: Add Status Query Parameter to Task List**
- **Description**: Instead of modifying `task next`, add `--status-pattern` flag to `task list`
- **Why Rejected**:
  - `task next` is semantic command for "get next available work" - better UX
  - AI orchestrators expect `next` command to return single task, not list
  - `task list` is for browsing, not work assignment

**Alternative 2: Create Separate Command per Workflow Phase**
- **Description**: Add `task next-development`, `task next-review`, `task next-qa` commands
- **Why Rejected**:
  - Doesn't scale to custom workflows (would need infinite commands)
  - Hard to discover which commands exist for a given workflow
  - Violates DRY principle (duplicate logic across commands)

**Alternative 3: Require Explicit Status Name in Filter**
- **Description**: Use `--status=ready_for_development` instead of pattern matching
- **Why Rejected**:
  - AI orchestrator would need workflow-specific configuration
  - Less flexible for custom workflows
  - Pattern matching (`ready_for_*`) is more intuitive

---

## Success Metrics

### Primary Metrics

1. **Workflow Phase Coverage**
   - **What**: Percentage of custom workflow statuses that `task next` can query
   - **Target**: 100% of statuses matching `ready_for_*` pattern
   - **Timeline**: Immediate upon feature release
   - **Measurement**: Test with 5+ different workflow configurations

2. **AI Orchestrator Adoption**
   - **What**: Percentage of AI orchestrator queries using workflow-aware features
   - **Target**: 80% of `task next` calls include `--agent` flag within 1 month
   - **Timeline**: 30 days post-release
   - **Measurement**: Log analysis of CLI command usage

3. **Query Performance**
   - **What**: 90th percentile query time for `task next` with pattern matching
   - **Target**: < 1 second for database with 10,000 tasks
   - **Timeline**: Performance testing before release
   - **Measurement**: Automated benchmark tests

---

### Secondary Metrics

- **Feature Next Usage**: 20% of PMs use `feature next` within first sprint (10 days)
- **Backward Compatibility**: 0 breaking changes for existing users (no workflow config)
- **Error Rate**: < 1% of queries fail due to workflow config issues

---

## Dependencies & Integrations

### Dependencies

- **Workflow Configuration Reader** (E13-F01): Must be implemented to read `.sharkconfig.json` workflow config
  - Provides APIs: `GetValidTransitions()`, `GetStatusMetadata()`, `GetAgentTypes()`
  - Feature can implement basic fallback if workflow service not available

- **Task Repository**: Uses existing `FilterCombined()` method
  - May need enhancement to support pattern matching (`status LIKE 'ready_for_%'`)
  - Current implementation filters by exact status value

- **Feature Repository**: Existing repository used for `feature next` command
  - No changes needed to repository layer
  - Business logic in command layer queries features + tasks

### Integration Requirements

- **AI Orchestrator**: This feature is specifically designed for AI orchestrator integration
  - Orchestrator polls `task next --agent=<type> --json` every 30 seconds
  - JSON output must include: task key, status, workflow_phase, agent_types
  - Exit code 0 when no work available (not an error)

- **Existing Task Commands**: Must maintain compatibility
  - `task start`, `task complete`, `task list` unaffected
  - All commands use same task repository
  - No breaking changes to data model

---

## Test Plan

### Unit Tests

1. **Task Next - Status Pattern Matching**
   - Test query returns tasks with `status = 'ready_for_development'`
   - Test query returns tasks with `status = 'ready_for_code_review'`
   - Test query returns tasks with `status = 'todo'` (backward compat)
   - Test query excludes tasks with `status = 'in_development'`

2. **Task Next - Agent Type Filtering**
   - Test filter by workflow metadata: `agent_types: ["backend"]`
   - Test filter by task field: `task.agent_type = 'backend'` (fallback)
   - Test multiple agent types: `agent_types: ["tech-lead", "code-reviewer"]`
   - Test no match returns empty result

3. **Feature Next - Feature Query**
   - Test returns first feature with available tasks (ordered by execution_order)
   - Test filters by agent type
   - Test skips features with no available tasks
   - Test returns null when no features have work
   - Test JSON output includes task count

### Integration Tests

1. **Workflow Compatibility Suite**
   - Test with default workflow (todo → in_progress → completed)
   - Test with custom 5-state workflow
   - Test with complex 10+ state enterprise workflow
   - Test with minimal 2-state workflow

2. **AI Orchestrator Simulation**
   - Create tasks in various `ready_for_*` statuses
   - Query `task next --agent=backend`
   - Verify correct task returned
   - Verify JSON output format matches specification

3. **Backward Compatibility**
   - Run tests with no `.sharkconfig.json` file
   - Verify fallback to `todo` status works
   - Verify fallback to `task.agent_type` field works
   - Verify warning logged when using fallback

### Performance Tests

1. **Query Performance Benchmark**
   - Seed database with 10,000 tasks across various statuses
   - Measure `task next --agent=backend` execution time
   - Target: < 1 second for 90th percentile
   - Compare to current implementation (should be equal or faster)

2. **Config Caching Performance**
   - Run `task next` 100 times in sequence
   - Verify config file read only once (cached)
   - Measure cache hit rate (should be 99%+)

### Manual Testing Checklist

- [ ] Run `task next --agent=backend` with custom workflow config
- [ ] Run `feature next E07 --agent=backend` and verify feature returned
- [ ] Run commands with `--json` flag and verify output structure
- [ ] Run commands without workflow config and verify fallback
- [ ] Test with various agent types (backend, frontend, qa, tech-lead)
- [ ] Test with no available work and verify message
- [ ] Verify existing commands (`task list`, `task start`) still work

---

*Last Updated*: 2026-01-11
