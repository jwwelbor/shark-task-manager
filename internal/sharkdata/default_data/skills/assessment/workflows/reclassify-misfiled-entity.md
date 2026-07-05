---
name: assessment-reclassify-misfiled-entity
mode: reclassify_misfiled_entity
---

# Workflow: Reclassify Misfiled Entity

**Purpose**: Mechanically convert a work item that was created under the
wrong entity type into the correct lighter-weight entity, without losing its
description or its history trail.

**When to use**: After the calling assessment prompt's own classification
step has already determined the target entity type (Change Card, Tech Debt,
Idea, Feature, or Task) — and, for Feature/Task targets, the parent
container the replacement entity will live under. This workflow covers only
the mechanical conversion itself, not the classification decision or the
parent-container search (those differ by source entity type and stay in the
calling prompt).

## Process

### Step 1: Copy the Description

Copy the original entity's description verbatim into the replacement
entity's description — the record of *why* the work exists must survive
the reclassification.

### Step 2: Create the Replacement Entity

Using the target type and container determined by the calling prompt, create
the replacement entity carrying over the copied description:

- **Change Card**: a standalone change card.
- **Tech Debt**: a standalone tech-debt item, tagged with the appropriate
  category (code-quality, architecture, dependency, testing, performance, or
  documentation).
- **Idea**: a standalone idea.
- **Feature**: under the chosen parent epic.
- **Task**: under the chosen enhancement feature.

### Step 3: Cross-Reference Both Directions

So the history is traceable from either entity:

1. Add a decision note on the original entity naming the new entity's key and
   title as its reclassification target.
2. Add a reference note on the new entity naming the original entity's key
   and title as what it was reclassified from.

### Step 4: Cancel the Original

Set the original entity's status to cancelled, with a reason recording the
new entity's key.

### Step 5: Return Structured Output

Return: `reclassified` (bool), `new_entity_key`, `new_entity_type`.
