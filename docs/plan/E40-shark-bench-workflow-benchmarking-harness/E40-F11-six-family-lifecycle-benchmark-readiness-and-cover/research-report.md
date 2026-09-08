---
research_schema: 2
rigor: complex
categories:
  - backend
  - data
  - workflow_operations
  - documentation
related_work: true
---

# E40-F11 research report: Six-family lifecycle benchmark readiness and coverage

## Scope

E40-F11 repairs the preflight/replay blockers exposed by the 2026-09-04
operator preflight and versions I-04 from four admitted delivery families
(`feature`, `bug`, `change_card`, `tech_debt`) to six (`epic`, `feature`,
`task`, `bug`, `change_card`, `tech_debt`). It touches the bench Python
harness (`bench/scripts/lib/e40_benchmark.py`), the I-04 scenario contract
(`bench/scenarios/scenarios.yaml`, `bench/scenarios/packages/*/package.yaml`),
its Go-side validator (`tests/contracts/e40_i04_scenario_contract_test.go`),
and the operator preflight/pilot/baseline flow. It does not touch
`internal/runner`, `internal/cli`, or any other production Shark package —
consistent with E40's "harness around shark, not inside it" constraint
(epic.md §Constraints). Vocabulary: **I-04** (the scenario package contract
this feature versions), **admitted family** (an `entity_family` value the I-04
schema and its Go validator accept), **root scenario** (a package whose
`entity_family` names the scenario's independently evaluable delivery root,
keyed by `scenario_id`), **spend gate** (an operator acknowledgement required
before any provider-backed call), **fail closed** (a preflight result of
`status=pass` only when every selected scenario and required input resolves
successfully — `pass_with_dry_run_limitations` does not authorize spend).

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F11-six-family-lifecycle-benchmark-readiness-and-cover/feature.md` (REQ-F-001 through REQ-F-014, observed-evidence section) and `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md` (E40-interaction-map.md I-04 row) define the family/scenario/spend-gate vocabulary and the boundary this feature must stay inside.
- [x] `affected_implementation_or_contract` — Evidence: `bench/scenarios/scenarios.yaml` (I-04 index, `schema_version: "1.0"`, currently four `scenarios:` entries); `bench/scenarios/packages/py-{bug-due-date-boundary,change-priority-scale,feature-recurring-tasks,techdebt-consolidate-validation}/package.yaml` (`entity_family:` field, D01-D05 applicability blocks); `tests/contracts/e40_i04_scenario_contract_test.go:26-27,872` (`e40I04SupportedSchemaVersion = "1.0"` and the hardcoded `want one of feature|bug|change_card|tech_debt` check — the exact line REQ-F-007 must change); `bench/scripts/lib/e40_benchmark.py:1909-2055` (`preflight_environment`/`cmd_preflight`/`_cmd_preflight_locked`, currently returns `pass_with_dry_run_limitations`) and lines 920/1013 (`entity_family` reads that key setup by family, the REQ-F-012 defect).
- [x] `related_work` — Evidence: parent epic research report (`docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md`, `RECOMMENDED OUTCOME: pass`) establishes E40's harness-around-shark boundary and the still-active E22 foundation; sibling `E40-F05-lifecycle-scenario-corpus-and-adapter-contract/feature.md` (produces I-04, "Consumes I-01. Produces I-04"); sibling `E40-F07-replayable-product-design-prelude/feature.md` (wraps the existing Shark Rider product-design action for D01-D05, feature-only); sibling `E40-F10-operator-workflow-and-retained-lifecycle-baseline/feature.md` ("one inspected provider-backed pilot per scenario family" — the exact mechanism REQ-F-013/F-014 must generalize to six); `E40-interaction-map.md` I-04 row and staged-edge table (F05 producer, F06/F07/F08 consumers, `contract-only` gate mode until each proves live use).
- [x] `pattern_contract` — Evidence: `tests/contracts/e40_i04_scenario_contract_test.go` is the authoritative I-04 shape validator (schema_version check line 98, required package keys line 850, family-conditioned `replay_reference`/D01 applicability checks lines 897-1035) — REQ-F-007/REQ-F-008 changes must land here, not as a parallel Python-only check. `bench/scenarios/scenarios.yaml`'s own header comment documents the "no field inherited from corpus.yaml" pattern (ADR-F05-01) that REQ-F-009's entity-graph extension must follow (new fields, not copied I-01 shape).
- [x] `dependency_impact` — Evidence: `bench/scripts/lib/e40_benchmark.py` `cmd_preflight`/`cmd_pilot`(2826)/`cmd_baseline`(2835) are the direct consumers of `scenarios.yaml` and each `package.yaml`'s `entity_family`; the submodule pin check (`git submodule status bench/fixture-py` → `2b030a9982a54473b023aebaaa35fe18bb5c42c6`) reproduces the feature's REQ-F-002 observed evidence exactly (differs from admitted `fixture.base_sha` `964fa68e4c9e0c4e0f3756d9efd78b888c558fd9`), confirming the submodule-vs-pinned-checkout defect is live in the current tree, not already fixed. `E40-F10-operator-workflow-and-retained-lifecycle-baseline/feature.md`'s per-family pilot-attestation requirement is the direct downstream dependent of REQ-F-013's six-family generalization.
- [x] `cross_boundary_risks` — Evidence: the I-04 contract crosses a Go/Python boundary — `tests/contracts/e40_i04_scenario_contract_test.go` (Go) validates YAML consumed by `bench/scripts/lib/e40_benchmark.py` (Python); REQ-F-007's schema bump must land in both or the two checkers silently diverge on what "six families" means. `E40-interaction-map.md`'s I-04 staged edge (`gate_mode: contract-only` until F06/F07/F08 each prove live use) means changing I-04's shape here also touches three downstream `activation_owner` obligations that are outside this feature's file set — REQ-F-007's "update every consumer" instruction is this real risk, not boilerplate. The fixture-submodule-vs-admitted-SHA gap (dependency_impact above) is itself a live cross-boundary risk: evaluator collection currently reads the repository's incidental submodule HEAD instead of an immutable checkout, which is exactly the failure REQ-F-002 reproduces and must close.
- [x] `alternatives` — Evidence: the feature could have kept `pass_with_dry_run_limitations` as an acceptable spend gate and treated dry-run stage-resolution failures as advisory (the status quo behavior in `bench/scripts/lib/e40_benchmark.py:1909-2055`); the feature file explicitly rejects this ("`pass_with_dry_run_limitations` is diagnostic information, not an acceptable spend gate for pilot or baseline execution", REQ-F-004) because the 2026-09-04 preflight reported provider readiness while two scenarios had not actually resolved — a live-reproduced counterexample to the alternative, not a hypothetical. The feature also considered generating tasks/descendants during triage but explicitly defers that to the feature workflow's own task-generation stage ("Out of scope").

## Capability map

| Capability | Brownfield evidence | Decision | E40-F11 responsibility |
|---|---|---|---|
| I-04 scenario package contract (`schema_version`, `entity_family` vocabulary) | `bench/scenarios/scenarios.yaml`; `tests/contracts/e40_i04_scenario_contract_test.go:26-27,872` | EXTEND | Bump `schema_version` and the Go validator's closed family enum from four to six (`epic`, `feature`, `task`, `bug`, `change_card`, `tech_debt`); update every consumer, README, and demo statement that assumes four (REQ-F-007). |
| Zero-provider preflight (`cmd_preflight`, `_cmd_preflight_locked`) | `bench/scripts/lib/e40_benchmark.py:1909-2055` | EXTEND | Make `status=pass` require every selected scenario to resolve and every required input to be valid; stop treating `pass_with_dry_run_limitations` as spend-eligible (REQ-F-004); add the fixed/bounded/worst-case call-plan output (REQ-F-006). |
| Admitted-fixture checkout binding for evaluator collection | Current submodule pin `2b030a9982a54473b023aebaaa35fe18bb5c42c6` vs. admitted `fixture.base_sha` `964fa68e4c9e0c4e0f3756d9efd78b888c558fd9` (reproduced live in this tree) | NEW | Add an immutable checkout path passed explicitly through every collector caller, verified against `fixture.base_sha` before collection, leaving the repository submodule byte-unchanged (REQ-F-002). |
| I-06 replay preparation for feature scenarios | `E40-F07-replayable-product-design-prelude/feature.md` (D01-D05 replay bundle, "Stop with `unresolved_gate` when the bundle lacks an authorized answer"); `bench/replay/i06-schema.yaml`; `bench/scripts/verify-replay-result.sh` | EXTEND | Add one operator-owned, spend-gated flow that previews calls, requires explicit acknowledgement, runs the genuine replay producer, and verifies against `bench/scripts/verify-replay-result.sh` — ordinary preflight only validates an existing result, never creates one (REQ-F-003). |
| Per-family pilot attestation before repeated baseline publication | `E40-F10-operator-workflow-and-retained-lifecycle-baseline/feature.md` ("one inspected provider-backed pilot per scenario family") | EXTEND | Generalize the attestation requirement from four families to six with no private family list in pilot, retention, aggregation, or report code (REQ-F-013). |
| D01-D05 applicability (feature-only) | `bench/scenarios/packages/py-change-priority-scale/package.yaml` and `py-techdebt-consolidate-validation/package.yaml` (`D01..D05: {applicable: false, reason: "... skip the product-design prelude"}`); `tests/contracts/e40_i04_scenario_contract_test.go:1029` (family-invariant check enforcing `feature` requires `applicable: true`) | REUSE (extend enum, not the invariant) | New `epic` and `task` packages must carry the same `applicable: false` pattern with a family-specific reason; the existing family-invariant check already generalizes once the enum is extended (REQ-F-008). |
| Scenario-root keying by `entity_family` (dictionary-style setup) | `bench/scripts/lib/e40_benchmark.py:920,1013` (`family = str(package["entity_family"])`, one-root-per-family assumption) | EXTEND | Re-key setup roots by `scenario_id` so same-family scenarios (e.g. two `feature` roots in the future) cannot silently overwrite each other; seed independent hierarchies per scenario (REQ-F-012). |
| Task-root and epic-root scenario packages | None — no `py-task-*` or `py-epic-*` package exists under `bench/scenarios/packages/` today | NEW | Admit `py-task-delete-task` (task-root, `draft -> development -> completed`, no descendants) and `py-epic-task-organization` (epic-root, two features, bounded task set, descendant-oracle union) as new packages conforming to the extended I-04 contract (REQ-F-010, REQ-F-011). |
| Expected-entity-graph contract for hierarchical scenarios | Not present in current `package.yaml` fields (verified against all four existing packages) | NEW | Add declarative fields for root family, allowed/required descendants, min/max counts, required terminal states, and unexpected-descendant handling, used both for structural evaluation and provider-call planning (REQ-F-009). |
| Retained 2026-09-04 operator root as evidence | `feature.md` §Observed evidence (retained at `/home/jwwel/e40-benchmarks/2026-09-04-e40-full-family`, preflight-result SHA-256 pinned) | REUSE, frozen | Preserve byte-for-byte; do not reclaim, retry, or promote; classify as `incomplete`; use a new external operator root for all F11 validation work (REQ-F-001, REQ-NF-003). |

## Findings

1. The submodule-versus-admitted-fixture-SHA defect described in the feature's
   observed evidence is live and independently reproducible in the current
   tree: `git submodule status bench/fixture-py` returns
   `2b030a9982a54473b023aebaaa35fe18bb5c42c6`, which does not match the four
   admitted packages' `fixture.base_sha`
   `964fa68e4c9e0c4e0f3756d9efd78b888c558fd9`. This is not a stale artifact
   from the retained operator root — it is the repository's present state,
   confirming REQ-F-002 has real work to do.

2. The I-04 family vocabulary is enforced identically in two places that must
   move together: `tests/contracts/e40_i04_scenario_contract_test.go:872`
   hardcodes `want one of feature|bug|change_card|tech_debt` in Go, and
   `bench/scripts/lib/e40_benchmark.py` reads `entity_family` at runtime in
   Python (lines 920, 1013) without its own independent enum check visible in
   this pass — the Go contract test is the actual gate. REQ-F-007's
   "update every ... schema validator" instruction maps to a concrete,
   single-file change plus every `package.yaml` and `scenarios.yaml` consumer.

3. `bench/scenarios/scenarios.yaml`'s own header comment (ADR-F05-01)
   documents a deliberate independence from `bench/corpus/corpus.yaml`'s (I-01)
   field names. REQ-F-009's expected-entity-graph extension should follow the
   same discipline: new fields on `package.yaml`, not fields borrowed from a
   different contract's shape.

4. The current preflight implementation
   (`bench/scripts/lib/e40_benchmark.py:1909-2055`) is the same code path
   that produced the 2026-09-04 `pass_with_dry_run_limitations` result the
   feature must stop treating as spend-eligible — this is an existing,
   unmodified function, not a hypothetical to design against. REQ-F-004 and
   REQ-F-005's failing-first regression tests (Required validation step 1)
   should target this function directly.

5. No `py-task-*` or `py-epic-*` scenario package exists under
   `bench/scenarios/packages/` today (confirmed by directory listing), and
   `scenarios.yaml` lists exactly the four families I-04 currently admits.
   REQ-F-010/REQ-F-011 are additive package creation, not a rework of
   existing packages — consistent with REQ-F-012's requirement that new
   scenario roots never reuse an existing epic or feature.

6. `E40-F10-operator-workflow-and-retained-lifecycle-baseline/feature.md`
   already states the per-family pilot-inspection requirement this feature
   must generalize ("one inspected provider-backed pilot per scenario family
   before any repeated baseline"). REQ-F-013 is a vocabulary and enumeration
   change against an already-designed mechanism, not new pilot-gating design.

7. The I-04 interaction-map row (`E40-interaction-map.md`) stages I-04 as a
   `contract-only` handoff with F06/F07/F08 as independent `activation_owner`s.
   Since this feature revisions I-04's shape (schema bump, new fields, six
   families), the "update every ... consumer" language in REQ-F-007 is a real
   cross-feature obligation the map itself names — this feature should not
   assume F06/F07/F08's own task files are already aware of the six-family
   shape.

## Decisions

1. **Treat the fixture-checkout binding fix (REQ-F-002) as prerequisite, not
   parallel, work.** It is independently reproducible today (Finding 1) and
   blocks evaluator collection for any new or existing package; sequence it
   before six-family admission work, matching the feature's own
   Implementation sequence (steps 2-3 before step 7).

2. **Change the I-04 family enum in exactly one authoritative place
   (`tests/contracts/e40_i04_scenario_contract_test.go`) and treat every other
   reference as a consumer to update, not a second source of truth.** This
   avoids the Go/Python divergence risk identified in `cross_boundary_risks`.

3. **Reuse `E40-F10`'s existing per-family pilot-attestation mechanism
   unchanged in shape; only its family enumeration changes.** No new pilot
   design is needed for REQ-F-013.

4. **New `py-task-*` and `py-epic-*` packages follow the existing D01-D05
   `applicable: false` pattern already used by `py-change-priority-scale` and
   `py-techdebt-consolidate-validation`**, rather than inventing a new
   non-applicability mechanism.

5. **REQ-F-009's entity-graph contract adds new `package.yaml` fields
   independent of I-01's shape**, per the documented ADR-F05-01 precedent in
   `scenarios.yaml`.

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F11-six-family-lifecycle-benchmark-readiness-and-cover/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F07-replayable-product-design-prelude/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-operator-workflow-and-retained-lifecycle-baseline/feature.md`
- `bench/scenarios/scenarios.yaml`
- `bench/scenarios/packages/py-bug-due-date-boundary/package.yaml`
- `bench/scenarios/packages/py-change-priority-scale/package.yaml`
- `bench/scenarios/packages/py-feature-recurring-tasks/package.yaml`
- `bench/scenarios/packages/py-techdebt-consolidate-validation/package.yaml`
- `tests/contracts/e40_i04_scenario_contract_test.go`
- `bench/scripts/lib/e40_benchmark.py`
- `bench/replay/i06-schema.yaml`
- `bench/scripts/verify-replay-result.sh`
- `git submodule status bench/fixture-py` (live command output)
- `internal/sharkdata/default_data/research/recipes.yaml`

RECOMMENDED OUTCOME: pass
