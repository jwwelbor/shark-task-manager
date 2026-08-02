---
research_schema: 2
entity_key: E38-F09
entity_type: feature
recipe: universal
rigor: complex
categories: [backend, workflow_operations, documentation]
related_work: true
---

# Research report — Provider-Neutral Coordination and Live Resume

## Scope

E38-F09 adds the Shark Attack v2 coordination protocol: independent
coordination-level (Direct/Batch/Council) and execution-topology
(Sequential/Parallel-with-ownership/Parallel-with-isolation) selection,
same-worker resume or bounded replacement, prompt-hash provenance at the host
adapter boundary, and Codex/Claude Code adapters. It extends the completed
F04 (skill/role protocol), F06 (role-aware pull), F07 (Rider loop), and F08
(integrity prerequisites) boundaries rather than reopening them. Shark and the
Rider parent remain the sole owners of selection, claims, heartbeats, notes,
transitions, kickbacks, and release; F09 adds no runtime, scheduler, claim
store, credential store, or second workflow engine.

The feature was rewound from `specification` to `research` on 2026-08-01
(see feature.md amendment) because its live-question design depended on a
first-class Question lifecycle that did not exist yet. E39 (Question and
Decision Workflow Management) shipped complete on 2026-07-31 (PR #145) and
publishes exactly that lifecycle as cross-epic contract **X-06**, naming
E38-F09 as its "blocked activation consumer." This research re-scopes F09
against E39's actual delivered API (`Q###` entities, serial
`shark next Q### --json` dispatch, `question_blocks` gate, and the F04 focused
read/producer-handoff surface) instead of the bespoke
question/responder/handoff/resolution shape assumed by the prior
specification pass.

Vocabulary: *coordination level* is Direct, Batch, or Council; *execution
topology* is Sequential, Parallel with ownership, or Parallel with isolation;
*control envelope* is the adapter result (`final|question|needs_council|
blocked_external|failed`) separate from the workflow's opaque
`recommended_outcome`; *same-worker resume* sends an answer to the still-live
dispatched worker; *bounded replacement* starts one new worker from an
immutable handoff when native resume is unsupported; and the live Question
loop now means: a worker's bounded question becomes (or references) a
first-class `Q###` Question routed through E39's serial responder workflow,
not a Shark-Attack-owned envelope-and-transcript store.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `feature.md` (triage breadcrumb and
  2026-08-01 amendment), `dev-artifacts/shark-attack-v2-plan/implementation-plan.md`
  §§2–3 (two-axis model, control envelope, live question-and-resume protocol),
  and `docs/plan/E39-question-and-decision-workflow-management/architecture.md`
  (Question lifecycle vocabulary: `open`, `answering`, `ready_for_resolution`,
  `resolved`, `withdrawn`, `superseded`).
- [x] `affected_implementation_or_contract` — Evidence: `internal/cli/commands/next.go`
  (`NextResponse` JSON has no `prompt_sha256`/`prompt_bytes` provenance field
  but already carries `QuestionBlock *services.QuestionBlock` from the live
  E39 gate); `internal/runner/dispatcher.go` (`AgentDispatcher` interface is
  `Dispatch`/`Name`/`BuildCommand` only — no capability discovery, follow-up,
  interrupt, isolation, or resume); `internal/services/question_service.go`
  (`ListOpenQuestionsByResponder`, `ListQuestionsBlocking`, `ReadQuestionFull`
  already implemented and live); `internal/cli/commands/question.go` (full
  `question` CLI surface — create/get/list/configure-workflow/respond/
  resolve/withdraw/supersede/open-by-responder/blocking-for/full — already
  registered); `skills/shark-rider/verbs/run.md` (exact `response.prompt`
  transport mandate already present, hash provenance not yet present);
  `skills/shark-attack/context/roster-schema.yaml` (still uses `model_tier:
  opus|high`, not the proposed `fast|balanced|deep` capability profile).
- [x] `related_work` — Evidence: `epic.md` §2 locked decisions and §3 scope,
  `E38-interaction-map.md` (I-04, I-06 through I-10, dependency order §"F09
  integrates F04-F08"), `E38-cross-epic-map.md`, F04/F07 `spec.md` and
  `research-report.md`, F08/F05/F10/F11 `feature.md`, E39 `architecture.md`,
  `E39-interaction-map.md`, `E39-cross-epic-map.md` (X-06), and E39-F04
  `spec.md` (REQ-F-004 "Publish the X-06 producer handoff").
- [x] `pattern_contract` — Evidence: `skills/shark-rider/verbs/run.md` lines
  19–255 establish keyed `shark next` as the sole prompt source and the
  parent-only claim/dispatch/advance/release loop; `internal/runner/
  {dispatcher,claude_dispatcher,codex_dispatcher}.go` establish the existing
  one-shot provider dispatch seam F09 must extend rather than replace;
  E39-F04 `spec.md` "Key technical decisions" table establishes the
  compact-vs-full read/actor-policy pattern F09's consumer coverage must
  reuse verbatim.
- [x] `dependency_impact` — Evidence: `E38-interaction-map.md` rows I-04
  (E38-F04 → F09), I-06 (F06 → F09), I-07 (F07 → F09), I-08 (F08 → F09), and
  I-10 (F09 → F10/F11 output); `E39-cross-epic-map.md` X-06 (E39-F04 → F09,
  "activation owner"); direct verification that `E38-F08`'s Shark status is
  `completed` in the live project database (`shark get E38-F08 --json`,
  history shows a full 2026-07-30 code_review→qa→approval cycle) while `git
  log --grep=F08` and file search show **no F08 implementation reachable from
  `main`** — the only F08 code (`internal/cli/commands/next.go` prompt
  provenance, `related_docs.go` tech-debt routing, `research/validator.go`,
  `workflow/tech-debt.yaml` single-gate collapse) exists solely in the
  unmerged branch/worktree `feature/E38-F05-council-artifacts`
  (`git merge-base --is-ancestor e079efd0 aa5a3438` → not an ancestor).
- [x] `cross_boundary_risks` — Evidence: `internal/cli/commands/next.go`
  crosses Shark prompt assembly into host adapters without hash provenance
  today; `skills/shark-rider/verbs/run.md` crosses host worker results back
  into parent-owned workflow mutation; `internal/runner/*dispatcher.go`
  cross into external provider CLIs (`claude`, `codex`) with no capability
  contract; the authored `skills/shark-attack/` tree and the embedded
  `internal/sharkdata/default_data/skills/shark-attack/` tree are currently
  byte-identical (verified with `diff -rq`) but `manifest.yaml:69` still
  declares `shark-attack` as `ownership: canonical` embedded content, so the
  authored-vs-embedded canonical direction (plan §8 decision 10) remains
  unresolved and any F09 skill edit must land in both trees or a parity gate.
- [x] `alternatives` — Evidence: `dev-artifacts/shark-attack-v2-plan/implementation-plan.md` §3 (capability
  detection vs. assumed provider commands), §"Live question-and-resume
  protocol" (same-worker follow-up vs. immutable handoff + replacement), §8
  (bespoke Shark-Attack question schema vs. E39 Question entity — resolved
  by the owner decision recorded in `feature.md`'s 2026-08-01 amendment in
  favor of E39); and the abandoned branch's own capability map
  (`git show feature/E38-F05-council-artifacts:...E38-F09.../research-report.md`)
  which chose a bespoke `QuestionControl`/`ControlEnvelope` type — now
  superseded evidence, cited only to show the alternative was tried and
  rejected, not to be revived.

## Capability map

| Capability | Source | Decision | Application to E38-F09 |
| --- | --- | --- | --- |
| Parent-owned keyed `next → claim → dispatch → advance → release` loop | `skills/shark-rider/verbs/run.md`; F07 `spec.md` REQ-F-001–003 (completed) | **REUSE** | Keep keyed `shark next` as the only prompt source. A live worker question must hold the parent claim/heartbeat without transitioning the dispatched entity; F09 adds question routing around this loop, not inside a new one. |
| Council roles, bounded messages, durable decisions/handoffs, escalation | F04 `spec.md`/`research-report.md` (completed); I-04 | **EXTEND** | Route material, non-routine questions through the existing chair/role escalation shape. Routine question/answer pairs stay in the live Question record (E39), not a second council transcript. |
| Role-aware read-only selection and `ClaimService` race boundary | F06 (completed); I-06 | **REUSE** | Capability/model preference and roster membership stay non-authoritative. F09 does not touch selection or claim eligibility. |
| `E38-F08` integrity prerequisites (non-mutating `shark plan`, tech-debt related-docs parity, single research-validator rule, one tech-debt implementation/gate state) | I-08; Shark status shows `completed`, but no implementation is reachable from `main` (see Findings #1) | **CONTRADICTS** | F09 cannot assume F08's Tranche A repairs are actually on trunk. Either re-verify/re-implement the specific F08 prerequisites F09 needs (prompt provenance seam, tech-debt parity) as part of F09's own scope, or block F09 until F08's status/implementation gap is reconciled — this is a decision for the parent loop, not something this research resolves. |
| E39 Question entity: `Q###` key, serial `shark next Q### --json` responder dispatch, `question_blocks` gate, focused reads (`open-by-responder`, `blocking-for`, `full`) | E39 `architecture.md`; E39-F01–F04 `spec.md`; **live on `main`** (`internal/services/question_service.go`, `internal/cli/commands/question.go`, `internal/models/question.go` all present and wired into `next.go` via `QuestionBlock`) | **REUSE** | This is the X-06 producer contract. F09 must build its live-question loop entirely on top of these public Question surfaces (create the Question, use `shark next Q###` to route the current responder, use `question respond`/`resolve` for closure, use `blocking-for`/`open-by-responder`/`full` for read access) instead of inventing a parallel envelope-and-transcript store. |
| X-06 cross-epic contract naming E38-F09 as "activation/consumer" | `E39-cross-epic-map.md`; E39-F04 `spec.md` REQ-F-004 | **NEW (for F09)** | E39-F04 published the producer side and explicitly states "E38-F09 remains blocked and must add its own live consumer coverage when E39 completes; this feature neither edits nor resumes E38-F09." F09 is the first and only owner of the consumer-side contract test and integration. |
| Existing Claude/Codex process dispatchers (`internal/runner/{dispatcher,claude_dispatcher,codex_dispatcher}.go`) | Live on `main` | **EXTEND** | Reuse the provider boundary for bounded spawn/argument transport, but add the missing capability/control-envelope contract (follow-up, progress, interrupt, isolation, resume, provenance) — `AgentDispatcher` today exposes only `Dispatch`/`Name`/`BuildCommand`. |
| Keyed-next JSON prompt payload (`internal/cli/commands/next.go`) | Live on `main` | **EXTEND** | Add `prompt_sha256` (and byte length) to the JSON response plus a byte-exact `--prompt-out <path>` flag. Confirmed absent today; `--field prompt` is documented elsewhere as not byte-exact (trailing newline). |
| Shark Attack authored (`skills/shark-attack/`) vs. embedded (`internal/sharkdata/default_data/skills/shark-attack/`) distribution | Both present, byte-identical today; `manifest.yaml:69` still declares embedded ownership `canonical` | **CONTRADICTS** | The canonical-direction decision (plan §8 #10) is still open on `main`. Any F09 skill-content change (roster capability profiles, question-routing workflow) must update both trees identically until a parity gate or explicit canonical decision exists, or risk silent drift. |
| Worker-owned child pulling (`skills/shark-attack/workflows/pull-by-role.md`) | Live on `main`, unchanged since F04/F07 | **CONTRADICTS** | No demonstrated live consumer (confirmed again — no Go runtime reference, only skill prose and F06's research report reference it). The prior branch's owner decision already resolved to exclude it from the v2 normal path; carry that resolution forward rather than re-litigating it. |
| Bespoke `QuestionControl`/`ControlEnvelope` question type from the abandoned branch (`internal/runner/parent_control.go` in `feature/E38-F05-council-artifacts`, unmerged) | Unmerged branch, dated 2026-07-30, predates E39 | **CONTRADICTS** | This is the exact "bespoke persistent question/responder/handoff/resolution flow" the feature.md amendment says is superseded by E39's Question entity. Do not resume or merge this branch's question/control-envelope code as-is; its non-question plumbing (control envelope for `final`/`needs_council`/`blocked_external`/`failed`, roster capability profiles, prompt-hash seam) may still be salvageable evidence but needs re-review against the current `main` baseline before reuse — it was never merged and is now ~2 days stale relative to `main`. |
| Additional-provider conformance (Copilot, Antigravity) | F10 `feature.md` | **NEW** | Out of F09's scope; F09 produces the provider-neutral contract plus installed Codex/Claude Code evidence only. F10 owns Copilot/Antigravity. |
| Real complicated-lifecycle release qualification | F11 `feature.md` | **NEW** | Out of F09's scope; F11 selects and runs the approved qualification target after F09/F10 land. |

## Findings

1. **F08's "completed" Shark status is not backed by any implementation on
   `main`.** `shark get E38-F08 --json` reports `status: completed` with a
   full, plausible 2026-07-30 history (code_review → qa → approval →
   completed, with distinct `tech-lead@anthropic/sonnet` / `qa@anthropic/sonnet`
   / `owner@human` agents and several fail/rework cycles). But `main` has no
   `E38-F08` `spec.md`, `research-report.md`, or tasks directory (only
   `feature.md`), and none of the described Tranche A code changes
   (`prompt_sha256`, tech-debt related-docs CLI routing, the research
   validator's single-line rule statement, the tech-debt workflow's single
   implementation/gate state) exist in the current tree. The only place this
   code exists is the commit `e079efd0` ("feat(e38): shark-attack v2 integrity
   prerequisites and provider-neutral coordination (F08/F09)") on the
   unmerged branch `feature/E38-F05-council-artifacts`, confirmed **not** an
   ancestor of `main`. That branch has its own separate `shark-tasks.db` file
   (dated 2026-07-25, distinct from the live project db dated today), so the
   branch's own workflow runs did not write into the live database either —
   the live "completed" status for F08 was set by some other, undocumented
   process against the live db. This is a genuine integrity gap upstream of
   F09: the interaction map's dependency order states "F09 integrates
   F04-F08," but F08's real repairs are not present for F09 to build on. This
   finding, not a design detail, is the most important thing this research
   surfaces — it is a decision for the parent loop / project owner, not
   something a research pass can silently paper over.

2. **E39 is fully live and already wired into the dispatch path.** Contrary
   to the state assumed by the pre-rewind specification pass, E39's Question
   entity, workflow, gate, and focused reads are not merely designed — they
   are implemented, tested, and already integrated into `shark next`'s JSON
   response (`QuestionBlock *services.QuestionBlock` in
   `internal/cli/commands/next.go`). F09's live-question loop should be a
   consumer of this existing, working surface, not a design exercise against
   an unbuilt one.

3. **The runner/dispatcher boundary is unchanged from the pre-rewind
   assessment.** `AgentDispatcher` still exposes only `Dispatch`, `Name`, and
   `BuildCommand`. There is no capability discovery, follow-up/resume,
   interrupt, isolation, or provenance surface. F09's adapter contract work
   is still fully required; nothing in this area regressed or was
   accidentally delivered elsewhere.

4. **The roster/skill-distribution open decisions from the original v2 plan
   remain open on `main`.** `roster-schema.yaml` still uses `model_tier:
   opus|high` (not `fast|balanced|deep`), and the authored/embedded
   `shark-attack` trees are still both present and byte-identical with the
   embedded copy still marked `canonical` in `manifest.yaml` — the plan's §8
   decision #10 (which tree is authoritative) has not been made. F09 should
   not assume either direction; a spec written against this research should
   either make and record that call explicitly or scope around it.

5. **An abandoned, un-mergeable prior attempt at F09 exists and predates
   E39.** The branch `feature/E38-F05-council-artifacts` contains a full
   previous F09 `spec.md`/`research-report.md`/task set and ~1,100 lines of
   Go (`parent_control.go`, `roster_profile.go`, `dispatcher.go` extensions,
   contract tests) implementing a bespoke `QuestionControl` type as the
   question/answer envelope. Its own council decision record
   (`d-e38-f09-v2-owner-gates-001.yaml`) shows real owner sign-off on several
   points (adapter preference vocabulary, sequential-by-default topology,
   manual-install-only skill distribution) that remain useful evidence even
   though the Question-shaped mechanism itself must be replaced by E39's
   entity. This body of work should be read for salvageable non-question
   decisions before starting F09 implementation, but must not be merged
   as-is: it is unmerged, ~2 days stale against `main`, and built against a
   question model the project has since explicitly rejected.

## Decisions

- **Build the live-question loop on E39's public Question API**, not a new
  Shark-Attack-owned envelope/transcript store: create/route/respond/resolve
  through `shark question ...` and `shark next Q###`, and use
  `blocking-for`/`open-by-responder`/`full` for read access, exactly as
  X-06 specifies. This supersedes the bespoke `QuestionControl` design from
  the abandoned branch and from the pre-rewind specification pass.
- **Treat F08's dependency as unresolved, not satisfied.** F09's specification
  pass must either (a) explicitly re-scope in the specific F08 integrity
  repairs it actually needs (at minimum, keyed-next prompt-hash provenance,
  since that is core F09 scope regardless of F08's status) and flag the
  broader F08 gap for separate remediation, or (b) escalate the F08
  status/implementation mismatch for an owner decision before proceeding.
  This research does not adjudicate which; it surfaces the gap as evidence.
- **Carry forward the prior owner decision** (`d-e38-f09-v2-owner-gates-001`)
  on non-question points where it does not conflict with the E39 pivot:
  `fast|balanced|deep` as non-authoritative capability-profile vocabulary
  with `model_tier` compatibility/warning; excluding worker-owned child
  pulling from the v2 normal path; manual/client-only distribution for
  Shark Attack (revisit only alongside the still-open authored/embedded
  canonical-direction decision); sequential-by-default execution with
  parallel requiring captured ownership/isolation evidence; and scoping
  installed-host verification to Codex and Claude Code only.
- **Do not merge or resume the abandoned branch's Go code directly.** Re-derive
  the control-envelope, roster-profile, and prompt-provenance implementation
  fresh against current `main`, reusing only the non-question design
  decisions and treating the branch as historical evidence.

## Sources

- `docs/plan/E38-shark-attack-team-orchestration/E38-F09-provider-neutral-coordination-and-live-resume/feature.md`
- `docs/plan/E38-shark-attack-team-orchestration/epic.md`
- `docs/plan/E38-shark-attack-team-orchestration/architecture.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-cross-epic-map.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-interaction-map.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F04-shark-attack-skill-and-role-protocol/spec.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F07-rider-execution-and-escalation/spec.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F08-shark-attack-v2-integrity-prerequisites/feature.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F05-reporting-and-operator-surface/{feature.md,assessment.md}`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F10-cross-provider-adapter-conformance/feature.md`
- `docs/plan/E38-shark-attack-team-orchestration/E38-F11-complicated-lifecycle-release-qualification/feature.md`
- `dev-artifacts/shark-attack-v2-plan/implementation-plan.md`
- `docs/plan/E39-question-and-decision-workflow-management/architecture.md`
- `docs/plan/E39-question-and-decision-workflow-management/E39-interaction-map.md`
- `docs/plan/E39-question-and-decision-workflow-management/E39-cross-epic-map.md`
- `docs/plan/E39-question-and-decision-workflow-management/E39-F02-serial-question-workflow-and-resolution-provenance/spec.md`
- `docs/plan/E39-question-and-decision-workflow-management/E39-F04-focused-question-read-surfaces-and-consumer-handof/spec.md`
- `internal/cli/commands/next.go`, `internal/cli/commands/plan.go`, `internal/cli/commands/related_docs.go`, `internal/cli/commands/question.go`
- `internal/runner/dispatcher.go`, `internal/runner/claude_dispatcher.go`, `internal/runner/codex_dispatcher.go`
- `internal/services/question_service.go`, `internal/models/question.go`
- `internal/research/validator.go`, `internal/sharkdata/default_data/workflow/tech-debt.yaml`
- `skills/shark-rider/verbs/run.md`, `skills/shark-attack/context/roster-schema.yaml`, `internal/sharkdata/default_data/manifest.yaml`
- Git evidence: `git log --all --grep=F08/F09`, `git merge-base --is-ancestor e079efd0 aa5a3438` (not an ancestor), branch `feature/E38-F05-council-artifacts` and worktree `.worktrees/E38-F05-council-artifacts/` (`spec.md`, `research-report.md`, `internal/runner/parent_control.go`, `docs/council/decisions/d-e38-f09-v2-owner-gates-001.yaml`), `shark get E38-F08 --json`, `shark history E38-F08 --json`

RECOMMENDED OUTCOME: pass
