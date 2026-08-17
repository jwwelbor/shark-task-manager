---
research_schema: 2
entity_key: E40-F09
entity_type: feature
recipe: universal
rigor: complex
categories:
  - backend
  - data
  - workflow_operations
  - documentation
related_work: true
---

# Research report: Calibrated evaluation and comparison identity

## Scope

F09 defines the post-stage and post-run evaluator that turns I-05 stage
evidence and I-07 lifecycle records into I-08 evaluation records. The research
focuses on deterministic artifact and execution checks, calibrated judgment for
planning/decomposition artifacts, exact candidate and workflow-policy identity,
review-finding normalization, independent versus sequential review comparison,
and fail-closed aggregate eligibility. It does not implement provider batches,
operator publication, or a new workflow engine; those boundaries remain with
F08, F10, and the owning Shark/Rider capabilities.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F09-calibrated-evaluation-and-comparison-identity/feature.md` (scope, acceptance boundary, contracts, and handoff) and `architecture.md#lifecycle-evaluation-record-contract` (I-08 truth separation).
- [x] `affected_implementation_or_contract` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md#lifecycle-evaluation-record-contract`, `#lifecycle-run-record-contract`, and `#workflow-value-attribution-contract`; `bench/evidence/i05-schema.yaml`; `internal/models/validation.go`; and `internal/sharkdata/default_data/prompts/feature/{code_review,qa,approval}.md`.
- [x] `related_work` — Evidence: upstream `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md`; sibling reports `E40-F05` through `E40-F08`; `E40-interaction-map.md` I-05/I-07/I-08 rows and staged edge; and `E40-cross-epic-map.md` X-12.
- [x] `pattern_contract` — Evidence: `bench/corpus/corpus.yaml` and `tests/contracts/e40_i02_artifact_contract_test.go` establish versioned, digest-bearing artifact contracts; `bench/scripts/aggregate-runs.sh` establishes invalid-input retention and no fabricated aggregate output; `skills/shark-rider/skills/deep-review/references/{angle-a-bugs,angle-b-behavior,angle-c-sibling,angle-d-cleanup,angle-e-tests,angle-f-standards,consolidator}.md` plus `scripts/get_diff.sh` define the pinned review bundle.
- [x] `dependency_impact` — Evidence: `architecture.md` I-05/I-07/I-08 producer-consumer contracts; `bench/evidence/i05-schema.yaml` invalidity reasons and candidate fields; `tests/contracts/e40_i04_scenario_contract_test.go` evaluator-only path containment; and `bench/scripts/aggregate-runs.sh` uniformity and eligibility behavior.
- [x] `cross_boundary_risks` — Evidence: `architecture.md` run-isolation and I-08 sections; `bench/evidence/i05-schema.yaml` (`isolation_violation`, `evaluator_access_out_of_order`, `candidate_field_missing`, `publication_eligible_conflict`); `E40-F06/research-report.md` three-root boundary; and `E40-F08/research-report.md` raw-review-finding versus adjudication boundary.
- [x] `alternatives` — Evidence: `shark-bench-design.md` lifecycle-v2 comparison section; `architecture.md` ADR-007 through ADR-010 and workflow-value-attribution contract; and `E40-F06/research-report.md` decision to fail closed rather than guess missing usage/model identity.

## Capability map

| Capability | Evidence | Decision | F09 responsibility |
|---|---|---|---|
| I-05 immutable stage evidence and evaluator isolation | F06 research report; `architecture.md#stage-evidence-and-isolation-contract` | REUSE | Consume snapshots, access lineage, candidate fields, and time/artifact evidence; do not recreate isolation or provider parsing. |
| I-07 lifecycle run and raw review-gate findings | F08 research report; `architecture.md#lifecycle-run-record-contract` | EXTEND | Evaluate the complete run, preserve raw findings, and add normalized/confirmed finding identity and eligibility. |
| I-04 scenario and held-back oracle identity | F05 research report; `architecture.md#lifecycle-scenario-package-contract` | REUSE | Use scenario, fixture, adapter, evaluator-reference, and oracle identities as immutable evaluation inputs. |
| Existing Phase 1 artifact aggregation | `bench/scripts/aggregate-runs.sh`; `tests/contracts/e40_i02_artifact_contract_test.go` | EXTEND | Reuse versioned/digest-bearing artifact discipline, but add lifecycle identity and reject mixed/incomplete records instead of averaging them. |
| Structured `review-finding` notes | `internal/models/validation.go`; feature QA/code-review/approval prompts | EXTEND | Treat reviewer fingerprint, severity, and disposition as evidence; assign independent normalized identity, confirmation source, gate, recurrence/duplicate links, and final disposition. |
| Repository-owned deep-review bundle | `skills/shark-rider/skills/deep-review/` and `scripts/get_diff.sh` | REUSE | Pin one digest over the skill, six angle prompts, consolidator, and diff-selection script; bundle drift invalidates comparison. |
| Candidate/workflow-policy comparison identity | `architecture.md#workflow-value-attribution-contract` and ADR-009 | NEW | Define exact candidate tree/diff/untracked/test-suite and enabled-gate/order/reviewer/fix-policy identity; branch or HEAD labels alone are insufficient. |
| Calibrated LLM judgment | F09 feature contract; `architecture.md#lifecycle-evaluation-record-contract` | NEW | Version rubric/prompt/model/config/reference digests, retain rationale/score/usage/cost, separate calibration from held-out evaluation, and never replace execution truth. |

F09 therefore extends the evidence and review-note capabilities established by
F06/F08 and creates the evaluation, normalization, comparison-identity, and
calibration capability consumed by F10.

## Findings

1. **I-08 must remain a three-truth record.** Structural checks answer whether
   the lifecycle artifacts are internally valid; the calibrated judge answers
   applicable planning/decomposition quality; the held-back oracle answers
   implementation correctness. Terminal Shark status, process success, or
   worker self-report cannot substitute for any of these. Evidence:
   `architecture.md#lifecycle-evaluation-record-contract`.

2. **The evaluator has two separate identity boundaries.** Scenario/fixture/
   adapter/Shark binary/installed content/rendered prompt/provider/model/effort/
   judge/reference/resource-policy identity governs evaluation compatibility.
   Candidate identity additionally includes the exact tree, diff, changed-path
   set, dirty and untracked manifest, and test-suite snapshot. Workflow-policy
   identity includes enabled gates, order, reviewer configuration, full
   deep-review bundle digest, and fix policy. Evidence: `feature.md`,
   `architecture.md#lifecycle-run-record-contract`, and ADR-009.

3. **Review notes are observations, not truth.** Existing prompts require
   structured gate, round, severity, defect class, fingerprint, criterion, and
   disposition metadata, while the model allowlist makes `review-finding`
   queryable. F09 must preserve these raw fields and independently normalize,
   deduplicate, confirm, link recurrence, and assign disposition. Precision and
   recall are valid only with a retained seeded truth set; otherwise report
   confirmed/unconfirmed yield, overlap, recurrence, and downstream escapes.
   Evidence: `internal/models/validation.go` and the three feature review
   prompts; upstream F08 report is the source for the raw-capture boundary.

4. **The fair review comparison has two modes.** Independent comparison runs
   QA and deep review against one frozen candidate and test snapshot with no
   fixes between them. Sequential comparison retains every intervening
   candidate and attributes only newly confirmed findings to the later gate.
   Stage order by itself cannot establish causal gate value. Evidence:
   `architecture.md#workflow-value-attribution-contract` and `feature.md`.

5. **Calibration must be isolated from evaluation.** A versioned rubric and
   prompt, judge model/configuration, human-scored calibration examples, and
   reference digests must be retained. Calibration examples cannot be drawn
   from the held-out evaluation set, and missing calibration, judge identity,
   oracle evidence, or required usage/model identity makes the record
   ineligible rather than silently imputing a value. Evidence:
   `architecture.md#lifecycle-evaluation-record-contract`, F06/F08 reports,
   and `bench/evidence/i05-schema.yaml`.

6. **F09 is downstream of isolation but upstream of publication.** F06 owns
   the three-root access boundary and F08 owns lifecycle capture; F09 consumes
   their immutable records, rejects drift/incompleteness, and owns aggregate
   eligibility. F10 may format and publish I-08 but may not weaken or
   recompute its verdict. Evidence: `E40-interaction-map.md` I-05/I-07/I-08
   rows and boundary rules.

## Decisions

1. Implement F09 as a new post-stage/post-run evaluator over I-05 and I-07
   artifacts. Reuse upstream schemas and access controls; do not add a second
   lifecycle engine, prompt assembler, claim store, or Question store.
2. Make identity comparison fail closed. Retain every invalid record with
   machine-readable divergence reasons; exclude it from aggregates. Matching
   branches, HEADs, or variant labels are not sufficient.
3. Keep structural checks, judge results, execution-oracle results, and review
   adjudication in separate fields. No composite score should conceal a failed
   oracle, missing evidence, or unconfirmed finding.
4. Normalize findings with an F09-owned identity namespace. Preserve all raw
   reviewer fields as evidence and use explicit duplicate/recurrence links,
   confirmation source, first-seen gate, resolution candidate, and disposition.
5. Pin the deep-review bundle as one digest over all required files. Any bundle
   or diff-selection change is workflow-policy drift and prevents comparison.
6. Define calibration as a separately versioned, human-scored reference set
   and rubric. Permit judge output only for applicable artifacts and report
   rationale, score, usage, and cost alongside calibration provenance.
7. Support independent frozen-candidate and sequential delivery comparisons,
   retaining candidate lineage and intervening fixes in both cases. Report
   quality, time, and cost separately; never collapse them into one efficiency
   score.

## Sources

- `internal/sharkdata/default_data/research/recipes.yaml` (universal v2 modules and complex rigor).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md` and `shark-bench-design.md` (parent goals and lifecycle-v2 PRD/design).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (I-05, I-07, I-08, ADR-007–ADR-010, and value-attribution contract).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` and `E40-cross-epic-map.md` (producer/consumer and X-12 boundaries).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F09-calibrated-evaluation-and-comparison-identity/feature.md` (feature scope and acceptance boundary).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F05-lifecycle-scenario-corpus-and-adapter-contract/research-report.md` (I-04 capability and oracle boundary).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F06-stage-evidence-and-evaluator-isolation/research-report.md` (I-05 isolation and evidence boundary).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F07-replayable-product-design-prelude/research-report.md` (I-06 replay boundary).
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F08-canonical-multi-entity-lifecycle-runner/research-report.md` (I-07 and raw finding capture boundary).
- `bench/evidence/i05-schema.yaml`, `bench/scripts/aggregate-runs.sh`, `bench/corpus/corpus.yaml`, and `tests/contracts/e40_i02_artifact_contract_test.go` (schemas, invalidity, identity, and aggregation precedents).
- `internal/models/validation.go` and `internal/sharkdata/default_data/prompts/feature/{code_review,qa,approval}.md` (review-finding fields and gate behavior).
- `skills/shark-rider/skills/deep-review/references/` and `skills/shark-rider/skills/deep-review/scripts/get_diff.sh` (review bundle contents and diff selection).
- `tests/contracts/e40_i04_scenario_contract_test.go` (evaluator-only path containment precedent).

RECOMMENDED OUTCOME: pass
