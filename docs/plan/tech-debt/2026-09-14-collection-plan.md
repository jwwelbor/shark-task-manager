# Tech-debt delivery plan — 2026-09-14

## Purpose

Group active tech debt by executable subsystem and verification surface. Each row is one branch and one pull request. Do not mix rows without updating this plan because the review and rollback boundaries are intentional.

## Delivery groups

| Branch | Tech debt | Scope and PR boundary |
| --- | --- | --- |
| `tech-debt/b064-i05-evaluation-regressions` | TD-226–TD-229 | B064 evaluation review follow-ups and I-05 regressions. |
| `tech-debt/e40-i04-admission-hardening` | TD-087–TD-099, TD-105–TD-111 | I-04 admission, containment, duplicate handling, and adapter scaffolding. |
| `tech-debt/e40-i04-coverage-docs` | TD-100–TD-104, TD-112–TD-118 | I-04 coverage and documentation contracts. |
| `tech-debt/e40-evidence-utilities` | TD-119, TD-125, TD-128 | Shared benchmark evidence helpers without CLI-contract changes. |
| `tech-debt/e40-lifecycle-safety` | TD-120–TD-122, TD-124, TD-126–TD-132 | Lifecycle/evidence-path safety and content-root correctness. TD-123 remains separate. |
| `tech-debt/e40-f10-symlink-safety` | TD-133–TD-149 | F10 retention, oracle, symlink, spend-gate, and lifecycle integration. |
| `tech-debt/core-runner-followups` | TD-150–TD-156, TD-216 | Core runner and sprint follow-ups plus project-root reuse. |
| `tech-debt/question-cli-ux` | TD-158 | Question-command information architecture and compatibility. |
| `tech-debt/sprint-admission` | TD-159–TD-161 | Sprint admission evaluation, override service extraction, and coverage. |
| `tech-debt/harness-contract` | TD-162–TD-170 | Harness validation, sanitization, runtime wiring, and test isolation. |
| `tech-debt/f02-artifact-accuracy` | TD-171–TD-175 | E34-F02 documentation, demo, and test-plan accuracy. |
| `tech-debt/gate-ingestion-reliability` | TD-176–TD-181 | Gate persistence errors, bounded input, conflict semantics, and envelope validation. |
| `tech-debt/gate-ingestion-concurrency` | TD-182–TD-193 | Gate ownership, transition guards, locks, and dispatcher output bounds. |
| `tech-debt/custom-key-dependencies` | TD-194–TD-196, TD-200 | Custom-key error handling and task-dependency persistence. |
| `tech-debt/impact-schema-contracts` | TD-197–TD-202, TD-204, TD-205 | Impact fidelity, tests, and E34 artifact pointers. |
| `tech-debt/integration-capture` | TD-207, TD-209–TD-211, TD-213–TD-215, TD-217–TD-220 | Integration validation, fresh-init classification, publishing, prompt, and coverage. |
| `tech-debt/integration-retry-recovery` | TD-212 | Fail-closed backfill recovery across run, event, candidate, and registration write windows. |
| `tech-debt/gate-kickback-reopen` | TD-221, TD-222 | Deduplicate first, then implement terminal-task reopening once. |
| `tech-debt/i05-time-ledger` | TD-223–TD-225 | Provider-active interval accounting and I-05 writer/test-plan conformance. |
| `tech-debt/f06-evidence-research` | TD-123 | Research-gated F06 evidence-root hardening. |

## Execution rules

1. Start each group in an isolated worktree based on current `origin/main`.
2. Run focused tests while developing, then `make fmt`, `make lint`, and `make test` before opening a PR.
3. Complete review, PR, CI, merge, tracker verification, and branch/worktree cleanup for each ready group.
4. Resolve TD-221/TD-222 duplication before code changes and retain the surviving record's evidence on the merged PR.
5. Consult the architect before changing any item whose current design may have superseded the original finding.
