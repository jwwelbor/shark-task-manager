---
feature_key: E38-F02-scheduler-and-claims
epic_key: E38
title: Scheduler and Claims
description: Execute planned children in dependency-safe bounded waves while preserving canonical dispatch, claim, lease, and resource-safety contracts.
status: superseded
superseded_by: E38-F06, E38-F07
---

# Scheduler and Claims

> Superseded on 2026-07-13. E38 does not add a scheduler or team-run runtime.
> Existing Shark claim/lease behavior is used by the Rider procedure and the
> role-aware pull feature.

This feature executes a confirmed team plan by selecting eligible dependency waves, enforcing bounded concurrency or explicit sequential fallback, detecting unsafe resource overlap, claiming children atomically, dispatching the canonical Shark-rendered prompt, recording semantic process results, and releasing child sessions safely. It is needed to gain parallel efficiency without duplicate work, false dependent completion, or worker-owned workflow transitions, and integrates with the F01 ledger, F04 role/communication contract, runner/dispatch seam, claims and leases, dependency checks, and aggregate routing. Dependencies: E38-F01 and E38-F04; execution order: 3; size: 5 (L). Consumes: I-01 from E38-F01 and I-04 from E38-F04; Produces: I-02 to E38-F03 and E38-F05, and I-05 to E38-F05, using architecture §§4.2, 4.3, and 4.6. Produces: X-01 for E22's dispatcher, process-result, worktree-capability, claim, and rendered-prompt contract.
