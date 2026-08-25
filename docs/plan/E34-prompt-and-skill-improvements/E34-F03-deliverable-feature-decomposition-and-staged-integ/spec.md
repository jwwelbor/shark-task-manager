# E34-F03 Specification: Deliverable Feature Decomposition and Staged Integration Acceptance

## Scope and traceability

This feature extends Epic E34's cross-feature interaction lifecycle enforcement
(Epic Requirements REQ-F-001 through REQ-F-009 and REQ-NF-001 through
REQ-NF-002). It also closes the deliverability and gate-integrity gap recorded
in the feature brief. See the epic PRD's **Goal** and **Success Criteria**, and
the feature research report's **Capability map** and **Decisions**; this
document does not restate their business context.

The capability map establishes the boundaries used here:

- Extend the existing I-## interaction-map lifecycle, feature specification,
  task-review, quality, and UAT content contracts.
- Reuse E34-F01's harness-aware rendering design without changing its renderer
  handshake, and produce the readiness vocabulary that E34-F02 consumes.
- Do not add a database model, workflow-engine behavior, entity type, or a
  persistent acceptance ledger.

## Requirements

### Functional requirements

#### REQ-F-001 — Require a deliverable feature boundary

Epic decomposition, feature specification, task review, and design validation
shall require every proposed feature to name a real trigger, observable result,
production path, complete UAT scenario, current prerequisites, and outputs for
later consumers. A criterion requiring a later feature shall move to that
activation owner or cause the slices to be redesigned; an invented throwaway
caller is not an acceptable substitute.

#### REQ-F-002 — Extend interaction-map rows with staged-edge disposition

The existing E34 I-## interaction-map contract shall gain a documented
deliverability disposition. `live` is the default. A `contract-only` row is
valid only when it is declared no later than feature specification and names:
counterpart entity keys and statuses, shared-contract evidence, activation
owner, closure key, and review basis. Task review and design validation shall
reject incomplete declarations and reverse build-order consumption shall be
reported as a decomposition warning.

#### REQ-F-003 — Preserve hard gate semantics

Quality and UAT content shall continue to treat an undeclared or `live` edge
without a production caller as blocking. Missing authentication, authorization,
current integrity guarantees, unsafe exposure, or an unmet current-feature
acceptance criterion shall remain blocking regardless of a future Shark key or
an owner override.

#### REQ-F-004 — Separate assessment from owner decision

The UAT contract shall preserve the independent assessor verdict and the owner
decision as distinct reported facts. A complete, predeclared `contract-only`
edge may be considered for an owner `override-accept` / Accept with Conditions
decision only after its explicit disposition is recorded; that decision shall
not rewrite the assessor verdict or turn the capability into a verified
production-path claim.

#### REQ-F-005 — Require activation-owner closure

The activation owner's UAT shall prove the real caller chain, shared contract,
production-path integration test, and a counterfactual test that fails if the
wiring is bypassed or removed. An epic completion check shall reject an open
internal activation obligation. An external obligation is allowed only with a
named future owner and a documented roadmap decision.

#### REQ-F-006 — Provide a single readiness evidence shape for E34-F02

The policy content shall define the reusable readiness evidence fields:
assessor verdict, owner decision, open conditions, gate mode, activation owner,
closure key, counterpart status, review basis, and demonstrability disposition.
E34-F02 may consume these fields for its `Demonstrated now`, `Not demonstrated
/ pending integration`, and `Accepted risks and overrides` classifications,
but this feature shall not implement demo generation or grant it acceptance
authority.

#### REQ-F-007 — Validate bundle integrity

Rendered-prompt validation shall confirm that the changed templates and includes
resolve through the shipped renderer. Focused checks shall confirm that the
documented E34-F03 to E34-F02 handoff files exist. Human review, not
string-matching decision tables, evaluates policy wording.

### Non-functional requirements

- **REQ-NF-001 — Backward compatibility:** Single-feature epics and epics with
  fewer than three features continue without interaction-map requirements.
- **REQ-NF-002 — No runtime expansion:** The change is embedded content and
  rendered-prompt validation only; it adds no database migration, schema,
  repository, service, CLI command, or workflow-status transition.
- **REQ-NF-003 — Security and integrity non-waiver:** The policy must make
  security, authorization, integrity, and unsafe-exposure failures explicit
  blockers, not conditional follow-ups.
- **REQ-NF-004 — Compatible rendering:** Prompt content must remain compatible
  with E34-F01 harness-aware prompt rendering and parse through the existing
  `templates.NewOrchestratorRenderer` test path.

### Acceptance criteria

- **AC-001:** A multi-feature epic's design/decomposition guidance rejects a
  feature whose acceptance depends on a later feature, unless its requirement
  is reassigned to a declared activation owner or the boundary is redesigned.
- **AC-002:** The interaction-map template and all authoring/review consumers
  require the exact `live`/`contract-only` disposition and every mandatory
  contract-only field from REQ-F-002.
- **AC-003:** Missing live wiring, authentication/authorization, integrity,
  unsafe exposure, or a current AC remains a blocking outcome even when a
  future owner or `override-accept` exists.
- **AC-004:** UAT guidance records assessor verdict, owner decision, and
  conditions separately; it never describes an overridden rejection or open
  contract-only obligation as verified end-to-end delivery.
- **AC-005:** Activation-owner UAT requires caller-chain evidence, shared
  contract evidence, a production-path integration test, a wiring-removal
  counterfactual, and closure before internal epic completion.
- **AC-006:** E34-F02 has one documented producer contract containing all nine
  REQ-F-006 fields and classifies an open activation obligation as pending
  integration.
- **AC-007:** Changed prompt templates render through the shipped bundle, and
  the documented E34-F03 to E34-F02 handoff files exist. Policy wording is
  reviewed against this specification by a human reviewer.
- **AC-008:** Existing interaction-lifecycle tests and the full repository
  quality gate pass without adding a runtime persistence surface.

### Out of scope

- A new workflow engine, runtime claim store, entity type, database schema, or
  automatic decision persistence.
- Weakening independent Codex red-team UAT, auto-approving work, or allowing a
  future key to waive a current security, integrity, exposure, or live-path
  failure.
- Rewriting historical WWGM verdicts, creating throwaway production callers,
  or implementing E34-F02's demo recipe.
- Requiring every feature to be independently deployable rather than
  independently demonstrable within a declared release scope.

## Architecture

### Component changes

The feature extends one established, file-based policy pipeline; it does not
introduce a runtime component:

| Surface | Files to modify | Change |
|---|---|---|
| Epic design and decomposition | `internal/sharkdata/default_data/prompts/epic/design.md`, `internal/sharkdata/default_data/prompts/epic/decomposition.md`, `internal/sharkdata/default_data/prompts/epic/feature_review.md` | Define deliverable boundaries, extend I-## row semantics, assign activation ownership, and reject unresolved internal closure. |
| Interaction-map source | `internal/sharkdata/default_data/skills/specification-writing/context/interaction-map-template.md` | Add the staged-edge fields and exact authoring rules while retaining stable I-## IDs and shared shape/test-pointer rules. |
| Feature authoring and task gate | `internal/sharkdata/default_data/prompts/feature/specification.md`, `internal/sharkdata/default_data/prompts/feature/task_generation.md`, `internal/sharkdata/default_data/prompts/feature/task_review.md` | Mirror the declared disposition into feature/task work and reject incomplete or mismatched activation ownership. |
| Test planning and quality | `internal/sharkdata/default_data/prompts/feature/test_planning.md`, `internal/sharkdata/default_data/prompts/feature/code_review.md`, `internal/sharkdata/default_data/prompts/feature/qa.md`, `internal/sharkdata/default_data/skills/quality/workflows/test-planning.md`, `internal/sharkdata/default_data/skills/quality/workflows/validate-design.md`, `internal/sharkdata/default_data/skills/quality/workflows/validate-tasks.md`, `internal/sharkdata/default_data/skills/quality/workflows/qa-testing.md` | Require production-caller and counterfactual proof only when a change adds or alters live runtime behavior. For prompt-only work, validate rendering, includes, and documented file references; review policy wording manually. |
| UAT policy | `internal/sharkdata/default_data/skills/uat/SKILL.md`, `internal/sharkdata/default_data/skills/uat/references/redteam-rubric.md`, `internal/sharkdata/default_data/agents/uat-agent.md` | Make verdict/owner-decision separation, non-waivable blockers, conditional disposition, and activation closure explicit. |
| E34-F02 consumer | `docs/plan/E34-prompt-and-skill-improvements/E34-F02-evidence-based-demo-script-skill/feature.md` | Keep the producer/consumer field set aligned; no demo implementation is included here. |
| Regression tests | `internal/cli/commands/interaction_prompts_test.go` | Render changed prompts through the shipped bundle and verify that the documented E34 handoff files exist. |

### Data model and persistence

There are no database, schema, migration, model, repository, service, or API
changes. The interaction map and feature/spec/task/UAT artifacts remain the
design-time source of truth, following the existing related-document contract.
The staged-edge disposition is structured markdown content, not a new persisted
Shark field. This is an intentional deviation from runtime enforcement: the
epic scope excludes an engine or claim-store change, and the existing workflow
already renders and validates these content bundles.

### Content/interface contract

For every I-## row that is `contract-only`, the map remains authoritative for
the counterpart identities and shared-contract evidence. Its producer and
consumer mirrors must copy this exact nine-field readiness shape and values:

| Field | Required contract |
|---|---|
| `assessor_verdict` | Independent UAT assessment, recorded without owner-decision rewrite |
| `owner_decision` | Separate approval or `override-accept` decision with conditions |
| `open_conditions` | Open activation and any recorded conditions remain visible |
| `gate_mode` | `contract-only` until E34-F02 proves live production-path use |
| `activation_owner` | E34-F02 |
| `closure_key` | E34-F02 |
| `counterpart_status` | Read live from Shark at review/UAT time; this map intentionally contains no copied current-state snapshot |
| `review_basis` | Accumulated E34 branch with the map and both feature specifications present |
| `demonstrability_disposition` | `pending-integration` until live wiring closes; no override makes it demonstrated-now |

The I-01 map table supplies E34-F03 and E34-F02 as counterpart identities and
the shared source/pointer. Those are staged-edge declaration metadata, not
readiness fields. Current lifecycle status must be read live from Shark at
review/UAT time; static documents must not claim a current status.

This is a documentation and prompt contract. It has no Go function signature or
external API. Existing content inclusion and rendering continue to use the
embedded-bundle pattern tested through
`templates.NewOrchestratorRenderer` in
`internal/cli/commands/interaction_prompts_test.go`.

### Key decisions

1. **Extend I-## rather than introduce an acceptance ledger.** This follows
   the capability map and Epic REQ-F-001 through REQ-F-009: interaction maps
   already supply stable ownership, shape, and shared-test references.
2. **Keep `live` as the default.** This preserves the current UAT rubric's
   rule that no call site, unregistered component, or unmounted route is a
   blocker. `contract-only` is a narrow, predeclared exception in review
   disposition, not a waiver of production evidence.
3. **Keep assessor verdict and owner decision separate.** This follows the
   existing UAT division of responsibility: Codex is the assessor and the user
   is the approver. A later decision cannot change observed evidence.
4. **Make activation closure a consumer-owned proof.** The feature that owns
   live wiring is the only place able to prove the full caller path and its
   counterfactual; earlier contract work must not manufacture a caller merely
   to satisfy an isolated gate.
5. **Use bundle-integrity checks, not policy simulators.** The affected
   implementation is embedded prompts and skills. Rendered-prompt validation
   catches broken templates and missing includes; focused file checks catch
   nonexistent documented references. Human review evaluates policy wording
   without turning prose into a simulated runtime rules engine.

### Integration with existing code

- The content bundle stays under `internal/sharkdata/default_data/`; no
  alternate host-file lookup is introduced.
- `internal/cli/commands/interaction_prompts_test.go` continues to render
  bundled prompts through `templates.NewOrchestratorRenderer` and uses
  `goldenVars()` to prove includes resolve. Its E34-F03 coverage also confirms
  that the documented interaction map and feature handoff files exist.
- The existing `shark related-docs add` pattern remains the registration path
  for the E34 interaction map. Implementation must create
  `docs/plan/E34-prompt-and-skill-improvements/E34-interaction-map.md` before
  requiring E34-F03 and E34-F02 to mirror its assigned IDs.
- E34-F02 consumes the readiness evidence shape read-only. It may not use a
  demo artifact, completion status, or owner override as a substitute for a
  closed production-path proof.

## Cross-feature interactions

**Produces: I-01** for E34-F02. The authoritative shape source is
[`E34-interaction-map.md#i-01-readiness-evidence-shape`](../E34-interaction-map.md#i-01-readiness-evidence-shape).
The shared structural contract-test pointer is
**TC-I-01-READINESS-SYMMETRY** at
`internal/cli/commands/interaction_prompts_test.go::TestI01ReadinessContract_TC_I_01_READINESS_SYMMETRY`;
E34-F03 publishes the nine-field readiness evidence shape and E34-F02 consumes
it read-only. Do not create a producer-only twin test or invent another I-## ID.

## Cross-epic integrations

No E34-specific cross-epic map exists, and the global product map has no X-##
row that names E34-F03. No X-## identifier is declared or invented by this
feature. If implementation discovers a real cross-epic handoff, it must first
be assigned in `docs/product/cross-epic-integration-map.md` and the E34
cross-epic map, with the required ownership, UX/CX note, and coverage or
progress-log deferral before any feature spec mirrors it.

## Delivery and verification plan

1. Create/register the E34 interaction map and update epic/design/decomposition
   guidance and its rendered tests.
2. Propagate the same staged-edge contract through feature specification, test
   planning, task generation/review, design validation, code review, and QA;
   validate bundle rendering, includes, golden snapshots, and documented file
   references.
3. Update UAT rubric, skill, and agent wording together; verify that the
   security/integrity non-waiver and assessor/owner separation use the same
   terms.
4. Update E34-F02's documented consumer contract after the map assigns the
   stable I-## identifier; do not implement the demo recipe.
5. Run `make fmt`, `make lint`, and `make test`. While iterating, run the
   focused renderer and handoff-reference checks, then retain the full quality
   gate as the final check.

## Exit-gate check

- Every REQ-F and AC above has a concrete content surface and testable proof.
- All changed file paths are enumerated in **Component changes**.
- No critical section contains a TBD; map IDs are deliberately unassigned only
  because the authoritative parent map does not yet exist, and the required
  creation/assignment sequence is explicit.
- No I-## or X-## ID has been invented.
- No proposal adds runtime persistence or weakens security, integrity, live
  wiring, or independent assessment requirements.
