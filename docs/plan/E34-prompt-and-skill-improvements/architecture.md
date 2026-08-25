---
type: architecture
epic: E34
last_updated: 2026-08-05
---

# E34 Workflow Quality Architecture

## Purpose

This architecture defines the shared contracts introduced by E34-F05 through
E34-F09. It keeps workflow policy in reusable bundle content, lifecycle
authority in the parent orchestrator, project commands in each project, and
durable evidence in existing Shark records.

## Design principles

1. **Parent owns state.** A dispatched worker returns bounded evidence. The
   parent binds it to the active entity and lease, persists it, applies
   validated kickbacks, advances by the configured outcome, and releases.
2. **Validate before object access or mutation.** Model output is untrusted.
   Decode the top-level shape, version, fields, bounds, and workflow outcome
   before any persistence call.
3. **One producer contract, many consumers.** Prompts and skills reference the
   shapes here instead of copying field lists.
4. **Evidence before escalation.** Defect recurrence and decision impact come
   from completed scopes and durable records, not round counts.
5. **Canonical policy, local commands.** Shark defines evidence requirements;
   a project defines the executable commands, environment, standards, and
   model choices that satisfy them.
6. **No silent authority changes.** Final integration review adds assurance but
   never rewrites an independent feature verdict.

## Component boundaries

| Component | Owns | Does not own |
|---|---|---|
| Gate worker | Gate analysis and one bounded result | Claims, status mutation, note persistence, release |
| Worker-control/GateResult parser | Canonical final-envelope selection, nested GateResult decoding, schema validation | Entity lookup or writes |
| Parent result coordinator | Entity/run/session binding, replay state, persistence order, kickbacks | Gate reasoning or workflow outcome definitions |
| Workflow service | Valid outcomes and target status resolution | Interpreting gate prose |
| Note and entity services | Durable typed notes and validated status operations | Reading unvalidated model output |
| Embedded quality content | General tier, sweep, planning, and review policy | Project-specific commands or standards |
| Project guidance | Exact validation commands, environment, standards, local policy | Canonical Shark semantics |
| Override drift service | Digests, baselines, classifications | Override content merge, deletion, or rewrite |

## Gate result flow

1. The parent reads the live entity, current route step, its `result_contract`,
   configured outcomes, durable `run_id`, active lease session, and worker
   retirement state.
2. The worker returns the existing single worker-control envelope with
   `kind: final`. It carries the opaque `recommended_outcome`, bounded common
   `evidence`, and, only for `result_contract: gate_result_v1`, one nested
   `gate_result` object. E34 adds no second marker or outcome field.
3. The parser validates the canonical worker-control top-level shape first,
   then the nested GateResult version, bounds, collection invariants, forbidden
   markers, and equality between its `outcome` and `recommended_outcome`.
4. The parent binds the result to entity key, entity type, source status, gate,
   durable `run_id`, and current session. The renewable session is associated
   provenance, not the replay identity; none of these fields are trusted from
   worker output.
5. The coordinator canonicalizes the validated envelope and computes an
   operation digest from the stable bound identity plus envelope bytes.
6. Replay reads a durable operation state. `transition_applied` plus the
   expected live target returns success; `persistence_complete` resumes the
   guarded transition; a partial persistence state resumes its next operation.
   A different digest for the same stable identity is a conflict.
7. The coordinator applies individually idempotent persistence operations in
   this order: gate summary/evidence, findings and sweeps, task kickbacks, then
   `persistence_complete`. Every operation carries the operation digest in
   metadata. A retry skips completed identical operations and resumes the rest.
8. The parent applies the workflow transition exactly once from the recorded
   source to the resolved target, verifies an already-applied identical target,
   then records `transition_applied`. It releases the lease only after terminal
   worker-retirement evidence; a crash after transition but before release
   resumes release rather than repeating the transition.

Cross-entity note and task writes do not need a new distributed transaction.
Validation completes before the first write, every write has a stable replay
identity, and transition is gated on the completed operation set.

## I-01 ReadinessEvidence v1

I-01 remains the E34-F03 producer-to-E34-F02 consumer handoff. Its canonical
shape is reproduced here so every E34 interaction resolves to architecture.

| Field | Meaning |
|---|---|
| `assessor_verdict` | Independent UAT assessment, unchanged by an owner decision |
| `owner_decision` | Separate approval or override decision with conditions |
| `open_conditions` | Unclosed activation or acceptance conditions |
| `gate_mode` | `live` or predeclared `contract-only` |
| `activation_owner` | Feature responsible for deferred production activation |
| `closure_key` | Shark key that closes the obligation |
| `counterpart_status` | Counterpart status read live from Shark at review time |
| `review_basis` | Accumulated branch and shared-contract evidence reviewed |
| `demonstrability_disposition` | Verified-now, pending-integration, or other evidence-grounded result |

## I-02 GateResult v1

The worker-owned JSON object has these fields. Parent-owned entity, source
status, and lease fields are deliberately absent.

| Field | Type | Required | Contract |
|---|---|---|---|
| `schema_version` | integer | Yes | Exactly `1` |
| `gate` | string | Yes | Current configured gate identifier |
| `outcome` | string | Yes | Exact key in the current workflow step's outcomes |
| `summary` | string | Yes | Trimmed bounded summary; no transcript or prompt content |
| `evidence` | array of `EvidenceRef` | Yes | At least one bounded pointer; no raw unrestricted logs |
| `findings` | array of `Finding` | No | Unique fingerprints within the result |
| `kickbacks` | array of `Kickback` | No | Unique entity keys within the result |
| `remediation_sweeps` | array of I-03 | No | Unique `class_key` values |
| `change_impacts` | array of I-04 | No | Unique `source_kind` plus `source_key`; planning gates only |
| `no_kickback_reason` | string | Conditional | Required when a failing result has no kickback |

`EvidenceRef` contains `kind`, `pointer`, and an optional bounded `summary`.
For executable evidence it also contains exact `command`, `working_directory`,
`exit_code`, runner-native `counts`, `expected_skips`, and
`unexpected_skips`. The pointer identifies the retained bounded log or report;
the envelope never embeds the log.

`Finding` contains `severity`, `class_key`, `class_statement`, `fingerprint`,
`affected_ids`, evidence pointers, `disposition`, and a conditional
`disposition_pointer`. Allowed dispositions are `open`, `fixed`,
`already_dispositioned`, `severity_conflict`, and `not_reproducible`.

`Kickback` contains `entity_key`, `target_status`, and `reason`. The target is
an opaque value validated against the target entity's workflow; no parser
hardcodes `development` or another project status.

Invariants:

- A pass outcome contains no `open` or `severity_conflict` blocking finding.
- A fail outcome with a rework target contains at least one kickback. A fail
  outcome without a rework target contains a non-empty
  `no_kickback_reason`. Findings may accompany either case but never substitute
  for the required routing explanation.
- Gate identity text is at most 256 bytes, summaries are at most 1,000 bytes,
  pointers are at most 2,048 bytes, each collection has at most 100 entries,
  and the canonical nested GateResult is at most 256 KiB. These constants live
  in one GateResult model package and are tested at limit-1, limit, and limit+1,
  including empty and aggregate-size cases.
- A disposition pointer is mandatory for `already_dispositioned` and
  `severity_conflict` when a prior decision exists.

## I-03 DefectClassSweep v1

| Field | Type | Contract |
|---|---|---|
| `class_key` | string | Stable normalized class identity |
| `class_statement` | string | One-line general class, not the point instance |
| `search_scope` | array | Concrete modules, patterns, contracts, and durable records searched |
| `prior_designs` | array | Prior TD, DEC, Question, note, spec, or standard pointers considered |
| `searched_count` | integer | Number of candidate sites evaluated |
| `matching_count` | integer | Number of class instances found |
| `instances` | array | Fingerprint, site pointer, disposition, and evidence for every match |
| `fixed_count` | integer | Matching instances repaired |
| `dispositioned_count` | integer | Matching instances covered by cited decisions |
| `open_count` | integer | Matching instances still open |
| `guard` | object | Kind, implementation pointer, counterfactual verification pointer, status |
| `status` | enum | `open` or `complete` |

For a complete sweep,
`matching_count = fixed_count + dispositioned_count`, `open_count = 0`, every
instance is represented exactly once, and the guard status is `verified`. A
future same-class instance is recurrence only when its fingerprint repeats or
its site is inside this completed search scope.

## I-04 ChangeImpactSet v1

| Field | Type | Contract |
|---|---|---|
| `source_kind` | enum | Question, tech debt, change card, ADR, state change, or design divergence |
| `source_key` | string | Durable Shark or ADR identity |
| `source_pointer` | string | Authoritative local record |
| `change_summary` | string | Bounded behavioral change |
| `affected_artifacts` | array | Path, artifact kind, invalidated text/contract, disposition, optional follow-up key |
| `affected_consumers` | array | Entity key, production caller path, AC IDs, and regression-test pointer |
| `shared_names` | array | Owning name and every producer/consumer usage checked |
| `verification` | array | Bounded evidence that amendments and follow-ups exist |
| `status` | enum | `accounted` or `incomplete` |

`accounted` means each affected artifact is amended or has an existing linked
follow-up key, each shipped consumer has an assigned regression test, and no
shared-name mismatch remains unexplained.

## I-05 CanonicalAdoptionManifest v1

E34-F08 produces this manifest after its canonical bundle and workflow changes
pass validation. E34-F09 consumes it before project reconciliation.

| Field | Type | Contract |
|---|---|---|
| `schema_version` | integer | Exactly `1` |
| `source_commit` | string | Canonical Shark commit containing the changes |
| `bundle_digest` | string | SHA-256 over normalized canonical path/digest pairs |
| `changed_paths` | array | Path, prior/current digest, change class, compatibility note |
| `workflow_changes` | array | Entity type, added/changed steps, and migration concern |
| `promoted_policies` | array | Policy key, canonical source path, and originating feature |
| `override_actions` | array | Counterpart path and recommended inspect/rebase/remove action |
| `validation_evidence` | array | Commands and retained report pointers |

The manifest describes canonical adoption work. It does not authorize changes
to a consuming project's overrides.

## Override baseline architecture

Shark stores digest provenance in
`shark-data/.shark-override-baselines.json`, outside the user-owned override
subtree. The file has a schema version and a map from normalized relative
canonical path to SHA-256. It contains no override bytes or summaries.

- Status compares override bytes with the current embedded canonical bytes and
  the recorded baseline.
- Upgrade and dry-run classify an override with no trustworthy baseline as
  `baseline_unknown`; neither operation creates or advances baseline metadata.
- Acknowledge records the current embedded canonical digest after explicit
  manual reconciliation.
- Missing, corrupt, untrusted, or mismatched metadata never implies current;
  it produces `baseline_unknown` or an actionable validation error.

## Compatibility and migration

- Route-step YAML gains `result_contract: legacy|gate_result_v1`. Omitted means
  `legacy`; unknown values fail workflow validation. Canonical quality-gate
  steps opt in explicitly, `shark next` exposes the resolved value, and both
  Rider and the core runner consume that same resolved field. Whole-file
  project overrides must deliberately adopt the new field.
- Once a gate selects `gate_result_v1`, a missing or malformed nested payload
  fails closed; it never falls back to legacy directives.
- Existing note records remain readable and need no database migration.
- Existing projects with no overrides receive unchanged upgrade behavior plus
  zero-valued override summary fields.
- Existing overrides classify as `baseline_unknown` until explicit operator
  acknowledgement records the reviewed current canonical counterpart.
- Route-based workflow additions are reconciled by whole-file override owners;
  Shark never patches project workflow YAML automatically.

## Security and privacy

- Reject absolute, escaping, symlinked, and non-regular override paths.
- Validate JSON top-level types before field access.
- Bound text and collections and reject known credential, rendered-prompt, and
  transcript markers.
- Store and print digests and relative pointers, not file content.
- Bind gate results to the parent-observed entity, source status, outcome set,
  and lease; ignore any worker attempt to assert those identities.

## Verification strategy

- Shared contract fixtures exercise Rider documentation and core runner code.
- Content tests verify every producer and consumer references the canonical
  interaction and no duplicate field list drifts.
- Service tests cover partial persistence, parent restart under a replacement
  session, exact/conflicting replay, crash after `persistence_complete`, crash
  after transition before release, and terminal worker retirement.
- Workflow tests cover each tier and final integration authority.
- Override tests cover all classifications, acknowledge-only baseline
  transitions, dry-run and mutating-upgrade override immutability, symlinks,
  corrupt metadata, deterministic output, and content non-disclosure.
