# Scope Boundaries

**Epic**: [Prompt and Skill Improvements](./epic.md)

---

## Overview

What is explicitly NOT included in this epic.

---

## Out of Scope

**1. Refactoring all skills in `~/.claude/skills`**
- **Why**: The skill extraction prompt (Feature Area 3) is a tool to enable extraction — it doesn't commit to extracting every skill in this epic
- **Future Consideration**: Each skill extraction is a candidate for a future feature or sub-task
- **Workaround**: Use the extraction prompt ad hoc as skills are touched during other epics

**2. Changes to shark's orchestration engine or workflow JSON files**
- **Why**: This epic improves the skill *content*, not the orchestration mechanism. Workflow JSON changes belong in the enhancements epic (E07).
- **Future Consideration**: Once skills are layered, the workflow engine can be updated to route to workflow-layer files directly

**3. Interaction contract enforcement for bug or change-card workflows**
- **Why**: Cross-feature interaction tracking applies to feature-level decomposition. Bug and change-card workflows operate at a different granularity.
- **Future Consideration**: Could be extended if complex bugs span multiple features

**4. Automated skill linting or CI checks**
- **Why**: Out of scope for this epic; enforcement is agent-driven via exit gates in skill workflows
- **Future Consideration**: A lint pass that checks for workflow-layer content in skill files

**5. Writing new skills from scratch**
- **Why**: This epic improves and layers existing skills, not creates new domains
- **Future Consideration**: New skill domains are independent epics

---

## Alternative Approaches Considered But Rejected

**Alternative: Single "mega-skill" with inline workflow control**
- **Description**: Keep skills monolithic but add clearer section markers
- **Pros**: No structural changes to existing files
- **Cons**: Doesn't enable orchestration routing; section markers drift without enforcement
- **Decision Rationale**: Doesn't solve the composability problem

**Alternative: Extract interaction contracts into a separate shark entity type**
- **Description**: Track `I-##` IDs as shark entities (like bugs or change-cards)
- **Pros**: Full shark tracking, searchable
- **Cons**: Heavyweight for design-time artefacts; markdown files suffice; adds DB schema complexity
- **Decision Rationale**: Interaction maps as markdown files registered via `related-docs` is sufficient

---

## Future Epic Candidates

| Future Epic Concept | Priority | Dependency |
|---|---|---|
| Extract `specification-writing` skill into layered architecture | High | This epic (extraction prompt) |
| Extract `quality` skill into layered architecture | High | This epic (extraction prompt) |
| Shark orchestration engine routing to workflow-layer files | Medium | Skill extraction complete |
| Automated skill layer validation (lint) | Low | Layered skills exist |

---

*See also*: [Requirements](./requirements.md)
