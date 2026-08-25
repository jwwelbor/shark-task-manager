---
feature_key: E32-F09-finish-skill-purity-pass-strip-remaining-shark-cli
epic_key: E32
title: Finish skill-purity pass — strip remaining shark CLI refs from embedded canonical skills
description: Complete the purity migration started in E32: remove bare shark CLI references from the five remaining embedded craft skills so they are tool-agnostic, route-returning skills per the E35 route-based redesign.
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

E32 extracted skill craft from workflow scaffolding and E35 introduced the route-based outcome model, but the final purity pass — removing shark CLI calls from the real craft-skill content files — was never completed. Historical planning recorded 56 references across six skills; the triage skill has since been retired and is absent from the canonical bundle. The remaining five craft skills (`internal/sharkdata/default_data/skills/`) still included CLI-specific plumbing in context and reference content. This violates the "tool-agnostic craft" contract established in E32/E35.

### Solution

Rewrite the 56 references to be tool-agnostic. Pure skills do their craft work and release a semantic outcome (`pass` / `fail` / `blocked`) for the engine to route. They must not invoke or name the shark CLI.

### Historical scope and current scan targets

| Skill | Historical count | Current status |
|---|---|---|
| specification-writing | 20 | embedded craft-skill scan target |
| triage | 13 | retired; absent from the canonical bundle |
| uat | 12 | embedded craft-skill scan target |
| assessment | 6 | embedded craft-skill scan target |
| research | 3 | embedded craft-skill scan target |
| quality | 2 | embedded craft-skill scan target |
| **Total** | **56 historical** | five active scan targets |

---

## Context

- The `_extracted/` scaffolding (48 files) was removed from the embed in the 2026-06-25 session — those were migration capture notes of shark plumbing being removed, not pure skill content.
- The on-disk `shark-data/skills/` tree preserves `_extracted/` dirs as a map of what plumbing each skill previously relied on — useful reference for the purity pass.
- The pure workflow files (e.g., `quality/workflows/review-code.md`) are already clean — the refs live in adjacent `context/` and reference files.
- "Contract-exempt" reference skills (brownfield-analysis, frontend-design) explicitly disclaim flow ownership; `pass/fail/blocked` outcomes don't apply to them, but shark CLI refs should still be removed.

---

## Acceptance Criteria

- [ ] The embedded craft-skill set owned by this feature contains no Shark platform names, bare `shark <verb>` invocations, or `/shark-rider` command forms: `specification-writing`, `uat`, `assessment`, `research`, and `quality`. The retired `triage` skill is absent from the canonical bundle.
- [ ] Workflow and orchestration skills are excluded because they own execution mechanics and may name the CLI. This feature does not broaden into their cleanup.
- [ ] Each rewritten context/reference file is tool-agnostic (no CLI names, command syntax, or platform-specific plumbing)
- [ ] Outcome-returning craft skills (`assessment` and `quality`) document the host-routable outcome contract (`pass` / `fail` / `blocked`), with `blocked` reserved for unavailable evidence or authority.
- [ ] `TestEmbedded_SkillsContainNoBareSharkCLIRefs` scans the owned craft-skill set and fails on any platform-specific reference, including `shark related-docs`, `shark sprint`, and `/shark-rider`.
- [ ] `make test` passes after changes (embed tests and rendered-prompt corpus)

---

## Out of Scope

- The on-disk `shark-data/` tree — addressed when CC-039 (hybrid embed/disk resolver) lands
- `skills/shark/` router files — they ARE the harness bootstrap and intentionally reference the CLI

---

*Last Updated*: 2026-06-25
