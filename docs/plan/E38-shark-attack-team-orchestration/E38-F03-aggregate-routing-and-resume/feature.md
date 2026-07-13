---
feature_key: E38-F03-aggregate-routing-and-resume
epic_key: E38
title: Aggregate Routing and Resume
description: Convert complete child outcomes into configured root workflow routing and make interrupted team runs safely resumable.
---

# Aggregate Routing and Resume

This feature aggregates success, failure, blocked, paused, cancelled, partial, skipped, and provider outcomes across planned children, routes the root through its configured semantic workflow outcome, and reconciles persisted item state and claims so resume dispatches only unfinished eligible work. It is needed to preserve workflow integrity and make partial execution diagnosable rather than silently successful, and integrates with the F01 ledger, F02 scheduler results, F04 role/communication contract, root workflow transitions, claim/session reconciliation, and operator reporting. Dependencies: E38-F01, E38-F02, and E38-F04; execution order: 4; size: 5 (L). Consumes: I-01 from E38-F01, I-02 from E38-F02, and I-04 from E38-F04; Produces: I-03 to E38-F05, using architecture §4.4. Produces: X-02 for E16/E35 configured workflow roles, boundaries, semantic outcomes, and root routing.
