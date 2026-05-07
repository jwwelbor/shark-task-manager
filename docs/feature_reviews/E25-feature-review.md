# Feature Decomposition Review: E25 — Add Tech-Debt Entity Type

**Date**: 2026-04-05
**Verdict**: PASS

---

## Requirements Coverage Matrix

| Epic Success Criterion | Feature | Coverage |
|------------------------|---------|----------|
| SC-1: TD CRUD via `shark td` CLI commands | E25-F01 | Full — F01 goal explicitly lists create, get, list, update, delete, triage, note, notes, context subcommands |
| SC-2: TD workflow with status transitions (`shark status advance/set`) | E25-F01 | Full — F01 goal specifies workflow config (level constant, default statuses, parser/validator entries) |
| SC-3: TD notes and context (`shark td note add`, `shark td context set`) | E25-F01 | Full — F01 goal includes entity type registration in ValidEntityTypes enabling notes/context |
| SC-4: TD-### key auto-detection in core commands | E25-F01 | Full — F01 goal lists all dispatch points: get, delete, status, list, update, context, notes, link, errors, render, analytics, workflow_show_action, run |
| SC-5: TD in search results and analytics | E25-F01 | Full — F01 goal mentions search integration via UNION ALL and analytics integration |
| SC-6: File path display on all entity creation commands | E25-F02 | Full — F02 goal lists all affected commands: epic, feature, create (unified), bug, change, and tech_debt |
| SC-7: DB migration safety (idempotent, no data loss) | E25-F01 | Full — F01 goal specifies schema version bump 10 to 11 |
| SC-8: JSON output via --json and --field flags | E25-F01 | Full — implied by following the Bug entity pattern which already supports --json/--field |

**Result**: All 8 success criteria are covered. No gaps.

---

## Feature Quality Assessment

### E25-F01: Tech-Debt Entity Implementation

- **Cohesion**: High. Represents the complete vertical slice of the tech-debt entity across all layers (model, DB, repo, service, CLI, workflow, search, analytics, templates).
- **Specificity**: The goal section is highly specific, naming exact files, schema versions, status names, and integration points. Sufficient for task generation.
- **Title accuracy**: Title accurately reflects content.

### E25-F02: Show File Location on Entity Create

- **Cohesion**: High. A single, focused UI improvement affecting creation commands across all entity types.
- **Specificity**: The goal section names all affected functions and files. States it is UI-only with no service/repo changes. Clear and actionable.
- **Title accuracy**: Title accurately reflects content.

**Result**: Both features are cohesive, specific, and appropriately scoped.

---

## Overlap Analysis

- F01 includes the new `runTechDebtCreate` in `tech_debt.go` which will need file path display.
- F02 modifies existing entity creation commands plus ensures the TD create command follows the same pattern.
- The F02 goal document explicitly acknowledges this boundary: "the last of which is implemented as part of E25-F01 but requires this feature's pattern for consistency."
- This is a natural integration point, not an overlap. F01 creates the command; F02 ensures all create commands (including the new one) display file paths.

**Result**: No problematic overlaps. Clear boundary between features.

---

## Ordering and Dependencies

| Feature | Execution Order | Dependencies |
|---------|----------------|--------------|
| E25-F01 | 1 | None — foundation feature |
| E25-F02 | 2 | Soft dependency on E25-F01 (needs tech_debt.go for end-to-end testing of TD create file path) |

- F01 must come first as it creates the tech-debt entity that F02 needs for complete testing.
- F02 can modify existing entity creation commands independently but needs F01 for the TD create command.
- No circular dependencies.

**Result**: Ordering is correct and reflects actual dependencies.

---

## Scope Alignment

- Both features map directly to epic requirements. No scope creep.
- Items explicitly listed as out of scope in the epic (auto-detection, metrics dashboard, promotion, bulk import, HTTP API, inter-entity dependencies) are not included in either feature.
- Feature granularity is appropriate: F01 is large but represents a single vertical entity implementation (matching the established Bug/Change-Card pattern); F02 is small and focused.

**Result**: Features stay within epic scope with appropriate granularity.

---

## Observations

1. F01 is a large feature (~7 new files, ~18 modified files) but this is inherent to adding a full entity type. The architecture doc and research report provide detailed implementation guidance that will make task generation straightforward.
2. Both feature PRD files have template placeholder sections (User Personas, User Stories, Requirements, Acceptance Criteria) that are unfilled. The Goal sections contain the substantive content. This is acceptable at the decomposition review stage — detailed requirements will be filled during the specification phase.
3. The architecture doc (architecture.md) and research report (research-report.md) are thorough and aligned with both features.

---

## Recommendations

None. The decomposition is clean, complete, and well-ordered.

---

*Reviewed by: Feature Decomposition Review Agent | Date: 2026-04-05*
