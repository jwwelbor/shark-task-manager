---
feature_key: E07-F01-remove-agent-requirement
epic_key: E07
title: remove agent requirement
description: 
---

# remove agent requirement

**Feature Key**: E07-F01-remove-agent-requirement

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
Currently, when creating a new task, the `--agent` flag is required and the agent type values are constrained to a predefined set (frontend, backend, api, testing, devops, general). This creates unnecessary friction when creating tasks that don't fit these categories or when the agent type needs to be determined later. Additionally, agent types are tightly coupled to template selection, making it inflexible to use custom templates or no template at all.

### Solution
Make the `--agent` field optional when creating tasks. Allow any string value for agent type (not just the predefined set). When an unknown agent type is specified, allow users to specify a custom template path via `--template` flag, or default to using the general template if no template is specified.

### Impact
- Reduce task creation friction by eliminating required agent field
- Enable flexible workflow where agent assignment happens after task creation
- Support custom agent types beyond the predefined list
- Improve extensibility for teams with custom agent roles or workflows

---

## User Personas

### Persona 1: Product Manager / Technical Lead

**Profile**:
- **Role/Title**: Product Manager or Technical Lead planning implementation work
- **Experience Level**: Experienced with task management CLIs, defining project structure
- **Key Characteristics**:
  - Creates task structures before implementation details are known
  - Works with diverse team structures that may not fit predefined agent types
  - Needs flexibility in workflow and task assignment

**Goals Related to This Feature**:
1. Quickly create task placeholders without needing to assign agents immediately
2. Support custom agent types specific to their team's workflow

**Pain Points This Feature Addresses**:
- Forced to specify agent type when creating tasks, even when assignment is TBD
- Cannot use custom agent types beyond the predefined set
- Template selection is too rigid and coupled to agent type

**Success Looks Like**:
Can create tasks rapidly without being blocked by required agent field. Can use custom agent types that match their team structure. Can specify custom templates or use defaults flexibly.

---

## User Stories

### Must-Have Stories

**Story 1**: As a product manager, I want to create tasks without specifying an agent type so that I can quickly define task structures before assignments are made.

**Acceptance Criteria**:
- [ ] `shark task create` command works without `--agent` flag
- [ ] Tasks created without agent have NULL agent_type in database
- [ ] Default general template is used when no agent type specified

**Story 2**: As a technical lead, I want to use custom agent type values so that I can match my team's specific roles.

**Acceptance Criteria**:
- [ ] Any string value is accepted for `--agent` flag
- [ ] Custom agent types are stored correctly in database
- [ ] System provides appropriate template fallback for unknown agent types

**Story 3**: As a user, I want to specify a custom template for task creation so that I can use my own task file formats.

**Acceptance Criteria**:
- [ ] New `--template` flag accepts path to custom template file
- [ ] Custom template is used when specified, regardless of agent type
- [ ] System validates template file exists before creating task

---

### Should-Have Stories

**Story 4**: As a user, when I use an unknown agent type, I want clear messaging about which template is being used so that I understand the file that was created.

**Acceptance Criteria**:
- [ ] CLI output indicates which template was selected
- [ ] Helpful message when falling back to general template

---

### Could-Have Stories

None identified for this feature.

---

### Edge Case & Error Stories

**Error Story 1**: As a user, when I specify a custom template that doesn't exist, I want to see a clear error message so that I can fix the path.

**Acceptance Criteria**:
- [ ] Error message includes the invalid path specified
- [ ] Suggests checking template path exists
- [ ] Task creation is aborted, no database entry created

---

## Requirements

### Functional Requirements

**Category: Task Creation**

1. **REQ-F-001**: Optional Agent Field
   - **Description**: The `--agent` flag must be optional when creating tasks via `shark task create`
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Remove `MarkFlagRequired("agent")` from task create command
     - [ ] Database accepts NULL for agent_type field
     - [ ] Task creation succeeds without agent specified

2. **REQ-F-002**: Unrestricted Agent Values
   - **Description**: Any string value must be accepted for agent type (not limited to predefined set)
   - **User Story**: Links to Story 2
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Validation removed for agent type enum
     - [ ] Custom agent strings stored in database
     - [ ] Agent type displayed correctly in task list/get commands

3. **REQ-F-003**: Custom Template Support
   - **Description**: Users can specify custom template file path via `--template` flag
   - **User Story**: Links to Story 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] New `--template` flag added to task create command
     - [ ] Template file validation before task creation
     - [ ] Custom template rendered correctly with task data

4. **REQ-F-004**: Template Fallback Logic
   - **Description**: When agent type is unknown or not specified, use general template; custom template overrides all
   - **User Story**: Links to Story 2, 3
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Priority: custom template > agent-specific template > general template
     - [ ] General template used when no agent specified
     - [ ] General template used for unknown agent types

---

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF-001**: Existing Behavior Preserved
   - **Description**: Existing tasks with agent types continue to work unchanged
   - **Measurement**: All existing tests pass
   - **Target**: 100% backward compatibility
   - **Justification**: Don't break existing workflows

**Usability**

1. **REQ-NF-002**: Clear User Feedback
   - **Description**: CLI provides clear feedback about template selection
   - **Implementation**: Output message indicating which template was used
   - **Justification**: Users should understand what file was created

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Create Task Without Agent**
- **Given** a user wants to create a task placeholder
- **When** they run `shark task create --epic=E01 --feature=F02 "Task Title"`
- **Then** the task is created successfully with NULL agent_type
- **And** the general template is used for the task file

**Scenario 2: Create Task With Custom Agent Type**
- **Given** a user has a custom agent role called "data-scientist"
- **When** they run `shark task create --epic=E01 --feature=F02 "Task Title" --agent=data-scientist`
- **Then** the task is created with agent_type="data-scientist"
- **And** the general template is used (fallback for unknown agent type)

**Scenario 3: Create Task With Custom Template**
- **Given** a user has a custom template at "my-templates/special-task.md"
- **When** they run `shark task create --epic=E01 --feature=F02 "Task Title" --template=my-templates/special-task.md`
- **Then** the task is created using the specified custom template
- **And** success message indicates custom template was used

**Scenario 4: Error Handling - Invalid Template Path**
- **Given** a user specifies a non-existent template path
- **When** they run `shark task create --epic=E01 --feature=F02 "Task Title" --template=invalid/path.md`
- **Then** an error message is displayed with the invalid path
- **And** task creation is aborted, no database record created

---

## Out of Scope

### Explicitly Excluded

1. **Changing existing agent type enum in database schema**
   - **Why**: Database already supports arbitrary strings via TEXT type
   - **Future**: No changes needed
   - **Workaround**: N/A - agent_type field already flexible

2. **Auto-assignment of agents based on task characteristics**
   - **Why**: Out of scope for this feature, focused on making field optional
   - **Future**: Could be a separate E07 feature if desired
   - **Workaround**: Manually assign agents via task update (future feature)

3. **Template auto-discovery from agent type**
   - **Why**: Adds complexity, fallback to general template is sufficient
   - **Future**: Could enhance template loader in future iterations
   - **Workaround**: Use `--template` flag for custom templates

---

### Alternative Approaches Rejected

**Alternative 1: Keep Agent Required, Add "TBD" Value**
- **Description**: Keep --agent required but allow special "TBD" value
- **Why Rejected**: Still forces users to specify something; making it optional is cleaner

**Alternative 2: Strict Agent Type Validation with Registry**
- **Description**: Maintain allowed agent types in config file or database table
- **Why Rejected**: Adds complexity and maintenance burden; unrestricted strings are simpler

---

## Success Metrics

### Primary Metrics

1. **Task Creation Flexibility**
   - **What**: Percentage of tasks created without agent specified
   - **Target**: >20% of tasks created use optional agent field
   - **Measurement**: Query tasks where agent_type IS NULL

---

### Secondary Metrics

- **Custom Agent Types**: Count of distinct agent_type values beyond predefined set
- **Template Usage**: Number of task creates using --template flag

---

## Dependencies & Integrations

### Dependencies

- **Task Create Command** (internal/cli/commands/task.go): Command flag changes
- **Task Creation Package** (internal/taskcreation/): Template selection logic
- **Database Schema**: agent_type field already allows NULL/any string

### Integration Requirements

None - purely internal changes to CLI and task creation logic

---

## Compliance & Security Considerations

No specific regulatory, data protection, or audit requirements for this feature.

---

*Last Updated*: 2025-12-17
