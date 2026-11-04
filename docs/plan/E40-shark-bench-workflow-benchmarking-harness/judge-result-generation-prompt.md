# Prompt: create the E40 feature judge result

Use this prompt in a fresh session to create the missing calibrated judge
result for the E40 lifecycle benchmark.

## Goal

Create a legitimate, file-backed `judge_result` for
`py-feature-recurring-tasks`, validate it against the E40-F09 contract, and
prepare a copied benchmark configuration that can use it in a fresh six-family
capture.

Do not run the full six-family capture until the judge result is complete and
validated.

## Non-negotiable evidence rules

- Do not fabricate a score, rationale, usage, cost, judge identity, digest, or
  calibration label.
- Do not reuse the synthetic judge values from
  `bench/scripts/tests/tc069_calibration_boundary_test.sh`.
- Do not reuse `bench/evaluation/review-truth-set.json`; that file calibrates
  review-finding confirmation, not planning/decomposition quality.
- Do not make D01-D05 non-applicable just to bypass the judge gate.
- Do not claim that a subscription provider cost is `$0` unless the provider
  contract or captured usage evidence supports that value.
- If human-scored calibration examples are unavailable, stop before writing an
  eligible result and ask for those labels. A schema-shaped guess is not valid
  benchmark evidence.

## Read first

From `/home/jwwel/projects/shark-task-manager`, read:

1. `AGENTS.md`, `CLAUDE.md`, and the applicable `.claude/rules/` files.
2. `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F09-calibrated-evaluation-and-comparison-identity/spec.md`.
3. `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F10-operator-workflow-and-retained-lifecycle-baseline/spec.md`.
4. `bench/scripts/evaluate-lifecycle.sh`, especially `run_judge()`.
5. `bench/scripts/run-lifecycle-batch.sh` and
   `bench/scripts/lib/e40_benchmark.py` for the `judge_result` path flow.
6. `bench/evaluation/i08-schema.yaml`.

Preserve all unrelated dirty files. Do not reset, clean, or overwrite the
working tree.

## Current benchmark inputs

Use this durable operator root for this walkthrough:

```text
/home/jwwel/projects/shark-task-manager-operator-work/e40-codex-six-family-20261005a
```

It is outside the Git checkout so its retained files do not become candidate
changes, and it is outside `/tmp` so routine temporary-file cleanup will not
remove it. Keep the human labels, raw judge outputs, result JSON, copied batch
configuration, validation output, and any new run retention under this root.

An earlier handoff named this old root and run:

```text
/tmp/e40-codex-six-family-20261001a/runs/codex-six-family-pilot-20261003a
```

That directory is absent in the current environment. Its evaluation and
prelude evidence are therefore unavailable here; do not treat its old
`missing_judge` result as freshly verified or edit the prior run in place if it
is later restored. The current scenario package still marks D01-D05 applicable.
No approved `judge-result.json` is present in this checkout or the new operator
root.

The current feature replay source is:

```text
bench/scenarios/packages/py-feature-recurring-tasks/evaluator/replay/reference-bundle.json
```

Its entry IDs in authoritative bundle order are:

```text
D01-01  problem_statement
D01-02  target_users
D02-03  success_metrics
D02-01  success_metrics
D02-02  constraints
D03-01  competitor_landscape
D03-02  market_sizing
D04-01  recurrence_library_options
D04-02  scheduling_algorithm_tradeoffs
D05-01  interview_mode_choice
D05-02  synthesis_confirmation
```

The raw-byte SHA-256 of that bundle in this checkout is
`6491a0dfb86155c03b3734b1c16c5d8e3c4cfb2f8c1df65b855a557498825b3b`.
Recompute it before use if the file changes. The current rubric and fixed judge
prompt raw-byte SHA-256 values are `2a2ab7eefde5a287a5b0aff3b56b0a01c0c8cc0d368e2eef77c227271d09ed42`
and `153a2487784fca3cb09253a032b90488a5c03e40f68521c8fa1e31f0476f1e14`.
Recompute those if either source file changes.

The I-08 `reference_digest` contract must identify exactly which authoritative
bytes it covers. The evaluator currently checks digest syntax but does not
prove that this value matches the bundle.

## Procedure

### 1. Confirm the gap

Run read-only checks that confirm:

- the feature package has D01-D05 marked applicable;
- if prior I-05/I-07/I-08 evidence is restored, the prior feature I-08 record
  has `missing_judge` (otherwise report that the old result is unavailable);
- the other truth blocks are not being substituted for the judge;
- no approved judge result already exists in the operator root or repository.

The old retained run is not available at the path above. Real evaluator
validation also needs a matching I-05 bundle and I-07 lifecycle record. Restore
those exact files or produce fresh feature-run evidence before claiming that
the judge result passed the real evaluator path.

### 2. Define the judge contract

Write down the rubric before scoring. It must assess the quality of the
planning/decomposition evidence, not implementation correctness. At minimum,
the rubric must distinguish:

- problem and target-user clarity;
- measurable success criteria and constraints;
- relevance and support of research claims;
- feasibility and trade-off reasoning;
- consistency of the final synthesis with the stated scope.

Record the exact rubric text or canonical rubric file and compute its lowercase
SHA-256 digest.

Create or identify the exact judge prompt and compute its lowercase SHA-256
digest. Record the actual judge model and configuration used.

### 3. Establish human calibration

Use a calibration set that is separate from the eleven feature evaluation IDs.
For each calibration example, retain:

- a stable `example_id`;
- the human score or label;
- the rubric version used;
- enough bounded source/reference identity to reproduce the calibration
  decision.

Have the human scorer review all six examples using
`calibration/v1/human-scores.md`. Copy
`calibration/v1/human-labels.template.json` to
`/home/jwwel/projects/shark-task-manager-operator-work/e40-codex-six-family-20261005a/judges/py-feature-recurring-tasks/human-labels.json`;
keep the five 0-4 grades and their rationales there. Set each numeric `label`
to `(P + M + R + F + S) / 20`, and record reviewer ID/date and completed review
provenance. The calibration-set digest is the canonical JSON digest of this
completed file, using sorted keys, compact separators, UTF-8, and
`ensure_ascii=False`.

Before judging the held-out feature, run the fixed judge prompt on the six
calibration examples **without supplying the human labels**. Retain the judge's
raw output, model/configuration, usage, and supported cost. Compare its
per-dimension and overall scores with the human labels; record human review of
disagreements before proceeding. Use the same actual judge model/configuration,
rubric, and prompt for calibration and held-out evaluation. Do not label
calibration examples by asking the judge to label itself.

If the session has no access to the human labels, stop and report exactly which
labels are needed. Do not emit a passing file.

### 4. Judge the feature replay

Apply the fixed `calibration/v1/judge-prompt.txt`, rubric, and judge
configuration to the D01-D05 replay evidence, in the bundle's actual entry
order shown above. Keep the calibration IDs disjoint from the evaluation IDs.
Retain the complete judge response (including dimension scores) beside the
result file. This replay-only score can satisfy the current judge-input gate;
it does not by itself support a prompt/model comparison. The performance rubric
requires scoring generated artifacts for that purpose. Record:

- the actual bounded score in the range `0` through `1`;
- a concise rationale tied to the rubric;
- the exact evaluation-set IDs used;
- actual token usage;
- actual judge cost, or an explicitly documented zero-cost subscription
  measurement if that is the provider’s recorded contract.

The score must reflect the planning evidence. Do not let the later passing
implementation oracle overwrite or inflate the planning score.

### 5. Write the result

Create the operator-side file at:

```text
/home/jwwel/projects/shark-task-manager-operator-work/e40-codex-six-family-20261005a/judges/py-feature-recurring-tasks/judge-result.json
```

Use this shape, replacing every placeholder with captured values:

```json
{
  "rubric_digest": "<64 lowercase hexadecimal characters>",
  "prompt_digest": "<64 lowercase hexadecimal characters>",
  "judge_model": "<actual judge model>",
  "configuration": "<actual judge configuration>",
  "calibration_set_digest": "<64 lowercase hexadecimal characters>",
  "calibration_examples": [
    {
      "example_id": "<human-scored calibration example>",
      "label": 0.0
    }
  ],
  "evaluation_set_ids": [
    "D01-01",
    "D01-02",
    "D02-03",
    "D02-01",
    "D02-02",
    "D03-01",
    "D03-02",
    "D04-01",
    "D04-02",
    "D05-01",
    "D05-02"
  ],
  "reference_digest": "<digest of the authoritative replay/reference set>",
  "rationale": "<bounded rubric-based rationale>",
  "score": 0.0,
  "usage": {
    "input_tokens": 0,
    "output_tokens": 0
  },
  "cost_usd": 0.0
}
```

The zero values above are placeholders only. Replace them with captured values;
never leave them merely because the schema accepts them.

### 6. Validate the result

Run JSON parsing and the real evaluator/validator paths available in the
current checkout. At minimum:

```bash
python3 -m json.tool \
  /home/jwwel/projects/shark-task-manager-operator-work/e40-codex-six-family-20261005a/judges/py-feature-recurring-tasks/judge-result.json

bench/scripts/tests/tc069_calibration_boundary_test.sh
```

Use a fresh output path under the durable operator root for any evaluator
replay, and pass the matching I-05/I-07 evidence and scenario package. Confirm
that the judge block reports:

```text
applicability: applicable
observed_result: pass
invalidity_reasons: []
```

Do not edit any retained run in place. Preserve prior or failed runs as
diagnostic evidence. TC-069 is a synthetic contract regression suite; its
fixture values are never calibration labels and do not validate provenance for
this specific result.

### 7. Prepare the next-run configuration

Copy the authoritative operator config instead of overwriting it, if it is
restored. Otherwise rebuild the config from the authoritative six-scenario
batch policy and record that the prior config was unavailable. Add this path
only to the copied config’s feature scenario:

```yaml
judge_result: /home/jwwel/projects/shark-task-manager-operator-work/e40-codex-six-family-20261005a/judges/py-feature-recurring-tasks/judge-result.json
```

Confirm that the copied config preserves the six admitted scenarios, the
feature replay result, and the explicit long wall-clock ceiling that the next
operator command will provide.

Do not start the six-family run in this session unless the user explicitly asks
for it. The next session should start it with a fresh run ID and an explicit
`--max-wall-clock-seconds 86400` ceiling.

## Completion criteria

Report one of these outcomes:

1. **Ready:** the judge JSON exists, its calibration provenance is real and
   disjoint, validation passes, and the copied config points to it. Include
   exact paths, digests, judge identity, score, usage, and cost.
2. **Needs human labels:** the required calibration examples or labels are
   missing. List the exact missing inputs and stop without creating an eligible
   result.
3. **Contract defect:** the repository cannot validate a legitimately captured
   result. Preserve the evidence, name the failing command and path, and do not
   weaken the evaluator.
