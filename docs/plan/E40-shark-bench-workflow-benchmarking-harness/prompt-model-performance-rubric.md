# Evaluate prompt and model performance against E40's goals

E40 asks whether a prompt, model, effort level, or stage assignment produces
more correct work and useful artifacts, and at what time and provider cost.
Paired comparisons should expose regressions and separate real changes from
run-to-run variation.

The [E40 business goals](epic.md#2-goals-and-success-criteria) define the
outcomes. This guide defines how to evaluate them; it does not claim every
measure is collected or that any comparison has passed. Report availability
from retained evidence.

## Define the comparison

Pair a baseline and variant within the same admitted scenario and repetition.
Record the intended change, changed stage, allowed change axes, prompt digest,
and effective provider, model, and effort. Hold other identity controls
constant. A model-only comparison holds prompt and workflow constant; a
prompt-only comparison holds model, effort, and workflow constant. A combined
change measures the bundle, not each change separately.

Judge the full lifecycle as the product result. Stage measures diagnose where
time, cost, or quality changed; a cheaper development stage is not a delivery
saving if it adds review or rework.

Use the scenario families admitted by current batch policy. F11 covers epic,
feature, task, bug, change-card, and tech-debt scenarios. Do not substitute a
leaf task for its parent lifecycle. D01-D05 replay applies to features; other
families retain their declared applicability.

## Establish the evidence boundary

These conditions decide whether a result supports comparison; they are not
subjective grades:

| Condition | E40 goals | Required evidence |
|---|---|---|
| Admitted, discriminative scenario | G1, G8 | Pinned fixture and adapter; base/reference admission results; declared final predicate and applicable stages. |
| Complete, isolated, reproducible inputs | G7, G9, G10, G14 | Frozen stage inputs and replay references; no evaluator truth at worker boundaries; matching scenario, toolchain, Shark binary, and reference identities. |
| Recorded lifecycle and resource policy | G2, G3, G11, G12 | Dispatch, claim/heartbeat, unchanged prompt handoff, semantic outcome, transition, release, and stop records; positive finite cost, time, generated-task, and provider-call ceilings. |
| Complete evaluation | G4, G13 | Independent structural, calibrated planning-judge, and held-back implementation-oracle evidence wherever applicable; actual model and required usage fields. |
| Inspectable retention | G15 | Spend acknowledgement for provider-backed execution, byte-preserved artifacts, and one verified, inspected pilot per selected family before repeated baseline publication. |
| Fair experiment and review comparison | G6, G14, G19 | Baseline-linked variant with declared change axes, unchanged controls, and exact candidate/test and workflow-policy identity for the selected review mode. |

Retain each missing judge, failed oracle, broken isolation boundary, or
incompatible identity as its named invalidity reason, not as a score. Keep
failed and incomplete attempts visible in the outcome inventory and out of
valid metric aggregates.

## Measure performance

Report quality, elapsed time, cost, and reliability separately. Do not combine
them into an overall model grade or efficiency score.

| Dimension | Goals | Measures and interpretation | Evidence and limits |
|---|---|---|---|
| Correctness and implementation quality | G1, G4, G13 | Final predicate; required F2P resolution; P2P preservation; applicable format, build, and test gates; acceptance criteria passed, failed, or unverified against the full criterion set; new standards violations by severity and class. LOC additions, deletions, and files touched, split by production/tests, diagnose patch scope only; never normalize quality, cost, or time by LOC. | Held-back oracle, criterion-linked structural checks, base ledgers, normalized findings, and candidate diff. Keep unknown criteria unverified; do not average away gaps. Workflow completion or a worker's success claim does not prove correctness. Lint does not cover every standard; exclude historical base violations. LOC, patch size, and token count do not establish quality. |
| Planning and decomposition quality | G9, G10, G13 | Five calibrated dimensions, reported separately, plus the planning-only mean required by the judge contract. | Judge generated planning artifacts against frozen stakeholder/research references with human calibration and fixed judge identity. Planning score does not replace oracle success. |
| Lifecycle reliability and budget | G2, G3, G11, G12, G15 | Completed, failed, stopped, and incomplete outcomes; first-pass yield when rounds support it; required/reached approval gates and attempts; gate rejections and recurrence; consistently defined rework rounds; unresolved gates; completion against cost, wall-time, generated-task, and provider-call ceilings. | I-07 dispatch/outcome/gate lineage, I-08 validity, and stop evidence. Heartbeats show activity, not useful progress. Classify human approval from explicit metadata, not gate names. A skipped gate is not lower overhead; a cheap failure or resource-limit stop is not successful budget delivery. |
| Elapsed time and coordination | G4, G16 | Lifecycle and stage time; provider work, tools/tests, queue/claim wait, replay/gate wait, retry/backoff, and unclassified time; longest gate-attempt span by scenario/gate/round; active human review seconds when independently observed. | I-05/I-07 non-overlapping interval ledger and retained I-08/F10 metrics. Reconcile partitions to wall time and disclose residuals; stage and interval totals are alternative partitions. A longest stage is not a longest review gate. Gate timing requires an exact gate/ledger join; a review-time observation must record the artifact and review event. Replay counts and wait do not measure reviewer attention; missing intervals are not zero. |
| Provider consumption and cost | G4, G12, G16 | Input/output tokens, observed calls, and observed USD cost by stage and lifecycle, including attributable judge and rework cost. Compare observed usage with ceilings; count each cost once. | Captured usage and supported pricing/contract evidence. Unavailable cost is not zero; a subscription alone does not prove zero cost. |
| Review-gate value | G17, G19 | Unique confirmed findings, duplicates, recurrence, overlap, unconfirmed findings, downstream escapes, and elapsed/provider/resolution cost by gate, severity, and defect class. | Independently normalized findings and exact candidate/test/policy lineage. More emitted findings do not prove more defects caught. Precision/recall require a retained truth set. Zero findings differ from failed collection; cost per confirmed finding is undefined when none are confirmed. |
| Artifact usefulness and interaction burden | G9, G10, G11, G18 | Produced documentation count/bytes and required-document completeness; artifact consumption, reuse, and orphaning; replay requests/responses, payload size, revisions, unresolved gates, and needed, repeated, or out-of-scope questions. Judge whether questions resolve missing decisions, repeat supplied context, or add unsupported scope; counts alone do not measure quality. | Typed producer/consumer edges, replay records, authorized inputs, and retained decisions. Exclude logs, transcripts, and code from document counts. Access shows consumption, not usefulness; volume alone is not quality or burden. Missing consumer edges mean incomplete telemetry; an explicit empty set means no recorded consumer. |
| Repeatability and detectable change | G5, G6, G7, G14 | Paired per-scenario deltas, baseline noise bands, repetitions, insufficient-repetition flag, retained reruns, and exact comparison identities. | Published bands and pinned identities. A delta inside or on the acceptance boundary means **no detectable effect**; these bands do not establish statistical significance. |

## Score planning artifacts with human calibration

The [planning rubric](E40-F09-calibrated-evaluation-and-comparison-identity/calibration/v1/rubric.md)
defines five equally weighted dimensions:

| Code | Dimension | Useful output | E40 connection |
|---|---|---|---|
| P | Problem and target-user clarity | Need, user, context, scope, and exclusions guide the work. | G10 stakeholder input; G13 artifact quality. |
| M | Measurable success criteria and constraints | Outcomes support acceptance checks; product and resource limits are explicit and consistent. | G12 limits; G13 acceptance/oracle truth. |
| R | Relevance and support of research claims | Evidence supports decisions; sources are identified; facts, assumptions, and open questions are distinct. | G9 source evidence; G10 research responses. |
| F | Feasibility and trade-off reasoning | Alternatives are weighed against constraints and risks; the selected approach is executable. | G11 descendants/lifecycle; G13 artifact quality. |
| S | Synthesis, scope, and decomposition consistency | Decisions and work trace to the need and criteria without dropped requirements or unsupported scope. | G9 traceability; G11 coverage; G13 quality; G18 downstream use. |

Use the calibration rubric's 0–4 anchors. Preserve each rating and its
evidence-linked rationale; normalize by dividing by four. The mean is a
**planning quality score only**, not an overall E40 score.

Score the artifact scope under evaluation. D01-D05 input, a feature
specification, and an executable task breakdown are different artifacts. Do
not penalize a scope that does not require decomposition; when it does, assess
coverage, dependencies, and executable acceptance criteria. Structural
validation owns required fields and graph correctness; the judge assesses
reasoning quality.

The [human calibration packet](E40-F09-calibrated-evaluation-and-comparison-identity/calibration/v1/README.md)
contains six fictional examples, blank ratings, and retained identities. Human
labels calibrate planning judgments; they do not supply time, token, cost, or
oracle measurements.

### Score generated outputs

Frozen stakeholder and research responses are shared inputs. Scoring them
alone evaluates the corpus, not the model or prompt.

Bind each planning score to the generated artifact and stage/candidate digest.
Use the same frozen references, rubric, and judge for both sides. The
[judge-generation prompt](judge-result-generation-prompt.md) currently names
eleven replay references as inputs; that reference-only scope cannot support a
prompt/model performance delta. This is a measurement boundary, not a claim
that output scoring is implemented.

## Run and interpret the comparison

1. Declare the decision, intended treatment, baseline/variant identities, and
   allowed change axes. A declared change does not excuse unrelated identity
   drift.
2. Hold scenario, replay, fixture/adapter, toolchain, evaluator/rubric/
   calibration, and resource policy constant. Retain actual prompt/model
   identities.
3. Run the admitted scenario matrix with declared repetitions and ceilings.
   Preserve all outcomes and inspect pilots; one pilot is not a repeated
   baseline.
4. Pair baseline and variant by scenario/repetition. Record variant minus
   baseline for each available measure. Less time/cost helps only with
   acceptable quality; more unique confirmed findings can help even when
   review takes longer.
5. Apply each published noise band. Report missing bands and insufficient
   repetitions; call a change detectable only when the band supports it.
6. State the trade-off for the decision. Do not rank models with one number or
   average unlike scenarios. Set any quality tolerance before reviewing
   results; this rubric supplies no default.

For independent QA-versus-deep-review comparisons, both gates inspect the same
frozen candidate and test suite without intervening fixes. For sequential
delivery, retain each candidate and attribute only newly confirmed findings to
the later gate. A prompt/model treatment that changes policy is not an
exact-policy control.

## Fill the comparison worksheet

Experiment ID: __________

Decision, treatment, and changed stage: __________

Baseline / variant manifests and allowed change axes: __________

Scenario/version / repetition / family: __________

Judge/rubric/calibration identity and generated-artifact references: __________

Eligibility and invalidity reasons for both sides: __________

| Measure | Baseline | Variant | Paired delta | Band / availability | Interpretation |
|---|---|---|---|---|---|
| Correctness, acceptance, regression, and standards | | | | | |
| Planning P / M / R / F / S and planning-only mean | | | | | |
| Lifecycle outcome, gates, first-pass yield, and rework | | | | | |
| Lifecycle time, interval partitions, longest gate/stage, and observed review time | | | | | |
| Tokens, provider calls, and observed cost | | | | | |
| Budget completion and limiting resource, if stopped | | | | | |
| Review findings: unique, duplicate, recurring, overlapping, escaped | | | | | |
| Artifacts/documents: completeness, consumption, reuse, orphaning | | | | | |
| Replay interactions and needed/repeated/out-of-scope questions | | | | | |
| Paired deltas, noise bands, repetitions, and identity validity | | | | | |

Recommendation and accepted quality/time/cost trade-off: __________

Evidence limits and next measurement needed: __________

Keep unavailable, not applicable, zero, and failed collection distinct. Link each
value to retained evidence. A diagnostic result must keep its ineligible label
and cannot serve as a publication-ready baseline.

## Source contracts

- [E40 epic](epic.md): business problem, G1-G19, constraints, and acceptance.
- [Architecture](architecture.md#workflow-value-attribution-contract): truth
  blocks, review attribution, interval/artifact evidence, and identity.
- [Design](shark-bench-design.md#4-per-metric-mechanics): collection and
  per-scenario comparison mechanics.
- [F09 specification](E40-F09-calibrated-evaluation-and-comparison-identity/spec.md):
  calibrated judgment, independent truth, finding confirmation, and comparison modes.
- [F10 specification](E40-F10-operator-workflow-and-retained-lifecycle-baseline/spec.md):
  baselines, noise bands, separate quality/time/cost, and reporting limits.
- [F11 specification](E40-F11-six-family-lifecycle-benchmark-readiness-and-cover/spec.md):
  six-family coverage and four-ceiling policy.
- [Operator comparison boundary](../../../bench/scripts/lib/e40_benchmark.py):
  validates baseline links, declared treatment axes, and unchanged controls.
