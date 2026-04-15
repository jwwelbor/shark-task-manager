---
feature_key: E07-F39
review_date: 2026-04-14
verdict: PASS
reviewer: task-decomposition-agent
---

# E07-F39 Task Decomposition Review

**Feature:** Remove legacy relationship tables and dual-path query code
**Verdict: PASS**

---

## 1. Requirements Coverage Matrix

| Spec Requirement | Covered By | Notes |
|---|---|---|
| REQ-F-001: No production code references legacy table identifiers | T-E07-F39-001 | AC-T1 grep assertion explicit in task |
| REQ-F-002: All dependency reads via `entity_relationships` | T-E07-F39-001 | dependency.go dual-path removal in scope |
| REQ-F-003: Template helpers source from `entity_relationships` | T-E07-F39-001 | helpers.go + new adapters in task scope |
| REQ-F-004: DROP TABLE migration + schema version 11→12 | T-E07-F39-001 | db.go migration + version bump explicit |
| REQ-F-005: Legacy models deleted | T-E07-F39-001 | All three model files listed for deletion |
| REQ-F-006: Viewer loads relationships on-demand per task | T-E07-F39-002 | GetRelationships per-task pattern explicit |
| REQ-F-007: taskRelAdapter / ViewerTaskRelationshipRepository / ViewerTaskRelationship / WithTaskRelRepo deleted | T-E07-F39-002 | All four symbols listed for deletion; AC-T1 grep assertion included |
| REQ-F-008: ViewerTask.Relationships []ViewerRelatedEntity with direction + type + key | T-E07-F39-002 | ViewerRelatedEntity struct; replace of DependsOn/BlockedBy/Blocks slices |
| REQ-F-009: shark link add round-trip through viewer | T-E07-F39-002 | Integration scenario IS-2 covered in test references |
| REQ-F-010: migrate_relationships.go removed | T-E07-F39-001 | File listed for deletion |
| REQ-N-001: make fmt && make lint && make test clean | Both tasks | AC-4/AC-7 referenced in both tasks; quality gate explicit in both |
| REQ-N-002: grep assertion passes | T-E07-F39-001 | AC-T1 criterion explicit |
| REQ-N-003: Backward-compat ViewerTask fields (open Q1) | T-E07-F39-002 | Open question flagged in spec §6; task references spec §3.4 where decision is documented |
| REQ-N-004: O(N) per-task GetRelationships calls | T-E07-F39-002 | IS-5 call-count test referenced |
| REQ-N-005: DROP migration uses version guard, not data check | T-E07-F39-001 | Migration function uses IF EXISTS; version bump provides guard |

**AC Coverage:**

| AC | Mapped to Task | Task AC Reference |
|---|---|---|
| AC-1 (legacy identifiers confined) | T-E07-F39-001 | AC-T1 + test-plan TC-1.1–1.5 |
| AC-2 (shark task deps end-to-end) | T-E07-F39-001 | test-plan TC-2.1–2.4 |
| AC-3 (link/unlink round-trip) | T-E07-F39-001 | test-plan TC-3.1–3.4 |
| AC-4 (tables absent after migration) | T-E07-F39-001 | test-plan TC-4.1–4.4 |
| AC-5 (viewer cross-entity links) | T-E07-F39-002 | test-plan TC-5.1–5.6 |
| AC-6 (deleted symbols absent) | T-E07-F39-002 | AC-T1 + test-plan TC-6.1–6.5 |
| AC-7 (make test clean) | Both | Referenced in both tasks |
| AC-8 (schema version + idempotency) | T-E07-F39-001 | test-plan TC-8.1–8.3 |

All 8 acceptance criteria and all 15 requirements are covered.

---

## 2. Task Quality

### T-E07-F39-001
- **Line count (non-frontmatter):** 49 — within 50-line limit
- **Code blocks:** None. Spec references use file:line notation, not copy-pasted code
- **References spec/test-plan:** Yes — "See spec.md AC-1, AC-2, AC-3, AC-4, AC-7, AC-8" and "See test-plan.md Section 1 (TC-1.1–TC-1.5, …)"
- **Scope coherence:** All modifications are in the repository layer, model layer, config/template helpers, and DB migration — tightly related, no scatter
- **Title accuracy:** Accurate — matches the deletion of legacy table references

### T-E07-F39-002
- **Line count (non-frontmatter):** 44 — within 50-line limit
- **Code blocks:** None. Spec references use section numbers and struct names
- **References spec/test-plan:** Yes — "See spec.md AC-5, AC-6, AC-7" and "See test-plan.md Section 1 (TC-5.1–TC-5.6, TC-6.1–TC-6.5, TC-7.1–TC-7.4)"
- **Scope coherence:** All modifications are in the viewer service layer and wiring — tightly related
- **Title accuracy:** Accurate — "Replace viewer adapter layer with on-demand entity service calls"

Both tasks PASS quality checks.

---

## 3. Ordering and Dependencies

- **T-E07-F39-001 (order=1):** No dependencies. Correctly first — establishes `entity_relationships` as the sole path and removes legacy repos/models that T-002 must not reference.
- **T-E07-F39-002 (order=2):** `dependencies: [T-E07-F39-001]`. Correctly second — viewer wiring requires `EntityRelationshipService` to be the canonical path (established by T-001) and references `EntityRelTaskKeyAdapter`/service patterns that T-001 ensures are the only available interfaces.

The dependency is directional and acyclic. No circular chains. Foundation changes (migration, adapter additions) precede consumer changes (viewer wiring). Valid DAG.

The spec's §4 implementation-order note ("Task 001 is a prerequisite for Task 002 only indirectly") is correctly reflected as a declared `dependencies` link without over-constraining the tasks.

---

## 4. Scope Alignment

- Both tasks stay entirely within the declared scope in feature.md and spec.md §2.4 (Out of Scope). No new relationship types, no dashboard aggregate queries, no CLI UX changes introduced.
- No scope creep detected. Every file listed for modification or deletion maps to a specific spec requirement.
- Task granularity is appropriate: T-001 is the data-layer cleanup (repositories, models, migration, template helpers); T-002 is the service/wiring cleanup (viewer). Each modifies a coherent set of files.
- Open question Q1 (REQ-N-003 backward-compat decision) is correctly flagged as a pre-implementation prerequisite in spec §6. Both tasks acknowledge this without trying to resolve it in the task spec itself.

---

## 5. Gaps Identified

None. All requirements and ACs map to at least one task. Test plan sections are cross-referenced with precision (section numbers, TC codes, IS codes). New test files are named explicitly in both task files.

---

## 6. Ordering Issues

None. The ordering is correct and well-justified.

---

## 7. Recommendations

No blocking issues. One advisory observation:

- **Open Question Q1 resolution timing:** spec §6 states Q1 (backward-compat ViewerTask JSON fields) must be resolved with the viewer frontend owner before T-002 implementation. This is correctly called out in spec §3.4. The developer picking up T-002 should verify Q1 is resolved before starting, or the task will need to make an arbitrary choice on REQ-N-003. This is a process note, not a task decomposition gap.

---

## Summary

Both tasks are well-formed, within line limits, reference the spec and test plan correctly, cover all requirements and acceptance criteria, are ordered correctly with valid dependencies, and stay within feature scope.

**Verdict: PASS — proceed to advance E07-F39.**
