---
feature_key: E36-F02-project-namespace-and-progress-record
epic_key: E36
title: Project namespace and progress record
description: /shark project bootstrap|brownfield-analysis|product-design namespace (menu, not sequence); rename project-init->bootstrap (alias old); file_templates/progress.md; derived-checklist + decision-log update protocol.
---

# Project namespace and progress record

**Feature Key**: E36-F02-project-namespace-and-progress-record | **Size**: M

**Epic**: [Project Layer and Consult Bridge](../../epic.md) | **Design**: [plan.md §1–3](../../../../../dev-artifacts/2026-06-29-project-entity-design/plan.md)

---

## Goal

### Problem

The pre-epic arc — bootstrapping architecture docs, analyzing an existing codebase, running product design — has no home in shark. Its commands (`/shark project-init`, `product-design`, `brownfield-analysis`) are hyphenated outliers in a CLI that otherwise groups commands (`admin <sub>`, `task <sub>`, `epic <sub>`). They are undiscoverable and give the arc no through-line. There is no lightweight way to see what pre-epic setup has been done or to record human decisions made along the way, yet any attempt to track this in a managed entity risks creating a second, stale copy of state (the managed entity and reality drift).

### Solution

Group the pre-epic activities under one `/shark project <activity>` verb namespace — a menu, not a sequence — implemented as skill-layer markdown in `verbs/project.md`. Activities remain independent; run what you need. Rename `project-init` → `project bootstrap` so there is exactly one `init` (`shark admin init` owns DB/config/`docs/plan/`); the old form works as a deprecation alias. Add a single advisory progress record — seeded from a `file_templates/progress.md` template to `docs/product/progress.md` — with two parts: a **derived checklist** regenerated from what's on disk (checked against artifact presence, rendered as `[x] / [~] / [ ]`), and an **append-only written decision log** for irreducible human narrative. When any project activity completes, the verb appends a decision-log entry and regenerates the checklist. Writing the file is a skill `Write` craft operation, not a shark mutation, so the checklist is always advisory and can never become a second source of truth.

### Impact

- The pre-epic PDLC arc has a consistent, grouped, discoverable command surface.
- `project bootstrap` cleanly separates architecture-doc generation from `shark admin init` (DB setup).
- Developers and agents can see at a glance what setup has been done without maintaining a manual ledger.
- The derived checklist is always accurate because it reads from disk; it cannot drift.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer starting a new project, I want to run `/shark project bootstrap` to generate architecture docs so that there is a single, clearly named command for that step that doesn't collide with `shark admin init`.

**Acceptance Criteria**:
- [ ] `/shark project bootstrap` invokes the architecture-doc generation activity (brownfield/greenfield auto-detection).
- [ ] `/shark project-init` (old form) produces identical behaviour and prints a deprecation notice.
- [ ] `shark admin init` is unaffected and continues to own DB/config/`docs/plan/`.

**Story 2**: As a developer, I want to run any of the three activities independently so that I don't have to follow a prescribed sequence when only one is needed.

**Acceptance Criteria**:
- [ ] `/shark project brownfield-analysis` runs the deep existing-codebase analysis without requiring bootstrap to have run first.
- [ ] `/shark project product-design` runs the D01–D14 vision → validated-concept arc independently.
- [ ] All three are listed when the user types `/shark project` with no activity argument.

**Story 3**: As a developer, I want a progress record that derives its checklist from disk artifacts so that I can see what's done without manually updating a file.

**Acceptance Criteria**:
- [ ] `docs/product/progress.md` is seeded from the template on first activity completion (if the file doesn't already exist).
- [ ] After any project activity, the checklist section reflects current artifact presence on disk.
- [ ] The presence of `[x]` in the checklist has no programmatic effect — it is advisory display only.

**Story 4**: As a developer, I want to append human decisions to the progress record so that I can record why things were done a certain way.

**Acceptance Criteria**:
- [ ] After each project activity, a timestamped entry is appended to the decision log section.
- [ ] The decision log section is human-editable and not overwritten by subsequent runs (append-only).

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: `/shark project` verb namespace
   - **Description**: `skills/shark/verbs/project.md` dispatches `/shark project bootstrap`, `/shark project brownfield-analysis`, and `/shark project product-design`. The namespace is a menu: activities are independent and do not assume each other ran. Invoking `/shark project` with no activity argument displays the available activities with a one-line description of each.
   - **Priority**: Must-Have

2. **REQ-F-002**: Rename `project-init` → `project bootstrap`; deprecation alias
   - **Description**: The architecture-doc-generation step is now invoked as `/shark project bootstrap`. `/shark project-init` continues to work as a deprecation alias throughout the window (prints a notice: "project-init is deprecated; use /shark project bootstrap"). There is exactly one `init` in the system: `shark admin init` owns DB/config/`docs/plan/` and is untouched.
   - **Priority**: Must-Have

3. **REQ-F-003**: `file_templates/progress.md` template
   - **Description**: A `file_templates/progress.md` template exists in the skill bundle. It contains: frontmatter with fields `track`, `stack_summary`, and `artifact_paths` (all optional convenience pointers, not enforced state); a `## Checklist` section (placeholder for derived content); and a `## Decision log` section (empty, timestamped append-only). When a project activity completes and `docs/product/progress.md` does not yet exist, the verb seeds it from this template.
   - **Priority**: Must-Have

4. **REQ-F-004**: Derived checklist
   - **Description**: The checklist section of the progress record is regenerated each time a project activity completes by inspecting disk artifacts. Minimum artifact set to check: `docs/architecture/` directory, `D01-vision-statement.md`, `D04-feasibility-report.md`. Each item renders as `[x]` (present), `[~]` (partial/directory exists but key files missing), or `[ ]` (absent), with a one-line note. The checklist is **advisory only**: no code path reads it back as authoritative state, and manual edits by a human are harmless.
   - **Priority**: Must-Have

5. **REQ-F-005**: Written decision log
   - **Description**: The decision log section is append-only and timestamped. When a project activity completes, the verb appends an entry in the form `- **YYYY-MM-DD** — <activity name>: <one-line summary of what was done or decided>`. Pre-existing entries are never modified or removed.
   - **Priority**: Must-Have

6. **REQ-F-006**: Progress record frontmatter
   - **Description**: The progress record's frontmatter carries `track`, `stack_summary`, and `artifact_paths` as lightweight convenience pointers. These are informational — the verb may update them on write, but they are not enforced state and their absence or staleness does not block any operation.
   - **Priority**: Should-Have

7. **REQ-F-007**: Update protocol — write is craft, not mutation
   - **Description**: Writing/updating `docs/product/progress.md` is a skill `Write` craft operation (the verb constructs the file content and writes it). It is not a shark `create` or `update` entity operation. There is no shark entity for the progress record; no database row is created.
   - **Priority**: Must-Have

### Non-Functional Requirements

1. **REQ-NF-001**: No schema impact
   - **Description**: No `projects` table, no `project_id` FK, no `CurrentSchemaVersion` bump, no migration. The project layer is skill-only with zero database footprint.

2. **REQ-NF-002**: `project` namespace membership rule
   - **Description**: Only pre-epic, one-time, human-driven activities that produce durable docs belong under `/shark project`. Recurring or queryable work (ops, deploys) is never added to this namespace (see E36-F03).

3. **REQ-NF-003**: Derived checklist is never authoritative
   - **Description**: No skill or Go code reads the checklist to determine project state. It is rendered for human display only. This ensures it can never drift into a shadow status-machine.

---

## Acceptance Criteria

**Scenario 1: Namespace dispatch**
- **Given** `verbs/project.md` is in the skill bundle
- **When** `/shark project bootstrap` is invoked
- **Then** the architecture-doc generation activity runs
- **And** `/shark project brownfield-analysis` and `/shark project product-design` each route to their respective activities independently

**Scenario 2: Deprecation alias**
- **Given** a user invokes `/shark project-init`
- **When** the verb is executed
- **Then** it behaves identically to `/shark project bootstrap` and prints: "project-init is deprecated; use /shark project bootstrap"
- **And** `shark admin init` is completely unaffected

**Scenario 3: Progress record seeding**
- **Given** `docs/product/progress.md` does not exist
- **When** a project activity (e.g., `bootstrap`) completes for the first time
- **Then** the file is created from the `file_templates/progress.md` template with the appropriate frontmatter placeholders, a checklist section, and an empty decision log section

**Scenario 4: Checklist derivation — partial state**
- **Given** `docs/architecture/` exists but `D01-vision-statement.md` and `D04-feasibility-report.md` are absent
- **When** the checklist is regenerated after an activity completes
- **Then** `docs/architecture/` shows `[x]`, `D01-vision-statement.md` shows `[ ]`, `D04-feasibility-report.md` shows `[ ]`

**Scenario 5: Decision log append**
- **Given** `docs/product/progress.md` exists with one prior decision log entry
- **When** a project activity completes
- **Then** a new timestamped entry is appended below the existing one
- **And** the prior entry is unchanged

**Scenario 6: Write is craft**
- **Given** any project activity completes
- **When** the progress record is updated
- **Then** no new shark entity is created in the database; the only change is the markdown file on disk

---

## Out of Scope

- **Any CLI `project` subcommand** — the layer is skill-only; `shark project` is not a Go command.
- **`shark project status` / `shark project advance`** — status-machine modeling of the project arc was rejected (see plan Appendix); the derived checklist is the advisory display.
- **Cross-project aggregation or a project registry** — one DB per project; fan-out is a later concern.
- **Viewer/API surfaces for a project entity** — no project entity exists.
- **Ops work in the project namespace** — recurring ops are E36-F03's convention (regular shark entities).

---

## Success Metrics

N/A — internal tooling; see epic scope.md exclusions.

---

*Last Updated*: 2026-06-29
