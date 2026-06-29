# Requirements

**Epic**: [Project Layer and Consult Bridge](./epic.md)
**Design source**: [`plan.md`](../../../dev-artifacts/2026-06-29-project-entity-design/plan.md)

This is a concise catalog. The plan holds the full rationale; requirements here
trace to its numbered sections (§1–§5).

---

## Guiding principles (from plan §Principles)

- **P1 — Skill-layer.** Project orchestration is markdown verbs; the CLI is a
  tool the skill calls. Exactly one Go change in this epic.
- **P2 — One DB per project.** Tenancy is the database boundary; shark models a
  single project per store. No `project` table, no `project_id`.
- **P3 — Progress is derived and advisory**, never an authoritative second copy
  of state. What can be computed from disk is computed; only the irreducible
  human narrative is written.
- **P4 — The `project` namespace is for one-time, pre-epic, human-driven setup
  that produces durable docs.** Anything recurring or queryable is a regular
  shark entity instead.

---

## Functional requirements

### Consult bridge (Slice 1 → E36-F01)

- **REQ-F1** — `agent list` and `skill list` print a `description` per entry,
  sourced from each entry's frontmatter `description:`. (plan §5)
- **REQ-F2** — Description population reuses the `Get` path's frontmatter parser,
  respects override > disk > embedded precedence, and is best-effort: a parse
  failure must never fail the list. (plan §5)
- **REQ-F3** — Description renders in both text and JSON list output. (plan §5)
- **REQ-F4** — `/shark consult <agent> [referent]` resolves the agent (exact
  name, else fuzzy-match against `agent list` names **and** descriptions; if
  omitted, show the menu), loads the persona via `shark agent get <agent>`
  (frontmatter stripped), and adopts it **inline** (turn-by-turn, not a
  background subagent). (plan §4)
- **REQ-F5** — A consult is **read-only by default**: it reads shark/docs for
  context as the persona would but does not mutate shark state unless explicitly
  asked. (plan §4)
- **REQ-F6** — Referent resolution: an explicit path is read; "this" with an
  obvious in-context artifact is used and named; ambiguous input is asked about
  or accepts pasted content. Never guess silently. (plan §4)
- **REQ-F7** — `query.md` gains a consult-intent recognizer: match
  *ask/have/consult `<agent>` to/about `<task>`*, extract agent + task, hand off
  to `consult.md`. Fire only on clear "talk to an agent" phrasing; fall back to
  the explicit form if the agent doesn't resolve. (plan §4)
- **REQ-F8** — `consult.md` degrades gracefully when the agent is absent (clear
  message + `agent list`); never hard-fail. (plan §4)

### Project namespace + progress record (Slice 2 → E36-F02)

- **REQ-F9** — `/shark project <activity>` namespace dispatches to `bootstrap`,
  `brownfield-analysis`, and `product-design`. It is a **menu, not a sequence**:
  activities are independent and run on demand. (plan §1)
- **REQ-F10** — `bootstrap` (architecture-doc generation, brownfield/greenfield
  detection) is the renamed `project-init`; `/shark project-init` continues to
  work as a **deprecation alias**. There remains exactly one `init`
  (`shark admin init` keeps DB/config/`docs/plan/`). (plan §1)
- **REQ-F11** — A progress record (seeded from a `file_templates/progress.md`
  template, conventionally `docs/product/progress.md`) has two parts:
  - a **derived checklist** regenerated from artifacts on disk (e.g.
    `docs/architecture/`, `D01-vision-statement.md`, `D04-feasibility-report.md`)
    rendered as `[x] / [~] / [ ]` with a one-line note; advisory only.
  - a **written decision log**: append-only, timestamped human entries that
    can't be derived. (plan §2)
- **REQ-F12** — Frontmatter on the progress record carries lightweight pointers
  (track, stack summary, artifact paths) as convenience, not enforced state.
  (plan §2)
- **REQ-F13** — Update protocol: when a `project` activity completes, the verb
  appends a decision-log entry and regenerates the checklist from disk. Writing
  the file is craft (a `Write`), not a shark mutation. (plan §2)

### Ops-as-entities convention (Slice 3 → E36-F03)

- **REQ-F14** — Recurring operational work (deploy, devops) is **not** a project
  activity and does not appear on the checklist. It is modeled as regular shark
  entities (tasks or change-cards, optionally under an "Ops" epic) so it keeps
  history, status, and queryability. Documented as convention; no new mechanism.
  (plan §3)

---

## Non-functional requirements

- **REQ-NF1 — No schema impact.** No `projects` table, no `project_id` FK, no
  migration, no `CurrentSchemaVersion` bump. (plan §Principles P2, Appendix)
- **REQ-NF2 — Minimal Go footprint.** Exactly one Go change: add
  `Description string` to `BundleContentEntry` and populate/render it. Quality
  gate (`make fmt && make lint && make test`) must pass. (plan §5)
- **REQ-NF3 — Backward compatibility.** `project-init` alias keeps working
  through the deprecation window; existing `agent get`/`spawn_agent` dispatch is
  unchanged. Agents stay canonical in the embed
  (`internal/sharkdata/default_data/agents/`); per-project tuning via
  `shark-data/overrides/agents/<name>.md` still works. (plan §1, §4)
- **REQ-NF4 — No source-of-truth duplication.** Both `consult` and
  `spawn_agent` resolve personas through `shark agent get`; the derived checklist
  is never read back as authoritative state. (plan §3, §4)
- **REQ-NF5 — Graceful degradation.** Missing bundle content (`shark agent get`
  / `shark skill get` failure) prints an "unavailable / coming soon" message and
  never hard-fails. (plan §4, SKILL.md router contract)

---

## Open questions

None blocking — the plan is a locked design with a rejected-alternatives
appendix. Implementation-detail decisions (exact checklist artifact list, NL
recognizer phrasing breadth) are deferred to each feature's design phase.
