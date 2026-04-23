---
epic_key: E28
document_type: feature-review
title: E28 Feature Decomposition Review
---

# E28 Feature Decomposition Review

**Epic:** E28 — Entity Tagging with Managed Vocabulary
**Reviewer:** Feature Decomposition Review (quality gate before task generation)
**Date:** 2026-04-22
**Features reviewed:** 7 (E28-F01 … E28-F07)

---

## Verdict

**PASS**

The seven features cover every PRD goal, success criterion, and UAT scenario with clear, non-overlapping boundaries and a correct dependency ordering. Two minor ambiguities are flagged as "clarify during task generation" — they do not block advancement.

---

## Requirements Coverage Matrix

### Epic Goals → Features

| # | Epic Goal (from §2) | Covered by |
|---|---|---|
| G1 | Uniform tagging across all six entity types (`epic`, `feature`, `task`, `bug`, `change-card`, `idea`) | F01 (schema + EntityTypeIdea), F04 (attach across all six entity services), F05 (query across all six list scopes) |
| G2 | Closed vocabulary — unregistered tags rejected at write time | F03 (vocabulary admin), F04 (reject at attach with SC-2 stderr shape) |
| G3 | Maintainer gate with sudo-style cache | F02 (gate package + config + set-password bootstrap), F03 (first consumer) |
| G4 | Queryable via `shark list --tag=` and `shark search --tag=` | F05 |
| G5 | Per-entity-type `tag_required_for` enforcement | F04 (`TagService.EnforceRequired` invoked in all six create paths) |
| G6 | Surface tags + filtering in E27 web viewer (read-only) | F06 |

All six goals mapped.

### Success Criteria → Features

| SC | Statement | Covered by |
|---|---|---|
| SC-1 | Register + apply tag across 6 types, `list --tag` returns all 6 | F01, F03 (register), F04 (apply), F05 (list) |
| SC-2 | Unregistered tag rejected with vocabulary listing + `shark tags add` command | F04 (error composition at attach time) |
| SC-3 | Non-maintainer blocked with `.sharkconfig.json` pointer in error | F02 (gate error type), F03 (consumes, surfaces to CLI) |
| SC-4 | Sudo-style 60s cache works across successive commands | F02 (cache store + window) |
| SC-5 | Rename atomic; `entity_tags` rows immutable | F03 (ADR-8 rename semantics) |
| SC-6 | `tag_required_for: ["task"]` enforced on task create but not epic create | F04 (service-layer enforcement in `EnforceRequired`) |
| SC-7 | Exactly two new tables, no per-entity join tables | F01 (schema) |
| SC-8 | Viewer displays tags + supports tag filter; no vocab-mutation UI | F06 (read-only integration) |
| SC-9 | Maintainer gate reusable; in a shared package; tag commands consume it | F02 (`internal/auth/maintainer` package, tag-agnostic API) |
| SC-10 | Docs complete (`shark tags` page, `.sharkconfig.json` fields, `--tag` on create/update/list/search, migration callout) | F07 |

All ten success criteria mapped.

### UAT Scenarios → Features

| UAT | Scenario | Covered by |
|---|---|---|
| UAT-1 | Register a tag and apply across 6 entity types | F01, F03, F04, F05 |
| UAT-2 | Unregistered tag → helpful error (vocabulary + exact command) | F04 |
| UAT-3 | Non-maintainer cannot modify vocabulary | F02, F03 |
| UAT-4 | Sudo cache burst within 60s | F02 |
| UAT-5 | Rename without per-entity migration | F03 |
| UAT-6 | `tag_required_for` enforcement per entity type | F04 |
| UAT-7 | Viewer shows tags + filters; no mutation UI | F06 |
| UAT-8 | Gate is reusable shared package | F02 (code-review-observable via package location and consumer shape) |
| UAT-9 | Documentation complete | F07 |

All nine UAT scenarios mapped.

---

## Feature Quality

- **F01 Tags Schema and Migration** — cohesive foundation: two tables + six triggers + indexes + version bump + `EntityTypeIdea`. No CLI surface. Clear scope.
- **F02 Reusable Maintainer Authorization Gate** — genuinely tag-agnostic; includes bootstrap helper (`shark admin maintainer set-password`) and `.sharkconfig.json` `maintainer` block. Could be lifted into another epic without modification.
- **F03 Tag Vocabulary Service and CLI** — vocabulary admin only (list/add/rm/rename). Consumes F02. `TagRepository` lives here; `EntityTagRepository` deferred to F04 — a deliberate split that prevents F03 from growing.
- **F04 Entity Tag Attachment and Enforcement** — polymorphic attach/detach, per-entity create/update CLI integration, `tag_required_for` enforcement, retroactive `<entity> tag add|rm`. Large but cohesive (it is "the write path for entity_tags, everywhere").
- **F05 Tag-Based Querying in List and Search** — pure read path. Adds `--tag` filter at every list scope + search + per-entity `Tags` field population for Get responses. Clean mirror of F04.
- **F06 Web Viewer Tag Integration** — consumes F05 read APIs. Explicitly read-only; no mutation UI. Boundary with F03 (vocabulary admin) is clean.
- **F07 Tagging Documentation** — one feature, one responsibility: user-facing docs across `docs/cli-reference/`.

Each feature is independently deliverable in the sense that once its dependencies are done, it stands alone. Titles accurately reflect content.

### No scope overlaps

- `TagRepository` in F03; `EntityTagRepository` in F04. Different tables.
- `TagService.AddTag/RemoveTag/RenameTag` (vocab mutation) in F03; `TagService.AttachMany/DetachOne/EnforceRequired` (entity links + enforcement) in F04; `TagService.EntityIDsByTags` (query) in F05. Clean method-level partition.
- Gate logic lives in F02; gate consumption lives in F03. F04/F05 do not invoke the gate (correct — they are not maintainer operations).
- Viewer read surface in F06; no CLI code in F06. No viewer code in F03/F04/F05.

---

## Ordering & Dependencies

Declared execution order:

```
F01 (schema)    ── blocks F02?F03, F04, F05, F06
F02 (gate)      ── can run parallel with F01; blocks F03
F03 (vocab)     ── depends on F01, F02; blocks F04
F04 (attach)    ── depends on F01, F03; blocks F05
F05 (query)     ── depends on F04; blocks F06
F06 (viewer)    ── depends on F05; parallel with F07
F07 (docs)      ── depends on F03, F04, F05, F06 (user-facing surface finalized)
```

- **No circular dependencies.** The chain is F01 → F03 → F04 → F05 → F06, with F02 as a parallel foundation and F07 trailing.
- **Foundation first.** F01 (tables) and F02 (gate, independent) ship before anything that needs tables or authorization.
- **Read after write.** F05 (query) is correctly gated behind F04 (attach) — no way to query tags that do not exist.
- **Viewer after CLI stabilizes.** F06 consumes the service-layer API that F05 finalizes.
- **Docs after surface stabilizes.** F07's parallel-with-F06 claim is qualified ("provided doc updates lag API/CLI stabilization") — this is the correct hedge.

F02 is explicitly labeled "depends on nothing" and can be developed in parallel with F01. That is correct: F02 touches `internal/auth/maintainer/`, `internal/config/`, and a new CLI bootstrap command; it does not touch `internal/db` or any entity services.

---

## Scope Alignment

- **No scope creep.** The explicit "out of scope" list from PRD §3 (dashboards, viewer vocabulary management, hierarchical tags, aliases, auto-suggest, search rework, visibility rules, bulk pattern tagging, tag metadata, freeform-tag migration) is not covered by any feature. No feature extends beyond v1.
- **No unnecessary features.** Every feature traces to a concrete SC / UAT. Removing any of the seven would leave one or more success criteria uncovered.
- **Granularity appropriate.** Seven features for a multi-layer, multi-entity, cross-cutting capability is reasonable — not too fine (features like "add `--tag` to create" alone would fragment), not too coarse (one mega-feature would be unsized).

---

## Gaps Identified

None that block decomposition. Two minor ambiguities to resolve during task generation:

1. **Ownership of `Config.TagRequiredFor` wiring.** F02's description lists only the `maintainer` object as its `.sharkconfig.json` addition, but F04's integration-points line says "`Config.TagRequiredFor` wiring from F02's config work." Task generation should explicitly assign the `TagRequiredFor` field addition to either F02 (as part of config struct expansion — more coherent) or F04 (as part of enforcement). The architecture doc §3.3 shows both fields together, so F02 is the natural home.

2. **Where `TagService.ValidateNames` lives.** F04 references "reuses `TagService.ValidateNames`" but F03's description does not name the method — it only describes "name normalization." Task generation should confirm `ValidateNames` is introduced in F03 (it is the natural location; F03 owns the vocabulary service) and consumed in F04.

Neither ambiguity changes the feature boundary; both are task-description-level clarifications.

---

## Overlaps Identified

None.

---

## Ordering Issues

None. The declared order is consistent with declared dependencies and with the architecture integration points.

One observation (not an issue): F02 ships a tiny CLI command (`shark admin maintainer set-password`) ahead of F03's larger `shark tags` command group. This is correct — F02's command is a prerequisite bootstrap for setting up the gate that F03 consumes, and is tag-agnostic.

---

## Recommendations

1. **Advance** E28 from `ready_for_feature_decomposition_review` to the next workflow status.
2. **During task generation (F04),** explicitly assign config wiring:
   - F02 tasks create `Config.Maintainer` **and** `Config.TagRequiredFor` fields (both are `.sharkconfig.json` additions; grouping them avoids a second config PR).
   - F04 tasks *consume* `Config.TagRequiredFor` in `TagService.EnforceRequired`.
3. **During task generation (F03),** the first task should introduce `TagService.ValidateNames(names []string)` explicitly so F04's dependency on it is unambiguous.
4. **During task generation (F01),** surface the "set `skip_migrations: false`" developer callout as part of the migration task's PR description template — per `.claude/rules/database-critical.md` — so it does not get lost between F01 shipping and the next developer running `shark` against an existing DB.
5. **Optional:** Consider whether F05's "populate `Tags` field on Get responses for all six entity types" might belong in F04 (it is a write-side effect: once attach happens, Get should return tags). The current placement in F05 is defensible ("all read-path changes in one feature") and is not a blocker; just flag for the tech lead during task generation.

---

*Last Updated*: 2026-04-22
