---
feature_key: E07-F03-move-task-files-to-feature-folder
epic_key: E07
title: move task files to feature folder
description: 
---

# move task files to feature folder

**Feature Key**: E07-F03-move-task-files-to-feature-folder

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
Currently, task files may be created in a location separate from their parent feature directory, making it difficult to navigate the project structure and understand the relationship between features and their tasks. This scattered structure makes it harder to find task files and reduces organizational clarity.

### Solution
When creating new task files, place them under the feature folder in a `tasks/` subdirectory following the pattern: `docs/plan/{epic-slug}/{feature-slug}/tasks/T-{epic-key}-{feature-key}-{number}-{task-slug}.md`. This creates a clear hierarchical structure where tasks are physically grouped with their parent feature.

### Impact
- Improved project organization with clear epic > feature > task hierarchy
- Easier navigation - all task files for a feature are in one place
- Better file discoverability
- Consistent directory structure across all features

---

## User Personas

### Persona 1: Developer / Project Navigator

**Profile**:
- **Role/Title**: Developer implementing tasks and navigating project structure
- **Experience Level**: Moderate, works with multiple features/tasks daily
- **Key Characteristics**:
  - Navigates file system to find task documentation
  - Values clear organizational structure
  - Wants related files grouped together

**Goals Related to This Feature**:
1. Quickly locate all tasks for a given feature
2. Understand project hierarchy from file structure alone

**Pain Points This Feature Addresses**:
- Task files scattered across different locations
- Difficult to find which tasks belong to which feature
- Unclear relationship between epics, features, and tasks in file system

**Success Looks Like**:
Can navigate to a feature folder and immediately see all its tasks in a `tasks/` subdirectory. File structure mirrors logical project hierarchy.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want task files created under their feature folder so that I can find all related tasks in one place.

**Acceptance Criteria**:
- [ ] New tasks created in `docs/plan/{epic-slug}/{feature-slug}/tasks/` directory
- [ ] Task filename follows pattern: `T-{epic-key}-{feature-key}-{number}-{task-slug}.md`
- [ ] Tasks subdirectory is automatically created if it doesn't exist

---

### Should-Have Stories

None identified.

---

### Could-Have Stories

None identified.

---

### Edge Case & Error Stories

**Error Story 1**: As a user, when the feature folder doesn't exist, I want a clear error so that I understand the issue.

**Acceptance Criteria**:
- [ ] Error message indicates feature folder not found
- [ ] Suggests verifying feature exists or running sync

---

## Requirements

### Functional Requirements

**Category: File System Organization**

1. **REQ-F-001**: Task File Location
   - **Description**: New task files must be created in `docs/plan/{epic-slug}/{feature-slug}/tasks/` directory
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Task creation code updated to use feature folder path
     - [ ] `tasks/` subdirectory created automatically if missing
     - [ ] File path stored correctly in database

2. **REQ-F-002**: Task Filename Pattern
   - **Description**: Task filenames must follow pattern `T-{epic-key}-{feature-key}-{number}-{task-slug}.md`
   - **User Story**: Links to Story 1
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Filename generation follows specified pattern
     - [ ] Task slug generated from task title
     - [ ] Number is properly zero-padded (001, 002, etc.)

---

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF-001**: Existing Task Files Unaffected
   - **Description**: Existing task files in old locations remain valid and accessible
   - **Measurement**: All existing task file paths continue to work
   - **Justification**: Don't break existing tasks, only affect new creations

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Create Task in Feature Folder**
- **Given** feature E01-F02 exists at `docs/plan/E01-epic-name/E01-F02-feature-name/`
- **When** user creates task with `shark task create --epic=E01 --feature=F02 "My Task"`
- **Then** task file is created at `docs/plan/E01-epic-name/E01-F02-feature-name/tasks/T-E01-F02-001-my-task.md`
- **And** tasks directory is created if it didn't exist

**Scenario 2: Auto-create Tasks Directory**
- **Given** feature folder exists but has no `tasks/` subdirectory
- **When** first task is created for that feature
- **Then** `tasks/` directory is automatically created
- **And** task file is placed inside it

---

## Out of Scope

### Explicitly Excluded

1. **Moving existing task files to new locations**
   - **Why**: Risk of breaking references, path changes in database, complexity
   - **Future**: Could provide migration tool if needed
   - **Workaround**: Existing tasks remain in current locations

2. **Automatic folder cleanup when tasks deleted**
   - **Why**: Out of scope for this feature, focused on creation only
   - **Future**: Could add in maintenance feature
   - **Workaround**: Manual cleanup if needed

---

### Alternative Approaches Rejected

**Alternative 1: Flat Structure with Prefix**
- **Description**: Keep all tasks in docs/plan/tasks/ with filename prefix
- **Why Rejected**: Doesn't provide hierarchical organization, harder to navigate

---

## Success Metrics

### Primary Metrics

1. **Directory Structure Consistency**
   - **What**: All new tasks created in correct feature folder
   - **Target**: 100% of new tasks follow new pattern
   - **Measurement**: File system inspection

---

## Dependencies & Integrations

### Dependencies

- **Task Creation Logic** (internal/taskcreation/): File path generation
- **Feature Folder Discovery**: Need to locate feature folder from epic/feature keys

### Integration Requirements

None

---

## Compliance & Security Considerations

None

---

*Last Updated*: 2025-12-17
