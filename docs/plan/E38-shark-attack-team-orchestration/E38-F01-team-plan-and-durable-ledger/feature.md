---
feature_key: E38-F01-team-plan-and-durable-ledger
epic_key: E38
title: Team Plan and Durable Ledger
description: Define a read-only, dependency-aware team plan and persist the durable run and item ledger needed for safe start and resume.
---

# Team Plan and Durable Ledger

This feature gives an operator a deterministic preview of an epic or feature's eligible children, dependency waves, workflow-resolved worker metadata, capability exclusions, execution mode, and stable plan hash, then persists the confirmed plan and per-child lifecycle records before any worker is dispatched. It is needed so team execution has an auditable source of truth across interruption and re-entry, and integrates with Shark child discovery, dependency data, workflow prompt metadata, claims, and the scheduler. Dependencies: none; execution order: 1; size: 5 (L). Produces: I-01 to E38-F02, E38-F03, and E38-F05, using the Team-run domain contract in architecture §4.2. X-## ownership: none.

