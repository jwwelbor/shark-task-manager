---
name: triage
description: Quick-capture and classify a work item discovered during development, routing it to the correct entity type (task, feature, bug, tech-debt, change-card, idea, or note) under the right parent. Searches existing entities first to avoid duplication, proposes a classification, and confirms before creating. Use when you encounter work that needs to be recorded for later, the user says "/triage", or something should be tracked but its home is unclear. This is a capture-and-classify tool, NOT a create-and-elaborate tool.
---

# triage — Capture, Classify, and Route Work Items

You are performing **intake triage**: understand a discovered work item, find where it
belongs, propose a classification, and (after confirmation) create it with just enough
context. Then **stop** — no subtasks, no PRDs, no cascading workflow.

## Step 0: Fix-Now Gate

Triage is for work that genuinely needs to be deferred. If the fix is trivial
(< ~20 lines, a typo/rename/null-check, a missing test, or it lives in a file already
edited in this branch) and it came from review/UAT of the current branch — **just fix it
now and stop**. Do not file an entity for work you can finish in place. When on the fence,
fix it. Otherwise proceed.

## Step 1: Understand

Identify **what** the work is, **where** it applies, and **why** it matters. If the
description is too vague to classify, ask exactly one clarifying question.

## Step 2: Search Existing Entities (dedup)

Before creating anything, search for existing coverage so you don't duplicate. Enumerate
existing items of each candidate type, not just a keyword search:

- Check project status and existing epics/features for structure
- List existing items of each candidate type (bugs, tasks, features, etc.) under the likely parent
- Search notes for related observations

If an entity already covers the work, prefer **adding a note** to it over creating a new one.

## Step 3: Classify

| Signal | Classification |
|--------|----------------|
| Something is broken or behaves wrong | **Bug** |
| Code works but should be improved (quality / architecture / dependency / testing / performance / docs) | **Tech Debt** |
| Process / infra / config change not tied to user value | **Change Card** |
| Atomic work fitting under an existing feature | **Task** (under that feature) |
| Multi-step work delivering user/stakeholder value | **Feature** (under best-fit epic) |
| Speculative / future concept not yet committed | **Idea** |
| Already tracked by an existing entity | **Note** (on that entity) |

Tie-breakers: prefer **task** over feature (cheaper to promote than demote); **bug** when
behavior is wrong, **tech-debt** when behavior is correct but the code is hard to maintain;
**tech-debt** when the work lives in code, **change-card** when it's process/infra; **idea**
while still exploratory.

## Step 4: Propose (Interactive — wait for confirmation)

```
## Triage Proposal
**Description**: <your understanding>
**Classification**: <Bug | Tech Debt | Task | Feature | Change Card | Idea | Note>
**Parent/Location**: <parent key + title, or N/A for standalone>
**Rationale**: <1–2 sentences>
**Existing coverage**: <entity key — title (status), if any → add note instead>
**Proposed title**: <concise title>
Create this? (or suggest changes)
```

**Do not create until the user confirms.**

## Step 5: Create

Create the entity of the appropriate type with the minimum required fields:

- **Bug**: title, severity level, optional link to parent entity, description with context
- **Tech Debt**: title, category (code-quality/architecture/dependency/testing/performance/documentation), severity, description
- **Task**: title, parent epic, parent feature
- **Feature**: title, parent epic
- **Change Card**: title, optional epic link, justification, description
- **Idea**: title, description
- **Note**: observation text, target entity key, type (comment/blocker/question/future)

After creating, link the entity to whatever made it relevant (the parent epic/feature/task)
so it doesn't float unanchored in the backlog.
Capture the assigned key from the create response. Then **stop**.
