# Requirements

**Epic**: [Prompt and Skill Improvements](./epic.md)

---

## Overview

Requirements for improving the skills and prompts in `~/.claude/skills` that drive shark's AI-orchestrated development workflows.

---

## Functional Requirements

### Feature Area 1: Cross-Feature Interaction Lifecycle Enforcement

Applies to 8 skill files across `specification-writing` and `quality`. Detailed spec: `/home/jwwel/.claude/plans/review-dev-artifacts-interaction-prompt-logical-sketch.md`.

**REQ-F-001**: Interaction map creation in epic design
- **Description**: When an epic decomposes into 3+ features, `write-epic.md` must produce `{{epicId}}-interaction-map.md` with stable `I-##` IDs
- **Acceptance Criteria**:
  - [ ] Interaction map produced automatically for multi-feature epics
  - [ ] Table schema: `| ID | Producer feature | Consumer feature(s) | Shape | Payload | Style |`
  - [ ] Each I-## shape resolves to a section in `architecture.md`
  - [ ] File registered in shark via `shark related-docs add`
  - [ ] Epic cannot advance to decomposition without the interaction map (exit gate)

**REQ-F-002**: Interaction wire preservation in decomposition
- **Description**: `decompose-epic.md` must read the interaction map and ensure every I-## has a producer and consumer feature
- **Acceptance Criteria**:
  - [ ] Each feature description names the I-## IDs it produces or consumes
  - [ ] Orphan wires (no producer or no consumer) cause FAIL at exit gate

**REQ-F-003**: Cross-feature interactions section in feature PRDs
- **Description**: `write-feature-prd.md` must include a "Cross-feature interactions" section for STANDARD/COMPLEX features under multi-feature epics
- **Acceptance Criteria**:
  - [ ] Produces and Consumes subsections present with I-## IDs verbatim from interaction map
  - [ ] Shape source and contract test pointers included
  - [ ] Producer and consumer reference the same shape source and contract test pointer

**REQ-F-004**: Contract test design in test planning
- **Description**: `test-planning.md` must design at least one contract test per I-## declared in the feature spec
- **Acceptance Criteria**:
  - [ ] TC name and file path match the contract test pointer in the feature spec
  - [ ] TC tagged with I-## ID
  - [ ] Same TC referenced by both producer and consumer (no twin tests)

**REQ-F-005**: I-## propagation in task generation
- **Description**: `write-task.md` must propagate I-## IDs from feature spec to task spec
- **Acceptance Criteria**:
  - [ ] Cross-feature interfaces use `I-##` (not `CONTRACT-###`)
  - [ ] Task spec includes "Integration Contracts > Cross-feature" subsection
  - [ ] Every I-## the feature declares appears in the task spec (exit gate)

**REQ-F-006**: Interaction-map closure gate in design validation
- **Description**: `validate-design.md` must verify all I-## have producer and consumer features that name them
- **Acceptance Criteria**:
  - [ ] Closure table printed: one row per I-## with producer, consumer(s), shape source, status
  - [ ] Orphan wires = FAIL

**REQ-F-007**: I-## mirror check in task validation
- **Description**: `validate-tasks.md` must verify I-## from feature spec appear in task specs with matching pointers
- **Acceptance Criteria**:
  - [ ] Producer and consumer task specs cite the same shape source
  - [ ] Each I-## has exactly one contract test pointer on both sides
  - [ ] Missing I-## in task specs = FAIL

**REQ-F-008**: Wiring coverage matrix in QA
- **Description**: `qa-testing.md` must include I-## rows in coverage matrix alongside CONTRACT-### rows
- **Acceptance Criteria**:
  - [ ] Columns: producer/consumer, contract test path, test-exists, test-passes
  - [ ] Missing or failing contract test = FAIL
  - [ ] On FAIL: producer task reopened in shark, blocker note added to consuming feature

**REQ-F-009**: Interaction map template
- **Description**: Create `skills/specification-writing/context/interaction-map-template.md`
- **Acceptance Criteria**:
  - [ ] Template includes table schema, I-## ID assignment rules, shape source linking convention, and example row
  - [ ] Referenced by `write-epic.md`

---

### Feature Area 2: Dev Artifacts Interaction Prompt

**REQ-F-010**: Structured prompt for reviewing dev-artifacts interactions
- **Description**: A reusable prompt that guides reviewing how interactions between dev-artifact sessions were structured, to surface patterns and anti-patterns
- **Acceptance Criteria**:
  - [ ] Prompt covers: what was produced, how the session was routed, what decisions were made
  - [ ] Prompt can be applied iteratively across multiple dev-artifact directories
  - [ ] Output format suitable for feeding into a skill improvement workflow

---

### Feature Area 3: Skill Extraction Workflow

**REQ-F-011**: Prompt for extracting workflow material from `~/.claude/skills` skills
- **Description**: A reusable prompt (`dev-artifacts/planning/skill-workflow-extraction-prompt.md`) that guides refactoring any monolithic skill into workflow / prompt / methodology / references layers
- **Status**: Already drafted — needs review and refinement
- **Acceptance Criteria**:
  - [ ] Covers 5-step analysis: inventory, classify, design structure, draft artifacts, identify hazards
  - [ ] Classification matrix includes all 6 disposition types (KEEP IN SKILL, MOVE TO PROMPT, etc.)
  - [ ] Output format produces actionable artifacts (workflow skeleton, prompt file list, refactored skill outline)
  - [ ] Works for any skill in `~/.claude/skills`, not shark-specific

---

## Non-Functional Requirements

**REQ-NF-001**: Backward compatibility
- Changes to existing skill files must not break existing workflows that omit interaction maps (i.e., solo-feature epics and epics with fewer than 3 features proceed without modification)

**REQ-NF-002**: No hardcoded status names
- Any shark CLI calls added to skills must use workflow-driven status names, not hardcoded strings

---

*See also*: [Scope](./scope.md)
