---
epic_key: E34
title: Prompt and Skill Improvements
size: L
---

# Prompt and Skill Improvements

**Epic Key**: E34-prompt-and-skill-improvements

---

## Goal

### Problem

The skills and prompts that power shark's AI-driven development workflows are monolithic: workflow control, step execution guidance, and reusable domain methodology are intermixed in single files. This makes skills hard to maintain, hard to reuse across contexts, and impossible to cleanly route through shark's orchestration layer. Additionally, skills lack mechanisms for enforcing cross-feature interaction contracts in multi-feature epics, leading to silent integration gaps that only surface during QA or post-merge.

### Solution

Refactor skills and prompts in `~/.claude/skills` into a layered architecture with clear separation between workflow definitions, prompt files, and reusable skill content. In parallel, add cross-feature interaction lifecycle enforcement to the specification and quality workflows so that integration wires get stable IDs, shape sources, and contract tests traceable from epic design through QA.

### Impact

- Skills become composable: workflow layers can be reused by shark's orchestration engine without pulling in step-execution concerns
- Cross-feature integration contracts are traceable from epic design to QA remediation
- Dev artifact review interactions are captured in a structured prompt that can be refined and tracked
- Future skill improvements can be made incrementally without touching unrelated workflow definitions

---

## Business Value

**Rating**: Medium

This epic improves the reliability and maintainability of the AI agent workflows that power all development in this project. Better layered skills reduce the cost of adding new workflow steps, catch integration failures earlier (preventing rework), and make shark's orchestration layer more capable of autonomous operation. The value is compounding — each skill improved multiplies across every epic that uses it.

---

## Epic Components

- **[Requirements](./requirements.md)** - Functional requirements by feature area
- **[Scope Boundaries](./scope.md)** - Out of scope items and future considerations

---

## Quick Reference

**Primary Users**: AI agents and Claude Code sessions operating within shark workflows

**Key Features**:
- Cross-feature interaction lifecycle enforcement across specification and quality skills
- Structured prompt for reviewing dev-artifacts interaction patterns
- Skill extraction workflow (layered architecture: workflow / prompt / methodology / references)

**Success Criteria**:
- Multi-feature epics automatically produce an interaction map with stable `I-##` IDs
- Cross-feature wires traceable from epic design through QA without manual lookup
- Any skill can be split into layers using the extraction prompt

**Timeline**: No fixed deadline — improvement-driven, features prioritized by workflow frequency

---

## Open Questions & Assumptions

1. **Which skills to extract first**
   - **Context**: The extraction prompt targets `~/.claude/skills` broadly; we need to prioritize
   - **Impact**: Determines feature decomposition order
   - **Recommendation**: Start with `specification-writing` and `quality` — highest workflow frequency

2. **Interaction map threshold (2 vs 3+ features)**
   - **Context**: The cross-feature interaction plan specifies "3+ features" as the trigger for requiring an interaction map
   - **Impact**: Whether enforcement applies to smaller epics
   - **Recommendation**: Keep the 3+ threshold; 2-feature epics rarely need formal interaction tracking

*NOTE: All items in this section MUST be reviewed interactively with the user before proceeding.*

---

*Last Updated*: 2026-06-22
