---
type: implementation-plan
epic: E34
source: wwgm-e04-review-gap-analysis
last_updated: 2026-08-05
---

# E34 Review Quality Improvement Plan

## Purpose

This plan translates WWGM's E04 review-gap proposal into reusable Shark
workflow capabilities and a bounded WWGM adoption path. It is the coverage
index for E34-F05 through E34-F09: every proposal item is implemented,
promoted, retained locally, deliberately modified, or explicitly rejected with
a reason.

## Evidence base

- WWGM `dev-artifacts/2026-08-04-1530-e04-review-gap-analysis/PROPOSAL.md`
- WWGM `dev-artifacts/2026-08-04-1530-e04-review-gap-analysis/INVENTORY.md`
- WWGM `shark-data/overrides/` and the current canonical counterparts
- Shark Rider's parent-owned run, task execution, and host adapter contracts
- Core runner dispatch, output parsing, transition, and Question handoff code
- Canonical feature and epic workflow YAML, quality prompts, and reusable
  Question/council skills
- E34-F03 staged-integration contracts and E34-F04 Question adoption
- E40 benchmark design, treated as later validation because implementation is
  underway and has only just started

## Systematic decisions

1. **Fix the authority boundary, not one prompt.** Gate workers return one
   structured result; Rider and the core runner persist it under the parent
   lease before transition.
2. **Escalate on evidence, not round number.** A repeated fingerprint or a new
   instance inside a claimed-complete class sweep is recurrence. Round-three
   and round-five rules are not adopted because they cannot distinguish new
   work from failed remediation.
3. **Keep canonical policy project-neutral.** Shark requires exact executable
   evidence and structural guards. WWGM supplies its Python commands,
   environment setup, standards, lint, and root agent guidance locally.
4. **Make final integration review additive.** It reviews the whole accumulated
   change but cannot silently supersede a rejected required feature gate.
5. **Treat overrides as versioned adoption surfaces.** Digests and explicit
   baselines expose drift; Shark never rewrites or merges user overrides.
6. **Reuse existing decision infrastructure.** Severity and materiality
   conflicts use E39 Questions and E38 councils. No recurrence store or new
   owner-approval configuration is added.
7. **Do not wait for E40.** E40 receives follow-up benchmark scenarios after
   its harness is ready; E34 implementation proceeds independently.

## Proposal coverage

| Proposal item | General disposition | Shark owner | WWGM disposition |
|---|---|---|---|
| P1 deterministic checks and truthful runner totals | ADAPT | E34-F08 defines exact command/exit/count/skip/log evidence without project commands | E34-F09's single WWGM item adds method-length, test-selection, test-DB, and unexpected-skip checks |
| P2 finding persistence and recurrence consumer | SPLIT AND ADAPT | E34-F05 owns structured parent persistence; E34-F06 owns evidence-based recurrence routing | Replaces local manual finding history after adoption |
| P2 round-three architect and round-five owner thresholds | REJECT | E34-F06 uses completed-sweep/guard evidence and existing Question/council materiality instead | No numeric hard stop added |
| P3 backward-looking rework and structural guards | PROMOTE | E34-F06 shared defect-class workflow | Project records and guards remain local inputs to the generic procedure |
| P4 rules index, selector, hook, and AGENTS channel | PARTIAL / DEFER | E34-F09 tracks drift and adoption; no generic selector/hook yet | Add a thin root `AGENTS.md`; defer `rules.py` and hook until multi-project or measured context evidence exists |
| P5 standards and bare-assert lint guard | KEEP LOCAL | E34-F06 defines generic guard closure; E34-F09 accounts for adoption | Add WWGM standards and scoped lint guard in the linked WWGM item |
| P6 lifecycle, state-transition, cross-feature, and naming checks | ADAPT AND PROMOTE | E34-F07 uses closed tables plus interaction/caller-path discovery instead of a one-FK-hop heuristic | WWGM consumes canonical planning rules |
| P7 decision propagation | PROMOTE | E34-F07 I-04 ChangeImpactSet | WWGM decisions/specs use the canonical sweep |
| P8 severity-conflict routing | PROMOTE | E34-F06 uses existing Question/council routing | Existing decisions remain evidence; new conflicts route durably |
| P9 whole-diff sweep and gate authority | ADAPT AND PROMOTE | E34-F08 adds an always-on epic integration review and explicit non-supersession rule | E34-F09 accounts for E04-F02's historical record correction in WWGM |
| Gap G staged edges and already-dispositioned findings | DO NOT REDO; CONSOLIDATE | E34-F03 remains staged-edge source; E34-F06/F08 promote reusable WWGM behavior | Resolve/link CC-008 and remove interim overrides after upstream adoption |
| STANDARD-tier artifact mismatch | RESOLVE EXISTING DECISION | E34-F08 pins STANDARD to merged review/QA and no standalone QA artifact | Resolve/link CC-007; do not add QA to every STANDARD feature |

## Feature plan

| Feature | Size | Primary deliverable | Depends on | Produces |
|---|---:|---|---|---|
| E34-F05 | 5 | GateResult v1, parent persistence, replay safety, Rider/core parity | None | I-02 |
| E34-F06 | 3 | Defect-class workflow, guard closure, recurrence/disposition/conflict routing | F05 | I-03 |
| E34-F07 | 3 | Closed state-space planning and decision impact propagation | F05 | I-04 |
| E34-F08 | 5 | Tier/evidence contract and epic final integration review | F06, F07 | I-05 |
| E34-F09 | 3 | Override drift CLI, baseline provenance, and WWGM reconciliation | F08 | WWGM adoption item and later E40 scenarios |

The dependency graph is stored in Shark and mirrored in
[E34-interaction-map.md](./E34-interaction-map.md).

## Override adoption decisions

### Promote upstream

- SIMPLE-lite and tier-consistent artifact expectations
- Declared staged-edge and final-closure checks
- Already-dispositioned recurrence and severity-conflict handling
- Cross-epic deferred-consumer closure where the consumer has not decomposed
- Generic executable evidence and defect-class guard requirements

### Retain in WWGM

- Epic workflow ordering
- `gpt-5.6-sol` review model assignments
- Method-length and test-selection implementations
- Test database environment setup and unexpected-skip policy
- Application coding standards, lint rules, and a thin root `AGENTS.md`

### Remove or rebuild

- Remove prompt and UAT rubric overrides once the promoted behavior ships and
  no local delta remains.
- Rebuild retained workflow overrides from the post-F08 canonical files so
  new steps, including `integration_review`, are not masked.
- Use drift status and I-05 to prove the action for every path. Do not infer
  safety from file age or a successful `shark admin upgrade` alone.

## Tracking rules

- F05–F09 remain in `draft` until implementation begins through Shark Rider.
- Their research reports are registered as feature related documents.
- Architecture, interaction map, and this plan are registered on E34.
- E34-F09 creates or reuses one WWGM change item only after I-05 exists. That
  item links or resolves CC-007 and CC-008 and includes the E04-F02 historical
  lifecycle reconciliation; it does not create a parallel set of cards.
- E40 work remains independent. Add benchmark scenarios after the harness
  exposes stable task/config fixtures.

## Explicit exclusions

- No automatic override merge, removal, or rewrite
- No global owner-approval config change
- No round-count escalation policy
- No separate QA artifact for STANDARD solely to satisfy a stale prompt
- No WWGM command, model, environment variable, or coding rule in canonical
  Shark content
- No upstream `rules.py` or editor hook without broader evidence
- No E40 dependency or benchmark target as an implementation gate

## Implementation handoff

Start with E34-F05. After it ships, E34-F06 and E34-F07 are independent and
may run in parallel if their live Shark dependencies and worktree state allow.
E34-F08 follows both; E34-F09 follows F08. Each feature's `feature.md` and
`research-report.md` contain the concrete requirements, acceptance scenarios,
implementation sequence, interactions, exclusions, and verification plan.
