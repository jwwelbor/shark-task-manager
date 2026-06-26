# E19-F08 — Sprint Workflow YAML: first-class lifecycle with embedded agent routing

**Epic:** E19 — Sprint Management & Planning System
**Complexity:** STANDARD
**Status:** spec

Sprint is the only entity type without an embedded `workflow.yaml`. This feature
adds a route-based sprint workflow (`planning → active → closing → archived`) to
the embedded canonical data, wires each step to a `spawn_agent` action backed by
the existing sprint skills, and removes the hardcoded sprint status strings and
the single-active-sprint constraint from `sprint_service.go` in favor of
`workflow.Service` phase lookups.

---

## Requirements

### Functional

- **REQ-F-001** — Ship an embedded route-based sprint workflow at
  `internal/sharkdata/default_data/workflow/sprint.yaml` using the `steps:`
  schema (per E35), with `start: planning` and steps `planning`, `active`,
  `closing`, `archived`, `on_hold`.
- **REQ-F-002** — The `planning` step routes its `pass` outcome to `active` and
  spawns the sprint-planning skill; `active` routes `pass` to `closing` and
  spawns the sprint-execution skill; `closing` routes `pass` to `archived` and
  spawns the sprint-analytics skill. Each workable step also defines `blocked`
  (→ `on_hold`).
- **REQ-F-003** — `archived` is terminal (`terminal: true`, phase `done`);
  `on_hold` is a parking step (`parking: true`, phase `paused`).
- **REQ-F-004** — Provide minimal but functional prompt stubs at
  `prompts/sprint/planning.md`, `prompts/sprint/active.md`,
  `prompts/sprint/closing.md`, referenced by the corresponding steps'
  `prompt:` field.
- **REQ-F-005** — Replace every hardcoded sprint status/phase string in
  `sprint_service.go` with `workflow.Service` lookups (table below). No literal
  `"planning"` / `"active"` / `"execution"` status strings remain in sprint
  business logic.
- **REQ-F-006** — Remove the single-active-sprint constraint entirely (no flag,
  no config) — delete the active-sprint check in `StartSprint()`. Multiple
  simultaneously active sprints become valid (parallel workstreams).
- **REQ-F-007** — `GetNextTask()` must tolerate more than one active sprint:
  selection picks deterministically (first active sprint by current ordering)
  rather than erroring on multiple actives. Backward-compatible: a single active
  sprint behaves exactly as before.
- **REQ-F-008** — The embedded-workflow loader test
  (`cc040_embedded_workflow_test.go`) verifies `sprint.yaml` loads and its status
  set matches the embedded YAML, instead of skipping sprint.

### Mapping: hardcoded → workflow.Service (sprint_service.go)

`s.workflowSvc` is already scoped to the sprint level (constructed via
`ForLevel(LevelSprint)`); call the methods directly.

| Function | Line | Current | Replacement |
|---|---|---|---|
| `DeleteSprint()` | ~412 | `!= "planning"` | `!= s.workflowSvc.GetInitialStatusString()` (compare case-insensitively) |
| `assignableSprintPhases()` | ~860 | `[]string{"planning","execution"}` | **delete function** |
| `assignableSprintStatuses()` | ~869 | phase loop over the deleted helper | iterate over the fixed phase labels `"planning"`, `"execution"` inline, calling `s.workflowSvc.GetStatusesByPhase(phase)` for each |
| `GetSprintBacklog()` | ~1093 | `== "active"` | membership in `s.workflowSvc.GetStatusesByPhase("execution")` |
| `StartSprint()` | ~455–467 | single-active check (`ListSprints{Status:"active"}` …) | **delete the entire block** |
| `ReorderAssignment()` | ~1464 | `!= "planning" && != "active"` | `!s.sprintAcceptsAssignments(status)` (reuses `assignableSprintStatuses()`) |
| `CloseSprintWithCarryover()` | ~2055 | `closeSprintStatusPtr("planning")` filter | `s.workflowSvc.GetStatusesByPhase("planning")[0]` |
| `CloseSprintWithCarryover()` | ~2081 | `SprintStatus("planning")` target | `models.SprintStatus(s.workflowSvc.GetInitialStatusString())` |
| `GetNextTask()` | ~1294 | `Status:"active"` filter | first status of `s.workflowSvc.GetStatusesByPhase("execution")` |

> **Phase-label convention:** the route logic keys off the **phase** labels
> `planning` and `execution`, not the status names. `assignableSprintPhases()` is
> removed, but the two phase labels it returned are still the contract the
> default `sprint.yaml` honors (`planning` step → phase `planning`, `active` step
> → phase `execution`). Custom workflows may rename statuses but should preserve
> these phase labels.

### Acceptance Criteria

(verbatim from feature.md, with clarifications)

- `shark status advance <sprint-key>` from `planning` triggers the
  sprint-planning agent/skill.
- `shark status advance <sprint-key>` from `active` triggers the
  sprint-execution agent/skill.
- `shark status advance <sprint-key>` from `closing` triggers the
  sprint-analytics agent/skill.
- `make test` passes with no regressions.
- Embedded workflow test passes for the sprint level (`sprint.yaml` loads; its
  status set matches the embedded YAML).
- Clarification: existing sprint CLI/service behavior (assign, reorder, backlog,
  close-with-carryover, next) continues to work unchanged for the
  single-active-sprint case.

### Out of Scope

- Adding `shark sprint next` disambiguation UX for multiple active sprints
  beyond deterministic first-match selection (prompt-driven convention is the
  agreed first cut — no interactive picker, no `--sprint` flag in this feature).
- Net-new sprint skills or agent definitions — the three sprint skills already
  exist under `internal/sharkdata/default_data/skills/`.
- Migrating any live sprint `status` values (`planning`/`active` remain the
  shipped status names; the existing single-active sprint keeps working).
- Workflow validator changes — sprint.yaml satisfies the existing route-based
  validation rules (start defined, core outcomes present, targets defined).

---

## Architecture

### Component changes

1. **`internal/sharkdata/default_data/workflow/sprint.yaml`** (new) — embedded
   route-based sprint workflow. Picked up automatically by the existing
   embedded-workflow loader (Pass 3) and by `workflow.EmbeddedWorkflowFilename("sprint")`.
2. **`internal/sharkdata/default_data/prompts/sprint/planning.md`** (new) —
   prompt stub for the planning step.
3. **`internal/sharkdata/default_data/prompts/sprint/active.md`** (new) — prompt
   stub for the active step.
4. **`internal/sharkdata/default_data/prompts/sprint/closing.md`** (new) — prompt
   stub for the closing step.
5. **`internal/services/sprint_service.go`** — apply the mapping table:
   delete `assignableSprintPhases()` and the `StartSprint()` active-sprint
   block; replace the eight hardcoded status/phase strings with
   `workflow.Service` calls; make `GetNextTask()` multi-active tolerant.
6. **`internal/services/sprint_service_test.go`** — drop/replace any test
   asserting the single-active constraint error; add coverage for the
   workflow-driven status lookups and multi-active `GetNextTask()` selection,
   using the existing mock pattern (`MockWorkflowService`-style function fields /
   mocked sprint repo). No real DB.
7. **`internal/config/cc040_embedded_workflow_test.go`** — remove the
   "sprint omitted" skip path; the existing
   `TestDefaultWorkflowDataLoader_Pass3_AllEmbeddedEntitiesLoaded` loop now finds
   `workflow/sprint.yaml` and validates its status set. Optionally add an
   explicit assertion that the `sprint` slot contains `planning` and `archived`.

### sprint.yaml (complete, copy-ready)

```yaml
version: "1.0"
start: planning
steps:
  planning:
    phase: planning
    color: gray
    display_token: PLN
    description: "Sprint scope being defined; entities assigned"
    progress_weight: 0
    responsibility: agent
    is_planning: true
    action: spawn_agent
    agent: sprint-planning
    provider: anthropic
    model: sonnet
    skills: [sprint-planning]
    prompt: sprint/planning.md
    outcomes:
      pass: active
      blocked: on_hold
  active:
    phase: execution
    color: yellow
    display_token: ACT
    description: "Sprint in progress; entities being pulled and worked"
    progress_weight: 0.5
    responsibility: agent
    action: spawn_agent
    agent: sprint-execution
    provider: anthropic
    model: sonnet
    skills: [sprint-execution]
    prompt: sprint/active.md
    outcomes:
      pass: closing
      blocked: on_hold
  closing:
    phase: review
    color: magenta
    display_token: CLS
    description: "Sprint wrapping up; retrospective and carryover review"
    progress_weight: 0.9
    responsibility: agent
    action: spawn_agent
    agent: sprint-analytics
    provider: anthropic
    model: sonnet
    skills: [sprint-analytics]
    prompt: sprint/closing.md
    outcomes:
      pass: archived
      blocked: on_hold
  archived:
    phase: done
    color: green
    display_token: ARC
    description: "Sprint closed and archived"
    progress_weight: 1.0
    responsibility: none
    action: archive
    terminal: true
  on_hold:
    phase: paused
    color: gray
    display_token: HLD
    description: "Sprint intentionally paused"
    progress_weight: 0
    responsibility: human
    action: pause
    parking: true
```

> **Naming note:** the step blocks reference `agent: sprint-planning` /
> `sprint-execution` / `sprint-analytics`, which are **skill** names (the skills
> exist under `default_data/skills/`); there are no matching files under
> `default_data/agents/`. This matches how the bug/task workflows reference
> agents that resolve through the bundle, and `skills:` lists the same name.
> Resolution is the harness/dispatch concern, not this feature's — but the
> implementer should confirm the dispatcher resolves a `spawn_agent` step whose
> `agent` names a skill (vs. an agent file) and adjust the `agent:` value to an
> existing agent file (e.g. `tech-director` or `product-manager`) if resolution
> requires an agent definition. The `skills:` and `prompt:` fields are the
> load-bearing routing; keep them as specified.

### Prompt stubs (minimal but functional)

These follow the existing `{{template "advance_preamble" .}}` convention (see
`prompts/bug/draft.md`) and `{{include: skills/...}}` convention (see
`prompts/task/development.md`). The sprint skills carry the real procedure;
stubs orient the agent and include the relevant skill.

**`prompts/sprint/planning.md`**
```markdown
{{template "advance_preamble" .}}

Plan sprint {{.id}}: "{{.title}}".

{{include: skills/sprint-planning/SKILL.md}}

Read the sprint plan and readiness data (`shark sprint plan {{.id}} --json`),
propose entity assignments, and confirm scope. Do NOT start the sprint — starting
is an explicit user action. When scope is agreed, release outcome `pass` to move
the sprint to active.
```

**`prompts/sprint/active.md`**
```markdown
{{template "advance_preamble" .}}

Drive active sprint {{.id}}: "{{.title}}" to completion.

{{include: skills/sprint-execution/SKILL.md}}

Pull entities via `shark sprint next` and dispatch each per the execution skill
until the sprint backlog is drained or no further progress is possible. When the
sprint is ready to wrap up, release outcome `pass` to move it to closing.
```

**`prompts/sprint/closing.md`**
```markdown
{{template "advance_preamble" .}}

Close out sprint {{.id}}: "{{.title}}".

{{include: skills/sprint-analytics/SKILL.md}}

Read `shark sprint summary {{.id}} --detailed` and velocity data, synthesize the
retrospective, and review carryover. Archiving is gated on explicit user
confirmation — release outcome `pass` only after the retro is confirmed.
```

### Integration points (exact signatures to call)

`s.workflowSvc` is the sprint-level `*workflow.Service`. Available methods:

```go
func (s *Service) GetStatusesByPhase(phase string) []string
func (s *Service) GetInitialStatusString() string
func (s *Service) GetTerminalStatuses() []string
func (s *Service) ValidateTransition(fromStatus, toStatus string) error
func (s *Service) IsValidTransition(currentStatus, targetStatus string) bool
func (s *Service) GetStatusMetadata(status string) StatusInfo
func (s *Service) IsTerminalStatus(status string) bool
```

Guidance for each call site:
- Compare status strings case-insensitively (`strings.EqualFold`), matching the
  existing `sprintAcceptsAssignments` convention.
- `GetStatusesByPhase("execution")` and `GetStatusesByPhase("planning")` return
  slices; guard against empty before indexing `[0]` (fall back to the literal
  status only if the slice is empty, or return a descriptive error — prefer the
  latter so a misconfigured workflow fails loudly).
- `assignableSprintStatuses()` keeps its dedup+order behavior; it now inlines the
  two phase labels rather than calling the deleted `assignableSprintPhases()`.

### Key technical decisions

- **Phase-keyed, not status-keyed.** Routing reads phases (`planning`,
  `execution`, `review`, `done`, `paused`) so custom workflows can rename
  statuses. The default `sprint.yaml` preserves these phase labels. (D: avoids
  re-hardcoding new status names.)
- **Delete the single-active constraint outright.** Per the agreed design, no
  flag or config — multiple active sprints are valid parallel workstreams.
  `StartSprint()` keeps only the `ValidateTransition` check.
- **`GetNextTask()` deterministic first-match.** With the constraint gone,
  multiple actives are possible; pick the first active sprint deterministically
  to preserve existing single-active behavior. Disambiguation UX is out of
  scope.
- **Stubs include skills, don't duplicate them.** The three sprint skills are the
  source of truth; prompts are thin orientation + `{{include:}}` to stay
  consistent with task/bug prompts and avoid drift.
- **Edit embedded canonical, not the deployed copy.** Per project memory, all
  new files go under `internal/sharkdata/default_data/`; `shark-data/` is a
  deployed copy that gets overwritten.

### Task breakdown (ordered)

1. **Create `sprint.yaml`** under `internal/sharkdata/default_data/workflow/`
   with the content above. Verify it loads via the embedded loader.
2. **Create the three prompt stubs** under
   `internal/sharkdata/default_data/prompts/sprint/`.
3. **Refactor `sprint_service.go`** — apply the full mapping table: delete
   `assignableSprintPhases()` and the `StartSprint()` active-sprint block; swap
   the eight hardcoded strings for `workflow.Service` calls; add empty-slice
   guards.
4. **Make `GetNextTask()` multi-active tolerant** — replace the
   single-active-required logic with deterministic first-active selection.
5. **Update `sprint_service_test.go`** — remove the single-active-constraint
   assertion, add workflow-lookup and multi-active selection coverage (mocks
   only, no DB).
6. **Update `cc040_embedded_workflow_test.go`** — drop the sprint skip path;
   assert the sprint slot loads and matches the embedded YAML. Run
   `make fmt && make lint && make test`.
