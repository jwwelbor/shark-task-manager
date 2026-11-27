---
feature_key: E41-F04-operate-and-benchmark-reusable-routing-profiles
epic_key: E41
title: Operate and benchmark reusable routing profiles
description: Phase 4. Add routing validation, migration tooling and refreshable local harness inventory. Adopt frozen routing profiles and compatible evidence in Rider and E40.
---

# Operate and benchmark reusable routing profiles

Add routing validation, migration tooling and the local inventory lifecycle in [REQ-F-08](../requirements.md). Adopt frozen routing profiles and compatible evidence in Rider and E40.

Rider owns the final inline attempt when eligible external routes are exhausted. See [the inline fallback contract](../02-architecture.md#rider-inline-fallback).

Depends on E41-F02.

This feature captures the scope for later decomposition.

## Acceptance

Verify profile-only comparisons, reproducible evidence, and rollback against R11–R13. Full strict-policy qualification also depends on E41-F03.

- [Phase 4 scope and dependencies](../08-implementation-phases.md)
- [Acceptance matrix](../09-test-plan.md#acceptance-matrix)
- [Epic charter](../epic.md)
