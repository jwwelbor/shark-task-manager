# Establish human calibration for the planning judge

Use [the E40 prompt and model performance rubric](../../../prompt-model-performance-rubric.md)
for the full evaluation across quality, time, cost, review value, and artifact
use. This packet creates the first human reference scores for its
planning/decomposition judge component.
Start with [the scoring worksheet](human-scores.md), which links to the rubric
and six examples. You can return your ratings in chat instead of editing files.

## Review the materials

- [Rubric](rubric.md): five equally weighted dimensions and common score anchors.
- [Examples](examples.md): six independent planning packets, each with its own
  bounded reference material and D01-D05 entries.
- [Worksheet](human-scores.md): blank ratings, reviewer identity, and rationale.
- [Judge prompt](judge-prompt.txt): fixed instructions for the later judge run.
- `examples.json`: the same examples with source and entry identities.
- `human-labels.template.json`: blank machine-readable human labels.
- `manifest.json`: file and canonical-example digests, version, and excluded
  evaluation IDs.

The examples and their reference material are newly authored fictional
scenarios. They are not actual interviews, market research, production runs, or
previous human ratings. Statements explicitly placed in each reference sheet
are facts of that fictional scenario. Statements added by the planning packet
still need support from that sheet. No suggested human grades are supplied.

This is a small initial calibration exercise, not evidence of broad judge
reliability. The six examples span three topics and different patterns of
clarity, support, feasibility, and scope consistency. E40-F09 requires a
nonempty, separate human-scored set; six is this packet's design choice, not a
new evaluator requirement.

## Score the examples

1. Read the rubric. Use the same rubric for all six examples.
2. Read one example's reference material and all five planning entries.
3. Grade each dimension from 0 through 4. Record a brief explanation tied to
   an entry or reference ID. Judge the evidence supplied, including missing
   support or conflicting scope.
4. Repeat for all examples. Leave a rating blank if you need clarification;
   a blank is missing evidence and must not become a zero.
5. Record your reviewer identity and review date. If another person scores the
   set, retain their independent worksheet and resolve disagreement explicitly.
6. Return the worksheet or the six rating lines in chat. Your submitted ratings
   become the human labels. The assistant may transcribe them and calculate
   their means, but must not select or change your grades.

The normalized score for a dimension is `grade / 4`. The overall label is
`(P + M + R + F + S) / 20`, matching the rubric's arithmetic mean. The rubric
defines planning quality; it does not define a minimum feature score for
publication.

## Run calibration after human scoring

Freeze the reviewed rubric and examples before scoring the held-out feature.
Fill a copy of `human-labels.template.json` from the actual submitted human
ratings. Record its canonical JSON digest only after labels and reviewer
provenance are complete. A draft template digest is not a calibration-set
digest. Canonical JSON uses sorted keys, compact separators, UTF-8, and
`ensure_ascii=False`; raw file digests cover exact bytes.

Use the same actual judge model, configuration, rubric, and prompt for
calibration and feature evaluation. Keep human labels out of the judge's first
calibration scoring input; compare its predictions with the retained human
labels afterward. Retain per-dimension differences and overall absolute
differences. Review material disagreement before proceeding. This packet adds
no numerical calibration-pass threshold to the existing evaluator contract.
Human review of discrepancies must be recorded; do not silently revise labels
to match the judge. If you revise the rubric, create a new version and obtain
human scores under that version before judging the feature.

Capture the actual provider/model configuration, invocation payload, token
usage, and supported cost from the judge run. This packet provides no model
identity, provider usage, cost, or eligible `judge_result`.

## Keep the feature held out

These calibration examples do not contain the recurring-task feature's eleven
entries, its implementation, or its oracle results. The excluded evaluation
IDs are `D01-01`, `D01-02`, `D02-01`, `D02-02`, `D02-03`, `D03-01`, `D03-02`,
`D04-01`, `D04-02`, `D05-01`, and `D05-02`. Distinct names alone would not make
copied evaluation content a separate calibration set.

The rubric and judge instructions preserve the exact bytes prepared before
held-out response inspection in the preceding session. These fictional
examples were authored afterward, so this is not a blind study of corpus
design; keep that limitation in any eventual calibration report. No grades
were selected from the held-out feature or from implementation success.

After genuine labels and captured judge outputs are available, return to
[the judge-result generation procedure](../../../judge-result-generation-prompt.md).
Validate the result through the real evaluator before copying the six-family
configuration. A benchmark capture is a later operation.
