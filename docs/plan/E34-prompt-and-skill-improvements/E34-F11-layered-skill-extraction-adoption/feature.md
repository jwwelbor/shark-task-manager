---
feature_key: E34-F11-layered-skill-extraction-adoption
epic_key: E34
title: Layered Skill Extraction Adoption
description: Adopt the existing dev-artifacts/planning/skill-workflow-extraction-prompt.md reusable prompt (dated 2026-06-22) as tracked REQ-F-006 reference tooling for migrating a ~/.claude/skills skill into workflow/prompt/methodology/reference layers. Resolves D-E34-LEGACY-PROMPTS-001 item 2 (skill-workflow-extraction prompt).
---

# Layered Skill Extraction Adoption

**Feature Key**: E34-F11-layered-skill-extraction-adoption

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)
- **Epic Architecture**: [Architecture](../../architecture.md) _(if available)_

---

## Goal

### Problem
D-E34-LEGACY-PROMPTS-001 requires the E34 decomposition owner to either give the
"skill-workflow-extraction prompt" a tracked Shark owner/path or explicitly
cancel it before decomposition can pass. The artifact already exists at
`dev-artifacts/planning/skill-workflow-extraction-prompt.md` (dated
2026-06-22) and directly implements REQ-F-006's layered-extraction concept
(workflow / prompt / methodology / reference), but no Shark feature or task
currently owns it — it is unowned planning collateral sitting in the repo.

### Solution
Give the existing prompt an explicit Shark owner and repository-path record
instead of authoring new content. This feature exists solely to close the
D-E34-LEGACY-PROMPTS-001 gap: confirm the artifact is current and make it
discoverable as intentionally-retained reference tooling.

### Impact
- D-E34-LEGACY-PROMPTS-001 item 2 is resolved with a real tracked path, not a
  deferred/open item.
- The prompt remains available for future skill-layering migrations without
  being mistaken for an orphaned/abandoned dev-artifact.

---

## User Stories

### Must-Have Stories

**Story 1**: As the E34 decomposition owner, I want the
skill-workflow-extraction prompt to have a tracked Shark owner and path so
that D-E34-LEGACY-PROMPTS-001 is resolved and the artifact isn't orphaned.

**Acceptance Criteria**:
- [x] `dev-artifacts/planning/skill-workflow-extraction-prompt.md` is named as
      the tracked artifact path in a Shark task under this feature.
- [ ] A short discoverability pointer to the artifact exists outside
      `dev-artifacts/` (e.g. referenced from this feature or a skill-authoring
      reference doc).

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Track the existing extraction prompt
   - **Description**: `dev-artifacts/planning/skill-workflow-extraction-prompt.md`
     has an explicit Shark task naming its repository path as the deliverable
     for REQ-F-006's legacy-prompt gap.
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Task references the exact file path.
     - [ ] No duplicate prompt content is authored.

---

## Acceptance Criteria

**Scenario 1: Adoption is discoverable**
- **Given** the skill-workflow-extraction prompt already exists in
  `dev-artifacts/planning/`
- **When** someone looks for reusable skill-layering guidance
- **Then** they can find a pointer to the tracked artifact from Shark (this
  feature) rather than discovering it only by chance repo browsing

---

## Out of Scope

1. **New prompt authoring**
   - **Why**: The artifact is already complete and dated 2026-06-22; this
     feature only tracks/adopts it.
   - **Future**: Revisions to the prompt itself are separate, future-tracked
     work if ever needed.

---

## Success Metrics

1. **D-E34-LEGACY-PROMPTS-001 closure**
   - **What**: Whether both legacy-prompt items have a tracked path or an
     explicit cancellation.
   - **Target**: Item 2 (this feature) has a tracked path; item 1 is recorded
     as cancelled via epic decision note.
   - **Measurement**: `shark get E34-F11` and the E34 decision note.

---

*Last Updated*: 2026-08-31
