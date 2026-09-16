# Residual tech-debt closure plan — 2026-09-16

## Current-state reconciliation

The live `shark td list --json` collection has 50 non-terminal records: 44
`identified`, five `research`, and TD-212 `in_progress`.  This is not a
re-run of the 139-record plan in `2026-09-14-collection-plan.md`: PRs #220
through #246 already delivered its branches.  The residual queue combines
tracker records whose terminal transition was missed with five discoveries
made after that plan.

The canonical per-key ownership and expected-status source is
`scripts/residual-closure-manifest.json`; the batch table below is a readable
grouping of that manifest, not a second source of truth. Each key below has
exactly one owner. An evidence-closure batch may move a
record to a terminal status only after the listed merged PR, current source,
and relevant tests prove the recorded scope.  A branch name is reserved only
for a batch that still needs a repository change; already merged branches are
not recreated merely to replay history.

## Waves and batches

| Wave | Batch | Keys | Delivery boundary and evidence |
| --- | --- | --- | --- |
| 0 | Tracker closure | TD-212 | Close from PRs #245 and #246, note 3331, manifest/restart-test evidence, and a current quality-gate check. The abandoned-claim durable witness remains separately deferred to I-2026-09-15-01. |
| 0 | I-05 source-pinning research | TD-131 | Complete research first; branch only if the report proves a current remediation is needed. Do not assume the former E40 lifecycle branch covers a distinct trust boundary. |
| 0 | I-08 measurement-contract research | TD-146, TD-147 | Treat elapsed/cost joins and comparison deltas as one upstream-contract investigation. Do not change the staged E40 contract without an explicit producer/consumer decision. |
| 0 | Question-command research | TD-158 | Complete compatibility and information-architecture research before grouping it with TD-234. |
| 0 | Gate lease-lifetime research | TD-182 | Verify the reported multi-step release premise before altering gate persistence. |
| 1 | Core-runner revalidation | TD-150, TD-153, TD-156 | Revalidate the merged `tech-debt/core-runner-followups` PR #227 against each residual scope; split any real gap into a new focused branch. |
| 1 | Harness revalidation | TD-165 | Revalidate the merged `tech-debt/harness-contract` PR #230. A real run-harness execution seam is separate from text-only coverage. |
| 1 | Gate-ingestion reliability | TD-176–TD-181 | Revalidate PR #232, with focused gatepersist, command, and workercontrol tests. |
| 1 | Gate-ingestion concurrency | TD-183–TD-193 | Revalidate PR #233 after TD-182 research. Keep concurrency/lifecycle invariants separate from reliability-only cleanup. |
| 1 | Custom-key and impact contracts | TD-194–TD-199 | Revalidate PRs #235 and #236. Schema decisions such as TD-199 require an explicit contract disposition, not title-based closure. |
| 1 | Dependency and artifact accuracy | TD-200–TD-202, TD-204, TD-205 | Revalidate PRs #234 and #236, including the production `task create --depends-on` path. |
| 1 | Integration capture | TD-207, TD-209, TD-210, TD-217–TD-220 | Revalidate PR #237 with integration, override, prompt, and parser tests. Do not fold TD-212 recovery into this batch. |
| 2 | Runner-ingestion documentation | TD-230 | New branch `tech-debt/runner-ingestion-docs`; add a runnable, contract-accurate apply-result/run-id example, a populated `EvidenceRef` example, and calibrated `gate_result.summary` guidance that prevents the 1000-byte rejection. Pin all three with targeted validation. |
| 2 | Review-test diagnostics | TD-231 | New branch `tech-debt/review-test-diagnostics`; improve the targeted assertion without broad test-framework refactoring. |
| 2 | Triage-routing contract | TD-233 | New branch `tech-debt/triage-routing-contract`; place the canonical contract in the embedded bundle; update `review-code.md`, `tech-lead.md`, and `consolidator.md` as consumers; extend the `TestEmbedded_` policy gate to reject direct severity-to-entity routing without the triage decision tree. |
| 2 | Question API error contract | TD-234 | New branch `tech-debt/question-api-error-contract`; make shared service error text neutral across CLI and HTTP, preserving both API and CLI assertions. |
| owner | User-global finish-feature wording | TD-232 | This is outside the repository and has no repository branch/PR. Update the user-global skill only under the requested global-skill scope, then verify its triage decision-tree wording and record the outcome in Shark. |

## Branch and merge gates

For every new repository branch: create an isolated worktree from current
`origin/main`, complete the keyed Shark workflow, run focused tests plus
`make fmt && make lint && make test`, use `/shark-rider deep-review high`,
then run `/finish-feature` through live green CI, squash merge, and cleanup.

For every evidence-closure batch: preserve the original PR and exact test
evidence in a Shark note, read back the resulting record, and do not claim it
resolved solely because a branch name appears in Git history.  Research items
remain outside implementation until their workflow produces a concrete,
reviewable decision.

An evidence-closure note must contain, for **each key**, the source and merge
commit, a current focused command with its expected assertion, the current
`make fmt && make lint && make test` result and SHA, and a production-entrypoint
check when the record concerns CLI or runtime wiring. Historical CI and PR
evidence are supplemental only. TD-165 specifically requires an isolated,
executed `shark run` harness path that proves resolver wiring; a source-text
scan cannot close it. A failed current check reopens delivery on a new focused
branch rather than allowing tracker closure.

## Mechanical reconciliation

The checked reconciler is
`docs/plan/tech-debt/scripts/reconcile-residual-tech-debt.go`. Capture the
live collection once, then run it without filtering or hand-editing the
input:

```bash
shark td list --json > /tmp/shark-td-live.json
go run docs/plan/tech-debt/scripts/reconcile-residual-tech-debt.go /tmp/shark-td-live.json
```

It normalizes the live non-terminal collection, expands the plan's exact
per-key ownership manifest, and exits non-zero for a missing, extra,
duplicate, or status-mismatched key. Persist its JSON output with the
timestamped closure evidence. At plan creation it reports `active: 50`,
`planned: 50`, `duplicates: []`, and `missing: []`.
