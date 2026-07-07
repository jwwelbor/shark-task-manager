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

## Step 2: Quick Scope Assessment

Before final classification, decide the smallest durable home for the work:

- **Note-sized**: an observation, decision, question, or breadcrumb on an existing entity.
- **Task-sized**: one atomic implementation or documentation change under an existing feature.
- **Feature-sized**: multi-step user or stakeholder value under an existing epic.
- **Standalone corrective work**: bug, tech-debt, or change-card that may link to an epic,
  feature, or task but does not belong as a child task.
- **Idea-sized**: speculative or product-direction work that is not ready for a committed
  epic/feature/task.

Prefer the narrowest classification that preserves the context. If the work is really a
new product direction, classify it as an idea or suggest `/shark vision "idea"` rather
than inventing an epic inside triage.

## Step 3: Search Existing Entities (dedup)

Before creating anything, search for existing coverage so you don't duplicate. Enumerate
existing items of each candidate type, not just a keyword search:

- `shark status` to understand current shape and active work.
- `shark list`, `shark list <epic>`, and `shark list <epic> <feature>` for likely
  epic/feature/task parents.
- `shark list bugs`, `shark list changes`, `shark list ideas`, and
  `shark list tech-debt` when those are candidate classifications.
- `shark notes search "<terms>"` for previous observations, blockers, decisions,
  and future-work breadcrumbs.

If an entity already covers the work, prefer **adding a note** to it over creating a new one.
In the proposal, name the existing coverage and recommend the note target.

## Step 4: Classify

| Signal | Classification |
|--------|----------------|
| Something is broken or behaves wrong | **Bug** |
| Code works but should be improved (quality / architecture / dependency / testing / performance / docs) | **Tech Debt** |
| Process / infra / config change (or "chore") not tied to user value | **Change Card** |
| Atomic work fitting under an existing feature | **Task** (under that feature) |
| Multi-step work delivering user/stakeholder value | **Feature** (under best-fit epic) |
| Speculative / future concept not yet committed | **Idea** |
| Already tracked by an existing entity | **Note** (on that entity) |

Tie-breakers: prefer **task** over feature (cheaper to promote than demote); **bug** when
behavior is wrong, **tech-debt** when behavior is correct but the code is hard to maintain;
**tech-debt** when the work lives in code, **change-card** when it's process/infra; **idea**
while still exploratory.

## Step 5: Propose (Interactive - wait for confirmation)

```
## Triage Proposal
**Description**: <your understanding>
**Scope assessment**: <note-sized | task-sized | feature-sized | standalone corrective work | idea-sized>
**Classification**: <Bug | Tech Debt | Task | Feature | Change Card | Idea | Note>
**Parent/Location**: <parent key + title, or N/A for standalone>
**Rationale**: <1–2 sentences>
**Existing coverage**: <none, or entity key + title + status + whether this should become a note there>
**Proposed title**: <concise title>
Create this? (or suggest changes)
```

Do not include entity identifiers in the proposal. Shark assigns the identifier
when the create command succeeds. **Do not create until the user confirms.**

## Step 6: Create

Create the entity of the appropriate type with the minimum required fields,
using Shark-native commands only:

- **Bug**:
  `shark create bug "<title>" --severity=<critical|high|medium|low> --description="<breadcrumb>"`
  plus `--linked-type=<epic|feature|task> --linked-key=<key>` when there is a known parent.
- **Tech Debt**:
  `shark create tech-debt "<title>" --category=<code-quality|architecture|dependency|testing|performance|documentation> --severity=<critical|high|medium|low> --description="<breadcrumb>"`
- **Task**:
  `shark create task <epic-key> <feature-key> "<title>"` or
  `shark create task <feature-key> "<title>"`.
- **Feature**:
  `shark create feature <epic-key> "<title>"`.
- **Change Card**:
  `shark create change "<title>" --justification="<why>" --description="<breadcrumb>"`
  plus `--epic=<key>` or `--feature=<key>` when there is a known anchor.
- **Idea**:
  `shark create idea "<title>" --description="<breadcrumb>"`.
- **Note**:
  `shark create note <target-key> "<breadcrumb>" --type=<comment|blocker|question|future|reference>`.

The created description or note should be a **breadcrumb, not a spec**: enough
context to recover what was found, where it was seen, why it matters, and any
obvious parent/link. Do not write acceptance criteria, implementation plans, PRDs,
or child task breakdowns during triage.

After creating, link or anchor the entity to whatever made it relevant (the
parent epic/feature/task) so it does not float unanchored in the backlog. Capture
the assigned key from the create response. Then **stop**.

## Anti-patterns

- Creating an entity for a trivial fix that should be handled immediately.
- Creating before the user confirms the proposal.
- Deduplicating only by keyword search instead of enumerating each candidate entity type.
- Proposing or reserving entity identifiers before Shark creates them.
- Writing a full spec, plan, task tree, or design document during triage.
- Creating an unanchored backlog item when a parent or note target is known.
- Treating vague input as classified work instead of asking one clarifying question.
