# E38-F02 Finite-State Regression Matrix

Status: planned follow-up from PR-128 review

## Objective

Prove the scheduler's safety-critical state partitions at the public
`Scheduler.Start`/`Scheduler.Resume` seams. Tests must exercise injected
claim, ledger, resolver, dispatcher, resource, clock, and event boundaries;
they must not test only private helpers or pre-sanitized outcomes.

## Matrix

| Area | Required partitions | Assertions |
| --- | --- | --- |
| Root ownership | initial heartbeat failure; periodic failure; cancellation; bounded diagnostic | no child claim/CAS/dispatch after loss; active claims released; planned children remain planned; run is paused/resumable |
| Dependencies | failed, blocked, paused, cancelled, skipped, satisfied external, unsatisfied external, missing target | dependent item is blocked/skipped with bounded reason; unrelated ready work continues |
| Dispatch gates | pause, human gate, terminal/archive, unresolved workflow, unresolved placeholder | no claim or dispatch; terminal action metadata is preserved |
| Worker outcomes | success, non-zero exit, provider-not-found, dispatcher error, result-plus-error, cancellation, panic | exact item outcome/evidence; cleanup; unrelated work continues |
| Claims and CAS | claim race, item CAS race, stale attempt, release false/error, replacement claim | one winner; no force steal; no replacement release; durable diagnostic |
| Resume and retry | completed item, stale claimed/running item, repeated resume, explicit retry, conflicting terminal result | no duplicate dispatch; stale claims reconciled; attempts and idempotency preserved |
| Capacity and events | sequential with oversized limit; parallel bounds; capability fallback; event allow-list | effective concurrency is correct; fallback reason durable; events contain no prompt/secret/transcript |

## Execution order

1. Add red tests at `Scheduler.Start` and `Scheduler.Resume` for root ownership,
   dispatch gates, and worker outcome partitions.
2. Add dependency, claim/CAS, and resume/retry decision-table cases.
3. Add capacity and event-boundary assertions, including race-enabled runs.
4. Run targeted `internal/team`, repository, dispatch, CLI, and contract tests.
5. Run `make fmt && make lint && make test` with writable Go caches when needed.

## Exit criteria

- Every row has at least one production-boundary test for each listed partition.
- Failure paths assert both durable state and absence of forbidden mutation.
- Tests cover the exact session, attempt, status, reason, and event fields.
- The existing E38-F02 review note is updated with the test names and results.
