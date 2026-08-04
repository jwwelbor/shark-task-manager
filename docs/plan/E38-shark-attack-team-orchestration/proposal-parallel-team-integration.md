---
title: Parallel Team Integration — run-agent-team × shark plan × shark-attack
status: proposal v4 (decision-complete; sprint contract added)
date: 2026-08-02
relates_to: E38-F09 (provider-neutral coordination and live resume), E39 (questions), /run-agent-team, /run-sprint-team
review: architect review incorporated 2026-08-02 (3 authority leaks fixed, live-resume trade made explicit); lease policy (30-min scoped TTL), event-bounded question holds, integrator role, and sprint contract (planning/execution/retro) recorded per owner + architect consults
---

# Proposal: Parallel Team Integration

**One sentence:** make `/run-agent-team` a *topology adapter* under E38's two-axis
model — `shark plan` (or the active sprint backlog, §5) selects the parallel
candidates, each teammate runs the ordinary keyed Rider loop as the parent of
its assigned entity, the coordinator owns integration (§4) and the council
interface, and the shark-attack council wraps the run — questions, escalations,
sprint planning, and retro.

Nothing here adds a runtime, scheduler, claim store, or workflow engine
(REQ-NF-001, ADR-007). Every deliverable is prompt/skill-layer content.

---

## 1. Problem

We have three coordination assets that do not currently compose:

| Asset | Owns | Gap |
|---|---|---|
| `shark plan` / `shark next` | Selection, dispatch, claims/leases, canonical prompts (TDD lives here), question gates | Read-only selection returns `parallel_candidates`, but nothing consumes them as a team |
| `/run-agent-team` | Parallel execution via the agent-teams primitive | Hand-rolls its own DAG/wave computation from `order` + `shark links`; teammates mutate shark state with **no claim**; single worktree only; no integration gate; references an `orchestration` skill that is not installed (it lives only under `dev-artifacts/shark-cleanup/`) |
| `shark-attack` (E38) | Judgment: roster, chair, durable decisions/handoffs/escalations, resume | Defines the council but names no concrete parallel-execution mode it wraps |

The /insights "parallel execution orchestrator" prompt is a one-session version
of the same idea with zero durable state and zero shark integration. Its two
genuinely missing mechanics — post-merge integration gating (now the
**integrator role**, §4) and the **closing report** — are incorporated below. Its TDD worker contract is
**not** incorporated: TDD instructions already arrive in the canonical dispatch
prompt (`shark next` → `response.prompt`), and duplicating them at the team
layer would create a second source of truth.

## 2. Proposed shape

### Roles and authority

- **Coordinator (the team lead, current session)** — bootstraps the team; owns
  *integration and the council interface*: merge-in via the integrator,
  responder routing and resolution for Questions, closing report. Named
  "coordinator" to avoid overloading the product-manager *persona* with this
  *process* role (see §5 role naming). The lead **never claims delivery
  entities while the team runs**. It does hold `Q###` leases (council interface) and, after
  team cleanup, it runs the ordinary single-entity keyed loop for the feature
  root itself to advance the feature.
- **Teammates (agent-teams primitive)** — each teammate is a full Claude Code
  instance running the **ordinary keyed Rider loop** for one assigned entity at
  a time: `shark next <key> --json` → `claim` → dispatch canonical prompt →
  `heartbeat` → `status advance --outcome …` → `release`. One parent per
  entity, exactly as ADR-005 requires, and exactly the REQ-F-017 re-entry rule:
  a selector supplies a key, `shark next <key>` supplies the dispatch.
  Teammates never select their own work from shark; they take keys from the
  team task list, which the lead fills from `shark plan`.
- **Workers** — subagents the teammate dispatches with the canonical prompt,
  craft-only, returning evidence and a control envelope (unchanged from F09).
- **Council (shark-attack)** — the wrap. Roster, chair, `docs/council/` ledger,
  escalation routes. Reached through the lead when a teammate reports
  `needs_council` or `CRITICAL_FAILURE:`; reached automatically for
  `question` envelopes via E39 (§3).

With teammates holding real shark leases, the agent-team task list is
demonstrably a coordination overlay, not a second claim store — the lease
underneath is authoritative.

### The loop (lead's procedure) — rolling refill, no barrier

```
1. shark plan <feature> --json          (no active sprint — free selection;
                                         active sprint: see §5 sprint mode)
     → select_task            : one candidate (degenerate wave)
     → parallel_candidates    : up to max_parallel_items candidates
2. Create one agent-team task per candidate key not already in flight.
   Task body: "Run the keyed Rider loop for <KEY>" + topology rules (§5).
3. Teammates claim team-tasks and run their Rider loops in parallel.
4. On EACH teammate completion (not on wave drain):
     a. integrator merge-in and closeout (§4) if running isolated worktrees
     b. re-run shark plan <feature> and top up idle teammates immediately
     c. a fail-routed entity (outcome sent it backward) is preferentially
        reassigned to the same teammate while it is alive — it has the context
5. Exit condition — verified, not inferred:
     shark plan returning no candidate is NOT success (it also happens when
     everything is question-blocked, paused, or claimed). The loop ends only
     when `shark list <epic> <feature> --json` shows every task terminal.
     Otherwise the lead reports exactly what is blocked/paused and on what.
6. All tasks terminal → closing report (§7) → clean up the team → lead runs
   the ordinary keyed loop on the feature root to advance the feature.
```

This **replaces** run-agent-team Steps 1–1.5 (read `order`, fetch `shark links`
per task, pick order-waves vs dep-graph vs empty-model). `shark plan` already
applies execution order, priority, dependency gating, question blocks, and the
`max_parallel_items` cap — deterministically, from the database. The skill
stops re-deriving state shark already owns; the old computation is deleted, not
kept as a fallback. The empty-model "architect-inference" branch survives as-is
(planning aid, not execution).

> **V-001 (verify during F12 spec):** confirm whether keyed hierarchy selection
> (`shark plan <feature>`) filters actively-claimed children. If yes, rolling
> refill is free; if no, the lead must dedupe candidates against in-flight keys
> (step 2 already words this defensively). This matters doubly for parked
> entities (§3), which are *both* claimed and question-gated: verify neither
> state alone is being relied on to keep them out of the ready set.

### Lease discipline (recorded decision — TTL stays on)

Persistent leases (`claim_ttl_seconds: 0`) were considered and **rejected**:
lease expiry is the only automatic liveness signal shark has, and this
proposal's own resume story depends on it — dead-teammate-with-live-lease is a
*routine* team scenario (agent-teams do not survive `/resume`), and with
expiry disabled every resume needs force-steal or session-ID forensics as the
normal path. TTL=0 would also break F09's own bounds (REQ-F-007/009/010 all
assume leases can expire) and, being a global key, would strand entities for
crashed solo Rider runs too. The two failure modes are asymmetric: false
expiry of a live teammate is fully preventable by heartbeats; slow reclaim of
a dead teammate is bounded, visible, and self-heals. So: bias the TTL up, keep
expiry on.

**Team runs use a 30-minute claim lease**: the lead exports
`SHARK_CLAIM_TTL_SECONDS=1800` at bootstrap; `.sharkconfig.json`'s
`claim_ttl_seconds` stays **unset** (the env var only applies while the config
key is absent — adding the key later silently kills the override). Lease
expiry is the run's liveness signal and its resume mechanism: after an
interrupted run, dead teammates' claims self-expire and their entities return
to the ready set within ≤30 minutes; no force-steal on the normal path.

- Teammates heartbeat at every state change and **at least every 10 minutes**
  while waiting (merge queue, question hold); workers are dispatched in the
  background so the teammate retains control to heartbeat.
- A lease that expires between merge and advance means the teammate died —
  re-dispatch is then the correct recovery, and the re-dispatched worker finds
  the already-merged work and returns `pass`.
- The gate-result→advance handoff is an explicit message contract: the lead
  sends the teammate `{entity_key, merge_commit, gate: green|red}`; on green
  the teammate advances (with `--session $SID --from-status <status>` when
  `advance_guard` is enabled) and releases; on red see §4.

### Mapping onto F09's axes

| F09 axis | Value | Realized by |
|---|---|---|
| Coordination | `Direct` | `/run` — no team, no council |
| Coordination | `Batch` | `/run-agent-team` without council roster |
| Coordination | `Council` | `/run-agent-team` + shark-attack wrap (this proposal) |
| Topology | `Sequential` | `/run`, `/run-sprint` — **the evidence-gated default (REQ-F-002)** |
| Topology | `Parallel with ownership` | Team, single shared worktree — requires recorded ownership evidence |
| Topology | `Parallel with isolation` | Team, per-teammate worktrees + integrator merge-in/closeout (§4) — requires isolation evidence, entered on file-conflict evidence |

Axes stay independent (REQ-F-001/002): a council can wrap a sequential run; a
batch can run isolated. Per REQ-F-002 the default is `Sequential` — a team
topology is only entered when ownership or isolation evidence is recorded, and
degrades back to `Sequential` when it cannot be produced. The agent-teams
primitive itself is a **host capability** (env var + version ≥ 2.1.32, Claude
Code only) discovered under REQ-F-012 *before* topology selection: capability
absent → documented fallback to `/run-sprint`, never an invented command.

## 3. Council wrap — what "answer problems" concretely means

When a teammate's worker returns a control envelope that is not `final`
(REQ-F-003):

- **`question`** → ownership follows F09 (the *entity parent* mints the
  Question): the **teammate** creates `Q###`, configures owner/responders,
  links it `question_blocks` to its entity (REQ-F-004), then applies the
  live-resume rule below. The **lead** owns responder routing (`shark next
  Q### --json`), transcribes answers via `shark question respond` under its
  `Q###` lease, and closes with `shark question resolve` (REQ-F-005/006).
  While the Question is open, shark's gate pauses that entity and *the rest of
  the team keeps running* — the payoff of per-entity claims. No bespoke
  question store.
- **`needs_council`** → chair convenes per `workflows/escalate.md`; outcome
  recorded in `docs/council/decisions/` or `escalations/`; entity resumes or
  parks.
- **`CRITICAL_FAILURE:`** (build/test/infra broken project-wide) → lead pauses
  all teammates, records the escalation in the durable ledger, surfaces to the
  owner.

### Live resume under a team — recorded decision: persistent hold, event-bounded

F09's headline capability is same-worker follow-up on a question answer
(REQ-F-008). A naive team flow (teammate releases the blocked entity and moves
on) silently forfeits the live worker and makes F09's *fallback* (replacement
worker from a bounded handoff) the team-topology *norm*. Recorded decision
(owner, architect-endorsed): **persistent hold with event bounds, not a
clock**.

A teammate holding a question-blocked entity keeps its worker live and waits
for the answer — no fixed timebox. Two event-based bounds apply:

1. **Starvation conversion** — when `shark plan` has a ready candidate and no
   teammate is idle, the lead directs the *longest-held* teammate (oldest
   worker context is most likely stale anyway; the rule stays deterministic)
   to convert to the bounded-handoff fallback: record the handoff (entity key,
   question, evidence pointers — no rendered prompt, REQ-NF-003), discard the
   worker, release the entity, take the new key. The Question stays open and
   the entity stays question-gated, so it cannot be re-dispatched until
   resolved; on resolution, re-plan surfaces it and exactly one replacement
   worker starts from the handoff (REQ-F-008 fallback, AC-021 semantics).
2. **Session boundary** — at run stop, lead resume, or team cleanup, every
   held entity is converted to a bounded handoff before shutdown. Live workers
   do not survive the session, and the protocol does not pretend otherwise: a
   "wait indefinitely" beyond the session boundary is an illusion, so the
   replacement path is what the owner gets there under any policy.

While parked, the teammate heartbeats its claim at least every 10 minutes and
re-checks Question status. The "remaining claim lease" bounds in
REQ-F-007/009 stay meaningful: heartbeats keep the lease alive during a hold —
the lease bounds *liveness*, not work duration.

**All-parked deadlock rule:** if every teammate is parked on a question and
`shark plan` has no ready candidate (everything gated), the starvation rule
never fires and the team correctly idles — the lead must then surface
"N open questions gate all remaining work: Q###, Q### …" to the owner rather
than sitting silent.

Resume of the *team* is inherited, not invented: agent-teams cannot restore
teammates after `/resume`, but shark claims/statuses and the council ledger are
durable, so a fresh lead re-runs the bootstrap and `shark plan` returns the
current ready set (shark-attack `resume.md`).

## 4. Integrator role — merge-in and worktree closeout (isolation topology only)

Recorded decision (owner, 2026-08-02): there is **no standing merge-referee
procedure**. Per-worktree discipline — TDD, commits, quality gate, reviews
diffing against `git merge-base main` — already lives in the canonical
dispatch prompts and is not duplicated at the team layer. What the prompts do
**not** cover (verified: no dispatch prompt performs a merge; the host-side
`finish-feature` skill covers only the single-branch endgame) is the
integration act itself. That gap is filled by a thin **integrator** role: an
agent the lead dispatches to merge in and close out completed worktrees.

1. One git worktree + branch per teammate (per assigned entity), created by the
   lead at dispatch.
2. When a teammate reports done, the lead dispatches the integrator for that
   worktree. The integrator determines merge order and strategy, merges the
   branch into the integration branch **serially** (one integrator, one
   integration branch), and runs `make fmt && make lint && make test` on the
   merged result — the one gate the worktree's own prompt-driven checks cannot
   have covered.
3. Merge conflicts the integrator can resolve mechanically, it resolves;
   anything judgment-bearing (conflicting designs, overlapping refactors) goes
   to the council, not to a silent resolution.
4. Red after a merge → one fix-forward worker scoped to the failure, on the
   integration branch, before merging anything else. Fix-forward craft is
   otherwise invisible to shark, so each fix-forward **must** leave a durable
   trace: a note on the feature (`shark create note <FEATURE> …`) and a council
   handoff naming the failure, the fix commit, and the evidence. Two
   consecutive fix-forward failures → council escalation, not a third retry.
5. Closeout: after a green merge, the integrator reviews the worktree's
   remaining contents before removal — **never force-remove a worktree
   unreviewed** — and prunes the merged branch.
6. Merge commits reference the entity key; the teammate's `status advance`
   happens only after its merge passes the gate (§2 lease-discipline contract —
   the gate result is the teammate's evidence). The integrator is git-craft
   only: it never touches shark state.

**Ownership topology (shared worktree)** does not get the referee, but needs
its own serialization rules (architect finding): commits and quality-gate runs
are mutually exclusive across teammates — one teammate commits/gates at a time
(lead-brokered turn), because concurrent `make test`, the git index, and
`shark-tasks.db` all collide in a single tree. File-scoped `git add` rules from
the current skill are kept verbatim.

## 5. Sprint contract — planning, execution, retro

Sprints enter the contract as two different kinds of thing: planning and retro
are **council ceremonies**; execution is a **coordinator selection boundary**.
Shark's constraint is respected throughout: only one sprint is active at a
time (`SprintService.StartSprint` enforces it) — multi-sprint value comes from
**pipelining**: the council plans S(n+1) while the team executes S(n).

**Sprint planning (council ceremony).** The product-manager chairs; the
scrum-master supplies capacity, velocity, and readiness evidence from the
sprint planning surfaces (`shark sprint plan / readiness / velocity /
capacity`); the business-analyst confirms acceptance-criteria readiness. The
proposed scope is recorded as a decision in `docs/council/decisions/`
**including the evidence snapshot** (so retro can compare planned vs. actual),
and entities are staged with `sprint add` while the sprint remains in a
planning state. **Only the owner starts a sprint**: the council proposes, the
owner activates. Roster roles gain no sprint authority beyond staging
reversible planning data.

**Sprint execution (coordinator selection).** While a sprint is active, the
sprint backlog is the coordinator's sole selection universe; with no active
sprint, free selection via `shark plan` applies unchanged (§2). Selection is
read-only and carries no dispatch metadata — the assigned teammate's keyed
`shark next` remains the only prompt/dispatch source. Mechanism note, verified
in code (`GetNextTask`, `internal/services/sprint_service.go:1512`): the
sprint selector inspects no claims and returns the *same top item* until its
status changes, so repeated `sprint next` calls cannot build a wave. The
coordinator therefore enumerates the sprint backlog and applies the selector's
documented order (sprint_order → execution_order → priority → assigned_at)
client-side, filtering in-flight and question-gated keys and taking the top
eligible item **per idle teammate role** (`--agent` filters by the workflow
step's role, not the roster).

*Upstream ask (recorded 2026-08-02; Go work owned outside F12 as its own
Shark feature): unify sprint selection under the plan surface rather than
patching the sprint selector. The sprint becomes a **selection root on
`shark plan`** — `shark plan sprint` returns the standard read-only selection
JSON (`parallel_candidates` capped by `max_parallel_items`, claim-aware and
question-gate-aware like the other collection roots) ordered by the sprint's
four-tier order; a keyed planning-state form (`shark plan S###`) gives the
planning ceremony a read-only execution preview; `shark sprint next` is
demoted to a compatibility alias returning the top candidate (equivalent to a
`--sequential` collapse) through a deprecation window. This supersedes the
earlier minimal ask (a claim-aware exclusion flag on `sprint next`). When it
ships, the client-side enumeration above collapses to one `shark plan sprint`
call and remains documented only as the fallback for older shark versions.*

**Entity-type rule.** Task, bug, change-card, and tech-debt items are assigned
to teammates directly — the keyed Rider loop is entity-agnostic. A **feature
or epic** item is expanded by the coordinator via `shark plan <key>` and run
as a sub-wave under the existing rules; **parent keys are never assigned to
teammates** (that would nest coordination inside a teammate and break
one-parent-per-entity).

**Sprint retro (council ceremony).** The scrum-master chairs after close;
`retro-sprint` produces its report in its existing location. The council
ledger records a bounded entry only — report pointer, adopted process
decisions, follow-up owners. Never copy the report into `docs/council/`
(bounded paths and metadata, never transcripts).

**Role naming.** The run lead is the **coordinator**: it participates in
ceremonies but chairs neither planning (product-manager) nor retro
(scrum-master); tie-breaks escalate to tech-director as always. When one
session wears several hats the separation is procedural, not enforced — its
value is ledger clarity (who decided as which role) and clean handoff when
the roles are separate agents.

**Not pursued:** concurrent active sprints — Go schema/service work with no
demonstrated need; pipelining delivers the multi-sprint value. Flagged as a
possible future feature owned outside F12.

## 6. Conflicts surfaced (not averaged)

1. **Teammates advancing shark with no claim.** The current run-agent-team task
   template has teammates run `shark status advance` directly and never claim.
   That predates the claim-lease model and violates ADR-005/F09 ownership.
   **Resolution:** teammates run the full keyed Rider loop as the parent of
   their assigned entity. The old template is flagged for cleanup, not kept
   alongside.
2. **Orchestration skill not installed / two homes.** `/run-agent-team`
   references `skills/orchestration/workflows/run-agent-team.md`, which exists
   only under `dev-artifacts/shark-cleanup/`, and the command file itself is
   host-side (`~/.claude/commands/`), outside the repo parity gate.
   **Resolution:** the canonical home is
   `skills/shark-attack/workflows/parallel-team.md` in the authored tree +
   embedded mirror (parity gate, REQ-F-016). The host command becomes a thin
   pointer at it and carries no procedure of its own.
3. **Hand-rolled DAG vs `shark plan`.** Both compute ready sets today.
   **Resolution:** `shark plan` wins (deterministic, DB-backed, question-aware).
   The order/links reading in the skill is deleted, not kept as fallback.
4. **Question minting: teammate vs lead.** v1 of this proposal centralized
   `Q###` creation at the lead; F09 assigns it to the entity parent.
   **Resolution:** F09 wins — the teammate mints and links; the lead routes and
   resolves (§3).
5. **`/run-sprint-team` vs sprint-mode coordinator.** The sprint execution
   contract (§5) *is* run-sprint-team generalized — one persistent team
   draining the sprint in sprint order, with feature grouping emerging
   naturally from feature expansion. Composing them would nest two team
   bootstraps. **Resolution:** supersede — `/run-sprint-team` is rewritten as
   a thin alias into the new workflow's sprint mode; the old group-by-feature
   procedure is flagged for cleanup.

## 7. Closing report

The lead ends every team run with:

```
| Entity | Teammate | Outcome | Merge commit | Gate result |
|--------|----------|---------|--------------|-------------|
| E##-F##-### | dev-1 | pass | abc1234 | green |
...
Waves: N   Wall-clock: H:MM   Questions raised/resolved: Q/Q   Fix-forwards: F
```

plus a shark note on the feature so the summary is durable, and council ledger
entries for anything escalated. (No serial-time estimate column — t-shirt-size
arithmetic is false precision; wall-clock and wave count carry the signal.)

## 8. Deliverables and sequencing

All prompt/skill-layer; no Go except where F09 already requires it. Proposed as
a follow-on feature (working title: **E38-F12 Parallel team topology**)
sequenced **after** F09's skill restructure (REQ-F-015) to avoid churning files
F09 renames. F12 inherits F09's foundation risk (F05/F08 report `completed`
with no trunk implementation), so its spec stays thin until F09 merges.

Phased to keep the first slice small:

**Phase 1 — ownership topology + council wrap + sprint contract**
1. `skills/shark-attack/workflows/parallel-team.md` — coordinator procedure
   (§2 loop, §3 council wrap, §5 sprint mode, §7 report) for the
   shared-worktree topology, including the serialization rules from §4.
   Embedded mirror synced (parity gate).
2. Revision of the host `/run-agent-team` command into a thin pointer;
   teammate task body becomes the keyed Rider loop; Steps 1–1.5 replaced by
   `shark plan`; capability preconditions kept.
3. Sprint planning and retro ceremony additions to the shark-attack council
   workflows; `/run-sprint-team` rewritten as a thin alias into sprint mode
   (§6 conflict 5).
4. Resolve V-001 (claim filtering in keyed plan) during spec. The upstream
   sprint plan-root ask (`shark plan sprint`, §5) is filed as **E19-F09
   Sprint Selection Root on shark plan** (created 2026-08-02, size M, this
   proposal linked as its related doc).

**Phase 2 — isolation topology**
4. The integrator role (§4): worktree-per-teammate creation, serial merge-in +
   post-merge gate, fix-forward protocol with durable traces, reviewed worktree
   closeout. Deferred until Phase 1 is exercised — current single-worktree pain
   is survivable short-term — but the surface is now thin (one agent role, not
   a standing referee procedure), so pulling it forward when parallel runs
   start is cheap.

**Not built:** no team runtime or dispatcher (primitive owns parallelism), no
second claim store (shark leases only), no writing synthetic edges or waves
back to shark, no per-teammate provider adapters beyond F09's Codex/Claude
references, no new tables or migrations.

## 9. Recorded decisions (2026-08-02, owner + architect)

1. **Lease policy:** keep TTL expiry; team runs use `SHARK_CLAIM_TTL_SECONDS=1800`
   exported by the lead at bootstrap; `claim_ttl_seconds` stays out of
   `.sharkconfig.json`. Persistent leases (TTL=0) rejected — expiry is the
   liveness signal and the resume mechanism (§2). Note: the earlier belief that
   TTL=0 was already configured was incorrect — the repo has always run the
   15-minute default, so adopting 1800 only makes behavior *more* forgiving.
2. **Question holds:** persistent live hold, bounded by events (starvation
   conversion, session boundary) rather than a clock (§3).
3. **Merge discipline:** no standing merge-referee procedure — per-worktree
   craft/gate/review discipline stays exclusively in the canonical dispatch
   prompts; the team layer adds only the thin **integrator** role for
   merge-in, post-merge gating, and reviewed worktree closeout (§4), deferred
   with the isolation topology to Phase 2.
4. **Sprint contract (owner-initiated, architect-endorsed):** sprint planning
   and retro are council ceremonies (product-manager chairs planning,
   scrum-master chairs retro; **the owner alone starts a sprint**); sprint
   execution is a coordinator selection boundary (active sprint backlog =
   sole selection universe — client-side ordered enumeration as the interim,
   collapsing to `shark plan sprint` once the sprint plan-root ships, §5);
   concurrent
   active sprints not pursued — the council pipelines S(n+1) planning during
   S(n) execution (§5). Supersedes `/run-sprint-team`'s group-by-feature
   procedure; resolves the former sprint-drift open question.

## 10. Open questions for the owner

None — the proposal is decision-complete and filed:

- **E38-F12 Parallel Team Topology** (created 2026-08-02, size M) carries the
  skill-layer work; this proposal is its governing design doc and stays the
  single source for the recorded decisions in §9. Thin spec until F09 merges
  (§8); driven through the workflow after F09.
- **E19-F09 Sprint Selection Root on shark plan** (created 2026-08-02,
  size M) carries the upstream Go work (§5 upstream ask).
