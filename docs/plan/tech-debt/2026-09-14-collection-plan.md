# Tech-debt delivery plan — 2026-09-14

## Purpose

The active queue contains 139 non-terminal tech-debt records.  This plan groups
them by executable subsystem and verification surface, rather than by their
creation order.  Each row is one branch and one pull request.  A PR must not
include work from another row without first updating this plan, because the
independent review and rollback boundaries are intentional.

## Delivery order

| Order | Branch | Tech debt | Scope and PR boundary |
| --- | --- | --- | --- |
| 1 | `tech-debt/b064-i05-evaluation-regressions` | TD-226, TD-227, TD-228, TD-229 | Current B064 review findings: stale oracle replacement, remove-or-wire dead lineage after evidence check, and two I-05 regression tests. |
| 2 | `tech-debt/e40-i04-admission-hardening` | TD-087–TD-099, TD-105–TD-111 | I-04 admission/adapter correctness, containment, duplicate handling, and shared adapter scaffolding. |
| 3 | `tech-debt/e40-i04-coverage-docs` | TD-100–TD-104, TD-112–TD-118 | I-04 coverage and documentation contracts only; no admission behavior change unless a test demonstrates it. |
| 4 | `tech-debt/e40-evidence-utilities` | TD-119, TD-125, TD-128 | Consolidate duplicated benchmark evidence helpers while preserving each script's CLI contract. |
| 5 | `tech-debt/e40-lifecycle-safety` | TD-120–TD-122, TD-124, TD-126–TD-132 | Lifecycle/evidence path safety, reproducibility, and content-root correctness. TD-123 remains exclusively in its research-gated row. |
| 6 | `tech-debt/e40-f10-symlink-safety` | TD-133–TD-149 | F10 retention, oracle, symlink, spend-gate, and lifecycle integration behavior. Split further only if tests show an incompatible contract boundary. |
| 7 | `tech-debt/core-runner-followups` | TD-150–TD-156, TD-216 | Core runner/sprint follow-ups plus the shared project-root implementation; TD-216 removes the duplicate root walk instead of merely changing its comment. |
| 8 | `tech-debt/question-cli-ux` | TD-158 | Focused question-command information architecture and compatibility review. |
| 9 | `tech-debt/sprint-admission` | TD-159–TD-161 | Sprint admission evaluation/override service extraction, lifecycle, and test-interface coverage. |
| 10 | `tech-debt/harness-contract` | TD-162–TD-170 | E34-F01 harness validation, sanitization, runtime wiring, and test isolation. |
| 11 | `tech-debt/f02-artifact-accuracy` | TD-171–TD-175 | E34-F02 documentation, demo, and test-plan accuracy only. |
| 12 | `tech-debt/gate-ingestion-reliability` | TD-176–TD-181 | Gate persistence error handling, bounded input, conflict semantics, command/service separation, and envelope validation. |
| 13 | `tech-debt/gate-ingestion-concurrency` | TD-182–TD-193 | Gate lifecycle ownership, transition guards, lock safety, and bounded dispatcher output. |
| 14 | `tech-debt/task-dependency-persistence` | TD-200 | Delivered by PR #234 (`tech-debt/custom-key-dependencies`): persist `--depends-on` task creation in the canonical relationship graph, with CLI-level regression coverage. |
| 15 | `tech-debt/custom-key-integrity` | TD-194–TD-196 | Delivered by PR #235: custom-key duplicate-check error handling, focused abstractions, and negative-path regression coverage. |
| 16 | `tech-debt/impact-schema-contracts` | TD-197–TD-202, TD-204, TD-205 | Impact schema fidelity, test coverage, and E34 artifact-pointer corrections. |
| 17 | `tech-debt/integration-capture` | TD-207, TD-209–TD-215, TD-217–TD-220 | Integration run validation, fresh-init classification, retry semantics, cancellation, atomic publishing, review prompt, and coverage. |
| 18 | `tech-debt/gate-kickback-reopen` | TD-221, TD-222 | First deduplicate the two identical records, then implement and test terminal-task kickback reopening exactly once. |
| 19 | `tech-debt/i05-time-ledger` | TD-223–TD-225 | Provider-active interval accounting and I-05 writer/test-plan conformance. |
| 20 | `tech-debt/f06-evidence-research` | TD-123 | Research-state, high-severity F06 evidence-root scan: confirm the current design before implementation; archive it if already superseded. |

## Execution rules

1. Work only from an isolated branch/worktree based on current `origin/main`.
2. Run the relevant focused test while developing, then `make fmt`, `make lint`,
   and `make test` before a PR is ready.
3. Run the full review/PR/CI/merge lifecycle through `/finish-feature` for each
   ready branch.  Do not claim a group closed merely because its individual
   tracker records were edited.
4. Resolve TD-221/TD-222 duplication before code changes and retain the
   surviving key's evidence on the merged PR.
5. For TD-123 and any item whose present architecture is uncertain, consult the
   architect and either record the current design decision or close it as
   superseded; do not implement a stale review suggestion.
