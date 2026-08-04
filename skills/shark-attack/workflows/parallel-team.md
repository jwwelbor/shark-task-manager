# Parallel team: topology adapter, council wrap, and integration

## Goal

Run a selected delivery set through ordinary keyed Shark Rider parents without
adding a scheduler, a second claim store, a second Question lifecycle, or a
new runtime. This procedure is a topology adapter: selection stays with Shark,
each teammate stays the sole parent of one delivery entity, and the coordinator
owns only topology, refill, integration, the council interface, and the
bounded closing report.

Read `context/operating-model.md` and the active host provider reference
before choosing a topology. Reuse `skills/shark-rider/verbs/run.md` for every
delivery loop, `route-question.md` for routine E39 Questions, `council.md` for
material matters, and `resume.md` for bounded replacement. Do not copy their
procedures or reconstruct a rendered worker prompt here.

## Authority and bootstrap

- The **coordinator** selects keys, records topology evidence, refills idle
  teammates, owns Question responder routing and resolution, invokes the
  isolation integrator, and writes the closing report. It never claims,
  advances, or releases a delivery entity.
- A **teammate** receives one concrete delivery key at a time and is that
  entity's sole ordinary keyed Rider-loop parent: `shark next <key> --json`,
  claim, dispatch the exact `response.prompt`, heartbeat, semantic
  `status advance --outcome`, and release. A selector supplies a key only;
  it never supplies a claimable prompt.
- A **worker** is craft-only and returns the existing F09 control envelope to
  its teammate. It does not mutate Shark.
- The **integrator** exists only for isolation topology and is git-only. It
  never mutates Shark state.

Before bootstrap, discover and record the active host's team, follow-up,
interrupt, and isolation capabilities from `providers/`. A missing required
capability or topology evidence resolves to `Sequential`; never invent a host
command. Export `SHARK_CLAIM_TTL_SECONDS=1800` for this team session only
when `.sharkconfig.json` has no `claim_ttl_seconds` key. Do not add or override
that config key, set TTL to zero, use force-steal as normal recovery, or create
a persistent lease/claim store. Teammates heartbeat at each state change and
at least every ten minutes while waiting.

## Select and dispatch

Outside an active sprint, first run the non-mutating, all-inclusive direct-child
precheck `/shark-rider query: list <root> --all --json`. Call `shark plan <root> --json` only
when that precheck shows at least one non-terminal direct child. This precheck
is mandatory before initial selection and every refill: when every child is
terminal, `shark plan` auto-advances its parent and is not safe as an
inspection call. Treat its `select_task` or `parallel_candidates` response as
selection, not dispatch. For each selected delivery key that is not already in
flight and is eligible under the resolved topology, assign exactly one teammate
to run the canonical keyed Rider loop. The coordinator does not claim a
delivery entity or build a worker prompt.

Use F09's independent two-axis decision table. `Sequential` is the default.
`Parallel with ownership` requires recorded disjoint ownership/write-scope
evidence; `Parallel with isolation` requires recorded isolation evidence.
Missing ownership evidence and missing isolation evidence independently
degrade to `Sequential`. Producer/consumer order remains binding under either
parallel topology: integrate and make the producer available before assigning
its consumer, even with isolation evidence.

For shared-worktree ownership execution, parallel craft is allowed only with
the recorded disjoint scopes. The coordinator brokers one mutually exclusive
turn for commits and for the complete `make fmt && make lint && make test`
gate. Keep file-scoped staging discipline. There is no standing merge-referee
role and no concurrent shared-worktree gate run.

On each teammate completion, not only when a wave drains, repeat the
non-terminal direct-child precheck before selection and refill an idle teammate
with only a key not already in flight. `shark plan <feature>` is claim-aware;
in-flight dedup remains a race defense. Prefer a fail-routed entity for its
still-live teammate where that does not violate Question gating or topology
order.

A no-candidate result is never itself success. For a hierarchy run, confirm
completion with the all-inclusive terminal query `/shark-rider query: list <epic> <feature>
--all --json`, which must show every direct task terminal. For a sprint run,
separately inspect `shark sprint backlog <sprint-key> --all --json` and require
every assigned task, bug, change-card, and tech-debt item to be terminal. Do
not infer sprint completion from a hierarchy query. Otherwise report the
paused, blocked, claimed, or Question-gated work and its cause.

## Question holds and council routing

For a worker `kind: question` envelope, classify it against the material
threshold in `council.md` before minting any Question or council artifact.
The branches are mutually exclusive:

- For a **routine** question, the entity-parent teammate mints, configures,
  and links the scoped `Q###` through `route-question.md`. The coordinator
  then claims that Question, routes responders, records responses, and
  resolves it under the Question lease.
- For a **material** question, route directly through `council.md`. Do not
  mint or claim a `Q###`. The entity-parent teammate retains the delivery
  entity lease and heartbeats while the council route is unresolved; the
  coordinator routes the council responders without taking delivery-entity
  ownership.

Ready unrelated work continues through rolling refill.

The teammate keeps a Question-held worker live and heartbeats at least every
ten minutes. Holds are event-bounded, never controlled by a fixed Question
wait timeout:

1. When selection has ready work and no teammate is idle, direct the
   deterministically longest-held teammate to convert to the existing bounded
   handoff fallback. Record only entity key, Question key, and bounded evidence
   pointers; discard the worker, release the entity, and refill with the ready
   key. The open Question continues to gate the held entity.
2. At a run stop, resume boundary, or team cleanup, convert every live hold to
   the same bounded handoff before worker cleanup.
3. If all remaining work is Question-gated and selection has no ready key,
   report the open `Q###` keys to the owner. Do not call this completion.

## Sprint mode

With an active sprint, its backlog is the sole selection universe. Until
E19-F09 provides `shark plan sprint`, enumerate the active backlog in its
documented order (`sprint_order`, `execution_order`, `priority`, `assigned_at`),
filter in-flight and Question-gated keys, and choose the top eligible key per
idle workflow role. Do not use free selection or repeated `sprint next` calls
to construct a wave. Assign task, bug, change-card, and tech-debt keys
directly. Before expanding a selected feature or epic, run its all-inclusive
non-terminal direct-child precheck; call `shark plan <key>` only when that
precheck passes, never to inspect an already-terminal parent.

Sprint planning and retro are council ceremonies documented in `council.md`.
The coordinator may participate but only the owner starts or closes a sprint.

## Isolation integrator

For `Parallel with isolation`, provision one worktree and branch per teammate.
After a teammate reports done, the coordinator invokes one integrator to merge
that branch into the integration branch serially, then runs `make fmt && make
lint && make test` on the merged result. The integrator may resolve only
mechanical conflicts; a judgment-bearing conflict routes to the council.

On a red post-merge gate, run one scoped fix-forward on the integration branch
before another merge. Persist a bounded feature note and council handoff naming
the failure, fix commit, and evidence. Two consecutive fix-forward failures
escalate to the council; do not attempt a third. After a green gate, review the
worktree contents before removal and prune only the reviewed merged branch.
The integrator reports entity key, merge commit, and gate result to the
teammate; only the teammate advances and releases after green evidence.

## Closing report

After all delivery tasks are terminal, persist one bounded feature note with:

| Entity | Teammate | Semantic outcome | Merge commit | Gate result |
|---|---|---|---|---|
| `<key>` | `<member>` | `<outcome>` | `<commit or n/a>` | `<green/red/n/a>` |

Also record wave count, wall-clock duration, raised/resolved Question counts,
and fix-forward count. Keep escalations in the existing council ledger. Never
persist a rendered prompt, credential, token, unrestricted worker transcript,
or a second Question record in this report or any handoff.

## Result

Every selected delivery entity remains under exactly one ordinary keyed Rider
parent, topology evidence controls opt-in parallelism, Questions retain E39
and council authority, and the coordinator reports verified terminal or
paused/blocked state without owning delivery claims.
