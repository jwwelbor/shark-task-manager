---
feature_key: E07-F02-make-task-title-like-feature
epic_key: E07
title: make task title like feature
description: 
---

# make task title like feature

**Feature Key**: E07-F02-make-task-title-like-feature

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
Task creation requires using `--title="Task Title"` flag syntax, which is inconsistent with how epic and feature creation work. Epic uses `shark epic create "Epic Title"` and feature uses `shark feature create --epic=E01 "Feature Title"` with positional arguments. This inconsistency creates friction and confusion for users who must remember different syntax patterns for similar operations.

### Solution
Change task creation to accept title as a positional argument, matching the epic/feature pattern: `shark task create --epic=E01 --feature=F02 "Task Title" --agent=backend`. Remove the `--title` flag requirement and parse the title from args[0].

### Impact
- Consistent CLI interface across epic/feature/task creation commands
- Reduced cognitive load for users learning the CLI
- Faster task creation with less typing
- More intuitive command syntax matching user expectations

---

## User Personas

### Persona 1: CLI User / Developer

**Profile**:
- **Role/Title**: Developer or Product Manager using shark CLI daily
- **Experience Level**: Moderate to high CLI experience, values consistency
- **Key Characteristics**:
  - Uses muscle memory for frequent CLI commands
  - Frustrated by inconsistent command patterns
  - Values efficiency and minimal typing

**Goals Related to This Feature**:
1. Quickly create tasks without remembering different syntax for each entity type
2. Use consistent patterns across all CLI commands

**Pain Points This Feature Addresses**:
- Must use `--title` for tasks but not for epics/features
- Syntax inconsistency breaks flow and requires mental context switching
- Extra typing required for common operations

**Success Looks Like**:
Can create epic, feature, and task using the same positional argument pattern for titles. Commands feel natural and consistent.

---

## User Stories

### Must-Have Stories

**Story 1**: As a CLI user, I want task creation syntax to match epic/feature creation so that I don't have to remember different patterns.

**Acceptance Criteria**:
- [ ] `shark task create --epic=E01 --feature=F02 "Task Title"` works without --title flag
- [ ] Title is parsed from positional arg[0]
- [ ] Command fails with clear error if title is missing

---

### Should-Have Stories

**Story 2**: As a user, when I forget to provide a title, I want a helpful error message so that I understand what's required.

**Acceptance Criteria**:
- [ ] Error message clearly states title is required
- [ ] Example usage is shown in error output

---

### Could-Have Stories

None identified.

---

### Edge Case & Error Stories

None beyond Story 2.

---

## Requirements

### Functional Requirements

**Category: CLI Argument Parsing**

1. **REQ-F-001**: Positional Title Argument
   - **Description**: Task create command must accept title as first positional argument (args[0])
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Change `Args: cobra.MinArgs(0)` to `Args: cobra.ExactArgs(1)`
     - [ ] Parse `title := args[0]` instead of flag
     - [ ] Remove `--title` flag and MarkFlagRequired

---

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF-001**: Clear Migration Message
   - **Description**: Users who try old syntax should get helpful error
   - **Measurement**: Error message clarity
   - **Justification**: Breaking change needs good UX

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Create Task With New Syntax**
- **Given** a user wants to create a task
- **When** they run `shark task create --epic=E01 --feature=F02 "Build Login Form"`
- **Then** the task is created with title "Build Login Form"
- **And** all other flags work as expected

**Scenario 2: Error on Missing Title**
- **Given** a user forgets the title
- **When** they run `shark task create --epic=E01 --feature=F02`
- **Then** an error message explains title is required
- **And** example usage is shown

---

## Out of Scope

### Explicitly Excluded

1. **Supporting both --title flag and positional arg**
   - **Why**: Adds complexity, better to have one clear way
   - **Future**: No - commit to positional argument pattern
   - **Workaround**: Use new positional syntax

---

### Alternative Approaches Rejected

**Alternative 1: Keep --title, Add Positional as Alias**
- **Description**: Support both syntaxes
- **Why Rejected**: Confusing to have two ways, complexity not worth it

---

## Success Metrics

### Primary Metrics

1. **Syntax Consistency**
   - **What**: All create commands use positional title argument
   - **Target**: 100% consistency across epic/feature/task
   - **Measurement**: Code review and testing

---

## Dependencies & Integrations

### Dependencies

- **Task Create Command** (internal/cli/commands/task.go): Argument parsing changes

### Integration Requirements

None

---

## Compliance & Security Considerations

None

---

*Last Updated*: 2025-12-17
