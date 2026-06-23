---
feature_key: E32-F08-supplemental-skill-library-consolidation-jaunty-pa
epic_key: E32
title: Supplemental skill-library consolidation (jaunty-panda)
description: Track execution of ~/.claude/plans/implement-home-jwwel-projects-shark-task-jaunty-panda.md as a supplemental skill-library consolidation stream. Batch A is complete; Batch B creates standalone methodology skills; Batch C audits original shark-coupled skills for layer purity.
size: L
---

# Supplemental skill-library consolidation (jaunty-panda)

**Feature Key**: E32-F08

---

## Epic

- **Epic PRD**: [Epic](../epic.md)
- **Execution Plan**: `~/.claude/plans/implement-home-jwwel-projects-shark-task-jaunty-panda.md`

---

## Goal

### Problem

E32 originally focused on the shark-coupled skills needed to make Shark 2.0 dispatch portable. The jaunty-panda plan extends the same three-layer architecture to a broader reusable skill library. Without tracking that work in shark, the repository can drift: supplemental skills appear under `shark-data/skills/` without a clear acceptance gate, while E32's core engine/migration work becomes harder to evaluate.

### Solution

Track the jaunty-panda plan as a supplemental E32 feature. Keep E32's core acceptance gates focused on engine, prompt, workflow, and original shark-coupled skill migration, while processing the broader skill library with the same layer rules:

- `shark-data/workflow/` owns routing, branching, status semantics, and agent selection.
- `shark-data/prompts/` owns workflow scaffolding, gates, mutations, and status movement.
- `shark-data/skills/` owns reusable domain methodology only.

### Impact

- Supplemental skill work is visible in shark state instead of only in a local plan file.
- Batch A completion and Batch B/C remaining work have concrete tasks.
- Final audits can distinguish canonical craft files from transient `_extracted/` scaffolding sidecars.

---

## Batch Status

### Batch A: Extract Existing Skills

Completed before this feature was created:

- `brownfield-analysis`
- `frontend-design`
- `product-design`

### Batch B: Create New Methodology Skills

Plan target:

- `org-standards`
- `clarification`
- `content-validation`
- `cross-artifact-analysis`
- `overconfidence-prevention`
- `breakdown-test`
- `status-tracker`

Current state at feature creation:

- Usable skill files present: `clarification`, `cross-artifact-analysis`, `breakdown-test`
- Placeholder directories only: `content-validation`
- Missing: `org-standards`, `overconfidence-prevention`, `status-tracker`

### Batch C: Audit Existing Shark-Coupled Skills

Audit these skills for layer purity and required file shape:

- `architecture`
- `assessment`
- `implementation`
- `quality`
- `research`
- `specification-writing`
- `test-driven-development`
- `uat`

---

## Acceptance Criteria

1. Batch B target skills each have a `SKILL.md` and at least one useful methodology workflow or context reference where appropriate.
2. Batch B skill files contain no `shark` commands.
3. Batch B skill files contain no owned status transition terms: `status advance`, `status set`, or `next-status`.
4. Batch C audit produces a concrete violation report for the original shark-coupled skills.
5. Batch C violations are either fixed or explicitly routed to E32-F04 as scaffolding migration work.
6. The final audit distinguishes canonical craft files from `_extracted/` sidecars; `_extracted/` sidecars are not counted as shipped methodology.
7. Shark notes on E32-F08 record batch completion and any follow-on blockers.

---

## Verification

Run after Batch B:

```bash
for skill in org-standards clarification content-validation cross-artifact-analysis overconfidence-prevention breakdown-test status-tracker; do
  test -f "shark-data/skills/$skill/SKILL.md" || echo "MISSING SKILL.md: $skill"
done

grep -R -n "shark " shark-data/skills/{org-standards,clarification,content-validation,cross-artifact-analysis,overconfidence-prevention,breakdown-test,status-tracker}/
grep -R -n -E "status advance|status set|next-status" shark-data/skills/{org-standards,clarification,content-validation,cross-artifact-analysis,overconfidence-prevention,breakdown-test,status-tracker}/
```

Run after Batch C:

```bash
for skill in architecture assessment implementation quality research specification-writing test-driven-development uat; do
  test -f "shark-data/skills/$skill/SKILL.md" || echo "MISSING SKILL.md: $skill"
done

grep -R -n "shark " shark-data/skills/{architecture,assessment,implementation,quality,research,specification-writing,test-driven-development,uat}/ --exclude-dir=_extracted
grep -R -n -E "status advance|status set|next-status" shark-data/skills/{architecture,assessment,implementation,quality,research,specification-writing,test-driven-development,uat}/ --exclude-dir=_extracted
```

---

## Out of Scope

- Rebuilding the Shark 2.0 engine.
- Repointing harness slash commands.
- Deciding final `_extracted/` sidecar disposition. That gate belongs to E32-F04, though this feature may report violations.

---

*Last Updated*: 2026-06-22
