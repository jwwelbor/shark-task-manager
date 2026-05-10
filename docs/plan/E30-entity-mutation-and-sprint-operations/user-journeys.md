# User Journeys

**Epic**: [Entity Mutation and Sprint Operations](./epic.md)

---

## Overview

This document maps the main workflows enabled by the epic: editing core entities, maintaining notes and dependencies, and performing actionable Sprint planning.

---

## Journey 1: Update An Entity Without Leaving The Viewer

**Persona**: Shark Maintainer

**Goal**: Correct a title, description, or other mutable field directly from the viewer.

**Preconditions**:
- The user has the target entity open in the viewer.
- The entity supports the requested fields.

### Happy Path

1. **Open the entity**
   - User action: Selects a task, feature, epic, bug, or change card in the viewer.
   - System response: Shows the entity detail view and current metadata.
   - Expected outcome: The user sees the current state before editing.

2. **Enter edit mode**
   - User action: Clicks an explicit edit control.
   - System response: Displays editable fields with validation hints.
   - Expected outcome: The user can see which fields are mutable.

3. **Change a field**
   - User action: Updates title or description.
   - System response: The UI tracks the change locally and marks the form as dirty.
   - Expected outcome: The user knows the change has not yet been saved.

4. **Save the update**
   - User action: Submits the change.
   - System response: Sends a mutation request and validates the payload.
   - Expected outcome: The record is updated and history is recorded.

5. **Review the result**
   - User action: Returns to the entity view.
   - System response: Refreshes the rendered data.
   - Expected outcome: The new value is visible immediately.

**Success Outcome**: The entity is updated in one view with a clear validation and history trail.

### Alternative Paths

**Alt Path A: Validation fails**
- **Trigger**: Missing required title or invalid field value.
- **Branch Point**: Step 4.
- **Flow**:
  1. The request is rejected with a clear validation message.
  2. The user corrects the field and resubmits.
- **Outcome**: No partial write is saved.

---

## Journey 2: Add Notes And Dependencies To A Work Item

**Persona**: Contributor

**Goal**: Attach notes and dependency links while finishing work.

**Preconditions**:
- The target work item exists.
- The user has the relevant references for the dependency or note.

### Happy Path

1. **Open the item**
   - User action: Selects a task or feature from the hierarchy or Sprint surface.
   - System response: Shows the entity drawer and existing notes/dependencies.
   - Expected outcome: The user can inspect current relationships.

2. **Add a note**
   - User action: Enters note text and saves.
   - System response: Creates the note and associates it with the entity.
   - Expected outcome: The note becomes part of the entity history.

3. **Link a dependency**
   - User action: Adds a dependency reference.
   - System response: Validates the dependency target and writes the relation.
   - Expected outcome: The dependency is reflected in the entity view.

4. **Recheck workflow**
   - User action: Reviews status or next-status information.
   - System response: Updates the workflow hints based on the new dependency state.
   - Expected outcome: The user understands whether status can advance.

**Success Outcome**: Notes and dependencies are captured without losing the entity context.

### Critical Decision Points

- Whether a dependency should be a blocking prerequisite or a reference-only relation.

---

## Journey 3: Shape Sprint Scope And Stage Work

**Persona**: Sprint Planner

**Goal**: Move candidate work into the sprint while keeping capacity visible.

**Preconditions**:
- Sprint mode is open.
- Candidate items and capacity data are available.

### Happy Path

1. **Open Sprint mode**
   - User action: Switches to Sprint from the header.
   - System response: Loads Overview by default.
   - Expected outcome: The user sees the active sprint at a glance.

2. **Switch to Plan**
   - User action: Selects the Plan subview.
   - System response: Shows filters, candidate backlog, and capacity data.
   - Expected outcome: The user can evaluate scope.

3. **Filter the backlog**
   - User action: Filters by status, agent type, or assignment.
   - System response: Narrows the candidate list.
   - Expected outcome: Only relevant work remains visible.

4. **Select items**
   - User action: Checks one or more items.
   - System response: Tracks the selection explicitly.
   - Expected outcome: The user knows which items are staged for action.

5. **Apply a planning action**
   - User action: Chooses stage, remove, or mark-ready.
   - System response: Validates and writes the change.
   - Expected outcome: Sprint scope changes are recorded.

**Success Outcome**: The planner can shape sprint scope and keep the reasoning visible.

### Alternative Paths

**Alt Path A: No items match the filter**
- **Trigger**: Filter combination is too narrow.
- **Branch Point**: Step 3.
- **Flow**:
  1. The view shows an empty-state message.
  2. The user clears or adjusts filters.
- **Outcome**: The planner recovers without leaving Sprint mode.

---

## Journey 4: Inspect Sprint Report And Jump Back To An Entity

**Persona**: Sprint Planner

**Goal**: Use report data to understand flow and then jump into a specific entity.

**Preconditions**:
- Sprint report data is available.
- An entity is selected in Sprint mode.

### Happy Path

1. **Open the Report view**
   - User action: Switches from Overview or Plan to Report.
   - System response: Renders burndown, velocity, and trend sections.
   - Expected outcome: The user gets historical context.

2. **Inspect a trend**
   - User action: Reads velocity or burndown entries.
   - System response: Shows values and summaries for the sprint.
   - Expected outcome: The user can compare current and historical progress.

3. **Open the selected entity**
   - User action: Clicks the jump-to-entity control.
   - System response: Opens the entity detail view.
   - Expected outcome: The user can review or edit the underlying work item.

**Success Outcome**: The report is not a dead-end; it leads back to the work item when needed.

---

*See also*: [Requirements](./requirements.md)
