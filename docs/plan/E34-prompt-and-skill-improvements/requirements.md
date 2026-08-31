# E34 Requirements

**Epic**: [Prompt and Skill Improvements](./epic.md)

## Overview

This catalog defines the epic-level capabilities. Feature files contain the
detailed field contracts, acceptance scenarios, implementation sequences, and
verification plans.

## Functional requirements

### Area 1: Cross-feature interaction lifecycle — E34-F03

**REQ-F-001 — Interaction map creation**

- Multi-feature epics produce a registered interaction map with stable I-##
  IDs, producers, consumers, shape sources, payloads, and styles.

**REQ-F-002 — Interaction preservation**

- Decomposition, feature specifications, tasks, test planning, review, QA, and
  UAT preserve every applicable I-## and one shared contract-test pointer.

**REQ-F-003 — Staged integration integrity**

- `live` remains the default. A predeclared `contract-only` edge records
  counterpart identity/status, shared evidence, activation owner, closure key,
  review basis, and demonstrability disposition without claiming live wiring.

### Area 2: Evidence and workflow composition — E34-F01/F02

**REQ-F-004 — Harness-aware rendering**

- Shark-owned prompt assembly remains compatible with supported providers and
  does not duplicate project or harness context unnecessarily.

**REQ-F-005 — Evidence-based demonstration**

- Demo claims distinguish runnable evidence, staged readiness, owner decision,
  open conditions, and demonstrability without granting acceptance authority.

**REQ-F-006 — Layered workflow extraction**

- Reusable skill content can be separated into workflow, prompt, methodology,
  and reference layers without breaking current consumers.
- **D-E34-LEGACY-PROMPTS-001 — RESOLVED 2026-08-31.** Neither legacy prompt was
  a shipped or owned deliverable in the F05-F09 packet. Both items are now
  resolved individually:
  - *Earlier ignored dev-artifact review prompt* — **CANCELLED**. An
    exhaustive search (`docs/`, `dev-artifacts/`, `shark search`) found no
    file, path, or specification for this artifact anywhere in the
    repository; it was referenced only in planning language and never
    materialized. No tracked work is created for it.
  - *skill-workflow-extraction prompt* — **TRACKED**. This artifact does
    exist, at `dev-artifacts/planning/skill-workflow-extraction-prompt.md`
    (dated 2026-06-22), and implements REQ-F-006's layered-extraction concept.
    It is now owned by `E34-F11-layered-skill-extraction-adoption`
    (`T-E34-F11-001`), which records the repository path and adoption
    pointer; no new prompt content was authored.
  - See the E34 epic decision note referencing D-E34-LEGACY-PROMPTS-001 for
    the full record.

### Area 3: Durable material Questions — E34-F04

**REQ-F-007 — Shared Question adoption**

- Decision-producing workflows create or reuse a linked Q### for material
  unresolved items and use the existing Question/council routing boundary.

**REQ-F-008 — Authoritative resolution**

- Question resolution points to the narrowest authoritative decision record and
  does not auto-close or silently block unrelated work.

### Area 4: Structured gate handoff — E34-F05

**REQ-F-009 — GateResult v1**

- Configured quality gates return one bounded canonical final JSON envelope:
  its outer fields own the opaque configured outcome and common evidence, and
  its versioned GateResult member owns findings, kickbacks, sweeps, impacts,
  and gate-specific summary without duplicate gate/outcome/evidence fields.
- Each structured route maps every opaque outcome key to a validated semantic
  role used only for success, route-owned rework, child-kickback rework,
  blocked, hold, or cancelled completeness rules.

**REQ-F-010 — Parent persistence before transition**

- Rider and the core runner validate, bind, and idempotently persist the gate
  result under its stable run identity and associated authorized parent session
  before any lifecycle transition.

**REQ-F-011 — Replay and failure safety**

- Exact replay is safe, conflicting replay fails, partial persistence resumes
  without duplication, and malformed structured output cannot advance work.
- The terminal result is write-once, and deterministic suboperation IDs let a
  restarted parent reconcile durable target records after every target
  commit/sidecar-update crash window.

**REQ-F-012 — Rider/core parity**

- Shared contract fixtures prove both execution paths accept, reject, persist,
  and route the same result shapes.

### Area 5: Defect-class completeness — E34-F06

**REQ-F-013 — Reusable class sweep**

- One canonical workflow defines class identity, search scope, enumeration,
  counts, instance evidence, dispositions, guard closure, and re-verification.

**REQ-F-014 — Backward-looking rework**

- Rework consults code, tests, decisions, tech debt, prior findings, specs, and
  standards before choosing or diverging from a repair design.

**REQ-F-015 — Evidence-based recurrence**

- Recurrence requires a repeated fingerprint or a new same-class instance
  inside a previously completed sweep; round number alone has no authority.

**REQ-F-016 — Conflict routing**

- Already-dispositioned findings remain visible without re-litigation absent
  new evidence, while severity conflicts use existing Questions or councils.

### Area 6: State and decision closure — E34-F07

**REQ-F-017 — Closed lifecycle specification**

- Behavior-bearing lifecycle and disposition fields have complete value and
  transition tables including failure, recovery, terminal, and invalid paths.

**REQ-F-018 — State-aware test design**

- State-transition and cross-entity decision-table techniques are mandatory
  when the behavior shape requires them.

**REQ-F-019 — Consumer impact discovery**

- Planning discovers consumers through I/X interactions and production caller
  paths, re-verifies shipped ACs, and rejects unexplained shared-name drift.

**REQ-F-020 — Decision propagation**

- Material Question, tech-debt, change, ADR, state, or design decisions account
  for every affected artifact and consumer through amendment or linked work.

### Area 7: Tier-consistent and integrated gates — E34-F08

**REQ-F-021 — Canonical tier matrix**

- SIMPLE uses feature/research and inline task evidence; STANDARD uses
  spec/test plan with merged review/QA; COMPLEX uses spec/test plan with
  separate review and QA; all tiers receive final UAT.

**REQ-F-022 — Executable evidence**

- Gate evidence records exact project-declared commands, working directory,
  exit status, runner counts, expected/unexpected skips, and bounded log
  pointers. Prose totals alone do not pass.

**REQ-F-023 — Epic integration review**

- Canonical epic workflow includes a final integration step over the complete
  accumulated diff and every completed/staged feature.

**REQ-F-024 — Integrated closure**

- Final review closes I/X interactions, I-03 sweeps/guards, I-04 impacts,
  findings, decisions, standards, and predicted debt.

**REQ-F-025 — Non-supersession authority**

- Final review adds a gate and cannot silently convert a rejected required
  feature gate into acceptance or introduce a global owner-approval setting.

### Area 8: Override drift and adoption — E34-F09

**REQ-F-026 — Drift status**

- `shark admin overrides status [--json]` emits deterministic digest-based
  current, upstream-changed, identical-redundant, orphaned, and
  baseline-unknown classifications.

**REQ-F-027 — Explicit baseline provenance**

- Shark stores canonical digests without override content, never advances a
  known baseline silently, and records a new baseline only after explicit
  operator acknowledgement.

**REQ-F-028 — Non-destructive upgrade visibility**

- Upgrade and dry-run include drift counts but never merge, rewrite, delete,
  disable, or expose override content.

**REQ-F-029 — WWGM reconciliation**

- One linked WWGM item promotes reusable content, removes stale overrides,
  rebases retained workflow policy, adds local safeguards, accounts for the
  E04-F02 record, and links or resolves CC-007/CC-008.

### Area 9: Proposal traceability

**REQ-F-030 — Complete accounting**

- Every E04 proposal item and current WWGM override has an owner and
  disposition in
  [E34-review-quality-improvement-plan.md](./E34-review-quality-improvement-plan.md).

**REQ-F-031 — Non-blocking benchmark follow-up**

- E40 receives later scenarios for tier routing, evidence fidelity,
  recurrence, integration closure, and override configurations without
  becoming an E34 dependency.

### Area 10: Product critical-path guard for delivery workflows — E34-F10

**REQ-F-032 — Durable product critical-path artifact**

- A durable product critical-path artifact records the current product-roadmap
  gate and the last passing production step, sourced from
  `docs/product/D01-vision-statement.md`, `docs/product/D02-success-criteria.md`,
  and `docs/plan/product-delivery-roadmap.md`.

**REQ-F-033 — Pre-dispatch delivery guard**

- Sprint planning/active/closing; epic assessment/decomposition/active; feature
  specification, test planning, task generation, task review, and approval; and
  task development completion reporting consult the guard before selecting or
  dispatching work, and report the current gate, proposed contribution,
  executable advancement evidence, unresolved prerequisites, and the
  disposition of side work.

**REQ-F-034 — Production-grade evidence only**

- Fixture, capture, hand-authored actor, contract-only, and component-suite
  evidence cannot satisfy a production product gate; only executable evidence
  against the live golden path advances it.

## Non-functional requirements

**REQ-NF-001 — Workflow compatibility**

- Existing non-gate output and projects without overrides remain compatible
  through explicit migration behavior.

**REQ-NF-002 — Workflow-driven status**

- Parsers and prompts preserve configured outcome and target-status values; no
  new hardcoded status route becomes authoritative.

**REQ-NF-003 — Parent authority**

- Dispatched workers never claim, advance, release, or force-set the entity
  being driven by the parent loop.

**REQ-NF-004 — Bounded and private evidence**

- Structured fields and collections are bounded and reject credentials,
  rendered prompts, transcripts, and unrestricted output.

**REQ-NF-005 — Project neutrality**

- Canonical policy does not require a specific language, test runner, database,
  provider, model, or project-local command.

**REQ-NF-006 — Deterministic verification**

- JSON ordering/fields, workflow routes, prompt rendering, digests, replay, and
  classification behavior are testable without an LLM policy simulator.

*See also*: [Scope boundaries](./scope.md)
