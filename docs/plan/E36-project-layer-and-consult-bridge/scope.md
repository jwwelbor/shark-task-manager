# Scope Boundaries

**Epic**: [Project Layer and Consult Bridge](./epic.md)
**Design source**: [`plan.md`](../../../dev-artifacts/2026-06-29-project-entity-design/plan.md) (see its Appendix for full rationale)

---

## In scope

- A `/shark project` verb namespace (`bootstrap`, `brownfield-analysis`,
  `product-design`) — skill-layer markdown dispatch.
- Rename `project-init` → `project bootstrap` with `project-init` kept as a
  deprecation alias.
- A derived-checklist + decision-log progress record + a
  `file_templates/progress.md` template.
- `/shark consult <agent>` advisor bridge (explicit form + NL recognizer in
  `query.md`), read-only by default.
- For the original F01-F03 scope, one Go change: `Description` on
  `BundleContentEntry`, populated from frontmatter and rendered in `agent list`
  / `skill list` (text + JSON).
- An "ops-as-entities" documentation convention.
- Bare `shark next` as a strictly read-only single-project portfolio-advice
  envelope with deterministic relationship layers, warnings, and an
  agent-facing `docs/product/` inspection prompt.
- State-aware `/shark-rider help` consumption of bare portfolio advice while
  keyed `shark next <key>` remains the dispatch API.
- The bounded Go read model, query repository, service, CLI branch, and Rider
  documentation required by E36-F04.

## Out of scope

- **Personas / user-journeys / success-metrics PRD files** — excluded: this is
  internal developer/agent tooling with no distinct external user types and no
  externally-measurable business outcomes. (Decision logged below.)
- Any CLI `project` subcommand, `project status`, or `project advance` — the
  layer is skill-only.
- Cross-project aggregation or a project registry — accepted as fan-out (open
  each DB), not needed today.
- Viewer/API surfaces for a project entity.
- Automatic root selection in Go, claiming, advancing, or dispatching from bare
  `shark next`.
- A persisted roadmap score/order field, a second workflow store under
  `docs/product/`, or general cross-entity implementation planning.
- `shark next --preview` or keyed-dispatch simulation. Keyed dispatch keeps its
  current normalization behavior.

---

## Rejected alternatives (from plan Appendix)

- **`project` as a shared-DB multi-instance entity** (a `projects` table +
  `project_id` FK on `epics`). Costs ~3–4.5k LOC and a SQLite table rebuild (a
  one-way door) to buy cross-project SQL we don't need. **Rejected — too heavy.**
- **`project` as a singleton workflow entity with a `project.yaml`
  status-machine.** Re-implements a lifecycle shark can't query and backdoors
  `project status`/`advance`. The derived checklist gives the same visibility
  without a second source of truth. **Rejected.**
- **Keep flat verbs / no namespace.** Leaves the pre-epic arc undiscoverable and
  inconsistent with `admin <sub>` / `task <sub>`. **Rejected.**
- **Relocate agents out of the embed into the skill tree.** Would split the
  source of truth for dual-use agents (injected *and* consulted); the consult
  bridge meets the need without moving definitions. **Rejected.**

## Accepted tradeoff

- **Cross-project aggregation is fan-out.** With one DB per project, "all my work
  across projects" means opening each DB, not one query. A registry-level
  fan-out command can be added later if it's ever needed.

---

## Decisions log (spec authoring)

- **2026-06-29** — Authored a lean epic index that references `plan.md` as the
  canonical design rather than a full multi-file PRD; `plan.md` is already a
  complete design doc with a rejected-alternatives appendix, and project
  convention favors concise specs for STANDARD-sized work.
- **2026-06-29** — EXCLUDED `personas.md`, `user-journeys.md`,
  `success-metrics.md`: internal tooling, no distinct external personas, no
  externally-measurable outcomes.
- **2026-06-29** — Stubbed three features (E36-F01/F02/F03) mirroring the plan's
  three implementation slices, smallest-first.
- **2026-07-20** — Registered E36-F04 as a later additive slice. Preserved the
  original plan's authority and boundaries for F01-F03, scoped the one-Go-change
  statement to those original slices, and retained the single-project,
  no-schema, advisory-document design.
