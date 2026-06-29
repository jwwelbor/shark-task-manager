---
epic_key: E36
title: Project Layer and Consult Bridge
description: Skill-layer pre-epic 'project' verb namespace (bootstrap / brownfield-analysis / product-design) with a derived-checklist + decision-log progress record, plus /shark consult <agent> advisor bridge. One Go change: agent/skill list descriptions. No schema changes.
---

# Project Layer and Consult Bridge

**Epic Key**: E36

> **Canonical design**: [`dev-artifacts/2026-06-29-project-entity-design/plan.md`](../../../dev-artifacts/2026-06-29-project-entity-design/plan.md).
> This index is the 3-minute summary; the plan is the authority (including the
> rejected-alternatives appendix). Where they ever disagree, the plan wins.

---

## Goal

### Problem
The work that happens *before* any epic exists — bootstrapping architecture
docs, analyzing an existing codebase, running product design — has no home in
shark. Its commands (`/shark project-init`, `product-design`,
`brownfield-analysis`) are flat, hyphenated outliers in a CLI that otherwise
groups (`admin <sub>`, `task <sub>`); they're undiscoverable and give the
pre-epic arc no through-line or progress record. Separately, there is no
first-class way to talk to shark's agent personas as advisors — the personas
exist only to be dispatched as `spawn_agent` workers.

### Solution
Add a **project layer** that lives in the shark *skill*, not the *CLI*: markdown
verbs the agent follows, leveraging the Go CLI only as a tool. Group the
pre-epic activities under one `/shark project <activity>` namespace (a menu, not
a sequence). Track the arc with a single advisory **progress record** — a
checklist *derived from artifacts on disk* plus an append-only human decision
log — so it can never drift into a second authoritative copy of state. Add
`/shark consult <agent>`, a skill verb that loads an agent persona via
`shark agent get` and adopts it inline as a read-only advisor. The only Go
change is populating a `description` field on `agent list` / `skill list`.

### Impact
- The pre-epic PDLC arc becomes a discoverable, consistent command surface
  (`/shark project bootstrap | brownfield-analysis | product-design`).
- Every future design conversation becomes a first-class command
  (`/shark consult <agent>`) instead of ad-hoc prompting.
- Zero schema cost: no `project` table, no `project_id`, no migration — tenancy
  stays a deployment concern (one DB per project, the natural Turso model).
- `agent list` / `skill list` become self-describing (name → description),
  enabling consult fuzzy-matching and the discovery menu.

---

## Business Value

**Rating**: High

Strategic developer-experience and onboarding improvement at trivial
implementation cost (skill-layer markdown + one Go field; no schema work). It
gives the pre-epic arc a home and a through-line, restores command-surface
consistency, and ships a reusable advisor pattern. The design was itself
produced by consulting two agent personas that converged independently — the
first proof that the consult bridge is worth building.

---

## Epic Components

- **[Requirements](./requirements.md)** — functional and non-functional requirements
- **[Scope Boundaries](./scope.md)** — out-of-scope items and rejected alternatives
- **Design source** — [`plan.md`](../../../dev-artifacts/2026-06-29-project-entity-design/plan.md) (attached via `related-docs`)

Personas, user-journeys, and success-metrics files are intentionally omitted —
this is internal developer/agent tooling with no distinct external user types or
externally-measurable business outcomes (exclusions logged in `scope.md`).

---

## Features

Three independent slices, smallest-first (decomposition from the plan's
implementation table):

| Feature | Slice | Touches | Size |
|---------|-------|---------|------|
| **E36-F01** | Consult bridge | `agent list`/`skill list` description (Go) + `verbs/consult.md` + `query.md` recognizer + `SKILL.md` allowlist row | S |
| **E36-F02** | Project namespace + progress record | `verbs/project.md` dispatch; rename `project-init` → `project bootstrap` (alias old); `file_templates/progress.md`; derived-checklist + decision-log updates in activity verbs | M |
| **E36-F03** | Ops-as-entities convention | docs + an "Ops epic" convention for deploy/devops; no new mechanism | S |

Slice 1 ships on its own. Slice 2 delivers the project layer. Slice 3 is mostly
convention.

---

## Quick Reference

**Primary Users**: Shark users and AI agents driving the PDLC (developers doing
project setup; agents running pre-epic activities and consulting personas).

**Key Capabilities**:
- `/shark project bootstrap | brownfield-analysis | product-design`
- Derived-checklist + decision-log progress record (advisory, never authoritative)
- `/shark consult <agent>` advisor bridge (explicit + NL forms)
- Self-describing `agent list` / `skill list`
- Recurring ops modeled as regular shark entities, not project activities

**Success Criteria**:
- The three slices land as skill-layer markdown + one Go field, no schema migration.
- `project-init` keeps working as a deprecation alias for `project bootstrap`.
- A consult never mutates shark state unless explicitly asked.

---

*Last Updated*: 2026-06-29
