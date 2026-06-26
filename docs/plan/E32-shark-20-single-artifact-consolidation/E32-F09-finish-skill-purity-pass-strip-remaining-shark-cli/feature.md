---
feature_key: E32-F09-finish-skill-purity-pass-strip-remaining-shark-cli
epic_key: E32
title: Finish skill-purity pass — strip remaining shark CLI refs from embedded canonical skills
description: Complete the purity migration started in E32: rewrite the 56 remaining shark CLI references in 6 embedded canonical skills so they are tool-agnostic, route-returning craft skills per the E35 route-based redesign.
size: M
---

# Finish skill-purity pass — strip remaining shark CLI refs from embedded canonical skills

**Feature Key**: E32-F09-finish-skill-purity-pass-strip-remaining-shark-cli

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

E32 extracted skill craft from workflow scaffolding and E35 introduced the route-based outcome model, but the final purity pass — removing shark CLI calls from the real skill content files — was never completed. 56 shark CLI references remain across 6 embedded canonical skills (`internal/sharkdata/default_data/skills/`), in context files and reference docs. Examples: `Set via \`shark update <key> --size=M\``, `The host stores these into shark context_data and advances...`. This means the embedded skills still describe shark-specific plumbing that pure skills must not reference — violating the "tool-agnostic craft" contract established in E32/E35.

### Solution

Rewrite the 56 references to be tool-agnostic. Pure skills do their craft work and release a semantic outcome (`pass` / `fail` / `blocked`) for the engine to route. They must not invoke or name the shark CLI.

### Scope by skill (refs remaining)

| Skill | Count | Location |
|---|---|---|
| specification-writing | 20 | context files |
| triage | 13 | context/reference |
| uat | 12 | context/workflow |
| assessment | 6 | context/reference |
| research | 3 | context/reference |
| quality | 2 | reference docs |
| **Total** | **56** | |

---

## Context

- The `_extracted/` scaffolding (48 files) was removed from the embed in the 2026-06-25 session — those were migration capture notes of shark plumbing being removed, not pure skill content.
- The on-disk `shark-data/skills/` tree preserves `_extracted/` dirs as a map of what plumbing each skill previously relied on — useful reference for the purity pass.
- The pure workflow files (e.g., `quality/workflows/review-code.md`) are already clean — the refs live in adjacent `context/` and reference files.
- "Contract-exempt" reference skills (brownfield-analysis, frontend-design) explicitly disclaim flow ownership; `pass/fail/blocked` outcomes don't apply to them, but shark CLI refs should still be removed.

---

## Acceptance Criteria

- [ ] `grep -r 'shark ' internal/sharkdata/default_data/skills/` returns 0 results
- [ ] Each rewritten context/reference file is tool-agnostic (no CLI names, command syntax, or platform-specific plumbing)
- [ ] Skills that describe outcome-returning workflows still document the outcome contract (`pass` / `fail` / `blocked`)
- [ ] `make test` passes after changes (embed tests)

---

## Out of Scope

- The on-disk `shark-data/` tree — addressed when CC-039 (hybrid embed/disk resolver) lands
- `skills/shark/` router files — they ARE the harness bootstrap and intentionally reference the CLI

---

*Last Updated*: 2026-06-25
