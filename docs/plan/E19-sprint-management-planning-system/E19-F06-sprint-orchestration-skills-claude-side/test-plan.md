# Test Plan: E19-F06 — Sprint Orchestration Skills (Claude-side)

**Created:** 2026-05-10
**QA Agent:** QA
**Feature Spec:** `docs/plan/E19-sprint-management-planning-system/E19-F06-sprint-orchestration-skills-claude-side/spec.md`
**Epic UAT Plan:** `docs/plan/E19-sprint-management-planning-system/uat-acceptance-plan.md`
**Status:** APPROVED

---

## Spec Drift Analysis

### Drift Findings

No significant drift detected. One sequencing constraint documented in the spec is explicitly flagged here:

1. **F03 field name TBD (REQ-F-023)**: The field that `shark get {KEY} --json` uses to expose sprint assignment is not yet finalized (depends on E19-F03 delivery). T-E19-F06-005 must check the actual F03 output before editing `run.md`. The test plan marks TC-X02 as blocked until F03 ships and the field name is confirmed. No other test cases are affected.

2. **Mode default**: The spec says `--mode` defaults to `interactive`. Auto-mode is intended for orchestrator use. Test cases cover both modes; interactive is the primary path.

### Traceability Matrix

| Feature AC / Epic Requirement | Spec AC | Test Cases | Notes |
|---|---|---|---|
| REQ-F-019: `/plan-sprint` registration | AC-1 | TC-P01 | File existence + frontmatter check |
| REQ-F-019: Refuses non-sprint keys | AC-2 | TC-P02 | Exact message verified |
| REQ-F-019: Advisory for non-planning status | AC-3 | TC-P03 | Asks user; does not block |
| REQ-F-019: Reads plan once at start | AC-4 | TC-P04 | Single shark call (review) |
| REQ-F-019: interactive mode — confirm per item | AC-5a | TC-P05 | No `shark sprint add` without confirmation |
| REQ-F-019: auto mode — greedy fill + one confirm | AC-5b | TC-P06 | Single confirmation gate |
| REQ-F-019: Readiness score delta on exit | AC-6 | TC-P07 | Delta reported |
| REQ-F-019: Does NOT call `shark sprint start` | AC-7 | TC-P08 | Forbidden call check (review) |
| REQ-F-019: All shark calls use `--json` | AC-8 | TC-P09 | Code review check |
| REQ-F-020: `/run-sprint` registration | AC-1 | TC-R01 | File existence check |
| REQ-F-020: Refuses non-sprint keys | AC-2 | TC-R02 | Same message pattern |
| REQ-F-020: Planning status — ask to start | AC-3 | TC-R03 | User confirmation gated |
| REQ-F-020: terminal status — exit with notice | AC-4 | TC-R04 | No silent no-op |
| REQ-F-020: Pull-loop body | AC-5 | TC-R05 | Loop: next → /run → repeat |
| REQ-F-020: `--agent` flag passed through | AC-6 | TC-R06 | Pass-through verified |
| REQ-F-020: `--max-iterations` cap | AC-7 | TC-R07 | Exits at cap with notice |
| REQ-F-020: Post-loop burndown + summary | AC-8 | TC-R08 | Both commands printed |
| REQ-F-020: Close requires user confirmation | AC-9 | TC-R09 | Never auto-closes |
| REQ-F-020: `--carryover` flag pass-through | AC-10 | TC-R10 | Value passed to shark sprint close |
| REQ-F-020: Delegates to `/run`, not own dispatch | AC-11 | TC-R11 | review: no inline dispatch |
| REQ-F-021: `/run-sprint-team` registration | AC-1 | TC-T01 | File existence check |
| REQ-F-021: Refuses non-sprint keys | AC-2 | TC-T02 | Same message pattern |
| REQ-F-021: Inherits /run-agent-team preconditions | AC-3 | TC-T03 | Precondition block present |
| REQ-F-021: Groups entities by feature key | AC-4 | TC-T04 | E##-F## extraction logic |
| REQ-F-021: Serial per-feature dispatch | AC-5 | TC-T05 | One `/run-agent-team` at a time |
| REQ-F-021: `--size` flag pass-through | AC-6 | TC-T06 | Passed to each /run-agent-team |
| REQ-F-021: `--features` filter | AC-7 | TC-T07 | Restricts to listed features |
| REQ-F-021: Standalones fall back to `/run` | AC-8 | TC-T08 | B###/CC-###/TD-### via /run |
| REQ-F-021: Burndown between feature groups | AC-9 | TC-T09 | shark sprint burndown called |
| REQ-F-021: Post-loop close prompt | AC-10 | TC-T10 | Same as /run-sprint gate |
| REQ-F-022: `/retro-sprint` registration | AC-1 | TC-A01 | File existence + frontmatter |
| REQ-F-022: Refuses non-sprint keys | AC-2 | TC-A02 | Exact message check |
| REQ-F-022: Refuses non-closed sprints | AC-3 | TC-A03 | Error message exact text |
| REQ-F-022: Data pull sequence | AC-4 | TC-A04 | summary → velocity → notes |
| REQ-F-022: Markdown report — 5 sections | AC-5 | TC-A05..TC-A09 | One TC per section |
| REQ-F-022: Writes to docs/sprints/{S###}-retro.md | AC-6 | TC-A10 | File path and creation |
| REQ-F-022: Prompts before overwrite | AC-7 | TC-A11 | Existing file gate |
| REQ-F-022: `--no-write` prints to stdout | AC-8 | TC-A12 | File not written |
| REQ-F-022: All shark calls use `--json` | AC-9 | TC-A13 | Code review check |
| REQ-F-023: `/run` advisory — field check | AC-1 | TC-X01 | Step 1 amendment present |
| REQ-F-023: `/run` advisory — message exact text | AC-2 | TC-X02 | Blocked until F03 field confirmed |
| REQ-F-023: `/run` advisory — continues execution | AC-3 | TC-X03 | No abort |
| REQ-F-023: No notice when no sprint | AC-4 | TC-X04 | Silent pass-through |
| REQ-F-024: `run-agent-team.md` cross-ref (command) | AC-1 | TC-D01 | One-line See-also present |
| REQ-F-024: `run-agent-team.md` cross-ref (skill) | AC-2 | TC-D02 | See-also in Usage section |
| REQ-F-024: No behavior change to /run-agent-team | AC-3 | TC-D03 | Only text added |
| REQ-F-025: Three new SKILL.md files | AC-1 | TC-M01 | File presence + frontmatter |
| REQ-F-025: SKILL.md style matches orchestration | AC-2 | TC-M02 | "What this is" paragraph |
| REQ-F-025: PIPELINE.md table row | AC-3 | TC-M03 | Row in SDLC table |
| REQ-F-026: product-manager.md subsection | AC-1 | TC-PM01 | Subsection under PRIMARY |
| REQ-F-026: No other agents changed | AC-2 | TC-PM02 | git diff scope check |
| REQ-NF-007: Idempotency — no duplicate adds | N/A | TC-NF01 | Second invocation invariant |
| REQ-NF-007: Idempotency — no double-close | N/A | TC-NF02 | Close-after-close check |
| REQ-NF-007: Idempotency — retro overwrite prompt | N/A | TC-NF03 | Prompt on existing retro file |
| REQ-NF-008: All mutating calls require confirmation | N/A | TC-NF04 | Enumeration of mutation gates |
| REQ-NF-009: All shark calls use `--json` | N/A | TC-NF05 | Full grep across all new files |
| REQ-NF-010: Zero shark-repo changes | N/A | TC-NF06 | git diff scope check |

---

## Acceptance Criteria Review

### Ambiguity Findings

**REQ-F-023 field name (TBD)**: The spec explicitly flags this as the one open item. Test case TC-X02 requires knowing what field `shark get {KEY} --json` returns for sprint assignment. The test plan marks TC-X02 as `BLOCKED_UNTIL_F03` and specifies that the developer must verify the actual JSON field name from a live F03-enabled shark build before implementing T-E19-F06-005.

**"session" scope for single shark plan read (AC-4 of REQ-F-019)**: The AC says "never re-reads in the same turn." Claude does not have turn-level introspection, so this AC is enforced by **code review of the workflow file**: the plan-sprint workflow must store the initial plan output in a variable and never call `shark sprint plan` again in the same workflow body unless the user has run an add/remove action (which would be triggered by user input, starting a new turn). TC-P04 is a review-only test case.

**"3–5 data-driven recommendations" (REQ-F-022 AC-5)**: The spec lists 5 pattern-match rules; the skill must pick the top 3–5 by signal strength. TC-A09 verifies (a) Recommendations section is non-empty, (b) the items reference actual data from the fixture (velocity figure, cycle-time, or carryover count), not generic placeholder text. Exact wording is not asserted — the observable is "contains a quantitative reference from the sprint data."

### Missing Coverage — None

All numbered ACs across REQ-F-019 through REQ-F-026 and all four non-functional requirements have at least one test case.

---

## ISTQB Technique Application (per AC)

This feature is Claude-side documentation/skill-prompt code, not Go code. The applicable ISTQB techniques map to verification methods rather than automated unit tests.

| AC / Requirement | Technique | Test Cases Generated | Rationale |
|---|---|---|---|
| REQ-F-019 AC-2: Refuses non-sprint keys | BVA + Equivalence Partitioning | TC-P02 (invalid: E07, E07-F01, S, random text; valid: S001, S999) | Input space: S### pattern. BVA at boundary (S = missing digits; S0001 = too many). EP: "non-sprint-looking key" is one class. |
| REQ-F-019 AC-3: Non-planning advisory | State Transition | TC-P03a (active), TC-P03b (completed), TC-P03c (planning — no advisory) | Sprint status is a state machine; AC behavior depends on status value |
| REQ-F-019 AC-5: Mode selection | Decision Table | TC-P05 (interactive, no confirm → no add), TC-P06 (auto, confirm → adds) | Conditions: mode (interactive/auto) × user-confirms (Y/N) → 4 cells |
| REQ-F-019 AC-7: Does NOT call sprint start | Attack-class enumeration | TC-P08 | Attack: does any branch in the workflow body contain `shark sprint start`? |
| REQ-F-020 AC-4: terminal status exit | Equivalence Partitioning | TC-R04a (completed), TC-R04b (archived), TC-R04c (cancelled) | Three terminal-status equivalence classes; each must produce the exit notice |
| REQ-F-020 AC-7: max-iterations cap | BVA | TC-R07a (cap reached = N), TC-R07b (loop exits before cap = N-1) | Boundary at the cap value: exactly at cap fires notice; one below exits normally |
| REQ-F-020 AC-9: Close requires confirmation | Attack-class enumeration | TC-R09 | Attack: does `shark sprint close` ever appear on a code path that does not pass through a user-confirmation gate? |
| REQ-F-021 AC-4: Group by feature key | Equivalence Partitioning | TC-T04 (task E07-F01-001 → E07-F01), TC-T08 (bug B003 → standalone), TC-T08b (CC-001 → standalone) | Three input classes: task (has feature parent), non-task with no parent, non-task with parent |
| REQ-F-021 AC-5: Serial dispatch | Attack-class enumeration | TC-T05 | Attack: does the workflow ever attempt to invoke two `/run-agent-team` calls without waiting for the first to return? |
| REQ-F-021 AC-7: `--features` filter | Decision Table | TC-T07 (feature in filter — dispatched), TC-T07b (feature not in filter — skipped), TC-T07c (filter empty — all dispatched) | Conditions: feature in filter list (Y/N) × filter provided (Y/N) |
| REQ-F-022 AC-3: Refuses non-closed sprints | State Transition | TC-A03a (planning), TC-A03b (active) | Non-closed states that must be rejected; completed/archived must pass |
| REQ-F-022 AC-5: Five report sections | Contract surface enumeration | TC-A05..TC-A09 | Five section headers are a fixed contract surface; each must exist and be non-empty |
| REQ-F-022 AC-7: Prompts before overwrite | State Transition | TC-A11 (file exists → prompt), TC-A11b (file absent → no prompt) | Pre-condition: file present/absent |
| REQ-F-023 AC-3: /run continues after advisory | State Transition | TC-X03 | State: advisory printed → execution continues (no abort state reachable from advisory) |
| REQ-NF-007: Idempotency | Attack-class enumeration | TC-NF01..TC-NF03 | Attack class: "second invocation" — what side effects does it produce? |
| REQ-NF-008: Mutation gates | Contract surface enumeration | TC-NF04 | Enumerate all shark sprint commands that mutate status or assignment; each must have a preceding confirmation prompt |
| REQ-NF-009: JSON-only shark consumption | Attack-class enumeration | TC-NF05 | Attack: does any skill file invoke a shark command without `--json` or `--field`? |

---

## ISO 25010 Coverage Matrix

| AC | Functional | Reliability | Usability | Security | Maintainability |
|---|---|---|---|---|---|
| REQ-F-019 (plan-sprint skill) | TC-P01..P09 | TC-NF01 (idempotency) | TC-P02 (error msg clarity), TC-P03 (advisory not abort) | TC-NF05 (no text scraping) | TC-NF10 (style mirrors orchestration) |
| REQ-F-020 (run-sprint skill) | TC-R01..R11 | TC-NF02 (no double-close), TC-R07 (max-iterations) | TC-R04 (exit notice), TC-R09 (confirm gate) | TC-NF04 (all mutating calls gated) | TC-R11 (delegates to /run, not own dispatch) |
| REQ-F-021 (run-sprint-team skill) | TC-T01..T10 | TC-T05 (serial dispatch — no concurrent teams) | TC-T08 (standalones fall back gracefully) | TC-NF04 (shared) | TC-T03 (preconditions reuse, not duplicate) |
| REQ-F-022 (retro-sprint skill) | TC-A01..A13 | TC-NF03 (overwrite prompt), TC-A04 (data pull order) | TC-A03 (clear refusal msg), TC-A09 (data-driven recs) | TC-NF05 (shared) | TC-A05..A09 (fixed 5-section layout) |
| REQ-F-023 (/run advisory) | TC-X01..X04 | TC-X03 (continues after advisory) | TC-X02 (one-line advisory exact text) | N/A | TC-X04 (silent when no sprint) |
| REQ-F-024 (cross-ref docs) | TC-D01..D03 | N/A | TC-D01..D02 (discoverability) | N/A | TC-D03 (no behavior change) |
| REQ-F-025 (registration) | TC-M01..M03 | N/A | TC-M02 (SKILL.md style) | N/A | TC-M03 (PIPELINE.md single row) |
| REQ-F-026 (PM agent) | TC-PM01..PM02 | N/A | TC-PM01 (PM knows to use sprint commands) | N/A | TC-PM02 (no other agents changed) |
| REQ-NF-007..010 (NFRs) | TC-NF01..NF06 | TC-NF01..03 | TC-NF04 | TC-NF05 | TC-NF06 |

### Coverage Gaps

- **Performance (no explicit NFR)**: These are skill files executed interpretively by Claude Code; performance is not a measurable property of the skill text. No performance test cases are required.
- **Accessibility**: N/A for CLI skill files.
- **Portability**: The skills run in any Claude Code environment that has the shark CLI on PATH. Path-portability is covered by `~/.claude/` relative paths (no hardcoded machine paths). Verified by code review only; no dedicated TC.

---

## Observability Design

This feature is Claude-side skill prompt code. There are no metrics, traces, or log lines produced by the skill files themselves — observability for the underlying shark CLI operations is owned by the CLI (E19-F01 through F05). The observable behaviors for this feature are:

| Behavior | Observable | Verification Method |
|---|---|---|
| Skill file registered and resolves | `/plan-sprint --help` (or equivalent) renders usage | TC-P01 (manual invocation) |
| Non-sprint key refusal | stderr / output contains refusal message | TC-P02, TC-R02, TC-T02, TC-A02 |
| User confirmation gate triggered | Prompt appears before any `shark sprint add/start/close` | TC-NF04 (code review + manual) |
| Advisory in `/run` fires for sprint-assigned entity | One-line notice printed to stdout before entity dispatch | TC-X02 (manual) |
| Retro file written | `docs/sprints/{S###}-retro.md` exists and has 5 sections | TC-A10 (file assertion) |
| Overwrite prompt | CLI asks before overwriting existing retro file | TC-A11 (manual) |

No structured logging or metrics instrumentation is required for Claude-side skill files.

---

## Caller-Path Contracts (per test case)

Because this feature is documentation/skill-prompt code (not Go code), "production caller" is the Claude Code skill execution engine. The entrypoint is the slash command `/plan-sprint S###`; the "function signature" is the command invocation. The lowest allowed mock seam is the shark CLI itself (replaced by a fixture sprint database during manual testing).

| TC | Production Entrypoint | Lowest Allowed Mock Seam | Forbidden Mocks | Counter-factual |
|---|---|---|---|---|
| TC-P02 | `/plan-sprint E07` (non-sprint key) | None — this is a text-match refusal in the workflow preamble | Do NOT test only valid keys; the refusal branch must be exercised | A buggy impl that skips key validation would call `shark sprint plan E07` and get a CLI error instead of the specified refusal message |
| TC-P05 | `/plan-sprint S001 --mode=interactive` (interactive, user says "no" to item) | Real `shark sprint plan S001 --json` against fixture sprint (planning status, 3 entities in backlog) | Do NOT mock the shark sprint plan call — the skill must parse real JSON output to surface the backlog items | A buggy impl that calls `shark sprint add` without waiting for confirmation would add items that the user declined |
| TC-P06 | `/plan-sprint S001 --mode=auto` | Real shark against fixture sprint | Do NOT mock the greedy-fill logic — the skill must iterate over the backlog JSON and select items; mock would bypass this | A buggy auto-mode that emits one `shark sprint add` per entity without a single final confirmation violates REQ-NF-008 |
| TC-P08 | `/plan-sprint S001 --mode=auto` (user confirms) | Real shark fixture | N/A — review-only | A buggy impl that calls `shark sprint start S001` after the user confirms the plan would violate the "does NOT call start" AC, silently starting the sprint |
| TC-R05 | `/run-sprint S001` (sprint active, 2 entities assigned) | Real `shark sprint next --json` against fixture sprint; real `/run` invocation (or stub fixture that completes immediately) | Do NOT mock the loop exit condition — `shark sprint next` returning empty JSON must cause the loop to exit (not error) | A buggy impl that never calls `/run` on the returned entity would stall indefinitely and never advance the sprint |
| TC-R07 | `/run-sprint S001 --max-iterations=2` (sprint with 3+ entities) | Real shark + fixture sprint with 3 entities | Do NOT mock the iteration counter — the skill's own loop counter must fire the cap; mocking would hide the missing-counter bug | A buggy impl with no counter would loop past N=2 and dispatch 3+ entities, defeating the cost-runaway protection |
| TC-R09 | `/run-sprint S001` (loop exits; user is asked whether to close) | Real shark + fixture | Do NOT stub the user prompt — it must be an actual interactive question (`"Close sprint S001 now? (yes/no)"`) | A buggy impl that calls `shark sprint close` before the user answers would close the sprint without confirmation |
| TC-T04 | `/run-sprint-team S001` (backlog: E07-F01-003, B003, CC-001) | Real `shark sprint backlog S001 --json` against fixture | Do NOT mock the entity-key parsing — the skill must extract `E##-F##` from `E##-F##-###` keys and place B###/CC-### in the standalone group | A buggy impl that groups B003 into a feature group would attempt `/run-agent-team B003`, which `/run-agent-team` rejects, crashing the loop |
| TC-T05 | `/run-sprint-team S001` (2 features in sprint) | Real shark + fixture with 2 feature groups | Do NOT simulate parallel dispatch — the skill must invoke `/run-agent-team E07-F01` and await completion before invoking `/run-agent-team E07-F02` | A buggy impl that spawns both teams concurrently would violate the one-team-per-session constraint, producing undefined behavior from the agent-teams primitive |
| TC-A03 | `/retro-sprint S001` (sprint status = `active`) | Real shark fixture with `active` sprint | Do NOT mock the status check — the skill must call `shark get S001 --json --field=status` and parse the result | A buggy impl that skips the status check would proceed to pull `--detailed` summary on an active sprint, which may produce incomplete data silently |
| TC-A05..TC-A09 | `/retro-sprint S004 --no-write` (sprint completed) | Real shark fixture: completed sprint with detailed summary, velocity history, 2 carryover entities with rejection notes | Do NOT mock `shark sprint summary --detailed --json` — the skill must parse real JSON output to populate each section | A buggy impl that produces a retro with the Velocity Context section empty (no velocity call made) would fail TC-A06 even if the file exists |
| TC-X02 | `/run E07-F01-001` (task assigned to active sprint S005) | Real shark fixture with task assigned to sprint | Blocked pending F03 field name — do NOT assume field name; developer must verify from live shark get output | A buggy impl that checks the wrong field name would silently skip the advisory for all sprint-assigned entities |
| TC-NF01 | `/plan-sprint S001` invoked twice in succession against same sprint at same lifecycle stage | Real shark fixture (planning, no entities added yet) | Do NOT mock the add call — the idempotency test requires the second invocation to NOT call `shark sprint add` again for items already confirmed in the first invocation | A buggy impl that re-displays the entire backlog on re-entry and asks for confirmation again would pass TC-P05 but fail TC-NF01 (user must re-confirm every item) |
| TC-NF04 | All four skills: enumerate every `shark sprint add`, `shark sprint start`, `shark sprint close`, `shark sprint remove` call site | Code review of all skill workflow files | N/A — this is a static review | A buggy impl where `shark sprint start` appears without a preceding user prompt would violate REQ-NF-008 |
| TC-NF05 | All four skill workflow files | grep `shark sprint` calls in skill files | Do NOT accept `--json` omission for human-readable-only commands — even `shark sprint burndown` is checked | A buggy impl that calls `shark sprint burndown S001` without `--json` violates REQ-NF-009 (even if the output is only displayed, not parsed) |

---

## Acceptance Test Cases

### Skill: `/plan-sprint` (REQ-F-019)

#### TC-P01 — Slash command registration
- **AC**: REQ-F-019 AC-1
- **Method**: File existence + content review
- **Setup**: Post-implementation file tree
- **Steps**:
  1. Verify `~/.claude/commands/plan-sprint.md` exists
  2. Verify it has valid YAML frontmatter with `description`
  3. Verify it references `sprint-planning/workflows/plan-sprint.md`
  4. Verify `~/.claude/skills/sprint-planning/SKILL.md` exists
  5. Verify `~/.claude/skills/sprint-planning/workflows/plan-sprint.md` exists
- **Expected**: All five paths exist; frontmatter valid; cross-reference correct
- **Edge cases**: Missing frontmatter closes the slash command entirely

#### TC-P02 — Refuses non-sprint keys
- **AC**: REQ-F-019 AC-2
- **Method**: Manual invocation
- **Test inputs**: `E07`, `E07-F01`, `E07-F01-001`, `S` (no digits), `S0001` (too many digits), `hello`
- **Setup**: Any shark project
- **Steps**:
  1. Invoke `/plan-sprint {each bad key}`
  2. Record output
- **Expected**: Output contains `"/plan-sprint only operates on sprints. Got: {KEY}"` for every invalid input; no shark commands are called
- **Edge cases**: `S123` should be accepted (valid); `s001` should be accepted (case-insensitive per shark conventions)

#### TC-P03 — Advisory for non-planning sprint status
- **AC**: REQ-F-019 AC-3
- **Method**: Manual, three sub-cases
- **Setup**: Fixture sprints: S010 (active), S011 (completed), S012 (planning)
- **Steps**:
  - TC-P03a: `/plan-sprint S010` — expect advisory about `active` status; user asked whether to continue
  - TC-P03b: `/plan-sprint S011` — expect advisory about `completed` status; user asked whether to continue
  - TC-P03c: `/plan-sprint S012` — no advisory; proceeds directly to plan view
- **Expected**: Only planning-status sprint proceeds silently; non-planning prints advisory and prompts
- **Edge cases**: User declines on TC-P03a → skill exits cleanly (no shark calls)

#### TC-P04 — Single plan read per turn (review-only)
- **AC**: REQ-F-019 AC-4
- **Method**: Code review of `plan-sprint.md` workflow body
- **Steps**:
  1. Read workflow file
  2. Count occurrences of `shark sprint plan {S###}` call
  3. Verify the result is stored in a named variable
  4. Verify no second call to `shark sprint plan` in the same workflow body (only `shark sprint readiness` in Step 4 is a second call, which is permitted)
- **Expected**: Exactly one `shark sprint plan` call at the start of the workflow

#### TC-P05 — Interactive mode: no add without explicit confirmation
- **AC**: REQ-F-019 AC-5a
- **Method**: Manual
- **Setup**: S001 in planning, 3 backlog entities (E07-F01-001, E07-F01-002, B003)
- **Steps**:
  1. `/plan-sprint S001 --mode=interactive`
  2. When asked about E07-F01-001: answer **no**
  3. When asked about E07-F01-002: answer **yes**
  4. When asked about B003: answer **no**
  5. Verify `shark sprint backlog S001`
- **Expected**: Only E07-F01-002 assigned; E07-F01-001 and B003 not assigned; readiness delta reported on exit
- **Edge cases**: User Ctrl-C mid-session — no partial assigns committed for unconfirmed items

#### TC-P06 — Auto mode: greedy fill + single confirmation
- **AC**: REQ-F-019 AC-5b
- **Method**: Manual
- **Setup**: S001 in planning, capacity backend=5 points, backlog: E07-F01-001 (size=3, backend), E07-F01-002 (size=3, backend), E07-F01-003 (size=2, frontend)
- **Steps**:
  1. `/plan-sprint S001 --mode=auto`
  2. Observe proposed plan (greedy: E07-F01-001 + E07-F01-003 fit; E07-F01-002 exceeds backend capacity)
  3. Confirm when asked
  4. Verify `shark sprint backlog S001`
- **Expected**: E07-F01-001 and E07-F01-003 assigned; E07-F01-002 not assigned (over backend capacity); only one confirmation prompt shown
- **Edge cases**: User declines single confirmation → no entities assigned

#### TC-P07 — Readiness delta reported on exit
- **AC**: REQ-F-019 AC-6
- **Method**: Manual (continuation of TC-P05 or TC-P06)
- **Steps**:
  1. Run `/plan-sprint S001` (any mode), assign at least 1 entity
  2. Observe exit output
- **Expected**: Output shows readiness score before planning (Step 2 value) and after planning (Step 4 value), with a delta (e.g., `+12 points`)

#### TC-P08 — Does NOT call `shark sprint start` (review + manual)
- **AC**: REQ-F-019 AC-7
- **Method**: Code review + manual
- **Code review**: grep `shark sprint start` in `plan-sprint.md` — must be absent
- **Manual**: Run TC-P06 from start to finish; verify `shark sprint get S001` shows status = `planning` (not `active`) after confirmation
- **Expected**: Sprint remains in `planning`; `shark sprint start` never called

#### TC-P09 — All shark calls use `--json` (review)
- **AC**: REQ-F-019 AC-8
- **Method**: Code review of `plan-sprint.md` and `commands/plan-sprint.md`
- **Steps**:
  1. List all `shark sprint …` calls in both files
  2. Verify each call includes `--json` or `--field=`
- **Expected**: Zero shark calls without `--json` or `--field`

---

### Skill: `/run-sprint` (REQ-F-020)

#### TC-R01 — Slash command registration
- **AC**: REQ-F-020 AC-1
- **Method**: File existence + content review
- **Steps**:
  1. Verify `~/.claude/commands/run-sprint.md` exists
  2. Verify `~/.claude/skills/sprint-execution/SKILL.md` exists
  3. Verify `~/.claude/skills/sprint-execution/workflows/run-sprint.md` exists
  4. Verify frontmatter and cross-reference correct in command file
- **Expected**: All paths exist with valid frontmatter

#### TC-R02 — Refuses non-sprint keys
- **AC**: REQ-F-020 AC-2
- **Method**: Manual
- **Test inputs**: Same set as TC-P02
- **Expected**: Same refusal message pattern as `/plan-sprint`

#### TC-R03 — Planning sprint: asks to start
- **AC**: REQ-F-020 AC-3
- **Method**: Manual
- **Setup**: S001 in `planning` status
- **Steps**:
  1. `/run-sprint S001`
  2. Skill asks: "Sprint S001 is in `planning`. Call `shark sprint start S001` first?"
  3a. User confirms → `shark sprint start S001` called → loop continues
  3b. User declines → skill exits
- **Expected**: Both paths work; no silent start without confirmation

#### TC-R04 — Terminal status exits with notice
- **AC**: REQ-F-020 AC-4
- **Method**: Manual, three sub-cases
- **Sub-cases**: TC-R04a (completed), TC-R04b (archived), TC-R04c (cancelled)
- **Expected**: Each exits with a clear notice (e.g., "Sprint S001 is completed. Nothing to run."); no silent no-op; exit code 0

#### TC-R05 — Pull-loop: next → /run → repeat
- **AC**: REQ-F-020 AC-5
- **Method**: Manual
- **Setup**: S001 active with 2 assigned tasks (E07-F01-001 in `ready_for_development`, E07-F01-002 in `ready_for_development`); `/run` is available and can dispatch tasks
- **Steps**:
  1. `/run-sprint S001`
  2. Observe loop: `shark sprint next` called → entity returned → `/run E07-F01-001` dispatched → on return, loop repeats
  3. Loop continues until `shark sprint next` returns empty
  4. Post-loop summary displayed
- **Expected**: Each entity dispatched in turn; loop exits cleanly when no entities remain; no entity skipped

#### TC-R06 — `--agent` flag passed through to `shark sprint next`
- **AC**: REQ-F-020 AC-6
- **Method**: Manual + code review
- **Code review**: Verify `shark sprint next` call in workflow includes `{--agent=X if provided}` pass-through
- **Manual**: `/run-sprint S001 --agent=backend` → observe that only backend entities are returned by `shark sprint next --agent=backend`
- **Expected**: Agent filter passed through; non-backend entities not dispatched in this session

#### TC-R07 — `--max-iterations` cap fires with notice
- **AC**: REQ-F-020 AC-7
- **Method**: Manual
- **Setup**: S001 active with 4 entities; `--max-iterations=2`
- **Steps**:
  1. `/run-sprint S001 --max-iterations=2`
  2. Observe: 2 iterations run; on 3rd, skill exits
  3. Verify notice printed: includes "max-iterations" or equivalent; reports sprint state
- **Expected**: Exactly 2 iterations; notice on cap; sprint still active (not closed)
- **TC-R07b** (boundary below cap): `--max-iterations=10`, sprint has 3 entities — loop exits at 3 with no cap notice

#### TC-R08 — Post-loop burndown and summary printed
- **AC**: REQ-F-020 AC-8
- **Method**: Manual (continuation of TC-R05)
- **Steps**:
  1. Complete TC-R05
  2. Observe post-loop output
- **Expected**: `shark sprint burndown S001` output printed, then `shark sprint summary S001` output printed, then close prompt

#### TC-R09 — Close requires user confirmation
- **AC**: REQ-F-020 AC-9
- **Method**: Manual + code review
- **Code review**: Verify no code path calls `shark sprint close` without a preceding interactive prompt
- **Manual**: After TC-R05 loop exit, observe prompt; answer **no** → sprint remains `active`; answer **yes** → `shark sprint close` called
- **Expected**: Sprint only closed when user explicitly confirms

#### TC-R10 — `--carryover` flag passed to `shark sprint close`
- **AC**: REQ-F-020 AC-10
- **Method**: Manual
- **Steps**:
  1. `/run-sprint S001 --carryover=backlog`
  2. After loop exits, confirm close
  3. Verify `shark sprint close S001 --carryover=backlog` was called (check sprint backlog — incomplete tasks unassigned)
- **Expected**: `--carryover=backlog` passed through; incomplete tasks returned to backlog

#### TC-R11 — Delegates to `/run`, not own dispatch (review)
- **AC**: REQ-F-020 AC-11
- **Method**: Code review of `run-sprint.md`
- **Steps**:
  1. Find the loop body in the workflow file
  2. Verify the entity dispatch call is `/run {entity_key}` (not a direct `shark status advance` or inline status manipulation)
- **Expected**: Loop body contains only a `/run` invocation, not embedded dispatch logic

---

### Skill: `/run-sprint-team` (REQ-F-021)

#### TC-T01 — Slash command registration
- **AC**: REQ-F-021 AC-1
- **Method**: File existence + content review
- **Steps**:
  1. Verify `~/.claude/commands/run-sprint-team.md` exists
  2. Verify `~/.claude/skills/sprint-execution/workflows/run-sprint-team.md` exists
  3. Verify frontmatter and SKILL.md cross-reference correct
- **Expected**: All paths exist with valid frontmatter

#### TC-T02 — Refuses non-sprint keys
- **AC**: REQ-F-021 AC-2
- **Method**: Manual
- **Test inputs**: Same set as TC-P02
- **Expected**: Same refusal message pattern

#### TC-T03 — Inherits /run-agent-team preconditions (review)
- **AC**: REQ-F-021 AC-3
- **Method**: Code review of `run-sprint-team.md`
- **Steps**:
  1. Verify Step 1 (Preconditions) in the workflow body contains the same env-var check (`CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`), claude-version check, branch check, clean-worktree check, and no-existing-team check as `run-agent-team.md`
  2. Verify failures abort before any team is spawned (abort in Step 1, before Step 4 dispatch loop)
- **Expected**: Identical precondition list; precondition failure does not spawn a partial team

#### TC-T04 — Groups entities by feature key
- **AC**: REQ-F-021 AC-4
- **Method**: Manual + code review
- **Setup**: Fixture sprint S001 active with: E07-F01-001 (task, feature E07-F01), E07-F02-001 (task, feature E07-F02), B003 (bug, no feature parent), CC-001 (change card, no feature parent)
- **Steps**:
  1. Code review: verify workflow extracts `E##-F##` from task keys using correct substring logic
  2. Manual: `/run-sprint-team S001 --features=E07-F01` → only E07-F01 group dispatched; B003 and E07-F02 skipped
  3. Manual (full): `/run-sprint-team S001` → E07-F01 dispatched, then E07-F02 dispatched, then B003 and CC-001 via plain `/run`
- **Expected**: Feature groups correctly formed; B003 and CC-001 in standalone group

#### TC-T05 — Serial feature dispatch (attack: no parallel teams)
- **AC**: REQ-F-021 AC-5
- **Method**: Code review
- **Steps**:
  1. In `run-sprint-team.md`, verify the feature dispatch loop is sequential (`for feature_key in feature_groups: /run-agent-team {feature_key}`)
  2. Verify there is no construct that would spawn two `/run-agent-team` calls concurrently (no background `&`, no parallel spawn pattern)
- **Expected**: Exactly one `/run-agent-team` call active at a time; loop is sequential

#### TC-T06 — `--size` flag passed to each `/run-agent-team`
- **AC**: REQ-F-021 AC-6
- **Method**: Code review of `run-sprint-team.md` + manual
- **Code review**: Verify every `/run-agent-team {feature_key}` call in the dispatch loop includes `[--size=N]` when `--size` was provided
- **Manual**: `/run-sprint-team S001 --size=3` → verify `/run-agent-team E07-F01 --size=3` invoked
- **Expected**: `--size` passed through to each feature-group dispatch

#### TC-T07 — `--features` filter restricts execution
- **AC**: REQ-F-021 AC-7
- **Method**: Manual
- **Setup**: Sprint S001 with entities in E07-F01, E07-F02, E07-F03
- **TC-T07**: `/run-sprint-team S001 --features=E07-F01,E07-F02` → E07-F03 skipped
- **TC-T07b**: `/run-sprint-team S001 --features=E07-F99` (no entities in F99) → no dispatch; exits cleanly
- **TC-T07c**: `/run-sprint-team S001` (no filter) → all three features dispatched
- **Expected**: Filter correctly limits dispatch to listed features

#### TC-T08 — Standalones dispatched via `/run` (not `/run-agent-team`)
- **AC**: REQ-F-021 AC-8
- **Method**: Code review + manual
- **Setup**: Fixture sprint with B003 and CC-001 as standalone entities (no feature parent)
- **Code review**: Verify standalone-group loop uses `/run {entity_key}`, not `/run-agent-team`
- **Manual**: `/run-sprint-team S001` → B003 dispatched via `/run B003`; no `/run-agent-team B003` attempt
- **Expected**: Standalones handled by `/run`; `/run-agent-team` never receives a non-feature key from this skill

#### TC-T09 — Burndown between feature groups
- **AC**: REQ-F-021 AC-9
- **Method**: Manual
- **Setup**: Sprint with 2 feature groups
- **Steps**:
  1. `/run-sprint-team S001`
  2. After E07-F01 group completes, observe output
- **Expected**: `shark sprint burndown S001` output printed between E07-F01 completion and E07-F02 dispatch start

#### TC-T10 — Post-loop close prompt
- **AC**: REQ-F-021 AC-10
- **Method**: Manual (continuation of TC-T07c)
- **Steps**:
  1. After all groups complete
  2. Observe: summary printed, then close prompt
  3. Decline → sprint stays active
- **Expected**: Identical close-gate behavior to `/run-sprint` TC-R09

---

### Skill: `/retro-sprint` (REQ-F-022)

#### TC-A01 — Slash command registration
- **AC**: REQ-F-022 AC-1
- **Method**: File existence + content review
- **Steps**:
  1. Verify `~/.claude/commands/retro-sprint.md` exists
  2. Verify `~/.claude/skills/sprint-analytics/SKILL.md` exists
  3. Verify `~/.claude/skills/sprint-analytics/workflows/retro-sprint.md` exists
  4. Verify frontmatter and cross-references
- **Expected**: All paths exist with valid frontmatter

#### TC-A02 — Refuses non-sprint keys
- **AC**: REQ-F-022 AC-2
- **Method**: Manual
- **Test inputs**: Same set as TC-P02
- **Expected**: Refusal output; no shark calls

#### TC-A03 — Refuses non-closed sprints
- **AC**: REQ-F-022 AC-3
- **Method**: Manual, two sub-cases
- **TC-A03a**: `/retro-sprint S001` where S001 is `active` → output contains: `/retro-sprint requires a completed or archived sprint. S001 is in status: active. Close the sprint first with /run-sprint or shark sprint close.`
- **TC-A03b**: `/retro-sprint S001` where S001 is `planning` → same pattern with status `planning`
- **Expected**: Exact refusal message; no data pull attempted

#### TC-A04 — Data pull sequence verified (review)
- **AC**: REQ-F-022 AC-4
- **Method**: Code review of `retro-sprint.md`
- **Steps**:
  1. Verify Step 2 calls `shark sprint summary {S###} --detailed --json` first
  2. Then calls `shark sprint velocity --json`
  3. Then for each entity in carryover/rejected lists: calls `shark notes {entity_key}`
- **Expected**: Sequence matches spec exactly; no calls out of order

#### TC-A05 — Outcome section present and populated
- **AC**: REQ-F-022 AC-5 (Outcome section)
- **Method**: Manual
- **Setup**: S004 completed; summary data: planned=10, completed=8, 2 carryover
- **Steps**:
  1. `/retro-sprint S004 --no-write`
  2. Check output for `## Outcome` section
- **Expected**: Section present; contains planned count, completed count; references Σ size and entity count

#### TC-A06 — Velocity Context section present and populated
- **AC**: REQ-F-022 AC-5 (Velocity Context)
- **Method**: Manual (same session as TC-A05)
- **Steps**:
  1. Check output for `## Velocity Context`
- **Expected**: Section present; shows this sprint velocity vs. trailing average; includes delta (e.g., `+3 tasks vs. trailing avg of 5`)

#### TC-A07 — Carryover Analysis section present and populated
- **AC**: REQ-F-022 AC-5 (Carryover Analysis)
- **Method**: Manual (same session)
- **Setup**: S004 has 2 carryover entities with rejection notes (at least one `blocker` note and one `rejection` note)
- **Steps**:
  1. Check output for `## Carryover Analysis`
- **Expected**: Section lists each carried-over entity; cites the note type and content source; does not say "no carryover notes found" when notes exist

#### TC-A08 — Cycle-Time Highlights section present
- **AC**: REQ-F-022 AC-5 (Cycle-Time Highlights)
- **Method**: Manual (same session)
- **Steps**:
  1. Check output for `## Cycle-Time Highlights`
- **Expected**: Section present; if `--detailed` summary includes cycle-time-by-phase data, it appears here; if data is absent, section contains an explanatory note (not empty)

#### TC-A09 — Recommendations: 3–5 items, data-driven
- **AC**: REQ-F-022 AC-5 (Recommendations)
- **Method**: Manual (same session)
- **Setup**: Fixture sprint S004 with velocity variance > 25% and at least one size≥8 entity completed
- **Steps**:
  1. Check output for `## Recommendations`
  2. Count items
  3. Verify each item references a quantitative value from the sprint data (velocity figure, cycle time, carryover count, size value)
- **Expected**: 3–5 bullet points; no generic placeholders ("consider improving X"); each references measurable data

#### TC-A10 — Retro file written to correct path
- **AC**: REQ-F-022 AC-6
- **Method**: Manual
- **Setup**: Fixture sprint S004 completed; `docs/sprints/` does not exist
- **Steps**:
  1. `/retro-sprint S004` (no `--no-write`)
  2. Confirm when prompted
  3. Check filesystem
- **Expected**: `docs/sprints/S004-retro.md` exists; `docs/sprints/` directory created; file contains 5-section markdown

#### TC-A11 — Prompts before overwriting existing retro file
- **AC**: REQ-F-022 AC-7
- **Method**: Manual
- **Setup**: `docs/sprints/S004-retro.md` already exists
- **TC-A11**: `/retro-sprint S004` → prompt asking whether to overwrite; user declines → file unchanged
- **TC-A11b**: `/retro-sprint S004` → user confirms overwrite → file updated
- **Expected**: Prompt fires on existing file; no silent overwrite

#### TC-A12 — `--no-write` prints to stdout, no file written
- **AC**: REQ-F-022 AC-8
- **Method**: Manual
- **Steps**:
  1. `/retro-sprint S004 --no-write`
  2. Verify retro markdown printed to stdout
  3. Verify `docs/sprints/S004-retro.md` does NOT exist (or is unchanged if it existed)
- **Expected**: No file I/O; full retro report on stdout

#### TC-A13 — All shark calls use `--json` (review)
- **AC**: REQ-F-022 AC-9
- **Method**: Code review of `retro-sprint.md`
- **Steps**: List all `shark …` calls; verify each includes `--json` or `--field=`
- **Expected**: Zero calls without `--json` or `--field`

---

### Cross-Cutting: `/run` Advisory (REQ-F-023)

#### TC-X01 — Step 1 amendment present in `run.md` (review)
- **AC**: REQ-F-023 AC-1
- **Method**: Code review of `~/.claude/skills/orchestration/workflows/run.md`
- **Steps**:
  1. Locate Step 1 ("Read State") in the file
  2. Verify it contains a check for the sprint field (or `shark sprint backlog` filter fallback)
  3. Verify the check is additive — original Step 1 logic is unchanged
- **Expected**: Amendment present; no other Step 1 logic removed

#### TC-X02 — Advisory message correct text (blocked until F03)
- **AC**: REQ-F-023 AC-2
- **Method**: Manual — BLOCKED until F03 ships and field name is confirmed
- **Pre-condition**: E19-F03 completed; developer verified field name from `shark get E07-F01-001 --json` output
- **Setup**: Task E07-F01-001 assigned to active sprint S005
- **Steps**:
  1. `/run E07-F01-001`
  2. Observe first line of output
- **Expected**: `Note: E07-F01-001 is in sprint S005 (active). For sprint-aware execution use /run-sprint S005. Continuing per-entity execution.`

#### TC-X03 — `/run` continues after advisory
- **AC**: REQ-F-023 AC-3
- **Method**: Manual (can be done in the same session as TC-X02 once unblocked)
- **Steps**:
  1. Observe TC-X02 advisory message printed
  2. Verify task dispatch continues (orchestrator_action is read; entity begins execution)
- **Expected**: No abort; execution proceeds normally after the one-line notice

#### TC-X04 — No notice when entity has no sprint
- **AC**: REQ-F-023 AC-4
- **Method**: Manual
- **Setup**: Task E07-F02-001 not assigned to any sprint
- **Steps**:
  1. `/run E07-F02-001`
  2. Observe Step 1 output
- **Expected**: No sprint-advisory line; execution proceeds silently through Step 1

---

### Cross-Cutting: Docs & Registration (REQ-F-024, REQ-F-025, REQ-F-026)

#### TC-D01 — `commands/run-agent-team.md` has See-also line
- **AC**: REQ-F-024 AC-1
- **Method**: File content review
- **Steps**: Read `~/.claude/commands/run-agent-team.md`; verify a line near the top reads approximately: `See also: \`/run-sprint-team\` for sprint-scoped multi-feature execution.`
- **Expected**: Line present; no other changes to the file

#### TC-D02 — `skills/orchestration/workflows/run-agent-team.md` has See-also
- **AC**: REQ-F-024 AC-2
- **Method**: File content review
- **Steps**: Read the file; find Usage section; verify See-also line present
- **Expected**: See-also in Usage section; no other content changed

#### TC-D03 — No behavior change to `/run-agent-team` itself
- **AC**: REQ-F-024 AC-3
- **Method**: Diff review
- **Steps**: `git diff HEAD~1 -- ~/.claude/commands/run-agent-team.md ~/.claude/skills/orchestration/workflows/run-agent-team.md`; verify only the See-also line is added
- **Expected**: Single-line addition per file; no other changes

#### TC-M01 — Three new SKILL.md files exist
- **AC**: REQ-F-025 AC-1
- **Method**: File existence check
- **Steps**:
  1. Verify `~/.claude/skills/sprint-planning/SKILL.md` exists
  2. Verify `~/.claude/skills/sprint-execution/SKILL.md` exists
  3. Verify `~/.claude/skills/sprint-analytics/SKILL.md` exists
- **Expected**: All three present

#### TC-M02 — SKILL.md files follow orchestration style
- **AC**: REQ-F-025 AC-2
- **Method**: Content review
- **Steps**:
  1. Read `~/.claude/skills/orchestration/SKILL.md` as reference
  2. For each new SKILL.md, verify: YAML frontmatter with `name` and `description`, one "What this is" paragraph
  3. Verify no significant deviation from the orchestration style
- **Expected**: Style consistent; frontmatter keys present

#### TC-M03 — PIPELINE.md table row added
- **AC**: REQ-F-025 AC-3
- **Method**: File content review
- **Steps**:
  1. Read `~/.claude/PIPELINE.md`
  2. Find "What 'Follow SDLC Flow' Covers" table
  3. Verify a row for "Sprint orchestration" exists with skills `sprint-planning`, `sprint-execution`, `sprint-analytics` and commands `/plan-sprint`, `/run-sprint`, `/run-sprint-team`, `/retro-sprint`
  4. Verify no PDLC diagram changed
- **Expected**: Single row added; no other table or diagram modified

#### TC-PM01 — `product-manager.md` subsection present
- **AC**: REQ-F-026 AC-1
- **Method**: File content review
- **Steps**:
  1. Read `~/.claude/agents/product-manager.md`
  2. Find "PRIMARY: Feature Execution Workflow" section
  3. Verify subsection "When a sprint is active" exists with the four `/sprint-*` commands listed
- **Expected**: Subsection present; content matches spec §Architecture section for this requirement

#### TC-PM02 — No other agent files changed
- **AC**: REQ-F-026 AC-2
- **Method**: Diff review
- **Steps**: `git diff HEAD~1 -- ~/.claude/agents/`; verify only `product-manager.md` has changes
- **Expected**: Zero changes to any other agent file

---

### Non-Functional Requirements

#### TC-NF01 — Idempotency: no duplicate adds
- **AC**: REQ-NF-007
- **Method**: Manual
- **Setup**: `/plan-sprint S001` run once; user confirmed item E07-F01-001; sprint backlog now contains E07-F01-001
- **Steps**:
  1. Run `/plan-sprint S001` again (same sprint, same lifecycle stage)
  2. Observe: E07-F01-001 already assigned; should not be re-presented for confirmation again
  3. Verify `shark sprint backlog S001` still shows exactly one instance of E07-F01-001
- **Expected**: Second invocation does not duplicate assignments; backlog unchanged

#### TC-NF02 — Idempotency: no double-close
- **AC**: REQ-NF-007
- **Method**: Manual + code review
- **Setup**: Sprint S001 completed (TC-R09 path)
- **Steps**:
  1. Run `/run-sprint S001` again on the already-completed sprint
  2. Observe TC-R04a behavior (terminal status exits with notice)
  3. Verify no `shark sprint close` attempted
- **Expected**: TC-R04 path triggered; sprint remains `completed`; no second close attempted

#### TC-NF03 — Idempotency: retro overwrite prompt
- **AC**: REQ-NF-007
- **Method**: TC-A11 covers this case — same test

#### TC-NF04 — All mutating calls require user confirmation
- **AC**: REQ-NF-008
- **Method**: Code review — exhaustive enumeration
- **Steps**:
  1. List every occurrence of these calls in all new skill files: `shark sprint add`, `shark sprint remove`, `shark sprint start`, `shark sprint close`
  2. For each, trace backward through the workflow to find the nearest preceding interactive prompt
  3. Verify no call site is reachable without passing through a prompt
- **Expected**: 100% of mutation call sites have a preceding confirmation prompt on every reachable code path

#### TC-NF05 — All shark calls use `--json` or `--field`
- **AC**: REQ-NF-009
- **Method**: Static analysis (grep)
- **Steps**:
  ```bash
  grep -rn "shark " ~/.claude/commands/plan-sprint.md \
    ~/.claude/commands/run-sprint.md \
    ~/.claude/commands/run-sprint-team.md \
    ~/.claude/commands/retro-sprint.md \
    ~/.claude/skills/sprint-planning/workflows/plan-sprint.md \
    ~/.claude/skills/sprint-execution/workflows/run-sprint.md \
    ~/.claude/skills/sprint-execution/workflows/run-sprint-team.md \
    ~/.claude/skills/sprint-analytics/workflows/retro-sprint.md \
    ~/.claude/skills/orchestration/workflows/run.md 2>/dev/null
  ```
  For each `shark …` line found, verify `--json` or `--field` appears on the same line
- **Expected**: Zero shark call lines missing `--json` or `--field`

#### TC-NF06 — Zero shark-repo changes
- **AC**: REQ-NF-010
- **Method**: Diff review
- **Steps**: `git diff HEAD~1 -- internal/ cmd/ docs/cli-reference/` in the shark repo
- **Expected**: Empty diff (no changes to any Go file, migration, or CLI reference doc)

---

## Integration Scenarios

These test cases verify cross-component boundaries. They map to UAT scenarios the epic expects F06 to satisfy.

### INT-F06-01: End-to-end sprint lifecycle via Claude commands (UAT-J2 + UAT-J3 + UAT-J4)

**Components interacting**: `/plan-sprint` → `shark sprint add` (F03) → `shark sprint start` (F02) → `/run-sprint` → `/run` (orchestration skill) → `shark sprint next` (F03) → `shark sprint close` (F02) → `/retro-sprint` → `shark sprint summary --detailed` (F04) + `shark sprint velocity` (F04)

**What to verify**: A single Claude session can execute the full journey:
1. `/plan-sprint S001` adds tasks in interactive mode
2. User manually runs `shark sprint start S001`
3. `/run-sprint S001` drives all tasks to completion
4. `/retro-sprint S001` generates a report with all 5 sections populated from real data

**UAT alignment**: Satisfies UAT-J2-10 (AI orchestrator sprint planning via JSON), UAT-J3 (monitoring during /run-sprint loop), UAT-J4-07 (summary JSON consumed by /retro-sprint)

### INT-F06-02: Advisory fires when `/run` encounters a sprint-assigned entity (UAT-EDGE-05 analog)

**Components interacting**: `/run` (orchestration skill, amended Step 1) → `shark get {KEY} --json` (F03 field) → one-line advisory

**What to verify**: TC-X02 (blocked until F03) — advisory text contains correct sprint key and status; execution continues; no double-dispatch

**UAT alignment**: Scenario 5 in spec.md feature-level AC; demonstrates coordination boundary between per-entity and sprint orchestrators

### INT-F06-03: `/run-sprint-team` with mixed entity types (Scenario 3 in spec.md)

**Components interacting**: `/run-sprint-team` → `shark sprint backlog S005 --json` (F03) → `/run-agent-team E07-F01` + `/run-agent-team E07-F02` → `/run B003`

**What to verify**: Feature groups dispatched serially; B003 standalone dispatched via `/run`; burndown displayed between groups; close prompt at end

**UAT alignment**: Scenario 3 in spec.md feature-level ACs; INT-01 (bug assignment in sprint) from UAT plan

### INT-F06-04: PM agent picks up sprint commands (discoverability)

**Components interacting**: `product-manager.md` agent prompt → human reading the PM agent instructions

**What to verify**: PM agent subsection "When a sprint is active" is present and correctly lists all four commands; PM would know to prefer `/run-sprint` over per-feature `/run E##-F##` when a sprint is active

**UAT alignment**: REQ-F-026; AI Orchestrator persona acceptance gate in UAT (§5)

---

## Test Infrastructure

### Existing test patterns to follow

- **Skill file structure**: Follow `~/.claude/skills/orchestration/SKILL.md` for SKILL.md format and `~/.claude/commands/run.md` for command file format
- **Review methodology**: sibling test plans (E19-F03, E19-F04, E19-F05) use code-review + manual invocation as the test execution method for skill-file features — follow this pattern
- **Fixture sprint**: E19-F01 through F05 deliver the shark CLI surface; tests require a real shark project with sprint data. Use `shark sprint create`, `shark sprint add`, and `shark sprint start` to set up fixtures manually.

### New test helpers needed

None. This feature is Claude-side skill files; no Go test helpers are added. Manual verification uses the live shark CLI against a local test project. The `TC-NF05` grep command is the only "automated" check and requires no test framework.

### Environment requirements

- `shark` CLI built from E19-F01 through F05 branch (sprint commands must be available)
- A local shark project with at least: 2 epics, 4 features, 10+ tasks in `ready_for_development` status, and `~/.claude/` with all existing skills present
- `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` set in environment for TC-T03 and TC-T05
- Claude Code version ≥ 2.1.32 for TC-T03

---

## Exit Gate Checklist

- [x] Every AC in REQ-F-019 through REQ-F-026 has at least one test case
- [x] Every test case has a caller-path contract (or review-only justification for non-Go, non-callable code)
- [x] Edge cases identified for each AC (refusal message inputs, terminal status sub-cases, BVA on max-iterations)
- [x] Integration scenarios cover cross-component boundaries (F03 dependency, orchestration skill delegation, PM agent discoverability)
- [x] Test patterns reference existing infrastructure (orchestration skill format, sibling test-plan methodology)
- [x] REQ-NF-007 through REQ-NF-010 each have dedicated test cases (TC-NF01..TC-NF06)
- [x] TC-X02 explicitly marked BLOCKED until F03 ships and field name confirmed
- [x] UAT alignment documented for each integration scenario

---

*Test Plan Complete — 2026-05-10*
