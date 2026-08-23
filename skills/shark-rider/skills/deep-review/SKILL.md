---
name: deep-review
description: >
  Adaptive multi-angle code review. Selects the strongest available runner: native Workflow,
  six dispatched angle reviewers, an out-of-band alternate-model CLI, or a clearly incomplete
  diagnostic result. Never labels fallback evidence as the canonical six-specialist review.
  Triggered by /deep-review, /comprehensive-review, and /pr-review.
---

# Deep Review

Use the capability ladder below in order. The persisted report must state the actual evidence
mode; a manual or partial review is never silently promoted to a six-angle automated review.

| Priority | Capability | `runner_mode` | Pre-merge status |
|---|---|---|---|
| 1 | Native `Workflow` | `canonical-six-angle` | Complete when all 6 specialists and the consolidator finish |
| 2 | Agent dispatch without `Workflow` | `dispatched-six-angle` | Complete when 6 angle reviewers and the consolidator finish |
| 3 | Alternate-model CLI | `adversarial-cli` | Complete when one bounded CLI review is structurally valid |
| 4 | Local/manual review | `manual-diagnostic` | Diagnostic only; stop before merge unless the user explicitly authorizes an exception |
| none | No valid automated evidence | `incomplete` | Stop before CI/merge |

## Required report metadata

Every persisted report, including failures, must contain these fields in its header or machine-
readable metadata block:

```text
runner_mode
specialists_completed
consolidator_completed
adversarial_model
fallback_reason
base_commit
diff_path
```

Use `scripts/adaptive_review.py` for capability selection and the CLI fallback. It emits an
explicit `INCOMPLETE` report when no valid automated reviewer is available, and rejects timeout,
non-zero exit, empty, malformed, and partial output. It runs CLI review in read-only/plan mode
with a bounded timeout. The host is supplied as `--host codex|claude` (or `DEEP_REVIEW_HOST`), so
Codex-hosted sessions prefer Claude and Claude-hosted sessions prefer Codex.

## Capture the review scope

```bash
project_root=$(git rev-parse --show-toplevel)
skill_dir="/home/jwwel/.claude/skills/shark-rider/skills/deep-review"
bash "$skill_dir/scripts/get_diff.sh" > /tmp/deep-review-scope.json
```

Retain `diff_path`, `base_commit`, `changed_files`, `changed_file_count`, `diff_shortstat`,
`project_root`, `coding_standards_path`, `branch`, and `review_output_path`. Review the complete
changed-file list unless the user explicitly narrows scope.

## Runner 1: native Workflow

When `Workflow` is available, invoke `scripts/review_workflow.js` with the scope metadata and the
requested effort (`low|medium|high|xhigh|max`). It runs all six angle prompts in parallel and the
consolidator. The returned report is compact on a clean PASS and detailed only for findings or
triage. Persist the returned body with `adaptive_review.py write-report` and:

Save it first; do not dump the full report inline on a clean PASS. Tell the user only a short verdict summary plus `review_output_path`.

```text
runner_mode=canonical-six-angle
specialists_completed=6
consolidator_completed=true
adversarial_model=none
fallback_reason=none
base_commit=<captured base commit>
diff_path=<captured diff path>
```

## Runner 2: agent dispatch without Workflow

Read `references/angle-a-bugs.md` through `references/angle-f-standards.md` and launch all six
angle agents in one parallel dispatch. Require each to return its JSON schema. Then run
`references/consolidator.md` with all six results. Persist as `dispatched-six-angle` only if all
six specialist results and the consolidator are valid; otherwise use `incomplete`.

## Runner 3: alternate-model CLI

Build one read-only adversarial prompt containing the diff path, complete changed-file list,
standards path, task context, and the following required output contract:

```text
REVIEWED SCOPE: <files and coverage>
FINDINGS: <table or explicit none>
<evidence and corrections>
VERDICT: PASS | PASS-with-triage | FAIL
```

Select and run it with:

```bash
python3 "$skill_dir/scripts/adaptive_review.py" select --host "${DEEP_REVIEW_HOST:-unknown}"
python3 "$skill_dir/scripts/adaptive_review.py" run-cli \
  --prompt-file /path/to/adversarial-prompt.txt \
  --project-root "$project_root" --diff-path "$diff_path" --base-commit "$base_commit" \
  --review-output-path "$review_output_path" --host "${DEEP_REVIEW_HOST:-unknown}"
```

The script records `adversarial_model`, `specialists_completed=0`, and
`consolidator_completed=true` only after a complete adversarial result validates. A successful
fallback is not a six-angle review and must be reported as `adversarial-cli`.

## Runner 4: manual diagnostic

A human/local review may be used to understand failures or prepare a later automated run. Record
`runner_mode=manual-diagnostic`, `specialists_completed=0`, `consolidator_completed=false`, and
why automated review was unavailable. It cannot satisfy the unattended pre-merge gate.

## Consolidation contract

`references/consolidator.md` accepts either six angle findings or one complete adversarial review.
It must state the evidence mode and use the matching coverage claim. It may say
“6-angle automated” only for `canonical-six-angle` or `dispatched-six-angle` with six completed
specialists. For fallback findings, fix every blocker/non-blocker or explicitly triage it before
CI/merge. An incomplete report cannot advance the gate.

## Flags and references

`--fix` is limited to confirmed one-line safe corrections (missing await, inverted boolean,
copy-paste variable, or loop-bound error). `--comment` may post verified findings as PR comments.
Read the angle prompts and consolidator only when that runner is selected:

| File | Focus |
|---|---|
| `references/angle-a-bugs.md` | Bugs and production caller chains |
| `references/angle-b-behavior.md` | Removed behavior and SOLID |
| `references/angle-c-sibling.md` | Contracts, siblings, round trips |
| `references/angle-d-cleanup.md` | Reuse, complexity, idioms |
| `references/angle-e-tests.md` | Tests and counter-factuals |
| `references/angle-f-standards.md` | Standards, security, error handling |
| `references/consolidator.md` | Verification, triage, evidence-mode report |
