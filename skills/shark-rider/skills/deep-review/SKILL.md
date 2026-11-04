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
| 2 | Host subagent dispatch without `Workflow` | `dispatched-six-angle` | Complete when 6 angle reviewers and the consolidator finish |
| 3 | Alternate/comparable CLI | `adversarial-cli` | Complete when one bounded comparable CLI review is structurally valid |
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

Determine host subagent-dispatch availability before running
`scripts/adaptive_review.py`. Host dispatch is available when the current
runtime provides a tool that can start independent subagents. Do not infer
that dispatch is unavailable because native `Workflow` is unavailable.

`scripts/adaptive_review.py` cannot detect or invoke host-native subagent
dispatch. Use it for the CLI fallback and for report persistence only. Do not
pass `--no-agent-dispatch-available` when host dispatch exists. If native
Workflow is unavailable and host dispatch exists, Runner 2 is mandatory; do
not use Runner 3 instead.

The script emits an explicit `INCOMPLETE` report when no valid automated
reviewer is available, and rejects timeout, non-zero exit, empty, malformed,
and partial CLI output. It runs CLI review in read-only/plan mode with a
bounded timeout. The host is supplied as `--host codex|claude` (or
`DEEP_REVIEW_HOST`), so Codex-hosted sessions prefer Claude and Claude-hosted
sessions prefer Codex. If the preferred alternate CLI is unavailable,
unauthenticated, exits non-zero, times out, or returns malformed output, retry
with the host's own comparable CLI when it is installed. In a Codex-hosted
session, this means falling back to `codex exec --sandbox read-only` when
Claude CLI cannot complete the review. Record the CLI that actually produced
the valid review in `adversarial_model` and preserve the fallback reason.

## Integrated workflow review

For a dispatched code-review gate, use the rendered prompt unchanged. For bug reviews, pass the accepted plan and revision, research, coverage evidence and prior findings as `repair_context`, together with `entity_type=bug`, to the native workflow; supply that same context to every host-dispatched angle and consolidator. Apply the
portable craft criteria included from `skills/quality/workflows/review-code.md`
as the canonical craft rubric alongside the angle assignments. The existing
parent retains its lease, heartbeats, transitions and result ingestion; angle
reviewers only supply evidence. Include requested and verified effective model
identity separately, marking unknown identity explicitly. Bind the complete
manifest, candidate tree and diff hash to every review and the accepted plan
revision when applicable. Reject stale or incomplete evidence. The scope must
include every unaccepted predecessor change; obtain the accepted baseline from
the plan rather than assuming the last commit. If capture cannot represent the
full candidate, stop and report incomplete. Pass the captured JSON as
`--scope-file` to `adaptive_review.py run-cli` or `write-report`; the writer
revalidates candidate, diff and manifest and persists their identities. Pass
`--requested-model` and `--effective-model` only when known; absent values are
recorded as unknown.

For a bug, set `REVIEW_BASE` to the independently approved `bug_review_baseline` commit and `REVIEW_PLAN_SHA256` to the 64 lowercase hex characters of the approved plan digest before capture. Every angle must include that digest as `candidate_identity.repair_plan_sha256`; supply the same scope field and `repair_context.plan_revision` (`sha256:<digest>`) to the native workflow. A report for a different plan revision or baseline cannot satisfy the gate.

On Windows, snapshot inventory delegates its confined filesystem reads to the installed Shark binary; it must include the matching `review-source-inventory` helper. Set `SHARK_REVIEW_INVENTORY_BIN` to that binary when it is not the `shark` on PATH. The parent still owns review dispatch and transitions.

## Capture the review scope

```bash
project_root=$(git rev-parse --show-toplevel)
skill_dir="/home/jwwel/.claude/skills/shark-rider/skills/deep-review"
REVIEW_BASE="<accepted-baseline-commit>" bash "$skill_dir/scripts/get_diff.sh" > /tmp/deep-review-scope.json
```

The capture requires a clean committed candidate. If relevant work is uncommitted,
the parent first enumerates every candidate path, including unaccepted predecessor
changes and relevant untracked files, and prepares an isolated snapshot without
altering the source worktree or index:

```bash
python3 "$skill_dir/scripts/prepare_candidate.py" --project-root "$project_root" \
  --destination /absolute/new/review-worktree --metadata-file /absolute/new/snapshot.json \
  path/to/first-candidate path/to/second-candidate
cd /absolute/new/review-worktree
REVIEW_BASE="<accepted-baseline-commit>" REVIEW_SOURCE_METADATA=/absolute/new/snapshot.json \
  bash "$skill_dir/scripts/get_diff.sh" > /tmp/deep-review-scope.json
```

Select paths explicitly; do not include unrelated work. Inspect the full diff and
manifest to ensure no relevant work was omitted and no unrelated work included.
The scope validator rechecks selected source files and source HEAD before accepting
the report. Retain the isolated worktree and snapshot metadata for review and QA;
never modify or clean the original worktree as part of capture. If a complete
snapshot cannot be prepared, report INCOMPLETE. Set `REVIEW_BASE` to the accepted
baseline when specified; otherwise omit it to use the merge base.

Retain `candidate_commit`, `candidate_tree`, `diff_sha256`, `diff_path`, `base_commit`, `changed_files`, `changed_file_count`, `diff_shortstat`,
`project_root`, `coding_standards_path`, `branch`, and `review_output_path`. Review the complete
changed-file list unless the user explicitly narrows scope.

## Runner 1: native Workflow

When `Workflow` is available, invoke `scripts/review_workflow.js` with the scope metadata and the
requested effort (`low|medium|high|xhigh|max`). It runs all six angle prompts in parallel and the
consolidator. Only a valid result from all six angles, complete changed-file coverage, and a valid consolidator report may be recorded as complete. An exception, partial result, coverage gap, or malformed report is INCOMPLETE; persist actual counts, or `unknown` when the host did not return them, and never replace it with a CLI pass. The returned report is compact on a clean PASS and detailed only for findings or
triage. Persist the returned body with `adaptive_review.py write-report` and:

The native result contains `report`, `angle_results`, and `decisions`; save these separately and pass the latter two with `--angle-results` and `--review-decisions` to `write-report`. For host dispatch, assemble the six returned objects under keys `a` through `f` and ask the consolidator for its candidate-bound decisions. Do not substitute caller-supplied counts for these artifacts. Save it first; do not dump the full report inline on a clean PASS. Tell the user only a short verdict summary plus `review_output_path`.

```text
runner_mode=canonical-six-angle
specialists_completed=6
consolidator_completed=true
adversarial_model=none
fallback_reason=none
base_commit=<captured base commit>
diff_path=<captured diff path>
```

## Runner 2: host-dispatched six-angle review

Use this runner whenever native Workflow is unavailable and the host exposes
subagent dispatch. Do not use the alternate-model CLI in that case.

1. Read `references/angle-a-bugs.md` through
   `references/angle-f-standards.md` and `references/consolidator.md`.
2. Dispatch all six angle agents in one batch through the host subagent tool.
   Give every agent the complete captured scope, its one assigned angle
   prompt, the canonical craft criteria (`shark skill get quality workflows/review-code.md` or its extracted file), and these constraints: read-only work; no edits, commits, pushes,
   or PR comments; return the required JSON schema with `candidate_identity` containing the exact captured commit, tree and diff hash; list every file inspected.
   Submit all six dispatches at once. The host may queue work when its
   concurrency limit is lower than six.
3. Wait for all six results. If any result is missing, malformed, timed out, has a mismatched candidate identity,
   or does not list the entire changed-file manifest, mark the review `incomplete` and skip to
   step 5 to persist it. Do not replace a failed dispatched review with the
   CLI fallback; the comparable-CLI retry rule applies when Runner 3's
   preferred alternate CLI is unavailable, not when a six-angle dispatch is
   incomplete.
4. Dispatch one read-only consolidator with `references/consolidator.md`, the
   captured scope (`diff_path`, `base_commit`), the complete changed-file
   list, and all six specialist results. Require the consolidator to verify
   coverage and return the required Markdown report and a decisions JSON object. The decisions object contains `candidate_identity` and a `findings` array; each finding has `file`, positive integer `line`, `severity`, `rule`, `summary`, `diagnosis`, `disposition`, and a nonempty `reason`. Every raw finding must have an exact matching decision. Before a pass, every blocker needs disposition `refuted` and an evidence-based reason; a clean PASS cannot retain an unrefuted finding. An empty findings array explicitly asserts no findings.
5. Persist with `scripts/adaptive_review.py write-report`, passing the
   captured `base_commit` and `diff_path` and the consolidator's verdict. If
   all six specialist results and the consolidator are valid, use
   `runner_mode=dispatched-six-angle`, `specialists_completed=6`,
   `consolidator_completed=true`, `adversarial_model=none`,
   `fallback_reason=native Workflow unavailable`. Otherwise use
   `runner_mode=incomplete` with `specialists_completed` and
   `consolidator_completed` reflecting what actually completed.

## Runner 3: alternate/comparable-model CLI

Build one read-only adversarial prompt containing the diff path, complete changed-file list,
standards path, task context, and the following required output contract:

```text
Return a JSON object containing `report` (the Markdown review, including
scope, findings and verdict) and `decisions` (the same candidate-bound
findings and disposition object required for six-angle consolidation).
```

Select and run it with:

```bash
python3 "$skill_dir/scripts/adaptive_review.py" select --host "${DEEP_REVIEW_HOST:-unknown}"
python3 "$skill_dir/scripts/adaptive_review.py" run-cli \
  --prompt-file /path/to/adversarial-prompt.txt \
  --project-root "$project_root" --diff-path "$diff_path" --base-commit "$base_commit" \
  --review-output-path "$review_output_path" --scope-file /tmp/deep-review-scope.json --host "${DEEP_REVIEW_HOST:-unknown}"
```

The runner tries the preferred opposite-host CLI first, then any installed
comparable CLI (including the host CLI) when the preferred process is
unavailable or its output fails validation. It records the CLI that actually
produced the valid result in `adversarial_model`,
`specialists_completed=0`, and `consolidator_completed=true` only after a
complete adversarial result validates. A successful fallback is not a
six-angle review and must be reported as `adversarial-cli` with the fallback
reason retained.

## Runner 4: manual diagnostic

A human/local review may be used to understand failures or prepare a later automated run. Record
`runner_mode=manual-diagnostic`, `specialists_completed=0`, `consolidator_completed=false`, and
why automated review was unavailable. It cannot satisfy the unattended pre-merge gate.

## Consolidation contract

`references/consolidator.md` accepts either six angle findings or one complete adversarial review.
It must state the evidence mode and use the matching coverage claim. It may say
“6-angle automated” only for `canonical-six-angle` or `dispatched-six-angle` with six completed
specialists. For fallback findings, fix every blocker or route it through the triage skill's decision tree (see `skills/triage/SKILL.md`'s routing contract; tech-debt is only one possible outcome, never a default) before CI/merge. An incomplete report cannot advance the gate.

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
