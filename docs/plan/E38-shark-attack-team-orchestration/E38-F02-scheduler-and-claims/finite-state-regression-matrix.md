# E38-F02 Finite-State Regression Matrix

This matrix is the regression checklist for T-E38-F02-005. Tests must enter at
the production scheduler boundary and use the Caller-Path Contracts in
`test-plan.md`.

| Surface | Required partitions | Contract source |
|---|---|---|
| Snapshot and dependencies | confirmed hash, drift, ready wave, failed/blocked/paused/cancelled/skipped/unsatisfied prerequisite | spec.md REQ-F-001–002; TC-001–003 |
| Claims and leases | single winner, exact session, root/child heartbeat, positive TTL, zero TTL, replacement claim | spec.md REQ-F-004, REQ-F-006–007; TC-004A, TC-006–007 |
| Dispatch gates | canonical resolve, pause, terminal, unresolved workflow, unresolved placeholder | spec.md REQ-F-005; TC-008; X-01 |
| Worker outcomes | success, non-zero exit, unavailable provider, cancellation, panic, persistence failure | spec.md REQ-F-008; TC-005–006 |
| Resume and attempts | completed item, stale running item, explicit retry, conflicting terminal result | spec.md REQ-F-009; TC-009 |
| Capacity and resources | sequential, bounded parallel, unknown ownership, overlap, unavailable capability | spec.md REQ-F-003, REQ-NFR-006; TC-001, TC-011 |
| Evidence and events | bounded contextual diagnostics, sensitive markers, allow-listed event fields, complete mixed result | spec.md REQ-F-010–011, REQ-NFR-005; TC-010, TC-012 |

The matrix is finite: each named partition must have an explicit pass/fail
assertion, and no test may broaden an acceptance criterion into an undefined
robustness claim.
