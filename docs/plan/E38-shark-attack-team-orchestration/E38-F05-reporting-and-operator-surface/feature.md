---
feature_key: E38-F05-reporting-and-operator-surface
epic_key: E38
title: Reporting and Operator Surface
description: Expose preview, start, resume, and summary operations with complete human and machine-readable team-run diagnostics.
---

# Reporting and Operator Surface

This feature provides the operator-facing attack plan, start, resume, and summary surfaces in human and JSON forms, reporting root and run identity, mode, concurrency, counts, per-child outcome or skip reason, evidence pointers, timing, and next action without exposing prompts or secrets. It is needed to make team execution actionable and diagnosable across success, pause, failure, cancellation, and fallback, and integrates with the F01 plan/ledger, F02 item and operator contracts, F03 aggregate routing, F04 council handoffs, and existing structured telemetry. Dependencies: E38-F01 through E38-F04; execution order: 5; size: 3 (M). Consumes: I-01 from E38-F01, I-02 from E38-F02, I-03 from E38-F03, and I-04 from E38-F04; Consumes: I-05 from E38-F02, using architecture §§4.2, 4.4, 4.5, and 4.6. Produces: X-04 for E23 run/root/child/wave/claim/session/duration/outcome telemetry.
