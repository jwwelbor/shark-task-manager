---
name: specification-writing
description: Authoritative source for all specification document generation workflows (Epics, Feature PRDs, Tasks). Provides standardized templates, procedures, and naming conventions for consistent documentation across all projects.
version: 2.0.0
inputs:
  - parent_spec_path: absolute path to parent spec (epic for feature, feature for task)
  - current_scope_path: absolute path to current scope document (optional)
  - related_research_path: absolute path to related research findings (optional)
  - design_docs_paths: list of design document paths (optional)
  - previous_spec_path: absolute path to a previous version for comparison (optional)
outputs:
  - selected_workflow: one of {decompose-epic, write-epic, write-feature-prd, write-task, plan-epic-ba-plan, plan-epic-tech-plan, plan-feature-ba-plan, plan-feature-tech-plan, check-ba-docs, check-tech-docs, refine-task-requirements}
  - specification_document: created or updated specification markdown
  - acceptance_criteria: structured list of acceptance criteria
  - dependencies: identified dependencies on other epics/features/tasks
  - decomposition: if applicable, child features/tasks decomposed from the spec
  - tech_plan: detailed implementation planning (if applicable)
---

# Specification Writing Skill

This skill provides technical product management and specification writing capabilities with standardized workflows and templates for creating three types of specification documents:

1. **Epic PRDs** — High-level product requirements for major initiatives
2. **Feature PRDs** — Detailed requirements for individual features within epics
3. **Implementation Tasks** — Agent-executable work items that implement features

## Suggested Upstream Input

An epic or feature PRD entering this skill should typically be grounded in upstream discovery/design artifacts (vision, success criteria, personas, journey maps, friction findings, validation verdicts). When such artifacts exist, every spec produced here should cite them so the SDLC inherits the validated scope and success bar.

If upstream artifacts are missing, **suggest** the user produce them first — but do not block. Some initiatives (small fixes, internal tooling, mid-stream pivots) legitimately enter the SDLC without a full discovery trail, and the user is the one who decides whether the upstream artifacts are needed for this piece of work.

## Feature-Level Sequence

The canonical SDLC feature sequence is:

```
feature.md (PRD)  →  wireframes.md  →  [prototype.md — optional]  →  architecture (02-…07-)  →  tasks  →  build
```

- **Feature PRD** is produced by `write-feature-prd.md` in this skill (business-analyst). User stories live as a section inside this file.
- **Wireframes** and **prototype** are produced by the **`feature-design` skill** (ux-designer, with cx-designer reviewing). Invoked **after the PRD is approved** and **before architecture/tasks** so wireframes shape downstream design.
- **Tasks** are produced by `write-task.md` (business-analyst). They MUST cite the wireframes (and prototype if present) by canonical path in their Design References sections — see `context/task-template.md`.

If a feature touches frontend code and wireframes are missing, `write-task.md` will stop and recommend running the feature-design workflow first.

## Workflow Selection

Based on what the user needs, invoke the appropriate workflow:

### Task-Level Refinement
**When**: A task needs BA or technical refinement.
**Invoke**: `workflows/refine-task-requirements.md`
**Output**: Updated PRD sections or design docs for the task.

### Epic Creation
**When**: User wants to document a new epic or major initiative.
**Invoke**: `workflows/write-epic.md`
**Output**: Multi-file epic documentation.

### Feature PRD Creation
**When**: User wants to document a feature within an existing epic.
**Invoke**: `workflows/write-feature-prd.md`
**Output**: Feature PRD.

### Plan-Gated Refinement (PLAN/ACT/CHECK pattern)
**When**: Called automatically from refinement templates via PLAN GATE instruction.
**Workflows** (in `workflows/plan/`):
- `epic-ba-plan.md` — Epic BA document selection decision tree
- `epic-tech-plan.md` — Epic tech document selection decision tree
- `feature-ba-plan.md` — Feature BA document selection (tier-aware, epic-incremental)
- `feature-tech-plan.md` — Feature tech document selection (contract-first)
- `check-ba-docs.md` — Shared BA validator: depth rules, pass/fail logic
- `check-tech-docs.md` — Shared tech validator: schema completeness, pass/fail logic

### Epic Decomposition
**When**: A refined epic must be broken into features.
**Invoke**: `workflows/decompose-epic.md`
**Output**: Feature set with dependency ordering and traceability matrix.

### Task Generation
**When**: Feature has completed architecture phase and needs implementation tasks.
**Invoke**: `workflows/write-task.md`
**Output**: A set of agent-executable task specs.

## Template Resources

All workflows reference these template files for consistent structure:

- `context/epic-template.md` — Epic document structure and sections
- `context/prd-template.md` — Feature PRD structure and sections
- `context/task-template.md` — Task structure and frontmatter
- `context/naming-conventions.md` — File and directory naming standards

## Quality Standards

All specification documents must:
- Be specific and actionable, avoiding vague language
- Include concrete examples where helpful
- Anticipate edge cases and error scenarios
- Define clear success metrics that are measurable
- Establish explicit boundaries to prevent scope creep
- Maintain consistent cross-references between files

---

*For detailed usage instructions, see [README.md](./README.md)*
